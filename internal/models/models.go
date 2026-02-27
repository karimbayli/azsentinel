package models

import "time"

// ProbeResult represents a single probe measurement from a node to a target.
type ProbeResult struct {
	Time           time.Time  `json:"time"`
	NodeID         string     `json:"node_id"`
	TargetURL      string     `json:"target_url"`
	TargetCategory string     `json:"target_category"`
	DNSResolveMs   int        `json:"dns_resolve_ms,omitempty"` // FIX R-5: DNS layer
	DNSResolvedIP  string     `json:"dns_resolved_ip,omitempty"`
	DNSError       string     `json:"dns_error,omitempty"`
	TCPDialMs      int        `json:"tcp_dial_ms"`
	TCPSuccess     bool       `json:"tcp_success"`
	TLSHandshakeMs int        `json:"tls_handshake_ms,omitempty"`
	TLSValid       *bool      `json:"tls_valid,omitempty"`
	TLSExpiry      *time.Time `json:"tls_expiry,omitempty"`
	CertIssuer     string     `json:"cert_issuer,omitempty"`
	HTTPStatus     int        `json:"http_status,omitempty"`
	TTFBMs         int        `json:"ttfb_ms,omitempty"`
	TotalMs        int        `json:"total_ms,omitempty"`
	NodeReliable   bool       `json:"node_reliable"`
	ErrorType      string     `json:"error_type,omitempty"`
	ErrorDetail    string     `json:"error_detail,omitempty"`
}

// ProbeBatch is what vantage nodes send to the central ingest endpoint.
type ProbeBatch struct {
	NodeID    string        `json:"node_id"`
	Region    string        `json:"region"`
	Timestamp time.Time     `json:"timestamp"`
	Version   string        `json:"version"`
	SentAt    int64         `json:"sent_at"` // FIX A-1: Unix timestamp for replay protection
	Nonce     string        `json:"nonce"`   // FIX A-1: Unique per-request nonce
	Results   []ProbeResult `json:"results"`
}

// BGPEvent represents a BGP route announcement or withdrawal.
type BGPEvent struct {
	Time      time.Time `json:"time"`
	ASN       int       `json:"asn"`
	Provider  string    `json:"provider"`
	Prefix    string    `json:"prefix"`
	EventType string    `json:"event_type"` // ANNOUNCE or WITHDRAW
	PeerAS    int       `json:"peer_as"`
	Collector string    `json:"collector"`
}

// CorrelationResult is a computed assessment for a target.
type CorrelationResult struct {
	Time          time.Time `json:"time"`
	TargetURL     string    `json:"target_url"`
	Status        string    `json:"status"` // HEALTHY, DEGRADED, PARTIAL_OUTAGE, MAJOR_OUTAGE
	Confidence    float64   `json:"confidence"`
	NodeSignal    float64   `json:"node_signal"`
	BGPSignal     float64   `json:"bgp_signal"`
	SocialSignal  float64   `json:"social_signal"`
	SignalsActive []string  `json:"signals_active"`
	TotalNodes    int       `json:"total_nodes"`
	FailingNodes  int       `json:"failing_nodes"`
}

// SocialSignal represents aggregated social media mention metrics.
type SocialSignal struct {
	Time           time.Time `json:"time"`
	WindowMinutes  int       `json:"window_minutes"`
	MentionCount   int       `json:"mention_count"`
	BaselineRate   float64   `json:"baseline_rate"`
	Ratio          float64   `json:"ratio"`
	SampleKeywords []string  `json:"sample_keywords"`
}

// Incident represents a detected outage or degradation event.
type Incident struct {
	ID             string     `json:"id"`
	TargetURL      string     `json:"target_url"`
	StartedAt      time.Time  `json:"started_at"`
	ResolvedAt     *time.Time `json:"resolved_at,omitempty"`
	PeakConfidence float64    `json:"peak_confidence"`
	PeakStatus     string     `json:"peak_status"`
	SignalsFired   []string   `json:"signals_fired"`
	Notes          string     `json:"notes,omitempty"`
}

// NodeHealth holds health info for a vantage node.
type NodeHealth struct {
	Time         time.Time `json:"time"`
	NodeID       string    `json:"node_id"`
	IsAlive      bool      `json:"is_alive"`
	BaselineOK   bool      `json:"baseline_ok"`
	ProbeCount   int       `json:"probe_count"`
	AvgLatencyMs int       `json:"avg_latency_ms"`
	BufferDepth  int       `json:"buffer_depth"`
	Version      string    `json:"version"`
}

// Target is a monitored endpoint.
type Target struct {
	URL           string `json:"url" yaml:"url"`
	Category      string `json:"category" yaml:"category"`
	Criticality   int    `json:"criticality" yaml:"criticality"`
	Enabled       bool   `json:"enabled" yaml:"enabled"`
	DisplayName   string `json:"display_name" yaml:"display_name"`
	DisplayNameEN string `json:"display_name_en" yaml:"display_name_en"`
	ParentSystem  string `json:"parent_system" yaml:"parent_system"`
}

// Node represents a vantage point.
type Node struct {
	NodeID  string `json:"node_id" yaml:"node_id"`
	Region  string `json:"region" yaml:"region"`
	Country string `json:"country" yaml:"country"`
	Enabled bool   `json:"enabled" yaml:"enabled"`
}

// TargetStatus is the current state of a target as reported by the API.
type TargetStatus struct {
	Target         Target            `json:"target"`
	Status         string            `json:"status"`
	Confidence     float64           `json:"confidence"`
	LastCheck      time.Time         `json:"last_check"`
	NodeBreakdown  []NodeProbeStatus `json:"node_breakdown,omitempty"`
	ActiveIncident *Incident         `json:"active_incident,omitempty"`
}

// NodeProbeStatus is the last probe result from a specific node for a target.
type NodeProbeStatus struct {
	NodeID     string `json:"node_id"`
	Region     string `json:"region"`
	TCPSuccess bool   `json:"tcp_success"`
	HTTPStatus int    `json:"http_status"`
	TotalMs    int    `json:"total_ms"`
	Reliable   bool   `json:"reliable"`
	Error      string `json:"error,omitempty"`
}

// Methodology contains the transparency documentation.
type Methodology struct {
	WhatWeMeasure      string             `json:"what_we_measure"`
	WhatWeDoNotMeasure string             `json:"what_we_do_not_measure"`
	ConfidenceModel    string             `json:"confidence_model"`
	Weights            map[string]float64 `json:"weights"`
	KnownLimitations   []string           `json:"known_limitations"`
	VantageNodes       []Node             `json:"vantage_nodes"`
}
