package interactions

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExecuteNeo4jQueriesFromFile_EarlyExits(t *testing.T) {
	dir := t.TempDir()
	emptyFile := filepath.Join(dir, "empty.yaml")
	if err := os.WriteFile(emptyFile, []byte("[]\n"), 0644); err != nil {
		t.Fatalf("failed to write fixture: %v", err)
	}
	malformedFile := filepath.Join(dir, "malformed.yaml")
	if err := os.WriteFile(malformedFile, []byte("not: [valid, query, list"), 0644); err != nil {
		t.Fatalf("failed to write fixture: %v", err)
	}

	tests := []struct {
		name     string
		filePath string
	}{
		{"missing file", filepath.Join(dir, "does-not-exist.yaml")},
		{"empty query list", emptyFile},
		{"malformed yaml", malformedFile},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ExecuteNeo4jQueriesFromFile(t.Context(), "neo4j://localhost:7687", "neo4j", "pass", "neo4j", tt.filePath, false)
			if err == nil {
				t.Errorf("ExecuteNeo4jQueriesFromFile(%q) expected error, got nil", tt.filePath)
			}
		})
	}
}
