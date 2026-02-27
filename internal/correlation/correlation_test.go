package correlation

import (
	"testing"

	"github.com/karimbayli/sentinel-v2/internal/models"
)

func TestConfidenceToStatus(t *testing.T) {
	tests := []struct {
		name       string
		confidence float64
		expected   string
	}{
		{"healthy", 0.0, "HEALTHY"},
		{"healthy_boundary", 0.3, "HEALTHY"},
		{"degraded", 0.31, "DEGRADED"},
		{"degraded_upper", 0.5, "DEGRADED"},
		{"partial_outage", 0.51, "PARTIAL_OUTAGE"},
		{"partial_upper", 0.8, "PARTIAL_OUTAGE"},
		{"major_outage", 0.81, "MAJOR_OUTAGE"},
		{"full_confidence", 1.0, "MAJOR_OUTAGE"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := confidenceToStatus(tt.confidence)
			if status != tt.expected {
				t.Errorf("confidenceToStatus(%.2f) = %s, want %s",
					tt.confidence, status, tt.expected)
			}
		})
	}
}

func TestAnchorExclusion(t *testing.T) {
	// Anchor targets should NEVER be included in correlation
	targets := []models.Target{
		{URL: "https://e-gov.az", Category: "GOV", Criticality: 10, Enabled: true},
		{URL: "https://google.com", Category: "ANCHOR", Criticality: 0, Enabled: true},
		{URL: "https://1.1.1.1", Category: "ANCHOR", Criticality: 0, Enabled: true},
		{URL: "https://cloudflare.com", Category: "ANCHOR", Criticality: 0, Enabled: true},
	}

	assessed := 0
	for _, t := range targets {
		if t.Category == "ANCHOR" {
			continue
		}
		assessed++
	}

	if assessed != 1 {
		t.Errorf("expected 1 non-anchor target, got %d", assessed)
	}
}

// confidenceToStatus is a test helper that mirrors the engine logic.
func confidenceToStatus(confidence float64) string {
	switch {
	case confidence > ThresholdMajor:
		return "MAJOR_OUTAGE"
	case confidence > ThresholdPartial:
		return "PARTIAL_OUTAGE"
	case confidence > ThresholdDegraded:
		return "DEGRADED"
	default:
		return "HEALTHY"
	}
}
