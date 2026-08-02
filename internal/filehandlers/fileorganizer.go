package filehandlers

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/tanq16/nits/utils"
)

// RunFileOrganizer groups files in the current directory into subdirectories
// by shared base name. It returns the number of files moved and the number
// of folders created (or that would be created, in dry-run mode).
func RunFileOrganizer(dryRun bool) (movedCount int, folderCount int, err error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get current directory: %w", err)
	}
	entries, err := os.ReadDir(currentDir)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to read directory: %w", err)
	}
	groups := make(map[string][]string)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		filename := entry.Name()
		base := extractBaseName(filename)
		if base != "" {
			groups[base] = append(groups[base], filename)
		}
	}
	filteredGroups := make(map[string][]string)
	for base, files := range groups {
		if len(files) > 1 {
			filteredGroups[base] = files
		}
	}
	if dryRun {
		dryRunMode(filteredGroups)
		return 0, len(filteredGroups), nil
	}
	for base, files := range filteredGroups {
		basePath := filepath.Join(currentDir, base)
		if err := os.MkdirAll(basePath, 0755); err != nil {
			return movedCount, folderCount, fmt.Errorf("failed to create directory %s: %w", basePath, err)
		}
		for _, filename := range files {
			srcPath := filepath.Join(currentDir, filename)
			dstPath := filepath.Join(basePath, filename)
			if err := os.Rename(srcPath, dstPath); err != nil {
				return movedCount, folderCount, fmt.Errorf("failed to move %s: %w", filename, err)
			}
			movedCount++
		}
		folderCount++
	}
	return movedCount, folderCount, nil
}

func extractBaseName(filename string) string {
	name := strings.TrimSuffix(filename, filepath.Ext(filename))
	re := regexp.MustCompile(`[_\-.\s]+`)
	parts := re.Split(name, -1)
	if len(parts) > 0 && parts[0] != "" {
		return parts[0]
	}
	return name
}

func dryRunMode(groups map[string][]string) {
	utils.PrintInfo(fmt.Sprintf("Found %d groups to create", len(groups)))
	for base, files := range groups {
		utils.PrintGeneric(fmt.Sprintf("  %s/ (%d files)", base, len(files)))
		displayCount := min(len(files), 5)
		for i := range displayCount {
			utils.PrintGeneric(fmt.Sprintf("    - %s", files[i]))
		}
		if len(files) > displayCount {
			utils.PrintGeneric(fmt.Sprintf("    ... and %d more", len(files)-displayCount))
		}
	}
}
