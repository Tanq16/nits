package generics

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/goccy/go-yaml"
)

func TestFormatTimeAgo(t *testing.T) {
	tests := []struct {
		name string
		d    time.Duration
		want string
	}{
		{"zero", 0, "just now"},
		{"negative (clock skew)", -5 * time.Minute, "just now"},
		{"seconds", 30 * time.Second, "just now"},
		{"minutes", 5 * time.Minute, "5m ago"},
		{"just under an hour", 59 * time.Minute, "59m ago"},
		{"hours", 3 * time.Hour, "3h ago"},
		{"days", 2 * 24 * time.Hour, "2d ago"},
		{"weeks", 10 * 24 * time.Hour, "1w ago"},
		{"months", 45 * 24 * time.Hour, "1mo ago"},
		{"years", 400 * 24 * time.Hour, "1y ago"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatTimeAgo(tt.d); got != tt.want {
				t.Errorf("formatTimeAgo(%v) = %q, want %q", tt.d, got, tt.want)
			}
		})
	}
}

func TestConvertData(t *testing.T) {
	dir := t.TempDir()
	composeFile := filepath.Join(dir, "compose.yaml")
	if err := os.WriteFile(composeFile, []byte("services:\n  app:\n    image: nginx\n"), 0644); err != nil {
		t.Fatalf("failed to write fixture: %v", err)
	}

	tests := []struct {
		name          string
		converterType string
		input         string
		wantErr       bool
	}{
		{"unsupported converter", "does-not-exist", "x", true},
		{"empty string input", "url", "", true},
		{"file converter missing file", "compose-docker", filepath.Join(dir, "missing.yaml"), true},
		{"valid string converter", "url", "hello world", false},
		{"valid file converter", "compose-docker", composeFile, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ConvertData(tt.converterType, tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ConvertData(%q, %q) err = %v, wantErr %v", tt.converterType, tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestJwtDecodeSegment(t *testing.T) {
	tests := []struct {
		name    string
		seg     string
		wantErr bool
	}{
		{"valid header, no padding needed", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9", false},
		{"valid payload, requires padding", "eyJzdWIiOiIxMjM0NTY3ODkwIn0", false},
		{"invalid base64", "not!!valid==base64", true},
		{"valid base64, invalid json", "bm90IGpzb24", true},
		{"empty segment", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := jwtDecodeSegment(tt.seg)
			if (err != nil) != tt.wantErr {
				t.Errorf("jwtDecodeSegment(%q) err = %v, wantErr %v", tt.seg, err, tt.wantErr)
			}
		})
	}
}

func TestJwtDecode(t *testing.T) {
	tests := []struct {
		name    string
		token   string
		wantErr bool
	}{
		{"too few parts", "onlyonepart", true},
		{"too many parts", "a.b.c.d", true},
		{"valid token", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.sig", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := jwtDecode(tt.token); (err != nil) != tt.wantErr {
				t.Errorf("jwtDecode(%q) err = %v, wantErr %v", tt.token, err, tt.wantErr)
			}
		})
	}
}

func TestFormatJWTValue(t *testing.T) {
	tests := []struct {
		name string
		in   any
		want string
	}{
		{"float64 whole number", float64(1516239022), "1516239022"},
		{"int64", int64(42), "42"},
		{"string", "hello", "hello"},
		{"bool", true, "true"},
		{"nil", nil, "<nil>"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatJWTValue(tt.in); got != tt.want {
				t.Errorf("formatJWTValue(%v) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestSplitCommand(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []string
	}{
		{"empty", "", nil},
		{"simple flags", " -d --name foo", []string{"-d", "--name", "foo"}},
		{"double quoted value with space", ` -e "KEY=some value"`, []string{"-e", "KEY=some value"}},
		{"single quoted value", ` -v 'a:b'`, []string{"-v", "a:b"}},
		{"unterminated quote keeps remainder", ` -e "unterminated`, []string{"-e", "unterminated"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitCommand(tt.in)
			if len(got) != len(tt.want) {
				t.Fatalf("splitCommand(%q) = %v, want %v", tt.in, got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("splitCommand(%q)[%d] = %q, want %q", tt.in, i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestConvertDockerToCompose(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	if err := convertDockerToCompose("not a docker command"); err == nil {
		t.Error("expected error for non 'docker run' prefixed input")
	}

	if err := convertDockerToCompose(`docker run -d --name web -p 8080:80 -e FOO=bar nginx`); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, err := os.ReadFile("docker-compose.yml")
	if err != nil {
		t.Fatalf("expected docker-compose.yml to be written: %v", err)
	}
	var cfg map[string]any
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("unmarshal compose: %v", err)
	}
	services, ok := cfg["services"].(map[string]any)
	if !ok {
		t.Fatalf("services missing: %v", cfg)
	}
	if _, hasApp := services["app"]; hasApp {
		t.Errorf("stale 'app' service should be deleted when --name is set; got %v", services)
	}
	web, ok := services["web"].(map[string]any)
	if !ok {
		t.Fatalf("expected 'web' service, got %v", services)
	}
	if web["image"] != "nginx" {
		t.Errorf("web.image = %v, want nginx", web["image"])
	}
}

func TestConvertDockerToComposeBooleanFlags(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	if err := convertDockerToCompose(`docker run -it --rm nginx bash`); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, err := os.ReadFile("docker-compose.yml")
	if err != nil {
		t.Fatalf("read compose: %v", err)
	}
	var cfg map[string]any
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	app := cfg["services"].(map[string]any)["app"].(map[string]any)
	if app["image"] != "nginx" {
		t.Errorf("image = %v, want nginx (boolean flags must not swallow the image)", app["image"])
	}
	if app["command"] != "bash" {
		t.Errorf("command = %v, want bash", app["command"])
	}
}

func TestResolveUnderRoot(t *testing.T) {
	root := t.TempDir()
	s := &MarkdownServer{Options: &MarkdownServerOptions{RootDir: root}}

	inside := filepath.Join(root, "doc.md")
	if err := os.WriteFile(inside, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}

	got, ok := s.resolveUnderRoot("doc.md")
	if !ok || got != inside {
		t.Errorf("resolveUnderRoot(doc.md) = %q, %v; want %q, true", got, ok, inside)
	}

	sibling := filepath.Join(filepath.Dir(root), filepath.Base(root)+"-private")
	if err := os.MkdirAll(sibling, 0755); err != nil {
		t.Fatal(err)
	}
	secret := filepath.Join(sibling, "secret.md")
	if err := os.WriteFile(secret, []byte("secret"), 0644); err != nil {
		t.Fatal(err)
	}

	escaped := "../" + filepath.Base(root) + "-private/secret.md"
	if got, ok := s.resolveUnderRoot(escaped); ok {
		t.Errorf("resolveUnderRoot(%q) = %q, true; want rejected", escaped, got)
	}
	if got, ok := s.resolveUnderRoot("../../../etc/passwd"); ok {
		t.Errorf("resolveUnderRoot(../../../etc/passwd) = %q, true; want rejected", got)
	}
}

func TestNextTaskID(t *testing.T) {
	tests := []struct {
		name  string
		store *TaskStore
		want  int
	}{
		{"empty store", &TaskStore{}, 1},
		{"single task", &TaskStore{Tasks: []TaskEntry{{ID: 5}}}, 6},
		{"non-sequential ids", &TaskStore{Tasks: []TaskEntry{{ID: 1}, {ID: 10}, {ID: 3}}}, 11},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := nextTaskID(tt.store); got != tt.want {
				t.Errorf("nextTaskID() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestFindAndRemoveTaskByID(t *testing.T) {
	store := &TaskStore{Tasks: []TaskEntry{{ID: 1, Task: "a"}, {ID: 2, Task: "b"}}}

	if task := findTaskByID(store, 2); task == nil || task.Task != "b" {
		t.Errorf("findTaskByID(2) = %v, want task 'b'", task)
	}
	if task := findTaskByID(store, 99); task != nil {
		t.Errorf("findTaskByID(99) = %v, want nil", task)
	}

	if !removeTaskByID(store, 1) {
		t.Error("removeTaskByID(1) = false, want true")
	}
	if len(store.Tasks) != 1 || store.Tasks[0].ID != 2 {
		t.Errorf("after removal, store.Tasks = %v, want only ID 2 remaining", store.Tasks)
	}
	if removeTaskByID(store, 1) {
		t.Error("removeTaskByID(1) on already-removed task = true, want false")
	}
}
