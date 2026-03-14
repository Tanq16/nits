package videohandlers

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type SmartEncodeOptions struct {
	Quality      string
	FPSDowngrade bool
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

func RunSmartEncode(inputFile string, opts SmartEncodeOptions) error {
	data, err := getVideoInfo(inputFile)
	if err != nil {
		return err
	}

	args, outputFile, err := buildFFmpegArgs(inputFile, data, opts)
	if err != nil {
		return err
	}

	fmt.Printf("Command: ffmpeg %s\n\n", strings.Join(args, " "))

	return runEncode(outputFile, data, args)
}

func buildFFmpegArgs(inputFile string, data *FFProbeOutput, opts SmartEncodeOptions) ([]string, string, error) {
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

	videoFlags := []string{"-c:v", "libx265", "-crf", crf}

	if videoStreams[0].stream.PixFmt == "yuv420p10le" {
		videoFlags = append(videoFlags, "-pix_fmt", "yuv420p10le")
		fmt.Println("→ 10-bit source detected, retaining pixel format")
	}

	if opts.FPSDowngrade {
		videoFlags = append(videoFlags, "-r", "30")
		fmt.Println("→ FPS downgrade to 30 enabled")
	}

	fmt.Printf("→ Video: libx265 CRF %s (%s quality)\n", crf, opts.Quality)

	var audioFlags []string
	audioStreams := filterStreams(data.Streams, "audio")

	if len(audioStreams) > 0 {
		selectedIdx := selectAudioStream(audioStreams)
		args = append(args, "-map", fmt.Sprintf("0:a:%d", selectedIdx))

		selected := audioStreams[selectedIdx]
		alreadyAACStereo := selected.stream.CodecName == "aac" && selected.stream.Channels == 2

		if alreadyAACStereo {
			audioFlags = append(audioFlags, "-c:a", "copy")
		} else {
			audioFlags = append(audioFlags, "-c:a", "aac", "-ac", "2")
		}

		lang := selected.stream.Tags.Language
		if lang == "" {
			lang = "und"
		}
		action := "→ AAC stereo"
		if alreadyAACStereo {
			action = "→ copy (already AAC stereo)"
		}
		if selected.stream.Tags.Title != "" {
			fmt.Printf("→ Audio: stream #%d (%s — %s) %s\n", selected.stream.Index, lang, selected.stream.Tags.Title, action)
		} else {
			fmt.Printf("→ Audio: stream #%d (%s) %s\n", selected.stream.Index, lang, action)
		}
	} else {
		fmt.Println("→ Audio: none")
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
			fmt.Printf("→ Subtitles: %d stream(s) (bitmap detected → MKV, copy)\n", len(subStreams))
		} else {
			subtitleFlags = append(subtitleFlags, "-c:s", "mov_text")
			fmt.Printf("→ Subtitles: %d stream(s) (text → MP4, mov_text)\n", len(subStreams))
		}
	} else {
		fmt.Println("→ Subtitles: none")
	}

	dir := filepath.Dir(inputFile)
	base := strings.TrimSuffix(filepath.Base(inputFile), filepath.Ext(inputFile))
	outputFile := filepath.Join(dir, base+".h265"+outputExt)

	args = append(args, videoFlags...)
	args = append(args, audioFlags...)
	args = append(args, subtitleFlags...)
	args = append(args, outputFile)

	fmt.Printf("→ Output: %s\n", outputFile)

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

func runEncode(outputFile string, data *FFProbeOutput, ffmpegArgs []string) error {
	totalDurationSecs := 0.0
	if data.Format.Duration != "" {
		totalDurationSecs, _ = strconv.ParseFloat(data.Format.Duration, 64)
	}

	ffmpegArgs = append(ffmpegArgs, "-progress", "pipe:1", "-nostats", "-loglevel", "error", "-y")

	cmd := exec.Command("ffmpeg", ffmpegArgs...)

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
				fmt.Fprintf(os.Stderr, "\r\033[K%s\n", line)
			}
		}
		errorChan <- hasErrors
	}()

	fmt.Printf("Encoding: %s | Duration: %s\n", outputFile, formatDuration(totalDurationSecs))
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
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
				drawProgressBar(percent, currentSecs, totalDurationSecs)
			} else {
				fmt.Printf("\rEncoding... %.1fs", currentSecs)
			}
		}
	}

	cmdErr := cmd.Wait()
	errorsDetected := <-errorChan

	if cmdErr != nil || errorsDetected {
		fmt.Println()
		if cmdErr != nil {
			return fmt.Errorf("ffmpeg encoding failed: %w", cmdErr)
		}
		return fmt.Errorf("encoding completed with errors (see messages above)")
	}

	fmt.Printf("\r\033[KEncoding completed in %s\n\n", time.Since(startTime))
	return nil
}

func drawProgressBar(percent float64, current, total float64) {
	width := 40
	completed := min(int((percent/100)*float64(width)), width)

	filled := strings.Repeat("━", completed)
	empty := strings.Repeat(" ", width-completed)

	fmt.Printf("\r[%s%s] %.1f%% (%.1fs / %.1fs)", filled, empty, percent, current, total)
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
