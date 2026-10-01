package fssync

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type Callbacks struct {
	OnInfo        func(msg string)
	OnGeneric     func(msg string)
	OnItemSuccess func(msg string)
	OnWarn        func(msg string, err error)
	OnSuccess     func(msg string)
	OnError       func(msg string, err error)
}

func (cb Callbacks) info(msg string) {
	if cb.OnInfo != nil {
		cb.OnInfo(msg)
	}
}

func (cb Callbacks) generic(msg string) {
	if cb.OnGeneric != nil {
		cb.OnGeneric(msg)
	}
}

func (cb Callbacks) itemSuccess(msg string) {
	if cb.OnItemSuccess != nil {
		cb.OnItemSuccess(msg)
	}
}

func (cb Callbacks) warn(msg string, err error) {
	if cb.OnWarn != nil {
		cb.OnWarn(msg, err)
	}
}

func (cb Callbacks) success(msg string) {
	if cb.OnSuccess != nil {
		cb.OnSuccess(msg)
	}
}

func (cb Callbacks) err(msg string, err error) {
	if cb.OnError != nil {
		cb.OnError(msg, err)
	}
}

type ManifestResponse struct {
	Files map[string]string `json:"files"`
}

type FileRequest struct {
	Paths []string `json:"paths"`
}

type FilesResponse struct {
	Files []FileContent `json:"files"`
}

type FileContent struct {
	Path    string `json:"path"`
	Content []byte `json:"content"`
}

type ModeResponse struct {
	Mode string `json:"mode"`
}

type UploadRequest struct {
	Files    []FileContent `json:"files"`
	ToDelete []string      `json:"to_delete,omitzero"`
}

type PathIgnorer struct {
	patterns []string
}

func NewPathIgnorer(ignoreStr string) *PathIgnorer {
	if ignoreStr == "" {
		return &PathIgnorer{patterns: []string{}}
	}
	parts := strings.Split(ignoreStr, ",")
	patterns := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			patterns = append(patterns, trimmed)
		}
	}
	return &PathIgnorer{patterns: patterns}
}

func (pi *PathIgnorer) IsIgnored(path string) bool {
	for _, pattern := range pi.patterns {
		if matched, _ := filepath.Match(pattern, filepath.Base(path)); matched {
			return true
		}
		if strings.Contains(path, pattern) {
			return true
		}
	}
	return false
}

func BuildManifest(rootDir string, ignorer *PathIgnorer) (map[string]string, error) {
	manifest := make(map[string]string)
	err := filepath.WalkDir(rootDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		relPath, err := filepath.Rel(rootDir, path)
		if err != nil {
			return err
		}
		if relPath == "." {
			return nil
		}
		relPath = filepath.ToSlash(relPath)
		if ignorer != nil && ignorer.IsIgnored(relPath) {
			return nil
		}
		hash, err := computeFileHash(path)
		if err != nil {
			return err
		}
		manifest[relPath] = hash
		return nil
	})
	return manifest, err
}

func computeFileHash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
