package utils

import (
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
