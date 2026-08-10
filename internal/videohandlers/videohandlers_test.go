package videohandlers

import (
	"slices"
	"strings"
	"testing"
)

func TestBuildFFmpegArgs_ResolutionScaling(t *testing.T) {
	tests := []struct {
		name       string
		width      int
		height     int
		pixFmt     string
		wantScaled bool
		wantFilter bool
		wantPixFmt string
	}{
		{
			name:       "4K video downscaled",
			width:      3840,
			height:     2160,
			pixFmt:     "yuv420p",
			wantScaled: true,
			wantFilter: true,
			wantPixFmt: "yuv420p",
		},
		{
			name:       "1440p video downscaled",
			width:      2560,
			height:     1440,
			pixFmt:     "yuv420p",
			wantScaled: true,
			wantFilter: true,
			wantPixFmt: "yuv420p",
		},
		{
			name:       "Vertical 1080x1920 phone video downscaled",
			width:      1080,
			height:     1920,
			pixFmt:     "yuv420p",
			wantScaled: true,
			wantFilter: true,
			wantPixFmt: "yuv420p",
		},
		{
			name:       "1080p exact kept unchanged",
			width:      1920,
			height:     1080,
			pixFmt:     "yuv420p",
			wantScaled: false,
			wantFilter: false,
			wantPixFmt: "yuv420p",
		},
		{
			name:       "720p kept unchanged",
			width:      1280,
			height:     720,
			pixFmt:     "yuv420p",
			wantScaled: false,
			wantFilter: false,
			wantPixFmt: "yuv420p",
		},
		{
			name:       "480p kept unchanged",
			width:      640,
			height:     480,
			pixFmt:     "yuv420p",
			wantScaled: false,
			wantFilter: false,
			wantPixFmt: "yuv420p",
		},
		{
			name:       "10-bit color preserved",
			width:      1920,
			height:     1080,
			pixFmt:     "yuv420p10le",
			wantScaled: false,
			wantFilter: false,
			wantPixFmt: "yuv420p10le",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			probe := &FFProbeOutput{
				Streams: []Stream{
					{
						Index:     0,
						CodecType: "video",
						CodecName: "h264",
						Width:     tt.width,
						Height:    tt.height,
						PixFmt:    tt.pixFmt,
					},
					{
						Index:     1,
						CodecType: "audio",
						CodecName: "aac",
						Channels:  2,
					},
				},
				Format: Format{
					Filename: "test.mkv",
					Duration: "120.0",
				},
			}

			args, outputFile, origW, origH, scaled, err := buildFFmpegArgs("/tmp/test.mkv", probe, EncodeCallbacks{})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if origW != tt.width || origH != tt.height {
				t.Errorf("got dimensions %dx%d, want %dx%d", origW, origH, tt.width, tt.height)
			}
			if scaled != tt.wantScaled {
				t.Errorf("got scaled %v, want %v", scaled, tt.wantScaled)
			}

			hasVf := slices.Contains(args, "-vf")
			if hasVf != tt.wantFilter {
				t.Errorf("has -vf flag = %v, want %v", hasVf, tt.wantFilter)
			}

			pixIdx := slices.Index(args, "-pix_fmt")
			if pixIdx == -1 || pixIdx+1 >= len(args) {
				t.Fatalf("missing -pix_fmt in args")
			}
			if args[pixIdx+1] != tt.wantPixFmt {
				t.Errorf("got pix_fmt %s, want %s", args[pixIdx+1], tt.wantPixFmt)
			}

			if !strings.HasSuffix(outputFile, "test.optimized.mp4") {
				t.Errorf("unexpected output file: %s", outputFile)
			}
		})
	}
}

func TestBuildFFmpegArgs_NoVideoStreams(t *testing.T) {
	probe := &FFProbeOutput{
		Streams: []Stream{
			{Index: 0, CodecType: "audio", CodecName: "aac"},
		},
	}

	_, _, _, _, _, err := buildFFmpegArgs("/tmp/audio_only.m4a", probe, EncodeCallbacks{})
	if err == nil {
		t.Fatal("expected error for input with no video streams, got nil")
	}
}

