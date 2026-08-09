package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
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

		utils.PrintRunning("Compressing images to WebP...")
		stats, err := imagehandlers.RunImgWebp(ctx, imgWebpFlags.dryRun, imgWebpFlags.workers)
		utils.ClearLines(1)
		if err != nil {
			utils.PrintFatal("Failed to compress images", err)
		}
		if stats.Processed == 0 {
			utils.PrintInfo("No images found to compress")
			return
		}

		utils.PrintSuccess(fmt.Sprintf("Processed %d image(s), saved %.2f MB", stats.Processed, float64(stats.TotalSavedBytes)/1024/1024))
		utils.PrintTable([]string{"Metric", "Value"}, [][]string{
			{"Total images processed", fmt.Sprintf("%d", stats.Processed)},
			{"Retained with Quality 98", fmt.Sprintf("%d", stats.Quality98)},
			{"Fallback to Quality 95", fmt.Sprintf("%d", stats.Quality95)},
			{"Images requiring Resizing", fmt.Sprintf("%d", stats.Resized)},
			{"Final WebP <= 190 KB", fmt.Sprintf("%d", stats.FinalUnder190)},
			{"Final WebP > 190 KB", fmt.Sprintf("%d", stats.FinalOver190)},
			{"Total storage space saved", fmt.Sprintf("%.2f MB", float64(stats.TotalSavedBytes)/1024/1024)},
		})

		if imgWebpFlags.dryRun && len(stats.DetailedLogs) > 0 {
			utils.PrintInfo("Dry run logs:")
			for _, entry := range stats.DetailedLogs {
				utils.PrintGeneric("  " + entry)
			}
		}
	},
}

var imgDedupeCmd = &cobra.Command{
	Use:   "img-dedup",
	Short: "Find duplicate images in CWD using perceptual hashing",
	Run: func(cmd *cobra.Command, args []string) {
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()

		utils.PrintRunning("Scanning images for perceptual duplicates...")
		groups, err := imagehandlers.FindDuplicates(ctx, imgDedupeFlags.hammingDistance, imgDedupeFlags.workers)
		utils.ClearLines(1)
		if err != nil {
			utils.PrintFatal("Failed to find duplicate images", err)
		}
		if len(groups) == 0 {
			utils.PrintSuccess("No duplicate images found")
			return
		}

		utils.PrintInfo(fmt.Sprintf("Found %d set(s) of duplicates", len(groups)))
		for i, group := range groups {
			best := group[0]
			duplicates := group[1:]
			utils.PrintGeneric(fmt.Sprintf("\nSET #%d", i+1))
			utils.PrintGeneric(fmt.Sprintf("  - KEEP  : %s (%dx%d)", best.Filename, best.Width, best.Height))
			var dupNames []string
			for _, d := range duplicates {
				dupNames = append(dupNames, fmt.Sprintf("%s (%dx%d)", d.Filename, d.Width, d.Height))
			}
			utils.PrintGeneric(fmt.Sprintf("  - DELETE: %s", strings.Join(dupNames, ", ")))
			cmdStr := "rm"
			for _, d := range duplicates {
				cmdStr += fmt.Sprintf(" %q", d.Filename)
			}
			utils.PrintGeneric(fmt.Sprintf("  - CMD   : %s", cmdStr))
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
