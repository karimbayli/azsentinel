package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/karimbayli/sentinel-v2/internal/models"
	"go.uber.org/zap"
)

// DB wraps a pgx connection pool for TimescaleDB operations.
type DB struct {
	pool   *pgxpool.Pool
	logger *zap.Logger
}

// New creates a new DB with a pgx connection pool.
func New(ctx context.Context, dsn string, logger *zap.Logger) (*DB, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse dsn: %w", err)
	}
	cfg.MaxConns = 20
	cfg.MinConns = 2
	cfg.MaxConnLifetime = 30 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping db: %w", err)
	}

	return &DB{pool: pool, logger: logger}, nil
}

// Close shuts down the connection pool.
func (db *DB) Close() {
	db.pool.Close()
}

// Pool returns the underlying pgx pool for advanced queries.
func (db *DB) Pool() *pgxpool.Pool {
	return db.pool
}

// ============================================================
// Probe Results
// ============================================================

// InsertProbeResults batch-inserts probe results.
func (db *DB) InsertProbeResults(ctx context.Context, results []models.ProbeResult) error {
	if len(results) == 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	batch := &pgx.Batch{}
	for _, r := range results {
		batch.Queue(`INSERT INTO probe_results
			(time, node_id, target_url, target_category,
			 dns_resolve_ms, dns_resolved_ip, dns_error,
			 tcp_dial_ms, tcp_success,
			 tls_handshake_ms, tls_valid, tls_expiry, cert_issuer, http_status,
			 ttfb_ms, total_ms, node_reliable, error_type, error_detail)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19)`,
			r.Time, r.NodeID, r.TargetURL, r.TargetCategory,
			r.DNSResolveMs, r.DNSResolvedIP, r.DNSError,
			r.TCPDialMs, r.TCPSuccess, r.TLSHandshakeMs, r.TLSValid,
			r.TLSExpiry, r.CertIssuer, r.HTTPStatus, r.TTFBMs,
			r.TotalMs, r.NodeReliable, r.ErrorType, r.ErrorDetail,
		)
	}

	br := db.pool.SendBatch(ctx, batch)
	defer br.Close()

	for i := 0; i < len(results); i++ {
		if _, err := br.Exec(); err != nil {
			return fmt.Errorf("insert probe result %d: %w", i, err)
		}
	}
	return nil
}

