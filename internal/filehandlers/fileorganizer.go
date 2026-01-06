package filehandlers

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/rs/zerolog/log"
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
	log.Info().Int("folders", len(filteredGroups)).Int("files", movedCount).Msg("Organized files")
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
	log.Info().Int("groups", len(groups)).Msg("Found groups to create")
	for base, files := range groups {
		log.Info().Str("folder", base).Int("files", len(files)).Msg("Would create folder")
		displayCount := min(len(files), 5)
		for i := range displayCount {
			log.Info().Str("file", files[i]).Msg("  Would move")
		}
		if len(files) > displayCount {
			log.Info().Int("remaining", len(files)-displayCount).Msg("  ... and more")
		}
	}
}
