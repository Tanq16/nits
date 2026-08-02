package cmd

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/tanq16/nits/internal/videohandlers"
	"github.com/tanq16/nits/utils"
)

var videoInfoCmd = &cobra.Command{
	Use:   "video-info <file>",
	Short: "Display detailed information about a video file using ffprobe",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := videohandlers.RunVideoInfo(args[0]); err != nil {
			utils.PrintFatal("Failed to get video info", err)
		}
	},
}

var videoEncodeFlags struct {
	quality      string
	fpsDowngrade bool
	noVideo      bool
}

var videoEncodeCmd = &cobra.Command{
	Use:   "video-encode <file>",
	Short: "Smart encode video to H.265 with automatic stream selection",
	Long: `Probes the input file, selects the best audio stream (rejecting commentary),
keeps all subtitles, picks the right container (MP4 or MKV), and encodes
video to libx265 with the chosen quality tier.

Output file is generated automatically as <basename>.h265.<mp4|mkv>.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()

		opts := videohandlers.SmartEncodeOptions{
			Quality:      videoEncodeFlags.quality,
			FPSDowngrade: videoEncodeFlags.fpsDowngrade,
			NoVideo:      videoEncodeFlags.noVideo,
		}
		if err := videohandlers.RunSmartEncode(ctx, args[0], opts); err != nil {
			utils.PrintFatal("Failed to encode video", err)
		}
	},
}

func init() {
	videoEncodeCmd.Flags().StringVarP(&videoEncodeFlags.quality, "quality", "q", "medium", "Quality tier: very-high, high, medium, low")
	videoEncodeCmd.Flags().BoolVar(&videoEncodeFlags.fpsDowngrade, "fps-downgrade", false, "Downgrade framerate to 30 fps")
	videoEncodeCmd.Flags().BoolVarP(&videoEncodeFlags.noVideo, "no-video", "V", false, "Copy video stream as-is without re-encoding")

	rootCmd.AddCommand(videoInfoCmd)
	rootCmd.AddCommand(videoEncodeCmd)
}
