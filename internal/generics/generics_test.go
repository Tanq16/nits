package generics

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
		{"valid string converter", "url", "hello world", false},
		{"valid file converter", "compose-docker", composeFile, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ConvertData(tt.converterType, tt.input)
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
			if _, err := jwtDecode(tt.token); (err != nil) != tt.wantErr {
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

func TestGenerateRandomString(t *testing.T) {
	tests := []struct {
		name       string
		length     int
		wantLength int
	}{
		{"default length", 0, 100},
		{"negative length", -5, 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GenerateRandomString(tt.length)
			if err != nil {
				t.Fatalf("GenerateRandomString(%d) error = %v", tt.length, err)
			}
			if len(got) != tt.wantLength {
				t.Errorf("GenerateRandomString(%d) len = %d, want %d", tt.length, len(got), tt.wantLength)
			}
		})
	}
}

func TestGenerateRUIDString(t *testing.T) {
	tests := []struct {
		name       string
		length     int
		wantLength int
	}{
		{"default for invalid <=0", 0, 18},
		{"default for invalid >30", 35, 18},
		{"max length 30", 30, 30},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GenerateRUIDString(tt.length)
			if err != nil {
				t.Fatalf("GenerateRUIDString(%d) error = %v", tt.length, err)
			}
			if len(got) != tt.wantLength {
				t.Errorf("GenerateRUIDString(%d) len = %d, want %d", tt.length, len(got), tt.wantLength)
			}
		})
	}
}

func TestGeneratePassPhrase(t *testing.T) {
	t.Run("clamp below 1", func(t *testing.T) {
		phrase, err := GeneratePassPhrase(0, true)
		if err != nil {
			t.Fatalf("GeneratePassPhrase error = %v", err)
		}
		parts := strings.Split(phrase, "-")
		if len(parts) != 3 {
			t.Errorf("got %d parts, want 3 in %q", len(parts), phrase)
		}
	})

	t.Run("clamp above 50", func(t *testing.T) {
		phrase, err := GeneratePassPhrase(51, true)
		if err != nil {
			t.Fatalf("GeneratePassPhrase error = %v", err)
		}
		parts := strings.Split(phrase, "-")
		if len(parts) != 3 {
			t.Errorf("got %d parts, want 3 in %q", len(parts), phrase)
		}
	})

	t.Run("simple has no capital and no digit", func(t *testing.T) {
		phrase, err := GeneratePassPhrase(3, true)
		if err != nil {
			t.Fatalf("GeneratePassPhrase error = %v", err)
		}
		parts := strings.Split(phrase, "-")
		if len(parts) != 3 {
			t.Errorf("got %d parts, want 3 in %q", len(parts), phrase)
		}
		for _, part := range parts {
			if part == "" {
				t.Fatalf("empty part in %q", phrase)
			}
			for _, r := range part {
				if r >= 'A' && r <= 'Z' {
					t.Errorf("simple phrase %q has a capital in %q", phrase, part)
				}
				if r >= '0' && r <= '9' {
					t.Errorf("simple phrase %q has a digit in %q", phrase, part)
				}
			}
		}
	})

	t.Run("default is one capital and one digit across the phrase", func(t *testing.T) {
		phrase, err := GeneratePassPhrase(3, false)
		if err != nil {
			t.Fatalf("GeneratePassPhrase error = %v", err)
		}
		parts := strings.Split(phrase, "-")
		if len(parts) != 3 {
			t.Errorf("got %d parts, want 3 in %q", len(parts), phrase)
		}
		capCount := 0
		digitCount := 0
		for _, part := range parts {
			if part == "" {
				t.Fatalf("empty part in %q", phrase)
			}
			if part[0] >= 'A' && part[0] <= 'Z' {
				capCount++
			}
			last := part[len(part)-1]
			if last >= '0' && last <= '9' {
				digitCount++
			}
		}
		if capCount != 1 {
			t.Errorf("got %d capitalized words, want 1 in %q", capCount, phrase)
		}
		if digitCount != 1 {
			t.Errorf("got %d words ending in a digit, want 1 in %q", digitCount, phrase)
		}
	})
}
