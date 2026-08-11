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
		maxRes     string
		wantScaled bool
		wantFilter bool
	}{
		{
			name:       "4K video downscaled to 1080p",
			width:      3840,
			height:     2160,
			maxRes:     "1080p",
			wantScaled: true,
			wantFilter: true,
		},
		{
			name:       "1440p video downscaled to 1080p",
			width:      2560,
			height:     1440,
			maxRes:     "1080p",
			wantScaled: true,
			wantFilter: true,
		},
		{
			name:       "1080p video downscaled to 720p",
			width:      1920,
			height:     1080,
			maxRes:     "720p",
			wantScaled: true,
			wantFilter: true,
		},
		{
			name:       "Vertical 1080x1920 phone video downscaled",
			width:      1080,
			height:     1920,
			maxRes:     "1080p",
			wantScaled: true,
			wantFilter: true,
		},
		{
			name:       "1080p exact kept unchanged on 1080p max",
			width:      1920,
			height:     1080,
			maxRes:     "1080p",
			wantScaled: false,
			wantFilter: false,
		},
		{
			name:       "720p kept unchanged on 1080p max",
			width:      1280,
			height:     720,
			maxRes:     "1080p",
			wantScaled: false,
			wantFilter: false,
		},
		{
			name:       "4K kept unchanged on none max",
			width:      3840,
			height:     2160,
			maxRes:     "none",
			wantScaled: false,
			wantFilter: false,
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
						PixFmt:    "yuv420p",
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

			opts := DefaultOptimizeOptions()
			opts.MaxRes = tt.maxRes

			args, outputFile, res, err := buildFFmpegArgs("/tmp/test.mkv", probe, opts, EncodeCallbacks{})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if res.OrigWidth != tt.width || res.OrigHeight != tt.height {
				t.Errorf("got dimensions %dx%d, want %dx%d", res.OrigWidth, res.OrigHeight, tt.width, tt.height)
			}
			if res.Scaled != tt.wantScaled {
				t.Errorf("got scaled %v, want %v", res.Scaled, tt.wantScaled)
			}

			hasVf := slices.Contains(args, "-vf")
			if hasVf != tt.wantFilter {
				t.Errorf("has -vf flag = %v, want %v", hasVf, tt.wantFilter)
			}

			// Verify 8-bit yuv420p is always enforced
			pixIdx := slices.Index(args, "-pix_fmt")
			if pixIdx == -1 || pixIdx+1 >= len(args) {
				t.Fatalf("missing -pix_fmt in args")
			}
			if args[pixIdx+1] != "yuv420p" {
				t.Errorf("got pix_fmt %s, want yuv420p", args[pixIdx+1])
			}

			// Verify CRF 30 by default
			crfIdx := slices.Index(args, "-crf")
			if crfIdx == -1 || crfIdx+1 >= len(args) {
				t.Fatalf("missing -crf in args")
			}
			if args[crfIdx+1] != "30" {
				t.Errorf("got crf %s, want 30", args[crfIdx+1])
			}

			if !strings.HasSuffix(outputFile, "test.optimized.mp4") {
				t.Errorf("unexpected output file: %s", outputFile)
			}
		})
	}
}

