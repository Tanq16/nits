package cmd

import (
	"github.com/spf13/cobra"
	"github.com/tanq16/nits/internal/filehandlers"
)

var organizeCmd = &cobra.Command{
	Use:   "organize",
	Short: "Organize files by grouping them into folders based on base name",
	Run: func(cmd *cobra.Command, args []string) {
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		filehandlers.RunOrganize(dryRun)
	},
}

var smartUnzipCmd = &cobra.Command{
	Use:   "smart-unzip",
	Short: "Unzip all zip files in the current directory",
	Long:  `Unzips any zip files in CWD, creating a new directory for each and unzipping contents into it. If the zip contains a single subdirectory, it will be flattened into the parent.`,
	Run: func(cmd *cobra.Command, args []string) {
		filehandlers.RunSmartUnzip()
	},
}

func init() {
	organizeCmd.Flags().BoolP("dry-run", "r", false, "Check without changes")
	rootCmd.AddCommand(organizeCmd)
	rootCmd.AddCommand(smartUnzipCmd)
}
