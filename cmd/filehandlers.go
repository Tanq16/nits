package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tanq16/nits/internal/filehandlers"
	"github.com/tanq16/nits/utils"
)

var fileUnzipperFlags struct {
	uuidNames bool
}

var fileUnzipperCmd = &cobra.Command{
	Use:   "file-unzipper",
	Short: "Unzip all zip files in the current directory",
	Long:  `Unzips any zip files in CWD, creating a new directory for each and unzipping contents into it. If the zip contains a single subdirectory, it will be flattened into the parent.`,
	Args:  cobra.NoArgs,
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

func init() {
	fileUnzipperCmd.Flags().BoolVarP(&fileUnzipperFlags.uuidNames, "uuid-names", "u", false, "Rename directories and files to UUIDs")
	rootCmd.AddCommand(fileUnzipperCmd)
}
