package slack

import (
	"testing"
)

// TestToStringPtr verifies that toStringPtr returns a non-nil pointer
// to a copy of the input string value.
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
			got := toStringPtr(tt.input)
			if got == nil {
				t.Fatalf("toStringPtr(%q) = nil; want non-nil pointer", tt.input)
			}
			if *got != tt.input {
				t.Errorf("*toStringPtr(%q) = %q; want %q", tt.input, *got, tt.input)
			}
		})
	}
}

// TestToStringPtr_IsCopy verifies that modifying the returned pointer's value
// does not affect the original variable (i.e. it is an independent copy).
func TestToStringPtr_IsCopy(t *testing.T) {
	original := "original"
	ptr := toStringPtr(original)
	*ptr = "modified"
	if original == "modified" {
		t.Error("toStringPtr did not return an independent copy; modifying *ptr changed the original")
	}
}

// TestToInt64Ptr verifies that toInt64Ptr returns a non-nil pointer
// to a copy of the input int64 value.
func TestToInt64Ptr(t *testing.T) {
	tests := []struct {
		name  string
		input int64
	}{
		{"positive value", 42},
		{"zero", 0},
		{"negative value", -1},
		{"large value", 9223372036854775807},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toInt64Ptr(tt.input)
			if got == nil {
				t.Fatalf("toInt64Ptr(%d) = nil; want non-nil pointer", tt.input)
			}
			if *got != tt.input {
				t.Errorf("*toInt64Ptr(%d) = %d; want %d", tt.input, *got, tt.input)
			}
		})
	}
}

// TestToInt64Ptr_IsCopy verifies that modifying the returned pointer's value
// does not affect the original variable.
func TestToInt64Ptr_IsCopy(t *testing.T) {
	var original int64 = 100
	ptr := toInt64Ptr(original)
	*ptr = 999
	if original == 999 {
		t.Error("toInt64Ptr did not return an independent copy; modifying *ptr changed the original")
	}
}
