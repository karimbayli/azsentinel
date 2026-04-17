package api

import (
	"net/http"
	"net/http/httptest"
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

func TestWithCORS(t *testing.T) {
	s := &Server{}

	// A dummy handler that returns 200 OK
	dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	corsHandler := s.withCORS(dummyHandler)

	// Test GET request
	reqGET := httptest.NewRequest(http.MethodGet, "/", nil)
	rrGET := httptest.NewRecorder()
	corsHandler.ServeHTTP(rrGET, reqGET)

	if rrGET.Code != http.StatusOK {
		t.Errorf("Expected status code 200 for GET, got %d", rrGET.Code)
	}
	if rrGET.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("Expected Access-Control-Allow-Origin: *, got %s", rrGET.Header().Get("Access-Control-Allow-Origin"))
	}
	if rrGET.Header().Get("Access-Control-Allow-Methods") != "GET, POST, OPTIONS" {
		t.Errorf("Expected Access-Control-Allow-Methods: GET, POST, OPTIONS, got %s", rrGET.Header().Get("Access-Control-Allow-Methods"))
	}

	// Test OPTIONS request (preflight)
	reqOPTIONS := httptest.NewRequest(http.MethodOptions, "/", nil)
	rrOPTIONS := httptest.NewRecorder()
	corsHandler.ServeHTTP(rrOPTIONS, reqOPTIONS)

	if rrOPTIONS.Code != http.StatusNoContent {
		t.Errorf("Expected status code 204 for OPTIONS, got %d", rrOPTIONS.Code)
	}
	if rrOPTIONS.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("Expected Access-Control-Allow-Origin: *, got %s", rrOPTIONS.Header().Get("Access-Control-Allow-Origin"))
	}
}
