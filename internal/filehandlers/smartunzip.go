package filehandlers

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

func RunFileUnzipper(uuidNames bool) {
	currentDir, _ := os.Getwd()
	entries, _ := os.ReadDir(currentDir)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".zip") {
			continue
		}
		base := strings.TrimSuffix(name, ".zip")
		if uuidNames {
			base = generateUUID()
		}
		basePath := filepath.Join(currentDir, base)
		log.Info().Str("zip", name).Str("directory", base).Msg("Processing zip file")
		os.MkdirAll(basePath, 0755)
		zipPath := filepath.Join(currentDir, name)
		newZipPath := filepath.Join(basePath, name)
		os.Rename(zipPath, newZipPath)
		err := extractZip(newZipPath, basePath)
		if err != nil {
			log.Error().Err(err).Str("zip", name).Msg("Failed to extract zip file")
			continue
		}
		os.Remove(newZipPath)
		if uuidNames {
			renameToUUIDs(basePath)
		}
		subEntries, _ := os.ReadDir(basePath)
		var visibleFiles []string
		for _, subEntry := range subEntries {
			if !strings.HasPrefix(subEntry.Name(), ".") {
				visibleFiles = append(visibleFiles, subEntry.Name())
			}
		}
		if len(visibleFiles) == 1 {
			subdirPath := filepath.Join(basePath, visibleFiles[0])
			if info, _ := os.Stat(subdirPath); info != nil && info.IsDir() {
				log.Info().Str("directory", base).Str("subdirectory", visibleFiles[0]).Msg("Flattening single subdirectory")
				subEntries2, _ := os.ReadDir(subdirPath)
				for _, subEntry := range subEntries2 {
					os.Rename(filepath.Join(subdirPath, subEntry.Name()), filepath.Join(basePath, subEntry.Name()))
				}
				os.Remove(subdirPath)
			}
		}
	}
}

func extractZip(zipPath, destDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()
	for _, f := range r.File {
		fpath := filepath.Join(destDir, f.Name)
		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, f.Mode())
			continue
		}
		os.MkdirAll(filepath.Dir(fpath), 0755)
		outFile, _ := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		rc, _ := f.Open()
		io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()
	}
	return nil
}

func generateUUID() string {
	ret, _ := uuid.NewRandom()
	return ret.String()
}

func renameToUUIDs(dir string) {
	entries, _ := os.ReadDir(dir)
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		oldPath := filepath.Join(dir, entry.Name())
		var newName string
		if entry.IsDir() {
			newName = generateUUID()
		} else {
			ext := filepath.Ext(entry.Name())
			newName = generateUUID() + ext
		}
		newPath := filepath.Join(dir, newName)
		os.Rename(oldPath, newPath)
		if entry.IsDir() {
			renameToUUIDs(newPath)
		}
	}
}
