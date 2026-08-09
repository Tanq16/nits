package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tanq16/nits/internal/filehandlers"
	"github.com/tanq16/nits/utils"
)

var fileOrganizerFlags struct {
	dryRun bool
}

var fileUnzipperFlags struct {
	uuidNames bool
}

var fileJSONUniqueFlags struct {
	path string
	key  string
}

var fileOrganizerCmd = &cobra.Command{
	Use:   "file-organizer",
	Short: "Group files into dirs based on base name. eg. goku_1.jpg, goku_2.jpg -> goku/",
	Run: func(cmd *cobra.Command, args []string) {
		if fileOrganizerFlags.dryRun {
			res, err := filehandlers.RunFileOrganizer(true)
			if err != nil {
				utils.PrintFatal("Failed to analyze directory", err)
			}
			if len(res.Groups) == 0 {
				utils.PrintInfo("No file groups found to organize")
				return
			}
			utils.PrintInfo(fmt.Sprintf("Found %d groups to create", len(res.Groups)))
			for base, files := range res.Groups {
				utils.PrintGeneric(fmt.Sprintf("  %s/ (%d files)", base, len(files)))
				displayCount := min(len(files), 5)
				for i := range displayCount {
					utils.PrintGeneric(fmt.Sprintf("    - %s", files[i]))
				}
				if len(files) > displayCount {
					utils.PrintGeneric(fmt.Sprintf("    ... and %d more", len(files)-displayCount))
				}
			}
			return
		}

		utils.PrintRunning("Organizing files into subdirectories...")
		res, err := filehandlers.RunFileOrganizer(false)
		utils.ClearLines(1)
		if err != nil {
			utils.PrintFatal("Failed to organize files", err)
		}
		utils.PrintSuccess(fmt.Sprintf("Organized %d file(s) into %d folder(s)", res.MovedCount, res.FolderCount))
	},
}

var fileUnzipperCmd = &cobra.Command{
	Use:   "file-unzipper",
	Short: "Unzip all zip files in the current directory",
	Long:  `Unzips any zip files in CWD, creating a new directory for each and unzipping contents into it. If the zip contains a single subdirectory, it will be flattened into the parent.`,
	Run: func(cmd *cobra.Command, args []string) {
		utils.PrintRunning("Unzipping archive files in current directory...")
		count, err := filehandlers.RunFileUnzipper(fileUnzipperFlags.uuidNames)
		utils.ClearLines(1)
		if err != nil {
			utils.PrintFatal("Failed to unzip files", err)
		}
		if count == 0 {
			utils.PrintInfo("No zip archives found in current directory")
			return
		}
		utils.PrintSuccess(fmt.Sprintf("Unzipped %d archive(s)", count))
	},
}

var fileJSONUniqueCmd = &cobra.Command{
	Use:   "file-json-uniq <file>",
	Short: "Remove duplicate items from a JSON slice based on a key",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		utils.PrintRunning(fmt.Sprintf("Deduplicating %s...", args[0]))
		err := filehandlers.RunJSONUnique(args[0], fileJSONUniqueFlags.path, fileJSONUniqueFlags.key)
		utils.ClearLines(1)
		if err != nil {
			utils.PrintFatal("Failed to deduplicate JSON", err)
		}
		utils.PrintSuccess(fmt.Sprintf("Deduplicated %s", args[0]))
	},
}

func init() {
	fileOrganizerCmd.Flags().BoolVarP(&fileOrganizerFlags.dryRun, "dry-run", "r", false, "Check without changes")
	fileUnzipperCmd.Flags().BoolVarP(&fileUnzipperFlags.uuidNames, "uuid-names", "u", false, "Rename directories and files to UUIDs")
	fileJSONUniqueCmd.Flags().StringVarP(&fileJSONUniqueFlags.path, "path", "p", "", "Path to the slice in the JSON (e.g. 'references')")
	fileJSONUniqueCmd.Flags().StringVarP(&fileJSONUniqueFlags.key, "key", "k", "", "Key to use for uniqueness (e.g. 'url')")
	fileJSONUniqueCmd.MarkFlagRequired("path")
	fileJSONUniqueCmd.MarkFlagRequired("key")
	rootCmd.AddCommand(fileOrganizerCmd)
	rootCmd.AddCommand(fileUnzipperCmd)
	rootCmd.AddCommand(fileJSONUniqueCmd)
}
