package bgp

import (
	"context"
	"encoding/json"
	"testing"

	"go.uber.org/zap/zaptest"
)

func TestProcessWithdrawals(t *testing.T) {
	logger := zaptest.NewLogger(t)

	// Since New expects a storage.DB (concrete), we pass nil.
	// We've updated bgp.go to handle nil db for testing purposes.
	m := New("ws://localhost", []string{"rrc00"}, []int{12345}, nil, logger)

	ctx := context.Background()
	collector := "rrc00"
	peerAS := 6789

	// 1. Send an announcement for a prefix, so origin ASN is stored.
	annMessage := risMessage{
		Type: "ris_message",
		Data: json.RawMessage(`{
			"timestamp": 1600000000.0,
			"peer": "192.0.2.1",
			"peer_asn": 6789,
			"host": "rrc00",
			"path": [6789, 12345],
			"announcements": [
				{
					"next_hop": "192.0.2.1",
					"prefixes": ["10.0.0.0/24"]
				}
			]
		}`),
	}
	annBytes, _ := json.Marshal(annMessage)
	m.processMessage(ctx, annBytes, collector)

	// Verify prefix state table is populated.
	knownOrigin := m.lookupPrefixOrigin("10.0.0.0/24", peerAS)
	if knownOrigin != 12345 {
		t.Fatalf("expected knownOrigin 12345, got %d", knownOrigin)
	}

	// 2. Send a withdrawal without AS path (typical behavior).
	wdMessage := risMessage{
		Type: "ris_message",
		Data: json.RawMessage(`{
			"timestamp": 1600000010.0,
			"peer": "192.0.2.1",
			"peer_asn": 6789,
			"host": "rrc00",
			"withdrawals": [
				{
					"prefixes": ["10.0.0.0/24"]
				}
			]
		}`),
	}
	wdBytes, _ := json.Marshal(wdMessage)
	m.processMessage(ctx, wdBytes, collector)

	// The recentEvents list should now have an ANNOUNCE and a WITHDRAW event, both associated with ASN 12345.
	m.mu.RLock()
	events := m.recentEvents
	m.mu.RUnlock()

	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}

	if events[1].EventType != "WITHDRAW" {
		t.Errorf("expected event type WITHDRAW, got %s", events[1].EventType)
	}

	if events[1].ASN != 12345 {
		t.Errorf("expected withdrawn event to be attributed to ASN 12345, got %d", events[1].ASN)
	}

	// 3. Fallback logic: withdrawal for unknown prefix but WITH an AS path.
	wdFallbackMsg := risMessage{
		Type: "ris_message",
		Data: json.RawMessage(`{
			"timestamp": 1600000020.0,
			"peer": "192.0.2.1",
			"peer_asn": 6789,
			"host": "rrc00",
			"path": [6789, 12345],
			"withdrawals": [
				{
					"prefixes": ["10.1.1.0/24"]
				}
			]
		}`),
	}
	wdFallbackBytes, _ := json.Marshal(wdFallbackMsg)
	m.processMessage(ctx, wdFallbackBytes, collector)

	m.mu.RLock()
	events = m.recentEvents
	m.mu.RUnlock()

	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(events))
	}
	if events[2].EventType != "WITHDRAW" {
		t.Errorf("expected event type WITHDRAW, got %s", events[2].EventType)
	}
	if events[2].ASN != 12345 {
		t.Errorf("expected fallback withdrawn event to be attributed to ASN 12345, got %d", events[2].ASN)
	}
	if events[2].Prefix != "10.1.1.0/24" {
		t.Errorf("expected prefix 10.1.1.0/24, got %s", events[2].Prefix)
	}
}
