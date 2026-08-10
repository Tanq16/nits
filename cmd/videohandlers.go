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

var videoOptimizeCmd = &cobra.Command{
	Use:     "video-optimize <file>",
	Aliases: []string{"video-opt"},
	Short:   "Optimize video file to H.265 (max 1080p, CRF 28, AAC 128k)",
	Long: `Optimizes a video file for size reduction using CPU H.265 encoding.
Videos with resolutions higher than 1080p are downscaled to fit within 1080p,
while lower resolutions are preserved. Encodes audio to 128 kbps AAC stereo.

Output file is saved as <basename>.optimized.mp4.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()

		inputFile := args[0]
		inputBase := filepath.Base(inputFile)

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

		res, err := videohandlers.RunVideoOptimize(ctx, inputFile, callbacks)
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
			resStr += " → max 1080p"
		} else {
			resStr += " (retained)"
		}

		spaceSavedStr := fmt.Sprintf("%s (%.1f%%)", videohandlers.FormatSize(float64(savedBytes)), savedPct)
		if savedBytes < 0 {
			spaceSavedStr = fmt.Sprintf("+%s", videohandlers.FormatSize(float64(-savedBytes)))
		}

		utils.PrintTable([]string{"Property", "Value"}, [][]string{
			{"Input Size", videohandlers.FormatSize(float64(res.InputBytes))},
			{"Optimized Size", videohandlers.FormatSize(float64(res.OutputBytes))},
			{"Space Saved", spaceSavedStr},
			{"Resolution", resStr},
			{"Duration", videohandlers.FormatDuration(res.DurationSec)},
			{"Output File", res.OutputFile},
		})
	},
}

func init() {
	rootCmd.AddCommand(videoOptimizeCmd)
}
