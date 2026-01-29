package filehandlers

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog/log"
)

func RunJSONUnique(targetFile, path, key string) error {
	absPath, _ := filepath.Abs(targetFile)
	data, err := os.ReadFile(absPath)
	if err != nil {
		log.Debug().Str("package", "filehandlers").Err(err).Str("file", targetFile).Msg("Failed to read file")
		return fmt.Errorf("failed to read file: %w", err)
	}
	var root map[string]any
	if err := json.Unmarshal(data, &root); err != nil {
		log.Debug().Str("package", "filehandlers").Err(err).Str("file", targetFile).Msg("Failed to parse JSON")
		return fmt.Errorf("failed to parse JSON: %w", err)
	}
	pathParts := strings.Split(path, ".")
	value, exists := getNestedValue(root, pathParts)
	if !exists {
		log.Debug().Str("package", "filehandlers").Str("path", path).Msg("Path not found")
		return fmt.Errorf("path '%s' not found in JSON", path)
	}
	slice, ok := value.([]any)
	if !ok {
		log.Debug().Str("package", "filehandlers").Str("path", path).Msg("Path not a slice")
		return fmt.Errorf("path '%s' does not point to a slice", path)
	}
	keyParts := strings.Split(key, ".")
	seen := make(map[string]bool)
	var unique []any
	for _, item := range slice {
		itemMap, ok := item.(map[string]any)
		if !ok {
			unique = append(unique, item)
			continue
		}
		keyValue, exists := getNestedValue(itemMap, keyParts)
		if !exists {
			unique = append(unique, item)
			continue
		}
		key := fmt.Sprintf("%v", keyValue)
		if !seen[key] {
			seen[key] = true
			unique = append(unique, item)
		}
	}
	setNestedValue(root, pathParts, unique)
	output, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		log.Debug().Str("package", "filehandlers").Err(err).Msg("Failed to marshal JSON")
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	if err := os.WriteFile(absPath, output, 0644); err != nil {
		log.Debug().Str("package", "filehandlers").Err(err).Str("file", targetFile).Msg("Failed to write file")
		return fmt.Errorf("failed to write file: %w", err)
	}
	log.Debug().Str("package", "filehandlers").Int("original", len(slice)).Int("unique", len(unique)).Msg("Deduplicated")
	return nil
}

func getNestedValue(obj map[string]any, parts []string) (any, bool) {
	current := any(obj)
	for _, part := range parts {
		m, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		val, exists := m[part]
		if !exists {
			return nil, false
		}
		current = val
	}
	return current, true
}

func setNestedValue(obj map[string]any, parts []string, value any) {
	if len(parts) == 1 {
		obj[parts[0]] = value
		return
	}
	next, ok := obj[parts[0]].(map[string]any)
	if !ok {
		return
	}
	setNestedValue(next, parts[1:], value)
}
