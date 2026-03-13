package tmp_test

import (
	"testing"

	"github.com/TouchBistro/gotham/sql/qb/tmp"
)

func TestToStringPtr(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"non-empty string", "hello"},
		{"empty string", ""},
		{"string with spaces", "hello world"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tmp.ToStringPtr(tt.input)
			if got == nil {
				t.Fatal("ToStringPtr() returned nil, want non-nil pointer")
			}
			if *got != tt.input {
				t.Errorf("ToStringPtr(%q) = %q, want %q", tt.input, *got, tt.input)
			}
		})
	}
}

func TestToInt64Ptr(t *testing.T) {
	tests := []struct {
		name  string
		input int64
	}{
		{"positive value", 42},
		{"zero value", 0},
		{"negative value", -1},
		{"large value", 9223372036854775807},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tmp.ToInt64Ptr(tt.input)
			if got == nil {
				t.Fatal("ToInt64Ptr() returned nil, want non-nil pointer")
			}
			if *got != tt.input {
				t.Errorf("ToInt64Ptr(%d) = %d, want %d", tt.input, *got, tt.input)
			}
		})
	}
}
