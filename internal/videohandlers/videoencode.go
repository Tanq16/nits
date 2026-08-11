package videohandlers

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

type EncodeCallbacks struct {
	OnInfo         func(msg string)
	OnProgress     func(label string, percent int)
	OnProgressDone func()
	OnError        func(msg string)
	OnSuccess      func(msg string)
}

func (cb EncodeCallbacks) info(msg string) {
	if cb.OnInfo != nil {
		cb.OnInfo(msg)
	}
}

func (cb EncodeCallbacks) progress(label string, percent int) {
	if cb.OnProgress != nil {
		cb.OnProgress(label, percent)
	}
}

func (cb EncodeCallbacks) progressDone() {
	if cb.OnProgressDone != nil {
		cb.OnProgressDone()
	}
}

func (cb EncodeCallbacks) errorMsg(msg string) {
	if cb.OnError != nil {
		cb.OnError(msg)
	}
}

func (cb EncodeCallbacks) success(msg string) {
	if cb.OnSuccess != nil {
		cb.OnSuccess(msg)
	}
}

type OptimizeOptions struct {
	Codec     string
	CRF       int
	MaxRes    string
	AudioMode string
	Preset    string
	ToneMap   string
}

func DefaultOptimizeOptions() OptimizeOptions {
	return OptimizeOptions{
		Codec:     "hevc",
		CRF:       30,
		MaxRes:    "1080p",
		AudioMode: "128k",
		Preset:    "medium",
		ToneMap:   "auto",
	}
}

type OptimizeResult struct {
	InputFile   string
	OutputFile  string
	InputBytes  int64
	OutputBytes int64
	DurationSec float64
	OrigWidth   int
	OrigHeight  int
	TargetRes   string
	Scaled      bool
	ToneMapped  bool
	Codec       string
	CRF         int
	Preset      string
	TimeTaken   time.Duration
}

type indexedStream struct {
	relIdx int
	stream Stream
}

var commentaryRegex = regexp.MustCompile(`(?i)commentary|director|cast`)

func RunVideoOptimize(ctx context.Context, inputFile string, opts OptimizeOptions, cb EncodeCallbacks) (*OptimizeResult, error) {
	inputStat, err := os.Stat(inputFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read input file: %w", err)
	}

	data, err := GetVideoInfo(inputFile)
	if err != nil {
		return nil, err
	}

	args, outputFile, res, err := buildFFmpegArgs(inputFile, data, opts, cb)
	if err != nil {
		return nil, err
	}

	cb.info(fmt.Sprintf("Command: ffmpeg %s", strings.Join(args, " ")))

	startTime := time.Now()
	if err := runEncode(ctx, outputFile, data, args, cb); err != nil {
		return nil, err
	}

	outputStat, err := os.Stat(outputFile)
	var outputBytes int64
	if err == nil {
		outputBytes = outputStat.Size()
	}

	durationSec := 0.0
	if data.Format.Duration != "" {
		durationSec, _ = strconv.ParseFloat(data.Format.Duration, 64)
	}

	res.InputBytes = inputStat.Size()
	res.OutputBytes = outputBytes
	res.DurationSec = durationSec
	res.TimeTaken = time.Since(startTime)

	return res, nil
}

