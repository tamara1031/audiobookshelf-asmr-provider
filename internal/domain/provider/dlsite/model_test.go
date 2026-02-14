package dlsite

import (
	"testing"
)

func TestNewRJCode_Valid(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"RJ123456", "RJ123456"},
		{"rj123456", "RJ123456"},
		{"RJ12345678", "RJ12345678"},
		{"  RJ123456  ", "RJ123456"},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			rj, err := NewRJCode(tc.input)
			if err != nil {
				t.Fatalf("NewRJCode(%q) returned error: %v", tc.input, err)
			}
			if rj.String() != tc.want {
				t.Errorf("NewRJCode(%q).String() = %q, want %q", tc.input, rj.String(), tc.want)
			}
		})
	}
}

func TestNewRJCode_Invalid(t *testing.T) {
	tests := []string{
		"",
		"RJ12345",     // too short
		"RJ123456789", // too long
		"XX123456",    // wrong prefix
		"hello",       // not an RJ code
		"123456",      // missing prefix
	}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			_, err := NewRJCode(input)
			if err == nil {
				t.Errorf("NewRJCode(%q) expected error, got nil", input)
			}
		})
	}
}
