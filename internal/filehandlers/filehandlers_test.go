package filehandlers

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

func TestExtractZipSlipRejected(t *testing.T) {
	tempDir := t.TempDir()
	zipFile := filepath.Join(tempDir, "slip.zip")
	destDir := filepath.Join(tempDir, "extracted")

	f, err := os.Create(zipFile)
	if err != nil {
		t.Fatalf("failed to create zip file: %v", err)
	}
	zw := zip.NewWriter(f)

	w, err := zw.Create("../outside.txt")
	if err != nil {
		t.Fatalf("failed to create entry: %v", err)
	}
	w.Write([]byte("malicious"))
	zw.Close()
	f.Close()

	if err := extractZip(zipFile, destDir); err != nil {
		t.Fatalf("extractZip returned error on slip: %v", err)
	}

	if _, err := os.Stat(filepath.Join(tempDir, "outside.txt")); err == nil {
		t.Errorf("zip slip file was written outside target destination!")
	}
}
