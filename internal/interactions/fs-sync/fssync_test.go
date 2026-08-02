package fssync

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPathIgnorerIsIgnored(t *testing.T) {
	tests := []struct {
		name      string
		ignoreStr string
		path      string
		want      bool
	}{
		{"no patterns", "", "anything", false},
		{"exact basename glob match", "*.log", "dir/app.log", true},
		{"substring match", ".git,node_modules", "node_modules/pkg/index.js", true},
		{"no match", "*.log", "dir/app.txt", false},
		{"whitespace trimmed patterns", " .git , node_modules ", ".git/config", true},
		{"all-empty pattern string ignored", " , ,", "anything", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pi := NewPathIgnorer(tt.ignoreStr)
			if got := pi.IsIgnored(tt.path); got != tt.want {
				t.Errorf("IsIgnored(%q) with ignore=%q = %v, want %v", tt.path, tt.ignoreStr, got, tt.want)
			}
		})
	}
}

func TestBuildManifest(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hello"), 0644); err != nil {
		t.Fatalf("failed to write fixture: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "sub"), 0755); err != nil {
		t.Fatalf("failed to mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "sub", "b.txt"), []byte("world"), 0644); err != nil {
		t.Fatalf("failed to write fixture: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "ignored.log"), []byte("noise"), 0644); err != nil {
		t.Fatalf("failed to write fixture: %v", err)
	}

	manifest, err := BuildManifest(dir, NewPathIgnorer("*.log"))
	if err != nil {
		t.Fatalf("BuildManifest() error = %v", err)
	}

	if _, ok := manifest["ignored.log"]; ok {
		t.Error("expected ignored.log to be excluded from manifest")
	}
	hashA, ok := manifest["a.txt"]
	if !ok || hashA == "" {
		t.Errorf("expected a.txt in manifest with a hash, got %q (ok=%v)", hashA, ok)
	}
	hashB, ok := manifest[filepath.ToSlash(filepath.Join("sub", "b.txt"))]
	if !ok || hashB == "" {
		t.Errorf("expected sub/b.txt in manifest with a hash, got %q (ok=%v)", hashB, ok)
	}
	if hashA == hashB {
		t.Error("distinct file contents produced the same hash")
	}

	manifest2, err := BuildManifest(dir, NewPathIgnorer("*.log"))
	if err != nil {
		t.Fatalf("BuildManifest() second run error = %v", err)
	}
	if manifest2["a.txt"] != hashA {
		t.Error("hash for identical content changed between runs")
	}
}

func TestCompareManifests(t *testing.T) {
	c := &Client{}
	tests := []struct {
		name        string
		server      map[string]string
		local       map[string]string
		wantRequest []string
		wantDelete  []string
	}{
		{
			name:        "identical manifests",
			server:      map[string]string{"a.txt": "h1"},
			local:       map[string]string{"a.txt": "h1"},
			wantRequest: nil,
			wantDelete:  nil,
		},
		{
			name:        "server has new file",
			server:      map[string]string{"a.txt": "h1", "b.txt": "h2"},
			local:       map[string]string{"a.txt": "h1"},
			wantRequest: []string{"b.txt"},
			wantDelete:  nil,
		},
		{
			name:        "local has extra file",
			server:      map[string]string{"a.txt": "h1"},
			local:       map[string]string{"a.txt": "h1", "extra.txt": "h3"},
			wantRequest: nil,
			wantDelete:  []string{"extra.txt"},
		},
		{
			name:        "changed hash requests re-fetch",
			server:      map[string]string{"a.txt": "h1-new"},
			local:       map[string]string{"a.txt": "h1-old"},
			wantRequest: []string{"a.txt"},
			wantDelete:  nil,
		},
		{
			name:        "empty manifests",
			server:      map[string]string{},
			local:       map[string]string{},
			wantRequest: nil,
			wantDelete:  nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			toRequest, toDelete := c.compareManifests(tt.server, tt.local)
			if !equalSets(toRequest, tt.wantRequest) {
				t.Errorf("toRequest = %v, want %v", toRequest, tt.wantRequest)
			}
			if !equalSets(toDelete, tt.wantDelete) {
				t.Errorf("toDelete = %v, want %v", toDelete, tt.wantDelete)
			}
		})
	}
}

func equalSets(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	seen := make(map[string]bool, len(a))
	for _, v := range a {
		seen[v] = true
	}
	for _, v := range b {
		if !seen[v] {
			return false
		}
	}
	return true
}
