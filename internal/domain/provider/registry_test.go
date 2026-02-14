package provider

import (
	"testing"
)

func TestNewAll_ReturnsProviders(t *testing.T) {
	providers := NewAll()

	if len(providers) == 0 {
		t.Fatal("NewAll() returned no providers")
	}

	for i, p := range providers {
		if p == nil {
			t.Errorf("provider at index %d is nil", i)
		}
		if p.ID() == "" {
			t.Errorf("provider at index %d has empty ID", i)
		}
	}
}
