package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/tanq16/nits/internal/videohandlers"
	"github.com/tanq16/nits/utils"
)

var videoOptimizeFlags struct {
	manual bool
}

var videoOptimizeCmd = &cobra.Command{
	Use:     "video-optimize <file>",
	Aliases: []string{"video-opt"},
	Short:   "Optimize video file to H.265 (max 1080p, CRF 30, 8-bit SDR, AAC 128k)",
	Long: `Optimizes a video file for size reduction using CPU H.265 encoding.
Videos with resolutions higher than 1080p are downscaled to fit within 1080p,
while lower resolutions are preserved. 10-bit HDR videos are automatically tone-mapped
to 8-bit SDR. Encodes audio to 128 kbps AAC stereo.

Use --manual to interactively configure CRF, resolution, audio, and presets.

Output file is saved as <basename>.optimized.mp4.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()

		inputFile := args[0]
		inputBase := filepath.Base(inputFile)

		opts := videohandlers.DefaultOptimizeOptions()

		if videoOptimizeFlags.manual {
			data, err := videohandlers.GetVideoInfo(inputFile)
			if err != nil {
				utils.PrintFatal("Failed to probe video", err)
			}

			// CRF prompt
			crfOptions := []string{
				"30 (Default — High Compression, ~65-75% size reduction)",
				"28 (Balanced Quality — Crisp 1080p details, ~50-65% size reduction)",
				"26 (Medium-High Quality — Recommended for action/sports)",
				"24 (High Quality — Near-source transparent)",
				"22 (Very High Quality — Archival quality)",
				"32 (Very High Compression — Smallest size, tutorials/talks)",
				"34 (Maximum Compression — Extreme compression)",
			}
			crfValues := []int{30, 28, 26, 24, 22, 32, 34}
			crfIdx, err := utils.PromptSelect("Select CRF quality/compression factor", crfOptions)
			if err != nil || crfIdx < 0 {
				utils.PrintInfo("Optimization cancelled")
				return
			}
			opts.CRF = crfValues[crfIdx]

			// Resolution prompt
			resOptions := []string{
				"1080p (Default — Max 1920x1080, keep if lower)",
				"720p (Max 1280x720, keep if lower)",
				"480p (Max 854x480 SD, keep if lower)",
				"Original (Preserve native resolution)",
			}
			resValues := []string{"1080p", "720p", "480p", "none"}
			resIdx, err := utils.PromptSelect("Select maximum resolution target", resOptions)
			if err != nil || resIdx < 0 {
				utils.PrintInfo("Optimization cancelled")
				return
			}
			opts.MaxRes = resValues[resIdx]

			// Audio prompt
			audioOptions := []string{
				"128 kbps (Default — AAC Stereo)",
				"160 kbps (Higher bitrate AAC Stereo)",
				"96 kbps (Lower bitrate AAC Stereo)",
				"No Audio (Strip all audio tracks)",
			}
			audioValues := []string{"128k", "160k", "96k", "none"}
			audioIdx, err := utils.PromptSelect("Select audio configuration", audioOptions)
			if err != nil || audioIdx < 0 {
				utils.PrintInfo("Optimization cancelled")
				return
			}
			opts.AudioMode = audioValues[audioIdx]

			// Preset prompt
			presetOptions := []string{
				"medium (Default — Balanced speed and compression)",
				"slow (Better compression, ~2x longer encode)",
				"fast (Faster encode, ~5-10% larger file)",
			}
			presetValues := []string{"medium", "slow", "fast"}
			presetIdx, err := utils.PromptSelect("Select H.265 encoder speed preset", presetOptions)
			if err != nil || presetIdx < 0 {
				utils.PrintInfo("Optimization cancelled")
				return
			}
			opts.Preset = presetValues[presetIdx]

			// Tone mapping prompt if HDR is detected
			var primaryVideo *videohandlers.Stream
			for _, s := range data.Streams {
				if s.CodecType == "video" {
					primaryVideo = &s
					break
				}
			}
			if primaryVideo != nil && videohandlers.IsHDRStream(*primaryVideo) {
				hdrOptions := []string{
					"Tone-map HDR to SDR 8-bit (Default — Prevents washed-out colors)",
					"Direct 8-bit conversion without tone mapping",
				}
				hdrIdx, err := utils.PromptSelect("HDR source detected: Select color processing", hdrOptions)
				if err != nil || hdrIdx < 0 {
					utils.PrintInfo("Optimization cancelled")
					return
				}
				if hdrIdx == 0 {
					opts.ToneMap = "yes"
				} else {
					opts.ToneMap = "no"
				}
			}
		}

		utils.PrintRunning(fmt.Sprintf("Optimizing %s...", inputBase))

		var printed atomic.Bool
		var firstTick atomic.Bool
		firstTick.Store(true)

		callbacks := videohandlers.EncodeCallbacks{
			OnInfo: func(msg string) {
				utils.PrintInfo(msg)
			},
			OnProgress: func(label string, percent int) {
				if !firstTick.Swap(false) {
					utils.ClearPreviousLine()
				}
				printed.Store(true)
				utils.PrintProgress(label, percent)
			},
			OnProgressDone: func() {
				if printed.Load() {
					utils.ClearPreviousLine()
				}
			},
			OnError: func(msg string) {
				utils.PrintIndentedError(msg, nil)
			},
		}

		res, err := videohandlers.RunVideoOptimize(ctx, inputFile, opts, callbacks)
		utils.ClearLines(1)
		if err != nil {
			utils.PrintFatal("Failed to optimize video", err)
		}

		savedBytes := res.InputBytes - res.OutputBytes
		savedPct := 0.0
		if res.InputBytes > 0 {
			savedPct = float64(savedBytes) / float64(res.InputBytes) * 100
		}

		outputBase := filepath.Base(res.OutputFile)
		utils.PrintSuccess(fmt.Sprintf("Optimized %s in %s (saved %.1f%%)", outputBase, res.TimeTaken.Round(time.Second), savedPct))

		resStr := fmt.Sprintf("%dx%d", res.OrigWidth, res.OrigHeight)
		if res.Scaled {
			resStr += fmt.Sprintf(" → %s", res.TargetRes)
		} else {
			resStr += " (retained)"
		}

		spaceSavedStr := fmt.Sprintf("%s (%.1f%%)", videohandlers.FormatSize(float64(savedBytes)), savedPct)
		if savedBytes < 0 {
			spaceSavedStr = fmt.Sprintf("+%s", videohandlers.FormatSize(float64(-savedBytes)))
		}

		colorProfile := "8-bit SDR"
		if res.ToneMapped {
			colorProfile = "Tone-mapped to 8-bit SDR"
		}

		utils.PrintTable([]string{"Property", "Value"}, [][]string{
			{"Input Size", videohandlers.FormatSize(float64(res.InputBytes))},
			{"Optimized Size", videohandlers.FormatSize(float64(res.OutputBytes))},
			{"Space Saved", spaceSavedStr},
			{"Resolution", resStr},
			{"CRF / Preset", fmt.Sprintf("%d / %s", res.CRF, res.Preset)},
			{"Color Format", colorProfile},
			{"Duration", videohandlers.FormatDuration(res.DurationSec)},
			{"Output File", res.OutputFile},
		})
	},
}

func init() {
	videoOptimizeCmd.Flags().BoolVarP(&videoOptimizeFlags.manual, "manual", "m", false, "Interactively choose CRF, resolution, audio, and preset options")
	rootCmd.AddCommand(videoOptimizeCmd)
}
