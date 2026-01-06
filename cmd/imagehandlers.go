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

var imgDedupeCmd = &cobra.Command{
	Use:   "img-dedup",
	Short: "Find duplicate images in CWD using perceptual hashing",
	Run: func(cmd *cobra.Command, args []string) {
		maxHammingDistance, _ := cmd.Flags().GetInt("hamming-distance")
		workers, _ := cmd.Flags().GetInt("workers")
		imagehandlers.RunImgDedupe(maxHammingDistance, workers)
	},
}

func init() {
	imgWebpCmd.Flags().BoolP("dry-run", "r", false, "Process images without deleting originals")
	imgDedupeCmd.Flags().IntP("hamming-distance", "d", 10, "Maximum Hamming distance for duplicate detection")
	imgDedupeCmd.Flags().IntP("workers", "w", 4, "Number of workers for parallel processing")
	rootCmd.AddCommand(imgWebpCmd)
	rootCmd.AddCommand(imgDedupeCmd)
}
