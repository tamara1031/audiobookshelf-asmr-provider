package provider

import (
	"testing"
)

func TestNewAll(t *testing.T) {
	providers := NewAll()
	if len(providers) == 0 {
		t.Errorf("expected at least one provider, got 0")
	}

	hasDLsite := false
	for _, p := range providers {
		if p.ID() == "dlsite" {
			hasDLsite = true
			break
		}
	}
	if !hasDLsite {
		t.Errorf("expected dlsite provider to be registered")
	}
}
