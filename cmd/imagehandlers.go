package cmd

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/tanq16/nits/internal/imagehandlers"
	"github.com/tanq16/nits/utils"
)

var imgWebpFlags struct {
	dryRun  bool
	workers int
}

var imgDedupeFlags struct {
	hammingDistance int
	workers         int
}

var imgWebpCmd = &cobra.Command{
	Use:   "img-webp",
	Short: "Compress all images in CWD to WebP format with quality optimization",
	Run: func(cmd *cobra.Command, args []string) {
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()

		if err := imagehandlers.RunImgWebp(ctx, imgWebpFlags.dryRun, imgWebpFlags.workers); err != nil {
			utils.PrintFatal("Failed to compress images", err)
		}
		utils.PrintSuccess("Image compression complete")
	},
}

var imgDedupeCmd = &cobra.Command{
	Use:   "img-dedup",
	Short: "Find duplicate images in CWD using perceptual hashing",
	Run: func(cmd *cobra.Command, args []string) {
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()

		if err := imagehandlers.RunImgDedupe(ctx, imgDedupeFlags.hammingDistance, imgDedupeFlags.workers); err != nil {
			utils.PrintFatal("Failed to find duplicate images", err)
		}
	},
}

func init() {
	imgWebpCmd.Flags().BoolVarP(&imgWebpFlags.dryRun, "dry-run", "r", false, "Process images without deleting originals")
	imgWebpCmd.Flags().IntVarP(&imgWebpFlags.workers, "workers", "w", 4, "Number of workers for parallel processing")
	imgDedupeCmd.Flags().IntVarP(&imgDedupeFlags.hammingDistance, "hamming-distance", "d", 10, "Maximum Hamming distance for duplicate detection")
	imgDedupeCmd.Flags().IntVarP(&imgDedupeFlags.workers, "workers", "w", 4, "Number of workers for parallel processing")
	rootCmd.AddCommand(imgWebpCmd)
	rootCmd.AddCommand(imgDedupeCmd)
}
