package filehandlers

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

func TestExtractZip(t *testing.T) {
	tempDir := t.TempDir()
	zipFile := filepath.Join(tempDir, "test.zip")
	destDir := filepath.Join(tempDir, "extracted")

	// Create a test zip file
	f, err := os.Create(zipFile)
	if err != nil {
		t.Fatalf("failed to create zip file: %v", err)
	}
	zw := zip.NewWriter(f)

	w, err := zw.Create("hello.txt")
	if err != nil {
		t.Fatalf("failed to create entry in zip: %v", err)
	}
	if _, err := w.Write([]byte("hello world")); err != nil {
		t.Fatalf("failed to write content: %v", err)
	}

	if err := zw.Close(); err != nil {
		t.Fatalf("failed to close zip writer: %v", err)
	}
	f.Close()

	// Extract zip
	if err := extractZip(zipFile, destDir); err != nil {
		t.Fatalf("extractZip failed: %v", err)
	}

	// Verify extracted file
	content, err := os.ReadFile(filepath.Join(destDir, "hello.txt"))
	if err != nil {
		t.Fatalf("failed to read extracted file: %v", err)
	}
	if string(content) != "hello world" {
		t.Errorf("expected 'hello world', got '%s'", string(content))
	}
}

func TestExtractZipSlipRejected(t *testing.T) {
	tempDir := t.TempDir()
	zipFile := filepath.Join(tempDir, "slip.zip")
	destDir := filepath.Join(tempDir, "extracted")

	f, err := os.Create(zipFile)
	if err != nil {
		t.Fatalf("failed to create zip file: %v", err)
	}
	zw := zip.NewWriter(f)

	// Attempt path traversal outside destDir
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
