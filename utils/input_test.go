package utils

import (
	"bufio"
	"strings"
	"testing"
)

func TestPromptSelectEmptyOptions(t *testing.T) {
	idx, err := PromptSelect("Pick one", []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if idx != -1 {
		t.Errorf("expected -1 for empty options, got %d", idx)
	}
}

func TestPromptMultiSelectEmptyOptions(t *testing.T) {
	sel, err := PromptMultiSelect("Pick multiple", []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sel != nil {
		t.Errorf("expected nil for empty options, got %v", sel)
	}
}

func TestPromptSelectForAI(t *testing.T) {
	GlobalForAIFlag = true
	defer func() {
		GlobalForAIFlag = false
		stdinScanner = nil
	}()

	tests := []struct {
		name        string
		input       string
		options     []string
		expectedIdx int
	}{
		{
			name:        "valid 1-based index 1",
			input:       "1\n",
			options:     []string{"first", "second", "third"},
			expectedIdx: 0,
		},
		{
			name:        "valid 1-based index 3",
			input:       "3\n",
			options:     []string{"first", "second", "third"},
			expectedIdx: 2,
		},
		{
			name:        "out of range high",
			input:       "5\n",
			options:     []string{"first", "second", "third"},
			expectedIdx: -1,
		},
		{
			name:        "out of range zero",
			input:       "0\n",
			options:     []string{"first", "second", "third"},
			expectedIdx: -1,
		},
		{
			name:        "non-numeric",
			input:       "abc\n",
			options:     []string{"first", "second", "third"},
			expectedIdx: -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdinScanner = bufio.NewScanner(strings.NewReader(tt.input))
			idx, err := PromptSelect("Pick one", tt.options)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if idx != tt.expectedIdx {
				t.Errorf("expected index %d, got %d", tt.expectedIdx, idx)
			}
		})
	}
}

func TestPromptMultiSelectForAI(t *testing.T) {
	GlobalForAIFlag = true
	defer func() {
		GlobalForAIFlag = false
		stdinScanner = nil
	}()

	tests := []struct {
		name        string
		input       string
		options     []string
		expectedMap map[int]bool
	}{
		{
			name:        "single selection",
			input:       "2\n",
			options:     []string{"apple", "banana", "cherry"},
			expectedMap: map[int]bool{1: true},
		},
		{
			name:        "multiple selections",
			input:       "1,3\n",
			options:     []string{"apple", "banana", "cherry"},
			expectedMap: map[int]bool{0: true, 2: true},
		},
		{
			name:        "none selection",
			input:       "none\n",
			options:     []string{"apple", "banana", "cherry"},
			expectedMap: nil,
		},
		{
			name:        "empty line",
			input:       "\n",
			options:     []string{"apple", "banana", "cherry"},
			expectedMap: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdinScanner = bufio.NewScanner(strings.NewReader(tt.input))
			sel, err := PromptMultiSelect("Pick fruits", tt.options)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(sel) != len(tt.expectedMap) {
				t.Errorf("expected %v, got %v", tt.expectedMap, sel)
			}
			for k, v := range tt.expectedMap {
				if sel[k] != v {
					t.Errorf("expected key %d to be %v in %v", k, v, sel)
				}
			}
		})
	}
}
