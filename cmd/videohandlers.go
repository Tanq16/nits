package cmd

import (
	"github.com/spf13/cobra"
	"github.com/tanq16/nits/internal/utils"
	"github.com/tanq16/nits/internal/videohandlers"
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
	input  string
	output string
	params string
}

var videoEncodeCmd = &cobra.Command{
	Use:   "video-encode",
	Short: "Encode video files using ffmpeg with custom parameters",
	Run: func(cmd *cobra.Command, args []string) {
		if videoEncodeFlags.input == "" || videoEncodeFlags.output == "" {
			utils.PrintFatal("Input (-i) and Output (-o) flags are required", nil)
		}
		if err := videohandlers.RunVideoEncode(videoEncodeFlags.input, videoEncodeFlags.output, videoEncodeFlags.params); err != nil {
			utils.PrintFatal("Failed to encode video", err)
		}
	},
}

func init() {
	videoEncodeCmd.Flags().StringVarP(&videoEncodeFlags.input, "input", "i", "", "Input video file (required)")
	videoEncodeCmd.Flags().StringVarP(&videoEncodeFlags.output, "output", "o", "", "Output video file (required)")
	videoEncodeCmd.Flags().StringVarP(&videoEncodeFlags.params, "params", "p", "", "FFmpeg encoding parameters (e.g., '-c:v libx264 -crf 23')")

	rootCmd.AddCommand(videoInfoCmd)
	rootCmd.AddCommand(videoEncodeCmd)
}
