package cmd

import (
	"errors"
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
	namesFile        string
}

var manualRenameCmd = &cobra.Command{
	Use:     "manual-rename",
	Aliases: []string{"mrename"},
	Short:   "Interactively rename files and directories one by one, optionally including directories, hidden files, and extensions",
	Args:    cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if manualRenameFlags.namesFile == "" && !u.StdinIsTerminal {
			u.PrintFatal("manual-rename needs --names-file, or an interactive terminal", nil)
		}

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
		var names []string
		if manualRenameFlags.namesFile != "" {
			data, err := os.ReadFile(manualRenameFlags.namesFile)
			if err != nil {
				u.PrintFatal("failed to read --names-file", err)
			}
			names = strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")
			if len(names) != len(items) {
				u.PrintFatal(fmt.Sprintf("--names-file has %d line(s) but there are %d item(s) to rename", len(names), len(items)), nil)
			}
		}

		u.PrintInfo("Renaming files...")
		renameCount := 0
		for i, entry := range items {
			oldName := entry.Name()
			var input string
			if names != nil {
				input = names[i]
			} else {
				input, err = u.PromptInput(oldName+" →", "new name (Enter to skip)")
				if err != nil {
					if errors.Is(err, u.ErrNoTerminal) {
						u.PrintFatal("manual-rename requires an interactive terminal", nil)
					}
					u.PrintFatal("TUI error", err)
				}
			}
			if strings.TrimSpace(input) == "" || strings.TrimSpace(input) == oldName {
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
	manualRenameCmd.Flags().BoolVar(&manualRenameFlags.includeDir, "include-dir", false, "Include directories in the rename operation")
	manualRenameCmd.Flags().BoolVar(&manualRenameFlags.hidden, "hidden", false, "Include hidden files and directories")
	manualRenameCmd.Flags().BoolVar(&manualRenameFlags.includeExtension, "include-extension", false, "Allow changing file extension")
	manualRenameCmd.Flags().StringVar(&manualRenameFlags.namesFile, "names-file", "", "File of new names, one per line in listing order (blank line skips)")
}
