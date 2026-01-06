package cmd

import (
	"github.com/spf13/cobra"
	"github.com/tanq16/nits/internal/imagehandlers"
)

var imgWebpCmd = &cobra.Command{
	Use:   "img-webp",
	Short: "Compress all images in CWD to WebP format with quality optimization",
	Run: func(cmd *cobra.Command, args []string) {
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		imagehandlers.RunImgWebp(dryRun)
	},
}

func init() {
	imgWebpCmd.Flags().BoolP("dry-run", "r", false, "Process images without deleting originals")
	rootCmd.AddCommand(imgWebpCmd)
}
