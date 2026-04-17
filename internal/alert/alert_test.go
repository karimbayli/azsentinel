package alert

import (
	"testing"
)

func TestIsRecovery(t *testing.T) {
	tests := []struct {
		name     string
		prev     string
		curr     string
		expected bool
	}{
		// Recovery cases
		{
			name:     "Recovery from MAJOR_OUTAGE to PARTIAL_OUTAGE",
			prev:     "MAJOR_OUTAGE",
			curr:     "PARTIAL_OUTAGE",
			expected: true,
		},
		{
			name:     "Recovery from MAJOR_OUTAGE to DEGRADED",
			prev:     "MAJOR_OUTAGE",
			curr:     "DEGRADED",
			expected: true,
		},
		{
			name:     "Recovery from MAJOR_OUTAGE to HEALTHY",
			prev:     "MAJOR_OUTAGE",
			curr:     "HEALTHY",
			expected: true,
		},
		{
			name:     "Recovery from PARTIAL_OUTAGE to DEGRADED",
			prev:     "PARTIAL_OUTAGE",
			curr:     "DEGRADED",
			expected: true,
		},
		{
			name:     "Recovery from PARTIAL_OUTAGE to HEALTHY",
			prev:     "PARTIAL_OUTAGE",
			curr:     "HEALTHY",
			expected: true,
		},
		{
			name:     "Recovery from DEGRADED to HEALTHY",
			prev:     "DEGRADED",
			curr:     "HEALTHY",
			expected: true,
		},
		// Non-recovery / degradation cases
		{
			name:     "Degradation from HEALTHY to DEGRADED",
			prev:     "HEALTHY",
			curr:     "DEGRADED",
			expected: false,
		},
		{
			name:     "Degradation from HEALTHY to MAJOR_OUTAGE",
			prev:     "HEALTHY",
			curr:     "MAJOR_OUTAGE",
			expected: false,
		},
		{
			name:     "Degradation from DEGRADED to PARTIAL_OUTAGE",
			prev:     "DEGRADED",
			curr:     "PARTIAL_OUTAGE",
			expected: false,
		},
		// Identical states
		{
			name:     "Same state MAJOR_OUTAGE",
			prev:     "MAJOR_OUTAGE",
			curr:     "MAJOR_OUTAGE",
			expected: false,
		},
		{
			name:     "Same state HEALTHY",
			prev:     "HEALTHY",
			curr:     "HEALTHY",
			expected: false,
		},
		// Edge cases / Unknown states
		// A non-existent state will yield 0 in the map.
		// "UNKNOWN" (0) < "MAJOR_OUTAGE" (3) -> true
		{
			name:     "Unknown current state (defaults to 0, acts as HEALTHY)",
			prev:     "MAJOR_OUTAGE",
			curr:     "UNKNOWN",
			expected: true,
		},
		// "MAJOR_OUTAGE" (3) < "UNKNOWN" (0) -> false
		{
			name:     "Unknown previous state (defaults to 0)",
			prev:     "UNKNOWN",
			curr:     "MAJOR_OUTAGE",
			expected: false,
		},
		// "UNKNOWN" (0) < "UNKNOWN" (0) -> false
		{
			name:     "Both states unknown",
			prev:     "UNKNOWN",
			curr:     "UNKNOWN",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isRecovery(tt.prev, tt.curr)
			if result != tt.expected {
				t.Errorf("isRecovery(%q, %q) = %v; want %v", tt.prev, tt.curr, result, tt.expected)
			}
		})
	}
}