func buildFFmpegArgs(inputFile string, data *FFProbeOutput, opts OptimizeOptions, cb EncodeCallbacks) ([]string, string, *OptimizeResult, error) {
	codec := strings.ToLower(opts.Codec)
	if codec != "av1" {
		codec = "hevc"
	}
	opts.Codec = codec

	if opts.CRF <= 0 {
		if codec == "av1" {
			opts.CRF = 32
		} else {
			opts.CRF = 30
		}
	}
	if opts.MaxRes == "" {
		opts.MaxRes = "1080p"
	}
	if opts.AudioMode == "" {
		opts.AudioMode = "128k"
	}
	if opts.Preset == "" {
		if codec == "av1" {
			opts.Preset = "6"
		} else {
			opts.Preset = "medium"
		}
	} else if codec == "av1" {
		switch strings.ToLower(opts.Preset) {
		case "medium":
			opts.Preset = "6"
		case "slow":
			opts.Preset = "4"
		case "fast":
			opts.Preset = "8"
		}
	}
	if opts.ToneMap == "" {
		opts.ToneMap = "auto"
	}

	args := []string{"-i", inputFile}

	videoStreams := filterStreams(data.Streams, "video")
	if len(videoStreams) == 0 {
		return nil, "", nil, fmt.Errorf("no video streams found in input")
	}

	primaryVideo := videoStreams[0].stream
	origWidth := primaryVideo.Width
	origHeight := primaryVideo.Height

	args = append(args, "-map", "0:v:0")

	var filterChain []string
	scaled := false
	maxW, maxH := 0, 0

	switch opts.MaxRes {
	case "720p":
		maxW, maxH = 1280, 720
	case "480p":
		maxW, maxH = 854, 480
	case "none":
		maxW, maxH = 0, 0
	default:
		maxW, maxH = 1920, 1080
	}

	if maxW > 0 && (origWidth > maxW || origHeight > maxH) {
		filterChain = append(filterChain, fmt.Sprintf("scale='min(%d,iw)':'min(%d,ih)':force_original_aspect_ratio=decrease,scale=trunc(iw/2)*2:trunc(ih/2)*2", maxW, maxH))
		scaled = true
		cb.info(fmt.Sprintf("Resolution: %dx%d downscaled to fit %s max", origWidth, origHeight, opts.MaxRes))
	} else {
		cb.info(fmt.Sprintf("Resolution: %dx%d (retained)", origWidth, origHeight))
	}

	isHDR := IsHDRStream(primaryVideo)
	toneMapped := false
	if opts.ToneMap == "yes" || (opts.ToneMap == "auto" && isHDR) {
		filterChain = append(filterChain, "format=gbrpf32le", "tonemap=hable:desat=0.5", "format=yuv420p")
		toneMapped = true
		cb.info("HDR detected: applying Hable tone-mapping to standard 8-bit SDR")
	}

	var videoFlags []string
	if len(filterChain) > 0 {
		videoFlags = append(videoFlags, "-vf", strings.Join(filterChain, ","))
	}

	if codec == "av1" {
		videoFlags = append(videoFlags, "-c:v", "libsvtav1", "-crf", strconv.Itoa(opts.CRF), "-preset", opts.Preset, "-svtav1-params", "tune=0", "-pix_fmt", "yuv420p", "-fps_mode", "cfr")
		cb.info(fmt.Sprintf("Video: AV1 (libsvtav1) CRF %d (preset %s, 8-bit yuv420p, CFR)", opts.CRF, opts.Preset))
	} else {
		videoFlags = append(videoFlags, "-c:v", "libx265", "-crf", strconv.Itoa(opts.CRF), "-preset", opts.Preset, "-pix_fmt", "yuv420p", "-fps_mode", "cfr")
		cb.info(fmt.Sprintf("Video: H.265 (libx265) CRF %d (preset %s, 8-bit yuv420p, CFR)", opts.CRF, opts.Preset))
	}

	var audioFlags []string
	audioStreams := filterStreams(data.Streams, "audio")

	if opts.AudioMode == "none" || len(audioStreams) == 0 {
		audioFlags = append(audioFlags, "-an")
		cb.info("Audio: none")
	} else {
		selectedIdx := selectAudioStream(audioStreams)
		args = append(args, "-map", fmt.Sprintf("0:a:%d", selectedIdx))
		audioFlags = append(audioFlags, "-c:a", "aac", "-b:a", opts.AudioMode, "-ac", "2", "-ar", "48000")

		selected := audioStreams[selectedIdx]
		lang := selected.stream.Tags.Language
		if lang == "" {
			lang = "und"
		}
		if selected.stream.Tags.Title != "" {
			cb.info(fmt.Sprintf("Audio: stream #%d (%s — %s) → AAC stereo %s 48kHz", selected.stream.Index, lang, selected.stream.Tags.Title, opts.AudioMode))
		} else {
			cb.info(fmt.Sprintf("Audio: stream #%d (%s) → AAC stereo %s 48kHz", selected.stream.Index, lang, opts.AudioMode))
		}
	}

	var subtitleFlags []string
	subStreams := filterStreams(data.Streams, "subtitle")

	if len(subStreams) > 0 {
		for i := range subStreams {
			args = append(args, "-map", fmt.Sprintf("0:s:%d", i))
		}
		subtitleFlags = append(subtitleFlags, "-c:s", "mov_text")
		cb.info(fmt.Sprintf("Subtitles: %d stream(s) → mov_text", len(subStreams)))
	}

	dir := filepath.Dir(inputFile)
	base := strings.TrimSuffix(filepath.Base(inputFile), filepath.Ext(inputFile))
	outputFile := filepath.Join(dir, base+".optimized.mp4")

	if outputFile == inputFile || strings.HasSuffix(base, ".optimized") {
		cleanBase := strings.TrimSuffix(base, ".optimized")
		outputFile = filepath.Join(dir, cleanBase+".optimized.1.mp4")
	}

	args = append(args, videoFlags...)
	args = append(args, audioFlags...)
	args = append(args, subtitleFlags...)
	args = append(args, "-avoid_negative_ts", "make_zero", "-movflags", "+faststart", outputFile)

	targetRes := opts.MaxRes
	if !scaled {
		targetRes = fmt.Sprintf("%dx%d", origWidth, origHeight)
	}

	return args, outputFile, &OptimizeResult{
		InputFile:  inputFile,
		OutputFile: outputFile,
		OrigWidth:  origWidth,
		OrigHeight: origHeight,
		TargetRes:  targetRes,
		Scaled:     scaled,
		ToneMapped: toneMapped,
		Codec:      codec,
		CRF:        opts.CRF,
		Preset:     opts.Preset,
	}, nil
}

