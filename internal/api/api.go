package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/karimbayli/sentinel-v2/internal/models"
	"github.com/karimbayli/sentinel-v2/internal/storage"
	"go.uber.org/zap"
)

// Server holds the REST API server state.
type Server struct {
	db         *storage.DB
	hmacSecret string
	nodes      []models.Node
	targets    []models.Target
	logger     *zap.Logger
	mux        *http.ServeMux
	version    string
	nodeSet    map[string]bool // FIX H-3: registered node IDs for validation

	// FIX A-1: Anti-replay nonce cache
	nonceMu    sync.Mutex
	nonceCache map[string]time.Time // nonce → expiry time
}

const (
	maxClockSkew    = 120 * time.Second // FIX A-1: max age for batch SentAt
	minProbeVersion = "1.0.0"           // FIX A-12: minimum acceptable probe version
)

// New creates a new API server.
func New(db *storage.DB, hmacSecret string, nodes []models.Node, targets []models.Target, logger *zap.Logger) *Server {
	// FIX H-3: Build registered node set for fast validation
	nodeSet := make(map[string]bool, len(nodes))
	for _, n := range nodes {
		nodeSet[n.NodeID] = true
	}

	s := &Server{
		db:         db,
		hmacSecret: hmacSecret,
		nodes:      nodes,
		targets:    targets,
		logger:     logger,
		mux:        http.NewServeMux(),
		version:    "1.0.0",
		nodeSet:    nodeSet,
		nonceCache: make(map[string]time.Time),
	}
	s.registerRoutes()

	// FIX A-1: Start nonce cache cleanup goroutine
	go s.cleanNonceCache()

	return s
}

// cleanNonceCache periodically removes expired nonces.
func (s *Server) cleanNonceCache() {
	for range time.Tick(5 * time.Minute) {
		now := time.Now()
		s.nonceMu.Lock()
		for k, expiry := range s.nonceCache {
			if now.After(expiry) {
				delete(s.nonceCache, k)
			}
		}
		s.nonceMu.Unlock()
	}
}

// Handler returns the HTTP handler for the API with rate limiting (FIX H-1).
func (s *Server) Handler() http.Handler {
	return s.withRateLimit(s.mux)
}

// registerRoutes sets up all API routes.
func (s *Server) registerRoutes() {
	// Health check
	s.mux.HandleFunc("GET /healthz", s.handleHealthz)

	// Public methodology
	s.mux.HandleFunc("GET /api/v1/methodology", s.handleMethodology)
	s.mux.HandleFunc("GET /methodology", s.handleMethodologyPage)

	// Probe data ingest (HMAC-authenticated)
	s.mux.HandleFunc("POST /api/v1/ingest/probe-batch", s.withHMAC(s.handleIngestProbeBatch))

	// Status
	s.mux.HandleFunc("GET /api/v1/status", s.handleStatus)
	s.mux.HandleFunc("GET /api/v1/status/{target}", s.handleStatusTarget)

	// Incidents
	s.mux.HandleFunc("GET /api/v1/incidents", s.handleIncidents)

	// BGP events
	s.mux.HandleFunc("GET /api/v1/bgp/events", s.handleBGPEvents)

	// Nodes
	s.mux.HandleFunc("GET /api/v1/nodes", s.handleNodes)

	// History
	s.mux.HandleFunc("GET /api/v1/history/{target}", s.handleHistory)

	// Export
	s.mux.HandleFunc("GET /api/v1/export/csv", s.handleExportCSV)

	// Static files
	s.mux.Handle("GET /", http.FileServer(http.Dir("static")))
}

// ============================================================
// Middleware
// ============================================================

// rateLimiter implements a simple per-IP sliding window rate limiter (FIX H-1).
type rateLimiter struct {
	mu       sync.Mutex
	clients  map[string]*clientWindow
	limit    int
	windowMs int64
}

