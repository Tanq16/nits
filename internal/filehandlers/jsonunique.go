package filehandlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func RunJSONUnique(targetFile, path, key string) error {
	absPath, _ := filepath.Abs(targetFile)
	data, err := os.ReadFile(absPath)
	if err != nil {
		return err
	}
	var root map[string]any
	if err := json.Unmarshal(data, &root); err != nil {
		return err
	}
	pathParts := strings.Split(path, ".")
	value, exists := getNestedValue(root, pathParts)
	if !exists {
		return errors.New("path not found in JSON")
	}
	slice, ok := value.([]any)
	if !ok {
		return errors.New("path does not point to a slice")
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
		return err
	}
	if err := os.WriteFile(absPath, output, 0644); err != nil {
		return err
	}
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
