package cmd

import (
	"github.com/spf13/cobra"
	"github.com/tanq16/nits/internal/setup"
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Check if required tools are installed",
	Run: func(cmd *cobra.Command, args []string) {
		setup.RunSetup()
	},
}

func init() {
	rootCmd.AddCommand(setupCmd)
}