func TestBuildFFmpegArgs_HDRToneMapping(t *testing.T) {
	tests := []struct {
		name          string
		colorTransfer string
		colorSpace    string
		pixFmt        string
		toneMapOpt    string
		wantToneMap   bool
	}{
		{
			name:          "HDR10 smpte2084 auto tone-mapped",
			colorTransfer: "smpte2084",
			colorSpace:    "bt2020nc",
			pixFmt:        "yuv420p10le",
			toneMapOpt:    "auto",
			wantToneMap:   true,
		},
		{
			name:          "HLG arib-std-b67 auto tone-mapped",
			colorTransfer: "arib-std-b67",
			colorSpace:    "bt2020nc",
			pixFmt:        "yuv420p10le",
			toneMapOpt:    "auto",
			wantToneMap:   true,
		},
		{
			name:          "SDR 10-bit not tone-mapped in auto",
			colorTransfer: "bt709",
			colorSpace:    "bt709",
			pixFmt:        "yuv420p10le",
			toneMapOpt:    "auto",
			wantToneMap:   false,
		},
		{
			name:          "Forced tone-mapping on SDR",
			colorTransfer: "bt709",
			colorSpace:    "bt709",
			pixFmt:        "yuv420p",
			toneMapOpt:    "yes",
			wantToneMap:   true,
		},
		{
			name:          "HDR with tone-mapping disabled",
			colorTransfer: "smpte2084",
			colorSpace:    "bt2020nc",
			pixFmt:        "yuv420p10le",
			toneMapOpt:    "no",
			wantToneMap:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			probe := &FFProbeOutput{
				Streams: []Stream{
					{
						Index:         0,
						CodecType:     "video",
						CodecName:     "hevc",
						Width:         1920,
						Height:        1080,
						PixFmt:        tt.pixFmt,
						ColorTransfer: tt.colorTransfer,
						ColorSpace:    tt.colorSpace,
					},
				},
				Format: Format{Filename: "hdr.mkv", Duration: "60"},
			}

			opts := DefaultOptimizeOptions()
			opts.ToneMap = tt.toneMapOpt

			args, _, res, err := buildFFmpegArgs("/tmp/hdr.mkv", probe, opts, EncodeCallbacks{})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if res.ToneMapped != tt.wantToneMap {
				t.Errorf("got ToneMapped=%v, want %v", res.ToneMapped, tt.wantToneMap)
			}

			vfIdx := slices.Index(args, "-vf")
			hasToneMapFilter := vfIdx != -1 && vfIdx+1 < len(args) && strings.Contains(args[vfIdx+1], "tonemap=hable")
			if hasToneMapFilter != tt.wantToneMap {
				t.Errorf("has tone map filter in -vf = %v, want %v", hasToneMapFilter, tt.wantToneMap)
			}
		})
	}
}

func TestBuildFFmpegArgs_ManualOptions(t *testing.T) {
	probe := &FFProbeOutput{
		Streams: []Stream{
			{Index: 0, CodecType: "video", CodecName: "h264", Width: 1920, Height: 1080, PixFmt: "yuv420p"},
			{Index: 1, CodecType: "audio", CodecName: "aac", Channels: 2},
		},
	}

	opts := OptimizeOptions{
		CRF:       24,
		MaxRes:    "720p",
		AudioMode: "none",
		Preset:    "slow",
		ToneMap:   "no",
	}

	args, _, res, err := buildFFmpegArgs("/tmp/sample.mp4", probe, opts, EncodeCallbacks{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.CRF != 24 || res.Preset != "slow" {
		t.Errorf("got CRF=%d, Preset=%s, want 24 / slow", res.CRF, res.Preset)
	}

	crfIdx := slices.Index(args, "-crf")
	if args[crfIdx+1] != "24" {
		t.Errorf("expected -crf 24, got %s", args[crfIdx+1])
	}

	presetIdx := slices.Index(args, "-preset")
	if args[presetIdx+1] != "slow" {
		t.Errorf("expected -preset slow, got %s", args[presetIdx+1])
	}

	if !slices.Contains(args, "-an") {
		t.Errorf("expected -an flag for AudioMode 'none', args: %v", args)
	}
}

func TestBuildFFmpegArgs_NoVideoStreams(t *testing.T) {
	probe := &FFProbeOutput{
		Streams: []Stream{
			{Index: 0, CodecType: "audio", CodecName: "aac"},
		},
	}

	_, _, _, err := buildFFmpegArgs("/tmp/audio_only.m4a", probe, DefaultOptimizeOptions(), EncodeCallbacks{})
	if err == nil {
		t.Fatal("expected error for input with no video streams, got nil")
	}
}

func TestBuildFFmpegArgs_OutputFileCollision(t *testing.T) {
	probe := &FFProbeOutput{
		Streams: []Stream{
			{Index: 0, CodecType: "video", CodecName: "h264", Width: 1280, Height: 720},
		},
	}

	_, outputFile, _, err := buildFFmpegArgs("/tmp/clip.optimized.mp4", probe, DefaultOptimizeOptions(), EncodeCallbacks{})
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
