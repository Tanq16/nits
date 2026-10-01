package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/tanq16/nits/internal/imagehandlers"
	"github.com/tanq16/nits/utils"
)

var imgDedupeFlags struct {
	hammingDistance int
	workers         int
}

var imgDedupeCmd = &cobra.Command{
	Use:   "img-dedup",
	Short: "Find duplicate images in CWD using perceptual hashing",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if imgDedupeFlags.workers < 1 {
			utils.PrintFatal("--workers must be at least 1", nil)
		}

		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()

		utils.PrintRunning("Scanning images for perceptual duplicates...")
		result, err := imagehandlers.FindDuplicates(ctx, imgDedupeFlags.hammingDistance, imgDedupeFlags.workers)
		utils.ClearLines(1)
		if err != nil {
			utils.PrintFatal("Failed to find duplicate images", err)
		}
		if len(result.Failures) > 0 {
			utils.PrintWarn(fmt.Sprintf("Could not read %d image(s)", len(result.Failures)), nil)
			for _, f := range result.Failures {
				utils.PrintIndentedWarn(filepath.Base(f.Path), f.Err)
			}
		}
		if result.Scanned == 0 {
			if len(result.Failures) == 0 {
				utils.PrintInfo("No images found")
			}
			return
		}
		groups := result.Groups
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
				cmdStr += " '" + strings.ReplaceAll(d.Filename, "'", `'\''`) + "'"
			}
			utils.PrintGeneric(fmt.Sprintf("  - CMD   : %s", cmdStr))
		}
	},
}

func init() {
	imgDedupeCmd.Flags().IntVarP(&imgDedupeFlags.hammingDistance, "hamming-distance", "d", 10, "Maximum Hamming distance for duplicate detection")
	imgDedupeCmd.Flags().IntVarP(&imgDedupeFlags.workers, "workers", "w", 4, "Number of workers for parallel processing")
}
