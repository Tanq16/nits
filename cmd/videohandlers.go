package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
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
		utils.PrintRunning(fmt.Sprintf("Probing %s...", args[0]))
		data, err := videohandlers.GetVideoInfo(args[0])
		utils.ClearLines(1)
		if err != nil {
			utils.PrintFatal("Failed to get video info", err)
		}

		sizeBytes, _ := strconv.ParseFloat(data.Format.Size, 64)
		durationSec, _ := strconv.ParseFloat(data.Format.Duration, 64)
		bitrate, _ := strconv.ParseFloat(data.Format.BitRate, 64)

		utils.PrintInfo("Container overview:")
		utils.PrintTable([]string{"Property", "Value"}, [][]string{
			{"Container", data.Format.FormatName},
			{"Size", videohandlers.FormatSize(sizeBytes)},
			{"Duration", videohandlers.FormatDuration(durationSec)},
			{"Bitrate", videohandlers.FormatBitrate(bitrate)},
		})

		var streamRows [][]string
		for _, s := range data.Streams {
			details := ""
			switch s.CodecType {
			case "video":
				fps := videohandlers.ParseFrameRate(s.AvgFrameRate)
				details = fmt.Sprintf("%dx%d @ %s fps (%s)", s.Width, s.Height, fps, s.PixFmt)
			case "audio":
				details = fmt.Sprintf("%d ch (%s) @ %s Hz", s.Channels, s.ChannelLayout, s.SampleRate)
			case "subtitle":
				details = s.Tags.Title
			}
			lang := s.Tags.Language
			if lang == "" {
				lang = "und"
			}
			streamRows = append(streamRows, []string{
				fmt.Sprintf("#%d", s.Index),
				strings.ToUpper(s.CodecType),
				strings.ToUpper(s.CodecName),
				details,
				strings.ToUpper(lang),
			})
		}

		utils.PrintInfo("Streams:")
		utils.PrintTable([]string{"Index", "Type", "Codec", "Details", "Lang"}, streamRows)
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
