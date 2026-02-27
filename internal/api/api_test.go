package api

import (
	"testing"
)

func TestComputeAndValidateHMAC(t *testing.T) {
	secret := "test-secret-key-12345"
	message := []byte(`{"node_id":"node-eu","results":[]}`)

	// Compute
	sig := ComputeHMAC(message, secret)
	if sig == "" {
		t.Fatal("ComputeHMAC returned empty signature")
	}

	// Validate — should pass with correct message
	if !validateHMAC(message, sig, secret) {
		t.Error("validateHMAC should return true for correct signature")
	}

	// Validate — should fail with wrong message
	wrongMsg := []byte(`{"node_id":"node-us","results":[]}`)
	if validateHMAC(wrongMsg, sig, secret) {
		t.Error("validateHMAC should return false for wrong message")
	}

	// Validate — should fail with wrong secret
	if validateHMAC(message, sig, "wrong-secret") {
		t.Error("validateHMAC should return false for wrong secret")
	}

	// Validate — should fail with tampered signature
	if validateHMAC(message, "tampered"+sig, secret) {
		t.Error("validateHMAC should return false for tampered signature")
	}
}

func TestComputeHMACConsistency(t *testing.T) {
	secret := "consistent-key"
	message := []byte("same message")

	sig1 := ComputeHMAC(message, secret)
	sig2 := ComputeHMAC(message, secret)

	if sig1 != sig2 {
		t.Errorf("ComputeHMAC should be deterministic: %s != %s", sig1, sig2)
	}
}
