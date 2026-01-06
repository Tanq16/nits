package cmd

import (
	"github.com/spf13/cobra"
	"github.com/tanq16/nits/internal/playground"
)

var playgroundCmd = &cobra.Command{
	Use:   "playground",
	Short: "Playground command for testing",
	Run: func(cmd *cobra.Command, args []string) {
		playground.Run()
	},
}

func init() {
	rootCmd.AddCommand(playgroundCmd)
}
