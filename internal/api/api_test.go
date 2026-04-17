package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/karimbayli/sentinel-v2/internal/models"
	"go.uber.org/zap/zaptest"
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

func TestAntiReplayNonce(t *testing.T) {
	logger := zaptest.NewLogger(t)
	// Create a Server without a real DB. We will stop testing logic before hitting the DB.
	nodes := []models.Node{
		{NodeID: "test-node", Enabled: true},
	}
	// Note: We use nil DB, which would panic if reached. But our test cases are designed
	// to trigger HTTP errors before DB insertion, or we will just use a minimal test.
	s := New(nil, "secret", nodes, nil, logger)

	// Mock nonce that has not expired
	s.nonceCache["valid-nonce"] = time.Now().Add(10 * time.Minute)
	// Mock nonce that has already expired
	s.nonceCache["expired-nonce"] = time.Now().Add(-10 * time.Minute)

	tests := []struct {
		name       string
		nonce      string
		wantStatus int
	}{
		{
			name:       "valid nonce - already in cache and not expired",
			nonce:      "valid-nonce",
			wantStatus: http.StatusConflict,
		},
		{
			name:       "expired nonce - in cache but expired",
			nonce:      "expired-nonce",
			wantStatus: 0, // In this test, it will bypass nonce check and then fail later or panic. We will catch panic.
		},
		{
			name:       "new nonce - not in cache",
			nonce:      "new-nonce",
			wantStatus: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			batch := models.ProbeBatch{
				NodeID:    "test-node",
				SentAt:    time.Now().Unix(),
				Nonce:     tt.nonce,
				Results:   []models.ProbeResult{{NodeID: "test-node", TargetURL: "http://example.com", Time: time.Now()}},
			}

			body, _ := json.Marshal(batch)

			req := httptest.NewRequest(http.MethodPost, "/api/v1/ingest/probe-batch", bytes.NewReader(body))
			req.Header.Set("X-Sentinel-Node", "test-node")

			sig := ComputeHMAC(body, "secret")
			req.Header.Set("X-Sentinel-Signature", sig)

			rr := httptest.NewRecorder()

			func() {
				defer func() {
					if r := recover(); r != nil {
						// Expected panic from nil db
					}
				}()
				s.mux.ServeHTTP(rr, req)
			}()

			if tt.wantStatus != 0 && rr.Code != tt.wantStatus {
				t.Errorf("Expected status %d, got %d", tt.wantStatus, rr.Code)
			}
			if tt.wantStatus == 0 && rr.Code == http.StatusConflict {
				t.Errorf("Expected not to get StatusConflict (409), but got it")
			}
		})
	}
}