type clientWindow struct {
	count    int
	windowAt int64
}

func newRateLimiter(requestsPerWindow int, windowDuration time.Duration) *rateLimiter {
	rl := &rateLimiter{
		clients:  make(map[string]*clientWindow),
		limit:    requestsPerWindow,
		windowMs: windowDuration.Milliseconds(),
	}
	// Cleanup stale entries periodically
	go func() {
		for range time.Tick(5 * time.Minute) {
			rl.cleanup()
		}
	}()
	return rl
}

func (rl *rateLimiter) allow(clientIP string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now().UnixMilli()
	cw, ok := rl.clients[clientIP]
	if !ok {
		rl.clients[clientIP] = &clientWindow{count: 1, windowAt: now}
		return true
	}

	// Reset window if expired
	if now-cw.windowAt > rl.windowMs {
		cw.count = 1
		cw.windowAt = now
		return true
	}

	cw.count++
	return cw.count <= rl.limit
}

func (rl *rateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now().UnixMilli()
	for k, cw := range rl.clients {
		if now-cw.windowAt > rl.windowMs*2 {
			delete(rl.clients, k)
		}
	}
}

// Global rate limiter: 120 requests per minute per IP
var globalRL = newRateLimiter(120, time.Minute)

// withRateLimit wraps the handler with per-IP rate limiting (FIX H-1).
func (s *Server) withRateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clientIP := r.RemoteAddr
		if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
			clientIP = strings.Split(fwd, ",")[0]
		}

		if !globalRL.allow(strings.TrimSpace(clientIP)) {
			s.errorJSON(w, http.StatusTooManyRequests, "rate limit exceeded")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// withHMAC validates HMAC-SHA256 signatures on ingest requests.
func (s *Server) withHMAC(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sig := r.Header.Get("X-Sentinel-Signature")
		if sig == "" {
			s.errorJSON(w, http.StatusUnauthorized, "missing signature")
			return
		}

		body, err := io.ReadAll(io.LimitReader(r.Body, 10*1024*1024))
		if err != nil {
			s.errorJSON(w, http.StatusBadRequest, "read body failed")
			return
		}
		r.Body.Close()

		if !validateHMAC(body, sig, s.hmacSecret) {
			s.logger.Warn("invalid HMAC signature",
				zap.String("remote_addr", r.RemoteAddr),
				zap.String("node", r.Header.Get("X-Sentinel-Node")))
			s.errorJSON(w, http.StatusForbidden, "invalid signature")
			return
		}

		// Reconstruct body for handler
		r.Body = io.NopCloser(strings.NewReader(string(body)))
		next(w, r)
	}
}

// validateHMAC verifies HMAC-SHA256 signature.
func validateHMAC(message []byte, signature, secret string) bool {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(message)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}

// ComputeHMAC generates HMAC-SHA256 signature for a payload.
func ComputeHMAC(message []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(message)
	return hex.EncodeToString(mac.Sum(nil))
}

// ============================================================
// Handlers
// ============================================================

func (s *Server) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"status":  "ok",
		"version": s.version,
		"time":    time.Now().UTC().Format(time.RFC3339),
	})
}

