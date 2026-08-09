package filehandlers

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type OrganizerResult struct {
	Groups      map[string][]string
	MovedCount  int
	FolderCount int
}

func RunFileOrganizer(dryRun bool) (*OrganizerResult, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(currentDir)
	if err != nil {
		return nil, err
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
		return &OrganizerResult{
			Groups:      filteredGroups,
			FolderCount: len(filteredGroups),
		}, nil
	}

	movedCount := 0
	folderCount := 0
	for base, files := range filteredGroups {
		basePath := filepath.Join(currentDir, base)
		if err := os.MkdirAll(basePath, 0755); err != nil {
			return nil, err
		}
		for _, filename := range files {
			srcPath := filepath.Join(currentDir, filename)
			dstPath := filepath.Join(basePath, filename)
			if err := os.Rename(srcPath, dstPath); err != nil {
				return nil, err
			}
			movedCount++
		}
		folderCount++
	}
	return &OrganizerResult{
		Groups:      filteredGroups,
		MovedCount:  movedCount,
		FolderCount: folderCount,
	}, nil
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
