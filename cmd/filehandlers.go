package cmd

import (
	"github.com/spf13/cobra"
	"github.com/tanq16/nits/internal/filehandlers"
)

var fileOrganizerCmd = &cobra.Command{
	Use:   "file-organizer",
	Short: "Organize files by grouping them into folders based on base name. eg. goku_1.jpg, goku_2.jpg -> goku/",
	Run: func(cmd *cobra.Command, args []string) {
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		filehandlers.RunFileOrganizer(dryRun)
	},
}

var fileUnzipperCmd = &cobra.Command{
	Use:   "unzipper",
	Short: "Unzip all zip files in the current directory",
	Long:  `Unzips any zip files in CWD, creating a new directory for each and unzipping contents into it. If the zip contains a single subdirectory, it will be flattened into the parent.`,
	Run: func(cmd *cobra.Command, args []string) {
		uuidNames, _ := cmd.Flags().GetBool("uuid-names")
		filehandlers.RunFileUnzipper(uuidNames)
	},
}

func init() {
	fileOrganizerCmd.Flags().BoolP("dry-run", "r", false, "Check without changes")
	fileUnzipperCmd.Flags().BoolP("uuid-names", "u", false, "Rename directories and files to UUIDs")
	rootCmd.AddCommand(fileOrganizerCmd)
	rootCmd.AddCommand(fileUnzipperCmd)
}
