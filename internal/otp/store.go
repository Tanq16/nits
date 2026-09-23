package otp

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"time"
)

func storePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".config", "nits")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	return filepath.Join(dir, "otp.json"), nil
}

func load() (map[string]Entry, error) {
	path, err := storePath()
	if err != nil {
		return nil, err
	}
	entries := map[string]Entry{}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return entries, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}

func save(entries map[string]Entry) error {
	path, err := storePath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

func Add(name, input string) error {
	entry, err := Parse(input)
	if err != nil {
		return err
	}
	entries, err := load()
	if err != nil {
		return err
	}
	if _, exists := entries[name]; exists {
		return fmt.Errorf("%q already exists", name)
	}
	entry.CreatedAt = time.Now()
	entries[name] = entry
	return save(entries)
}

func Get(name string) (string, error) {
	entries, err := load()
	if err != nil {
		return "", err
	}
	entry, ok := entries[name]
	if !ok {
		return "", fmt.Errorf("%q not found", name)
	}
	return entry.Code(time.Now())
}

func List() ([]string, map[string]Entry, error) {
	entries, err := load()
	if err != nil {
		return nil, nil, err
	}
	return slices.Sorted(maps.Keys(entries)), entries, nil
}

func Delete(name string) error {
	entries, err := load()
	if err != nil {
		return err
	}
	if _, ok := entries[name]; !ok {
		return fmt.Errorf("%q not found", name)
	}
	delete(entries, name)
	return save(entries)
}
