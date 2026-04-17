package bgp

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestMonitorRunCancellation(t *testing.T) {
	logger := zap.NewNop()

	// Create a monitor with fake collectors
	collectors := []string{"rrc00", "rrc01"}
	m := New("ws://fake", collectors, []int{12345}, nil, logger)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Run should return when context is cancelled, without blocking indefinitely
	done := make(chan struct{})
	go func() {
		m.Run(ctx)
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(1 * time.Second):
		t.Fatal("Run did not exit when context was cancelled")
	}
}

func TestLookupPrefixOrigin(t *testing.T) {
	m := &Monitor{
		prefixStates: make(map[string]*prefixState),
	}

	m.prefixStates["192.168.1.0/24:65000"] = &prefixState{
		OriginASN: 12345,
		State:     "ANNOUNCED",
		LastSeen:  time.Now(),
	}

	asn := m.lookupPrefixOrigin("192.168.1.0/24", 65000)
	if asn != 12345 {
		t.Errorf("expected 12345, got %d", asn)
	}

	asnNotFound := m.lookupPrefixOrigin("10.0.0.0/8", 65000)
	if asnNotFound != 0 {
		t.Errorf("expected 0, got %d", asnNotFound)
	}
}
