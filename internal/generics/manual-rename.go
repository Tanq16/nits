package generics

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog/log"

	u "github.com/tanq16/nits/utils"
)

func ManualRename(includeDir bool, hidden bool, includeExtension bool) error {
	currentDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}
	entries, err := os.ReadDir(currentDir)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	var items []os.DirEntry
	for _, entry := range entries {
		if !hidden && strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		if entry.IsDir() {
			if includeDir {
				items = append(items, entry)
			}
			continue
		}
		items = append(items, entry)
	}
	if len(items) == 0 {
		u.PrintWarn("no items found to rename", nil)
		return nil
	}
	log.Debug().Int("count", len(items)).Msg("items to process")

	renameCount := 0
	for _, entry := range items {
		oldName := entry.Name()
		input, err := u.PromptInput(oldName+" →", "new name (Enter to skip)")
		if err != nil || strings.TrimSpace(input) == "" || strings.TrimSpace(input) == oldName {
			u.PrintIndentedWarn(fmt.Sprintf("%s → (skipped)", oldName), nil)
			continue
		}
		newName := strings.TrimSpace(input)

		if !includeExtension && !entry.IsDir() {
			ext := filepath.Ext(oldName)
			if ext != "" && !strings.HasSuffix(newName, ext) {
				newName += ext
			}
		}
		if oldName == newName {
			u.PrintIndentedWarn(fmt.Sprintf("%s → (skipped)", oldName), nil)
			continue
		}

		oldPath := filepath.Join(currentDir, oldName)
		newPath := filepath.Join(currentDir, newName)
		log.Debug().Str("old", oldName).Str("new", newName).Msg("renaming")

		if err := os.Rename(oldPath, newPath); err != nil {
			u.PrintIndentedError(fmt.Sprintf("%s → %s", oldName, newName), err)
			continue
		}
		u.PrintIndentedSuccess(fmt.Sprintf("%s → %s", oldName, newName))
		renameCount++
	}

	if renameCount == 0 {
		u.PrintWarn("no items were renamed", nil)
		return nil
	}
	u.PrintSuccess(fmt.Sprintf("%d item(s) renamed", renameCount))
	return nil
}
