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

func init() {
	rootCmd.AddCommand(videoInfoCmd)
}