func TestBuildFFmpegArgs_AudioSelectionAndSubtitles(t *testing.T) {
	probe := &FFProbeOutput{
		Streams: []Stream{
			{Index: 0, CodecType: "video", CodecName: "hevc", Width: 1920, Height: 1080},
			{
				Index:     1,
				CodecType: "audio",
				CodecName: "ac3",
				Tags:      Tags{Title: "Director's Commentary", Language: "eng"},
			},
			{
				Index:     2,
				CodecType: "audio",
				CodecName: "aac",
				Tags:      Tags{Title: "Main Stereo", Language: "eng"},
			},
			{
				Index:     3,
				CodecType: "subtitle",
				CodecName: "subrip",
				Tags:      Tags{Language: "eng"},
			},
		},
	}

	args, _, _, _, _, err := buildFFmpegArgs("/tmp/movie.mkv", probe, EncodeCallbacks{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Stream index 2 (audio relative index 1) should be selected because index 1 is commentary
	hasAudioMap := slices.Contains(args, "0:a:1")
	if !hasAudioMap {
		t.Errorf("expected map 0:a:1 for non-commentary audio stream, args: %v", args)
	}

	// Subtitle should be mapped and converted to mov_text
	hasSubMap := slices.Contains(args, "0:s:0")
	if !hasSubMap {
		t.Errorf("expected map 0:s:0, args: %v", args)
	}

	subCodecIdx := slices.Index(args, "-c:s")
	if subCodecIdx == -1 || subCodecIdx+1 >= len(args) || args[subCodecIdx+1] != "mov_text" {
		t.Errorf("expected -c:s mov_text, args: %v", args)
	}
}

func TestBuildFFmpegArgs_OutputFileCollision(t *testing.T) {
	probe := &FFProbeOutput{
		Streams: []Stream{
			{Index: 0, CodecType: "video", CodecName: "h264", Width: 1280, Height: 720},
		},
	}

	_, outputFile, _, _, _, err := buildFFmpegArgs("/tmp/clip.optimized.mp4", probe, EncodeCallbacks{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if outputFile != "/tmp/clip.optimized.1.mp4" {
		t.Errorf("expected collision resolution /tmp/clip.optimized.1.mp4, got %s", outputFile)
	}
}

func TestFormatHelpers(t *testing.T) {
	tests := []struct {
		name     string
		bytes    float64
		wantSize string
	}{
		{"bytes", 500, "500 B"},
		{"kilobytes", 2048, "2.00 KB"},
		{"megabytes", 10485760, "10.00 MB"},
		{"gigabytes", 1073741824, "1.00 GB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatSize(tt.bytes)
			if got != tt.wantSize {
				t.Errorf("FormatSize(%v) = %s, want %s", tt.bytes, got, tt.wantSize)
			}
		})
	}

	if got := FormatBitrate(128000); got != "0.13 Mbps" {
		t.Errorf("FormatBitrate(128000) = %s, want 0.13 Mbps", got)
	}

	if got := FormatDuration(125.5); got != "2m5s" && got != "2m5.5s" {
		t.Logf("FormatDuration(125.5) = %s", got)
	}

	if got := ParseFrameRate("60000/1001"); got != "59.94" {
		t.Errorf("ParseFrameRate(60000/1001) = %s, want 59.94", got)
	}
	if got := ParseFrameRate("30/1"); got != "30.00" {
		t.Errorf("ParseFrameRate(30/1) = %s, want 30.00", got)
	}
}

func TestIsErrorLine(t *testing.T) {
	tests := []struct {
		line    string
		isError bool
	}{
		{"[info] Video stream detected", false},
		{"[warning] non-standard frame rate", false},
		{"[error] Failed to open codec", true},
		{"Conversion failed", true},
		{"Cannot allocate memory", true},
		{"frame=  120 fps= 24 q=28.0 size=    1024kB", false},
	}

	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			got := isErrorLine(tt.line)
			if got != tt.isError {
				t.Errorf("isErrorLine(%q) = %v, want %v", tt.line, got, tt.isError)
			}
		})
	}
}
