package genericsCmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tanq16/nits/internal/generics"
	u "github.com/tanq16/nits/utils"
)

var manualRenameFlags struct {
	includeDir       bool
	hidden           bool
	includeExtension bool
}

var ManualRenameCmd = &cobra.Command{
	Use:     "manual-rename",
	Aliases: []string{"mrename"},
	Short:   "Interactively rename files and directories one by one, optionally including directories, hidden files, and extensions",
	Args:    cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		currentDir, err := os.Getwd()
		if err != nil {
			u.PrintFatal("failed to get current directory", err)
		}
		items, err := generics.GetRenameCandidates(currentDir, manualRenameFlags.includeDir, manualRenameFlags.hidden)
		if err != nil {
			u.PrintFatal("failed to read directory", err)
		}
		if len(items) == 0 {
			u.PrintWarn("no items found to rename", nil)
			return
		}

		u.PrintInfo("Renaming files...")
		renameCount := 0
		for _, entry := range items {
			oldName := entry.Name()
			input, err := u.PromptInput(oldName+" →", "new name (Enter to skip)")
			if err != nil || strings.TrimSpace(input) == "" || strings.TrimSpace(input) == oldName {
				u.PrintIndentedWarn(fmt.Sprintf("%s → (skipped)", oldName), nil)
				continue
			}
			newName := generics.ComputeNewName(oldName, input, entry.IsDir(), manualRenameFlags.includeExtension)
			if oldName == newName {
				u.PrintIndentedWarn(fmt.Sprintf("%s → (skipped)", oldName), nil)
				continue
			}

			oldPath := filepath.Join(currentDir, oldName)
			newPath := filepath.Join(currentDir, newName)

			if err := os.Rename(oldPath, newPath); err != nil {
				u.PrintIndentedError(fmt.Sprintf("%s → %s", oldName, newName), err)
				continue
			}
			u.PrintIndentedSuccess(fmt.Sprintf("%s → %s", oldName, newName))
			renameCount++
		}

		if renameCount == 0 {
			u.PrintWarn("no items were renamed", nil)
			return
		}
		u.PrintSuccess(fmt.Sprintf("%d item(s) renamed", renameCount))
	},
}

func init() {
	ManualRenameCmd.Flags().BoolVarP(&manualRenameFlags.includeDir, "include-dir", "d", false, "Include directories in the rename operation")
	ManualRenameCmd.Flags().BoolVarP(&manualRenameFlags.hidden, "hidden", "H", false, "Include hidden files and directories")
	ManualRenameCmd.Flags().BoolVarP(&manualRenameFlags.includeExtension, "include-extension", "x", false, "Allow changing file extension")
}

