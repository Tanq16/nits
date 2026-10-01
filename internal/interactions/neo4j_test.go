package interactions

import (
	"os"
	"path/filepath"
	"strings"
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
		wantErr  string
	}{
		{"missing file", filepath.Join(dir, "does-not-exist.yaml"), "failed to read query file"},
		{"empty query list", emptyFile, "no queries found"},
		{"malformed yaml", malformedFile, "failed to parse YAML"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ExecuteNeo4jQueriesFromFile(t.Context(), "neo4j://localhost:7687", "neo4j", "pass", "neo4j", tt.filePath, false)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("ExecuteNeo4jQueriesFromFile(%q) err = %v, want error containing %q", tt.filePath, err, tt.wantErr)
			}
		})
	}
}