func (s *Server) handleMethodology(w http.ResponseWriter, _ *http.Request) {
	m := models.Methodology{
		WhatWeMeasure: "External reachability of configured targets from multiple vantage points " +
			"across different geographic locations. We perform TCP connection, TLS handshake, " +
			"and HTTP GET probes from each node every 60 seconds.",
		WhatWeDoNotMeasure: "Internal ISP backbone performance, internal network peering points, " +
			"geo-blocked content assessment, DNS resolution quality from within Azerbaijan, " +
			"or any measurement requiring internal vantage points.",
		ConfidenceModel: "Weighted multi-signal correlation: node failure agreement (0.5), " +
			"BGP route withdrawal from RIPE RIS (0.3), and social media mention spikes (0.2). " +
			"Confidence >0.8 = MAJOR_OUTAGE, >0.5 = PARTIAL_OUTAGE, >0.3 = DEGRADED, ≤0.3 = HEALTHY.",
		Weights: map[string]float64{
			"multi_node_failure":   0.5,
			"bgp_route_withdrawal": 0.3,
			"social_mention_spike": 0.2,
		},
		KnownLimitations: []string{
			"Cannot observe internal ISP backbone failures directly",
			"BGP data relies on RIPE RIS collector coverage; some events may be delayed or missed",
			"Social signal analysis depends on Telegram channel coverage and may have false positives",
			"Node reliability is determined by anchor checks; a node may be unreliable due to its own connectivity",
			"Geographic coverage is limited to the deployed vantage node locations",
			"During national outages, the central server in Azerbaijan may be unreachable; data is buffered at vantage nodes",
		},
		VantageNodes: s.nodes,
	}
	s.jsonResponse(w, http.StatusOK, m)
}

func (s *Server) handleMethodologyPage(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "static/methodology.html")
}

func (s *Server) handleIngestProbeBatch(w http.ResponseWriter, r *http.Request) {
	var batch models.ProbeBatch
	if err := json.NewDecoder(r.Body).Decode(&batch); err != nil {
		s.errorJSON(w, http.StatusBadRequest, fmt.Sprintf("parse body: %v", err))
		return
	}

	if len(batch.Results) == 0 {
		s.errorJSON(w, http.StatusBadRequest, "empty batch")
		return
	}

	// FIX H-3: Validate that node_id is a registered node
	if !s.nodeSet[batch.NodeID] {
		s.logger.Warn("rejected probe batch from unregistered node",
			zap.String("node_id", batch.NodeID),
			zap.String("remote_addr", r.RemoteAddr))
		s.errorJSON(w, http.StatusForbidden, "unregistered node_id")
		return
	}

	// FIX A-1: Anti-replay — validate timestamp freshness
	if batch.SentAt > 0 {
		batchTime := time.Unix(batch.SentAt, 0)
		skew := time.Since(batchTime)
		if skew < -maxClockSkew || skew > maxClockSkew {
			s.logger.Warn("rejected stale/future batch",
				zap.String("node_id", batch.NodeID),
				zap.Duration("skew", skew))
			s.errorJSON(w, http.StatusBadRequest, "batch timestamp too old or in future")
			return
		}
	}

	// FIX A-1: Anti-replay — check nonce uniqueness
	if batch.Nonce != "" {
		s.nonceMu.Lock()
		if _, exists := s.nonceCache[batch.Nonce]; exists {
			s.nonceMu.Unlock()
			s.logger.Warn("rejected replayed batch (duplicate nonce)",
				zap.String("node_id", batch.NodeID),
				zap.String("nonce", batch.Nonce))
			s.errorJSON(w, http.StatusConflict, "duplicate nonce (replay detected)")
			return
		}
		s.nonceCache[batch.Nonce] = time.Now().Add(2 * maxClockSkew)
		s.nonceMu.Unlock()
	}

	// FIX A-5: Validate individual probe timestamps
	now := time.Now().UTC()
	maxAge := 10 * time.Minute
	for i, pr := range batch.Results {
		if pr.Time.After(now.Add(60 * time.Second)) {
			// Clamp future timestamps to now
			batch.Results[i].Time = now
		} else if pr.Time.Before(now.Add(-maxAge)) {
			// Allow up to 10 min old (for buffered results), log older ones
			s.logger.Debug("probe result has old timestamp",
				zap.String("node_id", batch.NodeID),
				zap.Time("result_time", pr.Time))
		}
	}

	s.logger.Info("received probe batch",
		zap.String("node_id", batch.NodeID),
		zap.Int("results", len(batch.Results)))

	// Store results
	if err := s.db.InsertProbeResults(r.Context(), batch.Results); err != nil {
		s.logger.Error("failed to store probe results", zap.Error(err))
		s.errorJSON(w, http.StatusInternalServerError, "storage error")
		return
	}

	// Store node health
	health := models.NodeHealth{
		Time:       time.Now().UTC(),
		NodeID:     batch.NodeID,
		IsAlive:    true,
		BaselineOK: true,
		ProbeCount: len(batch.Results),
		Version:    batch.Version,
	}

	// Check baseline OK from results
	for _, r := range batch.Results {
		if !r.NodeReliable {
			health.BaselineOK = false
			break
		}
	}

	// Calculate avg latency
	totalLatency := 0
	count := 0
	for _, r := range batch.Results {
		if r.TotalMs > 0 {
			totalLatency += r.TotalMs
			count++
		}
	}
	if count > 0 {
		health.AvgLatencyMs = totalLatency / count
	}

	if err := s.db.InsertNodeHealth(r.Context(), health); err != nil {
		s.logger.Error("failed to store node health", zap.Error(err))
	}

	s.jsonResponse(w, http.StatusOK, map[string]string{"status": "accepted"})
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	targets, err := s.db.GetEnabledTargets(r.Context())
	if err != nil {
		s.errorJSON(w, http.StatusInternalServerError, "failed to get targets")
		return
	}

	var statuses []models.TargetStatus
	for _, t := range targets {
		ts := models.TargetStatus{Target: t, Status: "HEALTHY"}
		cr, err := s.db.GetLatestCorrelation(r.Context(), t.URL)
		if err == nil && cr != nil {
			ts.Status = cr.Status
			ts.Confidence = cr.Confidence
			ts.LastCheck = cr.Time
		}

		// Check for active incident
		inc, err := s.db.GetOpenIncident(r.Context(), t.URL)
		if err == nil && inc != nil {
			ts.ActiveIncident = inc
		}

		statuses = append(statuses, ts)
	}

	s.jsonResponse(w, http.StatusOK, statuses)
}

