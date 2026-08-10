package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tanq16/nits/internal/setup"
	"github.com/tanq16/nits/utils"
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Check if required tools are installed",
	Run: func(cmd *cobra.Command, args []string) {
		utils.PrintRunning("Checking required tools...")
		tools := setup.CheckTools()
		utils.ClearLines(1)

		allInstalled := true
		for _, t := range tools {
			if !t.Found {
				allInstalled = false
				break
			}
		}

		if allInstalled {
			utils.PrintSuccess("All required tools are installed")
			for _, t := range tools {
				utils.PrintIndentedSuccess(fmt.Sprintf("%s (%s)", t.Name, t.Command))
			}
			return
		}

		utils.PrintWarn("Some required tools are missing", nil)
		for _, t := range tools {
			if t.Found {
				utils.PrintIndentedSuccess(fmt.Sprintf("%s (%s)", t.Name, t.Command))
			} else {
				utils.PrintIndentedError(fmt.Sprintf("%s (expected: %s)", t.Name, t.Command), nil)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(setupCmd)
}
