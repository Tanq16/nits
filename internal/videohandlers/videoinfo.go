package videohandlers

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type FFProbeOutput struct {
	Streams []Stream `json:"streams"`
	Format  Format   `json:"format"`
}

type Stream struct {
	Index         int    `json:"index"`
	CodecType     string `json:"codec_type"`
	CodecName     string `json:"codec_name"`
	Width         int    `json:"width,omitempty"`
	Height        int    `json:"height,omitempty"`
	BitRate       string `json:"bit_rate,omitempty"`
	AvgFrameRate  string `json:"avg_frame_rate,omitempty"`
	RFrameRate    string `json:"r_frame_rate,omitempty"`
	PixFmt        string `json:"pix_fmt,omitempty"`
	Channels      int    `json:"channels,omitempty"`
	ChannelLayout string `json:"channel_layout,omitempty"`
	SampleRate    string `json:"sample_rate,omitempty"`
	Tags          Tags   `json:"tags,omitempty"`
}

type Format struct {
	Filename   string `json:"filename"`
	Duration   string `json:"duration"`
	Size       string `json:"size"`
	BitRate    string `json:"bit_rate"`
	FormatName string `json:"format_name"`
}

type Tags struct {
	Language string `json:"language,omitempty"`
	Title    string `json:"title,omitempty"`
	BPS      string `json:"BPS,omitempty"`
}

func RunVideoInfo(inputFile string) error {
	cmd := exec.Command("ffprobe",
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		inputFile,
	)

	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to run ffprobe: %w", err)
	}

	var data FFProbeOutput
	if err := json.Unmarshal(output, &data); err != nil {
		return fmt.Errorf("failed to parse ffprobe output: %w", err)
	}

	printOverview(data.Format)
	printStreams(data.Streams)
	return nil
}

func printOverview(f Format) {
	sizeBytes, _ := strconv.ParseFloat(f.Size, 64)
	durationSec, _ := strconv.ParseFloat(f.Duration, 64)
	bitrate, _ := strconv.ParseFloat(f.BitRate, 64)
	fmt.Printf(" Container: %s  |  Size: %s  |  Duration: %s  |  Bitrate: %s\n",
		f.FormatName, formatSize(sizeBytes), formatDuration(durationSec), formatBitrate(bitrate))
	fmt.Println("")
}

func printStreams(streams []Stream) {
	fmt.Println("STREAMS:")
	for _, s := range streams {
		switch s.CodecType {
		case "video":
			fps := parseFrameRate(s.AvgFrameRate)
			bitrate := ""
			if s.Tags.BPS != "" {
				br, _ := strconv.ParseFloat(s.Tags.BPS, 64)
				bitrate = fmt.Sprintf(" | Bitrate: %s", formatBitrate(br))
			}
			fmt.Printf(" [VIDEO #%d] %s\n", s.Index, strings.ToUpper(s.CodecName))
			fmt.Printf("   %dx%d | %s fps | %s%s\n", s.Width, s.Height, fps, s.PixFmt, bitrate)

		case "audio":
			lang := s.Tags.Language
			if lang == "" {
				lang = "und"
			}
			bitrate := ""
			if s.Tags.BPS != "" {
				br, _ := strconv.ParseFloat(s.Tags.BPS, 64)
				bitrate = fmt.Sprintf(" | %s", formatBitrate(br))
			}
			fmt.Printf(" [AUDIO #%d] %s | %s\n", s.Index, strings.ToUpper(s.CodecName), strings.ToUpper(lang))
			fmt.Printf("   %d ch (%s) | %s Hz%s\n", s.Channels, s.ChannelLayout, s.SampleRate, bitrate)
			if s.Tags.Title != "" {
				fmt.Printf("   %s\n", s.Tags.Title)
			}

		case "subtitle":
			lang := s.Tags.Language
			if lang == "" {
				lang = "und"
			}
			fmt.Printf(" [SUB #%d] %s | %s", s.Index, strings.ToUpper(s.CodecName), strings.ToUpper(lang))
			if s.Tags.Title != "" {
				fmt.Printf(" | %s", s.Tags.Title)
			}
			fmt.Println()
		}
		fmt.Println()
	}
}

func formatSize(bytes float64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%.0f B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", bytes/float64(div), "KMGTPE"[exp])
}

func formatBitrate(bps float64) string {
	return fmt.Sprintf("%.2f Mbps", bps/1000000)
}

func formatDuration(seconds float64) string {
	d := time.Duration(seconds) * time.Second
	return d.String()
}

func parseFrameRate(fr string) string {
	parts := strings.Split(fr, "/")
	if len(parts) == 2 {
		num, _ := strconv.ParseFloat(parts[0], 64)
		den, _ := strconv.ParseFloat(parts[1], 64)
		if den > 0 {
			return fmt.Sprintf("%.2f", num/den)
		}
	}
	return fr
}

func RunVideoEncode(inputFile, outputFile, params string) error {
	data, err := getVideoInfo(inputFile)
	if err != nil {
		return err
	}

	totalDurationSecs := 0.0
	if data.Format.Duration != "" {
		totalDurationSecs, _ = strconv.ParseFloat(data.Format.Duration, 64)
	}

	paramArgs := []string{}
	if params != "" {
		paramArgs = strings.Fields(params)
	}

	args := []string{"-i", inputFile}
	args = append(args, paramArgs...)
	args = append(args, outputFile, "-progress", "pipe:1", "-nostats", "-loglevel", "error", "-y")

	cmd := exec.Command("ffmpeg", args...)

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

	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			if line != "" {
				fmt.Fprintf(os.Stderr, "\r\033[K%s\n", line)
			}
		}
	}()

	fmt.Printf("Encoding: %s -> %s\n", inputFile, outputFile)
	if totalDurationSecs > 0 {
		fmt.Printf("Duration: %s\n\n", formatDuration(totalDurationSecs))
	}

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

	if err := cmd.Wait(); err != nil {
		fmt.Println()
		return fmt.Errorf("ffmpeg encoding failed: %w", err)
	}

	fmt.Printf("\r\033[K✓ Encoding completed successfully\n")
	return nil
}

func drawProgressBar(percent float64, current, total float64) {
	width := 40
	completed := int((percent / 100) * float64(width))
	if completed > width {
		completed = width
	}

	filled := strings.Repeat("━", completed)
	empty := strings.Repeat(" ", width-completed)

	fmt.Printf("\r[%s%s] %.1f%% (%.1fs / %.1fs)", filled, empty, percent, current, total)
}

func getVideoInfo(inputFile string) (*FFProbeOutput, error) {
	cmd := exec.Command("ffprobe",
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		inputFile,
	)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run ffprobe: %w", err)
	}

	var data FFProbeOutput
	if err := json.Unmarshal(output, &data); err != nil {
		return nil, fmt.Errorf("failed to parse ffprobe output: %w", err)
	}

	return &data, nil
}