func (s *Server) handleStatusTarget(w http.ResponseWriter, r *http.Request) {
	target := r.PathValue("target")
	if target == "" {
		s.errorJSON(w, http.StatusBadRequest, "missing target")
		return
	}

	// URL decode the target path
	targetURL := target
	if !strings.HasPrefix(targetURL, "http") {
		targetURL = "https://" + targetURL
	}

	// Get target info — use exact match only (FIX M-9)
	targets, err := s.db.GetEnabledTargets(r.Context())
	if err != nil {
		s.errorJSON(w, http.StatusInternalServerError, "failed to get targets")
		return
	}

	var found *models.Target
	for _, t := range targets {
		if t.URL == targetURL {
			found = &t
			break
		}
	}

	if found == nil {
		s.errorJSON(w, http.StatusNotFound, "target not found")
		return
	}

	ts := models.TargetStatus{Target: *found, Status: "HEALTHY"}

	// Get latest correlation
	cr, err := s.db.GetLatestCorrelation(r.Context(), found.URL)
	if err == nil && cr != nil {
		ts.Status = cr.Status
		ts.Confidence = cr.Confidence
		ts.LastCheck = cr.Time
	}

	// Get per-node breakdown
	probes, err := s.db.GetLatestProbeByNodeAndTarget(r.Context(), found.URL)
	if err == nil {
		for _, p := range probes {
			nps := models.NodeProbeStatus{
				NodeID:     p.NodeID,
				TCPSuccess: p.TCPSuccess,
				HTTPStatus: p.HTTPStatus,
				TotalMs:    p.TotalMs,
				Reliable:   p.NodeReliable,
			}
			if p.ErrorDetail != "" {
				nps.Error = p.ErrorDetail
			}
			// Find region from nodes config
			for _, n := range s.nodes {
				if n.NodeID == p.NodeID {
					nps.Region = n.Region
					break
				}
			}
			ts.NodeBreakdown = append(ts.NodeBreakdown, nps)
		}
	}

	// Active incident
	inc, err := s.db.GetOpenIncident(r.Context(), found.URL)
	if err == nil && inc != nil {
		ts.ActiveIncident = inc
	}

	s.jsonResponse(w, http.StatusOK, ts)
}