// GetRecentProbeResults returns recent probe results for a target.
func (db *DB) GetRecentProbeResults(ctx context.Context, targetURL string, window time.Duration) ([]models.ProbeResult, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	since := time.Now().UTC().Add(-window)
	rows, err := db.pool.Query(ctx, `
		SELECT time, node_id, target_url, target_category,
		       dns_resolve_ms, dns_resolved_ip, dns_error,
		       tcp_dial_ms, tcp_success,
		       tls_handshake_ms, tls_valid, tls_expiry, cert_issuer, http_status,
		       ttfb_ms, total_ms, node_reliable, error_type, error_detail
		FROM probe_results
		WHERE target_url = $1 AND time >= $2
		ORDER BY time DESC`, targetURL, since)
	if err != nil {
		return nil, fmt.Errorf("query probe results: %w", err)
	}
	defer rows.Close()

	var results []models.ProbeResult
	for rows.Next() {
		var r models.ProbeResult
		if err := rows.Scan(&r.Time, &r.NodeID, &r.TargetURL, &r.TargetCategory,
			&r.DNSResolveMs, &r.DNSResolvedIP, &r.DNSError,
			&r.TCPDialMs, &r.TCPSuccess, &r.TLSHandshakeMs, &r.TLSValid,
			&r.TLSExpiry, &r.CertIssuer, &r.HTTPStatus, &r.TTFBMs,
			&r.TotalMs, &r.NodeReliable, &r.ErrorType, &r.ErrorDetail); err != nil {
			return nil, fmt.Errorf("scan probe result: %w", err)
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

// GetLatestProbeByNodeAndTarget returns the latest probe result per node for a target.
func (db *DB) GetLatestProbeByNodeAndTarget(ctx context.Context, targetURL string) ([]models.ProbeResult, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	rows, err := db.pool.Query(ctx, `
		SELECT DISTINCT ON (node_id)
		       time, node_id, target_url, target_category,
		       dns_resolve_ms, dns_resolved_ip, dns_error,
		       tcp_dial_ms, tcp_success,
		       tls_handshake_ms, tls_valid, tls_expiry, cert_issuer, http_status,
		       ttfb_ms, total_ms, node_reliable, error_type, error_detail
		FROM probe_results
		WHERE target_url = $1 AND time >= now() - interval '10 minutes'
		ORDER BY node_id, time DESC`, targetURL)
	if err != nil {
		return nil, fmt.Errorf("query latest probes: %w", err)
	}
	defer rows.Close()

	var results []models.ProbeResult
	for rows.Next() {
		var r models.ProbeResult
		if err := rows.Scan(&r.Time, &r.NodeID, &r.TargetURL, &r.TargetCategory,
			&r.DNSResolveMs, &r.DNSResolvedIP, &r.DNSError,
			&r.TCPDialMs, &r.TCPSuccess, &r.TLSHandshakeMs, &r.TLSValid,
			&r.TLSExpiry, &r.CertIssuer, &r.HTTPStatus, &r.TTFBMs,
			&r.TotalMs, &r.NodeReliable, &r.ErrorType, &r.ErrorDetail); err != nil {
			return nil, fmt.Errorf("scan latest probe: %w", err)
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

// GetProbeHistory returns probe results for a target within a time range.
func (db *DB) GetProbeHistory(ctx context.Context, targetURL string, hours int) ([]models.ProbeResult, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	since := time.Now().UTC().Add(-time.Duration(hours) * time.Hour)
	rows, err := db.pool.Query(ctx, `
		SELECT time, node_id, target_url, target_category,
		       dns_resolve_ms, dns_resolved_ip, dns_error,
		       tcp_dial_ms, tcp_success,
		       tls_handshake_ms, tls_valid, tls_expiry, cert_issuer, http_status,
		       ttfb_ms, total_ms, node_reliable, error_type, error_detail
		FROM probe_results
		WHERE target_url = $1 AND time >= $2
		ORDER BY time DESC
		LIMIT 5000`, targetURL, since)
	if err != nil {
		return nil, fmt.Errorf("query probe history: %w", err)
	}
	defer rows.Close()

	var results []models.ProbeResult
	for rows.Next() {
		var r models.ProbeResult
		if err := rows.Scan(&r.Time, &r.NodeID, &r.TargetURL, &r.TargetCategory,
			&r.DNSResolveMs, &r.DNSResolvedIP, &r.DNSError,
			&r.TCPDialMs, &r.TCPSuccess, &r.TLSHandshakeMs, &r.TLSValid,
			&r.TLSExpiry, &r.CertIssuer, &r.HTTPStatus, &r.TTFBMs,
			&r.TotalMs, &r.NodeReliable, &r.ErrorType, &r.ErrorDetail); err != nil {
			return nil, fmt.Errorf("scan probe history: %w", err)
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

// ============================================================
// BGP Events
// ============================================================

// InsertBGPEvent inserts a single BGP event.
func (db *DB) InsertBGPEvent(ctx context.Context, e models.BGPEvent) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := db.pool.Exec(ctx, `
		INSERT INTO bgp_events (time, asn, provider, prefix, event_type, peer_as, collector)
		VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		e.Time, e.ASN, e.Provider, e.Prefix, e.EventType, e.PeerAS, e.Collector)
	return err
}

// GetRecentBGPWithdrawals returns BGP withdrawals for watched ASNs within a window.
func (db *DB) GetRecentBGPWithdrawals(ctx context.Context, asns []int, window time.Duration) ([]models.BGPEvent, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	since := time.Now().UTC().Add(-window)
	placeholders := make([]string, len(asns))
	args := make([]interface{}, 0, len(asns)+1)
	args = append(args, since)
	for i, asn := range asns {
		placeholders[i] = fmt.Sprintf("$%d", i+2)
		args = append(args, asn)
	}

	query := fmt.Sprintf(`
		SELECT time, asn, provider, prefix, event_type, peer_as, collector
		FROM bgp_events
		WHERE event_type = 'WITHDRAW' AND time >= $1 AND asn IN (%s)
		ORDER BY time DESC`, strings.Join(placeholders, ","))

	rows, err := db.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query bgp withdrawals: %w", err)
	}
	defer rows.Close()

	var events []models.BGPEvent
	for rows.Next() {
		var e models.BGPEvent
		if err := rows.Scan(&e.Time, &e.ASN, &e.Provider, &e.Prefix,
			&e.EventType, &e.PeerAS, &e.Collector); err != nil {
			return nil, fmt.Errorf("scan bgp event: %w", err)
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

// GetBGPEvents returns BGP events within a number of hours.
func (db *DB) GetBGPEvents(ctx context.Context, hours int) ([]models.BGPEvent, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	since := time.Now().UTC().Add(-time.Duration(hours) * time.Hour)
	rows, err := db.pool.Query(ctx, `
		SELECT time, asn, provider, prefix, event_type, peer_as, collector
		FROM bgp_events WHERE time >= $1
		ORDER BY time DESC LIMIT 1000`, since)
	if err != nil {
		return nil, fmt.Errorf("query bgp events: %w", err)
	}
	defer rows.Close()

	var events []models.BGPEvent
	for rows.Next() {
		var e models.BGPEvent
		if err := rows.Scan(&e.Time, &e.ASN, &e.Provider, &e.Prefix,
			&e.EventType, &e.PeerAS, &e.Collector); err != nil {
			return nil, fmt.Errorf("scan bgp event: %w", err)
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

// ============================================================
// Correlation Results
// ============================================================

// InsertCorrelationResult inserts a correlation assessment.
func (db *DB) InsertCorrelationResult(ctx context.Context, r models.CorrelationResult) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	signalsJSON, err := json.Marshal(r.SignalsActive)
	if err != nil {
		return err
	}

	_, err = db.pool.Exec(ctx, `
		INSERT INTO correlation_results
			(time, target_url, status, confidence, node_signal, bgp_signal,
			 social_signal, signals_active, total_nodes, failing_nodes)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		r.Time, r.TargetURL, r.Status, r.Confidence,
		r.NodeSignal, r.BGPSignal, r.SocialSignal,
		signalsJSON, r.TotalNodes, r.FailingNodes)
	return err
}

// GetLatestCorrelation returns the most recent correlation result for a target.
func (db *DB) GetLatestCorrelation(ctx context.Context, targetURL string) (*models.CorrelationResult, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var r models.CorrelationResult
	err := db.pool.QueryRow(ctx, `
		SELECT time, target_url, status, confidence, node_signal, bgp_signal,
		       social_signal, signals_active, total_nodes, failing_nodes
		FROM correlation_results
		WHERE target_url = $1
		ORDER BY time DESC LIMIT 1`, targetURL).Scan(
		&r.Time, &r.TargetURL, &r.Status, &r.Confidence,
		&r.NodeSignal, &r.BGPSignal, &r.SocialSignal,
		&r.SignalsActive, &r.TotalNodes, &r.FailingNodes)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query latest correlation: %w", err)
	}
	return &r, nil
}

// ============================================================
// Social Signals
// ============================================================

// InsertSocialSignal inserts an aggregated social signal measurement.
func (db *DB) InsertSocialSignal(ctx context.Context, s models.SocialSignal) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := db.pool.Exec(ctx, `
		INSERT INTO social_signals (time, window_minutes, mention_count, baseline_rate, ratio, sample_keywords)
		VALUES ($1,$2,$3,$4,$5,$6)`,
		s.Time, s.WindowMinutes, s.MentionCount, s.BaselineRate, s.Ratio, s.SampleKeywords)
	return err
}

// GetLatestSocialSignal returns the most recent social signal.
func (db *DB) GetLatestSocialSignal(ctx context.Context) (*models.SocialSignal, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var s models.SocialSignal
	err := db.pool.QueryRow(ctx, `
		SELECT time, window_minutes, mention_count, baseline_rate, ratio, sample_keywords
		FROM social_signals ORDER BY time DESC LIMIT 1`).Scan(
		&s.Time, &s.WindowMinutes, &s.MentionCount, &s.BaselineRate, &s.Ratio, &s.SampleKeywords)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query latest social signal: %w", err)
	}
	return &s, nil
}

// GetSocialBaseline returns the average mention rate over the baseline period.
func (db *DB) GetSocialBaseline(ctx context.Context, days int) (float64, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	since := time.Now().UTC().Add(-time.Duration(days) * 24 * time.Hour)
	var avg float64
	err := db.pool.QueryRow(ctx, `
		SELECT COALESCE(AVG(mention_count), 0) FROM social_signals WHERE time >= $1`, since).Scan(&avg)
	if err != nil {
		return 0, fmt.Errorf("query social baseline: %w", err)
	}
	return avg, nil
}

// ============================================================
// Incidents
// ============================================================

// UpsertIncident creates or updates an incident.
func (db *DB) UpsertIncident(ctx context.Context, inc models.Incident) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := db.pool.Exec(ctx, `
		INSERT INTO incidents (id, target_url, started_at, resolved_at, peak_confidence, peak_status, signals_fired, notes)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		ON CONFLICT (id) DO UPDATE SET
			resolved_at = EXCLUDED.resolved_at,
			peak_confidence = GREATEST(incidents.peak_confidence, EXCLUDED.peak_confidence),
			peak_status = EXCLUDED.peak_status,
			signals_fired = EXCLUDED.signals_fired,
			notes = EXCLUDED.notes`,
		inc.ID, inc.TargetURL, inc.StartedAt, inc.ResolvedAt,
		inc.PeakConfidence, inc.PeakStatus, inc.SignalsFired, inc.Notes)
	return err
}

// GetOpenIncident returns the currently open incident for a target, if any.
func (db *DB) GetOpenIncident(ctx context.Context, targetURL string) (*models.Incident, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var inc models.Incident
	err := db.pool.QueryRow(ctx, `
		SELECT id, target_url, started_at, resolved_at, peak_confidence, peak_status, signals_fired, notes
		FROM incidents
		WHERE target_url = $1 AND resolved_at IS NULL
		ORDER BY started_at DESC LIMIT 1`, targetURL).Scan(
		&inc.ID, &inc.TargetURL, &inc.StartedAt, &inc.ResolvedAt,
		&inc.PeakConfidence, &inc.PeakStatus, &inc.SignalsFired, &inc.Notes)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query open incident: %w", err)
	}
	return &inc, nil
}

// GetIncidents returns incidents within a time range.
func (db *DB) GetIncidents(ctx context.Context, from, to time.Time) ([]models.Incident, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	rows, err := db.pool.Query(ctx, `
		SELECT id, target_url, started_at, resolved_at, peak_confidence, peak_status, signals_fired, notes
		FROM incidents
		WHERE started_at >= $1 AND started_at <= $2
		ORDER BY started_at DESC LIMIT 500`, from, to)
	if err != nil {
		return nil, fmt.Errorf("query incidents: %w", err)
	}
	defer rows.Close()

	var incidents []models.Incident
	for rows.Next() {
		var inc models.Incident
		if err := rows.Scan(&inc.ID, &inc.TargetURL, &inc.StartedAt, &inc.ResolvedAt,
			&inc.PeakConfidence, &inc.PeakStatus, &inc.SignalsFired, &inc.Notes); err != nil {
			return nil, fmt.Errorf("scan incident: %w", err)
		}
		incidents = append(incidents, inc)
	}
	return incidents, rows.Err()
}

// ============================================================
// Node Health
// ============================================================

// InsertNodeHealth inserts a node health record.
func (db *DB) InsertNodeHealth(ctx context.Context, h models.NodeHealth) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := db.pool.Exec(ctx, `
		INSERT INTO node_health (time, node_id, is_alive, baseline_ok, probe_count, avg_latency_ms, buffer_depth, version)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		h.Time, h.NodeID, h.IsAlive, h.BaselineOK, h.ProbeCount, h.AvgLatencyMs, h.BufferDepth, h.Version)
	return err
}

// GetLatestNodeHealth returns the most recent health record for each node.
func (db *DB) GetLatestNodeHealth(ctx context.Context) ([]models.NodeHealth, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	rows, err := db.pool.Query(ctx, `
		SELECT DISTINCT ON (node_id)
		       time, node_id, is_alive, baseline_ok, probe_count, avg_latency_ms, buffer_depth, version
		FROM node_health
		WHERE time >= now() - interval '10 minutes'
		ORDER BY node_id, time DESC`)
	if err != nil {
		return nil, fmt.Errorf("query node health: %w", err)
	}
	defer rows.Close()

	var health []models.NodeHealth
	for rows.Next() {
		var h models.NodeHealth
		if err := rows.Scan(&h.Time, &h.NodeID, &h.IsAlive, &h.BaselineOK,
			&h.ProbeCount, &h.AvgLatencyMs, &h.BufferDepth, &h.Version); err != nil {
			return nil, fmt.Errorf("scan node health: %w", err)
		}
		health = append(health, h)
	}
	return health, rows.Err()
}

// ============================================================
// Targets and Nodes
// ============================================================

// SyncTargets upserts targets from config into the database.
func (db *DB) SyncTargets(ctx context.Context, targets []models.Target) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	batch := &pgx.Batch{}
	for _, t := range targets {
		batch.Queue(`
			INSERT INTO targets (url, category, criticality, enabled, display_name)
			VALUES ($1,$2,$3,$4,$5)
			ON CONFLICT (url) DO UPDATE SET
				category = EXCLUDED.category,
				criticality = EXCLUDED.criticality,
				enabled = EXCLUDED.enabled,
				display_name = EXCLUDED.display_name`,
			t.URL, t.Category, t.Criticality, t.Enabled, t.DisplayName)
	}

	br := db.pool.SendBatch(ctx, batch)
	defer br.Close()

	for i := 0; i < len(targets); i++ {
		if _, err := br.Exec(); err != nil {
			return fmt.Errorf("sync target %d: %w", i, err)
		}
	}
	return nil
}

// SyncNodes upserts nodes from config into the database.
func (db *DB) SyncNodes(ctx context.Context, nodes []models.Node) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	batch := &pgx.Batch{}
	for _, n := range nodes {
		batch.Queue(`
			INSERT INTO nodes (node_id, region, country, enabled)
			VALUES ($1,$2,$3,$4)
			ON CONFLICT (node_id) DO UPDATE SET
				region = EXCLUDED.region,
				country = EXCLUDED.country,
				enabled = EXCLUDED.enabled`,
			n.NodeID, n.Region, n.Country, n.Enabled)
	}

	br := db.pool.SendBatch(ctx, batch)
	defer br.Close()

	for i := 0; i < len(nodes); i++ {
		if _, err := br.Exec(); err != nil {
			return fmt.Errorf("sync node %d: %w", i, err)
		}
	}
	return nil
}

// GetEnabledTargets returns all enabled targets.
func (db *DB) GetEnabledTargets(ctx context.Context) ([]models.Target, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	rows, err := db.pool.Query(ctx, `
		SELECT url, category, criticality, enabled, display_name
		FROM targets WHERE enabled = true ORDER BY criticality DESC`)
	if err != nil {
		return nil, fmt.Errorf("query targets: %w", err)
	}
	defer rows.Close()

	var targets []models.Target
	for rows.Next() {
		var t models.Target
		if err := rows.Scan(&t.URL, &t.Category, &t.Criticality, &t.Enabled, &t.DisplayName); err != nil {
			return nil, fmt.Errorf("scan target: %w", err)
		}
		targets = append(targets, t)
	}
	return targets, rows.Err()
}

// GetEnabledNodes returns all enabled nodes.
func (db *DB) GetEnabledNodes(ctx context.Context) ([]models.Node, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	rows, err := db.pool.Query(ctx, `
		SELECT node_id, region, country, enabled
		FROM nodes WHERE enabled = true`)
	if err != nil {
		return nil, fmt.Errorf("query nodes: %w", err)
	}
	defer rows.Close()

	var nodes []models.Node
	for rows.Next() {
		var n models.Node
		if err := rows.Scan(&n.NodeID, &n.Region, &n.Country, &n.Enabled); err != nil {
			return nil, fmt.Errorf("scan node: %w", err)
		}
		nodes = append(nodes, n)
	}
	return nodes, rows.Err()
}

// StreamExportCSV streams probe results for the last 7 days directly to an io.Writer
// (FIX H-2: avoids loading 100k+ rows into memory).
func (db *DB) StreamExportCSV(ctx context.Context, w io.Writer) error {
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	since := time.Now().UTC().Add(-7 * 24 * time.Hour)
	rows, err := db.pool.Query(ctx, `
		SELECT time, node_id, target_url, target_category, tcp_dial_ms, tcp_success,
		       tls_handshake_ms, http_status, ttfb_ms, total_ms, node_reliable,
		       error_type, error_detail
		FROM probe_results
		WHERE time >= $1
		ORDER BY time DESC
		LIMIT 100000`, since)
	if err != nil {
		return fmt.Errorf("query export csv: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var (
			t              time.Time
			nodeID         string
			targetURL      string
			targetCategory string
			tcpDialMs      int
			tcpSuccess     bool
			tlsHandshakeMs int
			httpStatus     int
			ttfbMs         int
			totalMs        int
			nodeReliable   bool
			errorType      string
			errorDetail    string
		)
		if err := rows.Scan(&t, &nodeID, &targetURL, &targetCategory,
			&tcpDialMs, &tcpSuccess, &tlsHandshakeMs,
			&httpStatus, &ttfbMs, &totalMs, &nodeReliable,
			&errorType, &errorDetail); err != nil {
			return fmt.Errorf("scan export: %w", err)
		}
		fmt.Fprintf(w, "%s,%s,%s,%s,%d,%t,%d,%d,%d,%d,%t,%s,%s\n",
			t.Format(time.RFC3339), nodeID, targetURL, targetCategory,
			tcpDialMs, tcpSuccess, tlsHandshakeMs, httpStatus,
			ttfbMs, totalMs, nodeReliable, errorType, errorDetail)
	}
	return rows.Err()
}
