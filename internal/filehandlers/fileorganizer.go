package filehandlers

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/tanq16/nits/internal/utils"
)

func RunFileOrganizer(dryRun bool) {
	currentDir, _ := os.Getwd()
	entries, _ := os.ReadDir(currentDir)
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
		return
	}
	movedCount := 0
	for base, files := range filteredGroups {
		basePath := filepath.Join(currentDir, base)
		os.MkdirAll(basePath, 0755)
		for _, filename := range files {
			srcPath := filepath.Join(currentDir, filename)
			dstPath := filepath.Join(basePath, filename)
			os.Rename(srcPath, dstPath)
			movedCount++
		}
	}
	utils.PrintSuccess(fmt.Sprintf("Organized %d files into %d folders", movedCount, len(filteredGroups)))
	log.Debug().Str("package", "filehandlers").Int("folders", len(filteredGroups)).Int("files", movedCount).Msg("Organized")
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
	log.Debug().Str("package", "filehandlers").Int("groups", len(groups)).Msg("Dry run")
}
