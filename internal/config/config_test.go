package config

import (
	"os"
	"testing"
)

func TestExpandEnv(t *testing.T) {
	// Set up environment variables for testing
	os.Setenv("TEST_VAR1", "value1")
	os.Setenv("TEST_VAR2", "")
	defer os.Unsetenv("TEST_VAR1")
	defer os.Unsetenv("TEST_VAR2")

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Basic expansion",
			input:    "Hello ${TEST_VAR1}",
			expected: "Hello value1",
		},
		{
			name:     "Bash-style default, var set",
			input:    "Hello ${TEST_VAR1:-default}",
			expected: "Hello value1",
		},
		{
			name:     "Bash-style default, var unset",
			input:    "Hello ${UNSET_VAR:-default}",
			expected: "Hello default",
		},
		{
			name:     "Bash-style default, var empty",
			input:    "Hello ${TEST_VAR2:-default}",
			expected: "Hello default",
		},
		{
			name:     "Multiple variables",
			input:    "${TEST_VAR1} and ${UNSET_VAR:-default2}",
			expected: "value1 and default2",
		},
		{
			name:     "No variables",
			input:    "Hello world",
			expected: "Hello world",
		},
		{
			name:     "Unset variable without default",
			input:    "Hello ${UNSET_VAR}",
			expected: "Hello ",
		},
		{
			name:     "Default value is empty",
			input:    "Hello ${UNSET_VAR:-}",
			expected: "Hello ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := expandEnv(tt.input)
			if result != tt.expected {
				t.Errorf("expandEnv(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}
