package utils

import (
	"errors"
	"testing"
)

func TestPromptEmptyOptions(t *testing.T) {
	saved := StdinIsTerminal
	StdinIsTerminal = true
	t.Cleanup(func() { StdinIsTerminal = saved })

	if idx, err := PromptSelect("Pick one", nil); !errors.Is(err, ErrNoOptions) || idx != -1 {
		t.Errorf("PromptSelect(nil) = %d, %v; want -1, ErrNoOptions", idx, err)
	}
	if sel, err := PromptMultiSelect("Pick multiple", nil); !errors.Is(err, ErrNoOptions) || sel != nil {
		t.Errorf("PromptMultiSelect(nil) = %v, %v; want nil, ErrNoOptions", sel, err)
	}
}
