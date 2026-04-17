package bgp

import (
	"testing"
	"time"
)

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