func (s *Server) handleIncidents(w http.ResponseWriter, r *http.Request) {
	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")

	from := time.Now().UTC().Add(-24 * time.Hour)
	to := time.Now().UTC()

	if fromStr != "" {
		if t, err := time.Parse(time.RFC3339, fromStr); err == nil {
			from = t
		}
	}
	if toStr != "" {
		if t, err := time.Parse(time.RFC3339, toStr); err == nil {
			to = t
		}
	}

	incidents, err := s.db.GetIncidents(r.Context(), from, to)
	if err != nil {
		s.errorJSON(w, http.StatusInternalServerError, "failed to get incidents")
		return
	}

	if incidents == nil {
		incidents = []models.Incident{}
	}
	s.jsonResponse(w, http.StatusOK, incidents)
}

func (s *Server) handleBGPEvents(w http.ResponseWriter, r *http.Request) {
	hoursStr := r.URL.Query().Get("hours")
	hours := 24
	if hoursStr != "" {
		if h, err := strconv.Atoi(hoursStr); err == nil && h > 0 && h <= 168 {
			hours = h
		}
	}

	events, err := s.db.GetBGPEvents(r.Context(), hours)
	if err != nil {
		s.errorJSON(w, http.StatusInternalServerError, "failed to get bgp events")
		return
	}

	if events == nil {
		events = []models.BGPEvent{}
	}
	s.jsonResponse(w, http.StatusOK, events)
}

func (s *Server) handleNodes(w http.ResponseWriter, r *http.Request) {
	health, err := s.db.GetLatestNodeHealth(r.Context())
	if err != nil {
		s.errorJSON(w, http.StatusInternalServerError, "failed to get node health")
		return
	}

	if health == nil {
		health = []models.NodeHealth{}
	}
	s.jsonResponse(w, http.StatusOK, health)
}

func (s *Server) handleHistory(w http.ResponseWriter, r *http.Request) {
	target := r.PathValue("target")
	if target == "" {
		s.errorJSON(w, http.StatusBadRequest, "missing target")
		return
	}

	targetURL := target
	if !strings.HasPrefix(targetURL, "http") {
		targetURL = "https://" + targetURL
	}

	hoursStr := r.URL.Query().Get("hours")
	hours := 24
	if hoursStr != "" {
		if h, err := strconv.Atoi(hoursStr); err == nil && h > 0 && h <= 720 {
			hours = h
		}
	}

	results, err := s.db.GetProbeHistory(r.Context(), targetURL, hours)
	if err != nil {
		s.errorJSON(w, http.StatusInternalServerError, "failed to get history")
		return
	}

	if results == nil {
		results = []models.ProbeResult{}
	}
	s.jsonResponse(w, http.StatusOK, results)
}

// FIX H-2: CSV export now streams directly from database rows to HTTP response
// instead of loading all rows into memory.
func (s *Server) handleExportCSV(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=sentinel-export.csv")

	// Write CSV header
	fmt.Fprintln(w, "time,node_id,target_url,target_category,tcp_dial_ms,tcp_success,tls_handshake_ms,http_status,ttfb_ms,total_ms,node_reliable,error_type,error_detail")

	// Stream rows directly (FIX H-2)
	if err := s.db.StreamExportCSV(r.Context(), w); err != nil {
		s.logger.Error("csv export error", zap.Error(err))
	}
}

// ============================================================
// Helpers
// ============================================================

func (s *Server) jsonResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (s *Server) errorJSON(w http.ResponseWriter, status int, message string) {
	s.jsonResponse(w, status, map[string]string{"error": message})
}
