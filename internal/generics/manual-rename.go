package generics

import (
	"os"
	"path/filepath"
	"strings"
)

func GetRenameCandidates(dir string, includeDir bool, hidden bool) ([]os.DirEntry, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
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
	return items, nil
}

func ComputeNewName(oldName, input string, isDir bool, includeExtension bool) string {
	newName := strings.TrimSpace(input)
	if !includeExtension && !isDir {
		ext := filepath.Ext(oldName)
		if ext != "" && !strings.HasSuffix(newName, ext) {
			newName += ext
		}
	}
	return newName
}