func filterStreams(streams []Stream, codecType string) []indexedStream {
	var result []indexedStream
	for _, s := range streams {
		if s.CodecType == codecType {
			result = append(result, indexedStream{relIdx: len(result), stream: s})
		}
	}
	return result
}

func selectAudioStream(audioStreams []indexedStream) int {
	if len(audioStreams) == 1 {
		return 0
	}

	for i, as := range audioStreams {
		if isRejectedAudio(as.stream) {
			continue
		}
		lang := as.stream.Tags.Language
		if lang == "eng" || lang == "" {
			return i
		}
	}

	for i, as := range audioStreams {
		if isRejectedAudio(as.stream) {
			continue
		}
		return i
	}

	return 0
}

func isRejectedAudio(s Stream) bool {
	if commentaryRegex.MatchString(s.Tags.Title) {
		return true
	}
	if s.Disposition.Comment == 1 || s.Disposition.VisualImpaired == 1 {
		return true
	}
	return false
}

func runEncode(ctx context.Context, outputFile string, data *FFProbeOutput, ffmpegArgs []string, cb EncodeCallbacks) error {
	totalDurationSecs := 0.0
	if data.Format.Duration != "" {
		totalDurationSecs, _ = strconv.ParseFloat(data.Format.Duration, 64)
	}

	ffmpegArgs = append(ffmpegArgs, "-progress", "pipe:1", "-nostats", "-loglevel", "error", "-y")

	cmd := exec.CommandContext(ctx, "ffmpeg", ffmpegArgs...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start ffmpeg: %w", err)
	}

	errorChan := make(chan bool, 1)
	go func() {
		hasErrors := false
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			if line != "" && isErrorLine(line) {
				hasErrors = true
				cb.errorMsg(line)
			}
		}
		if err := scanner.Err(); err != nil {
			hasErrors = true
			cb.errorMsg(fmt.Sprintf("stderr stream error: %v", err))
		}
		errorChan <- hasErrors
	}()

	label := filepath.Base(outputFile)
	var currentPercent atomic.Int64
	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				cb.progress(label, int(currentPercent.Load()))
			}
		}
	}()

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		if ctx.Err() != nil {
			break
		}
		line := scanner.Text()
		parts := strings.Split(line, "=")

		if len(parts) == 2 && parts[0] == "out_time_us" {
			currentUs, _ := strconv.ParseFloat(parts[1], 64)
			currentSecs := currentUs / 1000000.0

			if totalDurationSecs > 0 {
				percent := (currentSecs / totalDurationSecs) * 100
				if percent > 100 {
					percent = 100
				}
				currentPercent.Store(int64(percent))
			}
		}
	}
	if err := scanner.Err(); err != nil && ctx.Err() == nil {
		cb.errorMsg(fmt.Sprintf("progress stream error: %v", err))
	}

	close(done)
	cb.progressDone()

	cmdErr := cmd.Wait()
	errorsDetected := <-errorChan

	if ctx.Err() != nil {
		return ctx.Err()
	}

	if cmdErr != nil || errorsDetected {
		if cmdErr != nil {
			return fmt.Errorf("ffmpeg encoding failed: %w", cmdErr)
		}
		return fmt.Errorf("encoding completed with errors (see messages above)")
	}

	return nil
}

func isErrorLine(line string) bool {
	line = strings.ToLower(line)
	if strings.Contains(line, "[info]") || strings.Contains(line, "[warning]") || strings.HasPrefix(line, "svt[info]") {
		return false
	}
	if strings.Contains(line, "error") || strings.Contains(line, "failed") || strings.Contains(line, "cannot") {
		return true
	}
	return false
}
