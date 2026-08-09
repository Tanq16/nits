package videohandlers

import (
	"bufio"
	"context"
	"fmt"
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

type SmartEncodeOptions struct {
	Quality      string
	FPSDowngrade bool
	NoVideo      bool
}

var qualityCRF = map[string]string{
	"very-high": "22",
	"high":      "24",
	"medium":    "26",
	"low":       "28",
}

var bitmapSubCodecs = map[string]bool{
	"hdmv_pgs_subtitle": true,
	"vobsub":            true,
	"dvd_subtitle":      true,
}

var commentaryRegex = regexp.MustCompile(`(?i)commentary|director|cast`)

type indexedStream struct {
	relIdx int
	stream Stream
}

func RunSmartEncode(ctx context.Context, inputFile string, opts SmartEncodeOptions, cb EncodeCallbacks) error {
	data, err := GetVideoInfo(inputFile)
	if err != nil {
		return err
	}

	args, outputFile, err := buildFFmpegArgs(inputFile, data, opts, cb)
	if err != nil {
		return err
	}

	cb.info(fmt.Sprintf("Command: ffmpeg %s", strings.Join(args, " ")))

	return runEncode(ctx, outputFile, data, args, cb)
}

func buildFFmpegArgs(inputFile string, data *FFProbeOutput, opts SmartEncodeOptions, cb EncodeCallbacks) ([]string, string, error) {
	args := []string{"-i", inputFile}

	videoStreams := filterStreams(data.Streams, "video")
	if len(videoStreams) == 0 {
		return nil, "", fmt.Errorf("no video streams found in input")
	}

	args = append(args, "-map", "0:v:0")

	crf, ok := qualityCRF[opts.Quality]
	if !ok {
		crf = qualityCRF["medium"]
	}

	var videoFlags []string

	if opts.NoVideo {
		videoFlags = []string{"-c:v", "copy"}
		cb.info("Video: copy (no re-encoding)")
	} else {
		videoFlags = []string{"-c:v", "libx265", "-crf", crf, "-fps_mode", "cfr"}

		if videoStreams[0].stream.PixFmt == "yuv420p10le" {
			videoFlags = append(videoFlags, "-pix_fmt", "yuv420p10le")
			cb.info("10-bit source detected, retaining pixel format")
		}

		if opts.FPSDowngrade {
			videoFlags = append(videoFlags, "-r", "30")
			cb.info("FPS downgrade to 30 enabled")
		}

		cb.info(fmt.Sprintf("Video: libx265 CRF %s (%s quality, CFR)", crf, opts.Quality))
	}

	var audioFlags []string
	audioStreams := filterStreams(data.Streams, "audio")

	if len(audioStreams) > 0 {
		selectedIdx := selectAudioStream(audioStreams)
		args = append(args, "-map", fmt.Sprintf("0:a:%d", selectedIdx))

		selected := audioStreams[selectedIdx]
		// Always re-encode audio to generate fresh timestamps from the encoder.
		// Copying audio preserves source timestamp imprecisions that cause A/V
		// drift in browsers (which lack real-time clock correction unlike VLC).
		audioFlags = append(audioFlags, "-c:a", "aac", "-ac", "2", "-ar", "48000")

		lang := selected.stream.Tags.Language
		if lang == "" {
			lang = "und"
		}
		if selected.stream.Tags.Title != "" {
			cb.info(fmt.Sprintf("Audio: stream #%d (%s — %s) → AAC stereo 48kHz", selected.stream.Index, lang, selected.stream.Tags.Title))
		} else {
			cb.info(fmt.Sprintf("Audio: stream #%d (%s) → AAC stereo 48kHz", selected.stream.Index, lang))
		}
	} else {
		cb.info("Audio: none")
	}

	var subtitleFlags []string
	subStreams := filterStreams(data.Streams, "subtitle")
	outputExt := ".mp4"

	if len(subStreams) > 0 {
		hasBitmap := false
		for _, ss := range subStreams {
			if bitmapSubCodecs[ss.stream.CodecName] {
				hasBitmap = true
				break
			}
		}

		for i := range subStreams {
			args = append(args, "-map", fmt.Sprintf("0:s:%d", i))
		}

		if hasBitmap {
			outputExt = ".mkv"
			subtitleFlags = append(subtitleFlags, "-c:s", "copy")
			cb.info(fmt.Sprintf("Subtitles: %d stream(s) (bitmap detected → MKV, copy)", len(subStreams)))
		} else {
			subtitleFlags = append(subtitleFlags, "-c:s", "mov_text")
			cb.info(fmt.Sprintf("Subtitles: %d stream(s) (text → MP4, mov_text)", len(subStreams)))
		}
	} else {
		cb.info("Subtitles: none")
	}

	dir := filepath.Dir(inputFile)
	base := strings.TrimSuffix(filepath.Base(inputFile), filepath.Ext(inputFile))
	outputFile := filepath.Join(dir, base+".h265"+outputExt)

	args = append(args, videoFlags...)
	args = append(args, audioFlags...)
	args = append(args, subtitleFlags...)
	args = append(args, "-avoid_negative_ts", "make_zero")
	args = append(args, outputFile)

	cb.info(fmt.Sprintf("Output: %s", outputFile))

	return args, outputFile, nil
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

	startTime := time.Now()
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
	cb.info(fmt.Sprintf("Encoding: %s | Duration: %s", label, FormatDuration(totalDurationSecs)))

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

	cb.success(fmt.Sprintf("Encoding completed in %s", time.Since(startTime).Round(time.Second)))
	return nil
}

func isErrorLine(line string) bool {
	line = strings.ToLower(line)
	if strings.Contains(line, "[info]") || strings.Contains(line, "[warning]") {
		return false
	}
	if strings.Contains(line, "error") || strings.Contains(line, "failed") || strings.Contains(line, "cannot") {
		return true
	}
	return false
}

