package bgp

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/karimbayli/sentinel-v2/internal/models"
	"github.com/karimbayli/sentinel-v2/internal/storage"
	"go.uber.org/zap"
)

// ASNProviders maps watched ASNs to their provider names.
var ASNProviders = map[int]string{
	29049: "Delta Telecom",
	31721: "Azertelecom",
	39232: "Bakcell",
	34377: "Azerfon/Nar Mobile",
	57021: "AZTELECOM",
}

// DefaultCollectors are RIPE RIS collectors with best visibility into AZ networks.
// FIX R-2: Multiple collectors instead of just rrc21.
var DefaultCollectors = []string{
	"rrc00", // Amsterdam — AMS-IX (major global peering)
	"rrc03", // Amsterdam — but peers with MSK-IX members
	"rrc13", // Moscow — MSK-IX (primary AZ peering point)
	"rrc20", // Zurich — DE-CIX/SwissIX
	"rrc21", // Paris — France-IX
}

// backoffSchedule defines the exponential backoff for reconnection.
var backoffSchedule = []time.Duration{
	5 * time.Second,
	10 * time.Second,
	30 * time.Second,
	60 * time.Second,
	120 * time.Second,
}

// prefixState tracks the BGP state of a prefix.
type prefixState struct {
	OriginASN    int
	Provider     string
	State        string // "ANNOUNCED", "WITHDRAWN"
	LastSeen     time.Time
	AnnouncedAt  time.Time
	WithdrawnAt  time.Time
	FlapCount    int       // Number of state changes within flapWindow
	FlapWindowAt time.Time // Start of current flap counting window
}

// Monitor connects to RIPE RIS Live WebSocket and watches for BGP events.
type Monitor struct {
	wsURL      string
	collectors []string
	watchASNs  map[int]bool
	db         *storage.DB
	logger     *zap.Logger

	mu             sync.RWMutex
	recentEvents   []models.BGPEvent
	maxRecentCount int
	cacheMaxAge    time.Duration

	// FIX R-1/R-16: Prefix → origin ASN state table
	prefixMu      sync.RWMutex
	prefixStates  map[string]*prefixState // key: "prefix:peerAS" for per-peer tracking
	flapWindow    time.Duration
	flapThreshold int
}

// New creates a new BGP monitor.
func New(wsURL string, collectors []string, asns []int, db *storage.DB, logger *zap.Logger) *Monitor {
	asnMap := make(map[int]bool)
	for _, a := range asns {
		asnMap[a] = true
	}

	// FIX R-2: Use default collectors if none specified
	if len(collectors) == 0 {
		collectors = DefaultCollectors
	}

	return &Monitor{
		wsURL:          wsURL,
		collectors:     collectors,
		watchASNs:      asnMap,
		db:             db,
		logger:         logger,
		maxRecentCount: 2000,
		cacheMaxAge:    15 * time.Minute,
		prefixStates:   make(map[string]*prefixState),
		flapWindow:     10 * time.Minute,
		flapThreshold:  5, // 5+ state changes in 10 min = flapping
	}
}

// HasRecentWithdrawals returns true if there are significant BGP withdrawals
// for watched ASNs within the given duration.
func (m *Monitor) HasRecentWithdrawals(window time.Duration) bool {
	return m.CountRecentWithdrawals(window) > 0
}

// CountRecentWithdrawals returns the number of distinct prefix withdrawals
// within the given window.
func (m *Monitor) CountRecentWithdrawals(window time.Duration) int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	cutoff := time.Now().UTC().Add(-window)
	seenPrefixes := make(map[string]bool)
	for _, e := range m.recentEvents {
		if e.EventType == "WITHDRAW" && e.Time.After(cutoff) {
			seenPrefixes[e.Prefix] = true
		}
	}
	return len(seenPrefixes)
}

// GetWithdrawnPrefixCount returns the number of prefixes currently in WITHDRAWN state.
func (m *Monitor) GetWithdrawnPrefixCount() int {
	m.prefixMu.RLock()
	defer m.prefixMu.RUnlock()

	count := 0
	for _, ps := range m.prefixStates {
		if ps.State == "WITHDRAWN" {
			count++
		}
	}
	return count
}

// Run starts the BGP monitor loop with automatic reconnection.
// FIX R-2: Launches one goroutine per collector.
func (m *Monitor) Run(ctx context.Context) {
	// Start cache cleanup goroutine
	go m.cacheCleanupLoop(ctx)
	go m.prefixStateCleanupLoop(ctx)

	var wg sync.WaitGroup
	for _, collName := range m.collectors {
		wg.Add(1)
		go func(coll string) {
			defer wg.Done()
			m.runCollector(ctx, coll)
		}(collName)
	}

	m.logger.Info("bgp monitor started",
		zap.Strings("collectors", m.collectors),
		zap.Int("watch_asns", len(m.watchASNs)))

	wg.Wait()
	m.logger.Info("bgp monitor stopped")
}

// runCollector manages the connection lifecycle for a single RIS collector.
func (m *Monitor) runCollector(ctx context.Context, collector string) {
	attempt := 0
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		err := m.connect(ctx, collector)
		if err != nil {
			backoff := backoffSchedule[min(attempt, len(backoffSchedule)-1)]
			m.logger.Error("bgp websocket disconnected, reconnecting",
				zap.String("collector", collector),
				zap.Error(err),
				zap.Duration("backoff", backoff),
				zap.Int("attempt", attempt+1))

			select {
			case <-ctx.Done():
				return
			case <-time.After(backoff):
			}
			attempt++
		} else {
			attempt = 0
		}
	}
}

// cacheCleanupLoop periodically removes expired events from the in-memory cache.
func (m *Monitor) cacheCleanupLoop(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.cleanExpiredEvents()
		}
	}
}

// prefixStateCleanupLoop periodically removes stale prefix states.
func (m *Monitor) prefixStateCleanupLoop(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.cleanStalePrefixStates()
		}
	}
}

// cleanExpiredEvents removes events older than cacheMaxAge.
func (m *Monitor) cleanExpiredEvents() {
	m.mu.Lock()
	defer m.mu.Unlock()

	cutoff := time.Now().UTC().Add(-m.cacheMaxAge)
	kept := make([]models.BGPEvent, 0, len(m.recentEvents))
	evicted := 0
	for _, e := range m.recentEvents {
		if e.Time.After(cutoff) {
			kept = append(kept, e)
		} else {
			evicted++
		}
	}

	if evicted > 0 {
		m.recentEvents = kept
		m.logger.Debug("bgp cache cleanup", zap.Int("evicted", evicted), zap.Int("remaining", len(kept)))
	}
}

// cleanStalePrefixStates removes prefix states not seen in over 1 hour.
func (m *Monitor) cleanStalePrefixStates() {
	m.prefixMu.Lock()
	defer m.prefixMu.Unlock()

	cutoff := time.Now().UTC().Add(-1 * time.Hour)
	evicted := 0
	for key, ps := range m.prefixStates {
		if ps.LastSeen.Before(cutoff) {
			delete(m.prefixStates, key)
			evicted++
		}
	}

	if evicted > 0 {
		m.logger.Debug("prefix state cleanup", zap.Int("evicted", evicted), zap.Int("remaining", len(m.prefixStates)))
	}
}

// connect establishes a WebSocket connection and processes messages.
func (m *Monitor) connect(ctx context.Context, collector string) error {
	m.logger.Info("connecting to RIPE RIS Live",
		zap.String("url", m.wsURL),
		zap.String("collector", collector))

	conn, _, err := websocket.DefaultDialer.DialContext(ctx, m.wsURL, nil)
	if err != nil {
		return fmt.Errorf("dial ws: %w", err)
	}
	defer conn.Close()

	// Build subscription data — subscribe to all updates for this collector to avoid RIPE rejecting complex regex paths.
	// Filtering by ASN is already handled locally in processMessage().
	subData := map[string]interface{}{
		"type": "UPDATE",
		"host": collector,
		"socketOptions": map[string]bool{
			"includeRaw": false,
		},
	}

	subscribe := map[string]interface{}{
		"type": "ris_subscribe",
		"data": subData,
	}
	if err := conn.WriteJSON(subscribe); err != nil {
		return fmt.Errorf("subscribe: %w", err)
	}

	m.logger.Info("subscribed to RIPE RIS Live",
		zap.String("collector", collector))

	// Keepalive: send periodic pings to prevent idle disconnect
	pingDone := make(chan struct{})
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					return
				}
			case <-pingDone:
				return
			case <-ctx.Done():
				return
			}
		}
	}()
	defer close(pingDone)

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		conn.SetReadDeadline(time.Now().Add(90 * time.Second))
		_, message, err := conn.ReadMessage()
		if err != nil {
			return fmt.Errorf("read message: %w", err)
		}

		m.processMessage(ctx, message, collector)
	}
}

// risMessage represents a RIPE RIS Live message.
type risMessage struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

// risUpdate represents BGP UPDATE data from RIPE RIS.
type risUpdate struct {
	Timestamp     float64 `json:"timestamp"`
	Peer          string  `json:"peer"`
	PeerASN       int     `json:"peer_asn"`
	Host          string  `json:"host"`
	Path          []int   `json:"path"`
	Announcements []struct {
		NextHop  string   `json:"next_hop"`
		Prefixes []string `json:"prefixes"`
	} `json:"announcements"`
	Withdrawals []struct {
		Prefixes []string `json:"prefixes"`
	} `json:"withdrawals"`
}

// processMessage parses and handles a single RIS Live message.
func (m *Monitor) processMessage(ctx context.Context, raw []byte, collector string) {
	var msg risMessage
	if err := json.Unmarshal(raw, &msg); err != nil {
		return
	}

	if msg.Type != "ris_message" {
		return
	}

	var update risUpdate
	if err := json.Unmarshal(msg.Data, &update); err != nil {
		return
	}

	// FIX R-4: Sub-second timestamp precision
	sec := int64(update.Timestamp)
	nsec := int64((update.Timestamp - float64(sec)) * 1e9)
	eventTime := time.Unix(sec, nsec).UTC()

	// FIX R-3: Extract ORIGIN ASN (last element of path, not any element)
	var originASN int
	if len(update.Path) > 0 {
		// Handle AS-path prepending: the origin is the last unique ASN
		originASN = update.Path[len(update.Path)-1]
	}

	// Process announcements — update prefix state table
	for _, ann := range update.Announcements {
		for _, prefix := range ann.Prefixes {
			// FIX R-3: Only process if the ORIGIN ASN is one we watch
			if originASN > 0 && m.watchASNs[originASN] {
				// FIX R-1: Track prefix → origin ASN mapping
				m.updatePrefixState(prefix, update.PeerASN, originASN, "ANNOUNCED", eventTime)

				event := models.BGPEvent{
					Time:      eventTime,
					ASN:       originASN,
					Provider:  ASNProviders[originASN],
					Prefix:    prefix,
					EventType: "ANNOUNCE",
					PeerAS:    update.PeerASN,
					Collector: collector,
				}
				m.storeEvent(ctx, event)
			}
		}
	}

	// Process withdrawals — FIX R-1: Look up origin ASN from prefix state table
	for _, wd := range update.Withdrawals {
		for _, prefix := range wd.Prefixes {
			// FIX R-1: Withdrawals have NO AS-path. Look up the origin ASN
			// from our prefix state table (populated by prior announcements).
			knownOrigin := m.lookupPrefixOrigin(prefix, update.PeerASN)
			if knownOrigin > 0 && m.watchASNs[knownOrigin] {
				m.updatePrefixState(prefix, update.PeerASN, knownOrigin, "WITHDRAWN", eventTime)

				event := models.BGPEvent{
					Time:      eventTime,
					ASN:       knownOrigin,
					Provider:  ASNProviders[knownOrigin],
					Prefix:    prefix,
					EventType: "WITHDRAW",
					PeerAS:    update.PeerASN,
					Collector: collector,
				}
				m.storeEvent(ctx, event)
			} else if knownOrigin == 0 && originASN > 0 && m.watchASNs[originASN] {
				// Fallback: some withdrawals DO include a path (implementation varies)
				m.updatePrefixState(prefix, update.PeerASN, originASN, "WITHDRAWN", eventTime)

				event := models.BGPEvent{
					Time:      eventTime,
					ASN:       originASN,
					Provider:  ASNProviders[originASN],
					Prefix:    prefix,
					EventType: "WITHDRAW",
					PeerAS:    update.PeerASN,
					Collector: collector,
				}
				m.storeEvent(ctx, event)
			}
		}
	}
}

// prefixKey builds a unique key for prefix state tracking.
func prefixKey(prefix string, peerAS int) string {
	return fmt.Sprintf("%s:%d", prefix, peerAS)
}

// lookupPrefixOrigin returns the last known origin ASN for a prefix from a specific peer.
// FIX R-1: This is the critical function that makes withdrawals work correctly.
func (m *Monitor) lookupPrefixOrigin(prefix string, peerAS int) int {
	m.prefixMu.RLock()
	defer m.prefixMu.RUnlock()

	key := prefixKey(prefix, peerAS)
	if ps, ok := m.prefixStates[key]; ok {
		return ps.OriginASN
	}
	return 0
}

// updatePrefixState updates the state machine for a prefix.
// FIX R-16: Tracks ANNOUNCED → WITHDRAWN → FLAPPING transitions.
func (m *Monitor) updatePrefixState(prefix string, peerAS, originASN int, newState string, eventTime time.Time) {
	m.prefixMu.Lock()
	defer m.prefixMu.Unlock()

	key := prefixKey(prefix, peerAS)
	ps, exists := m.prefixStates[key]

	if !exists {
		ps = &prefixState{
			OriginASN:    originASN,
			Provider:     ASNProviders[originASN],
			State:        newState,
			LastSeen:     eventTime,
			FlapWindowAt: eventTime,
		}
		if newState == "ANNOUNCED" {
			ps.AnnouncedAt = eventTime
		} else {
			ps.WithdrawnAt = eventTime
		}
		m.prefixStates[key] = ps
		return
	}

	// Detect state change
	if ps.State != newState {
		ps.FlapCount++

		// Reset flap count if outside the window
		if eventTime.Sub(ps.FlapWindowAt) > m.flapWindow {
			ps.FlapCount = 1
			ps.FlapWindowAt = eventTime
		}

		// Check for flapping
		if ps.FlapCount >= m.flapThreshold {
			m.logger.Warn("bgp prefix flapping detected",
				zap.String("prefix", prefix),
				zap.Int("origin_asn", originASN),
				zap.Int("flap_count", ps.FlapCount))
		}
	}

	ps.OriginASN = originASN
	ps.Provider = ASNProviders[originASN]
	ps.State = newState
	ps.LastSeen = eventTime

	if newState == "ANNOUNCED" {
		ps.AnnouncedAt = eventTime
	} else {
		ps.WithdrawnAt = eventTime
	}
}

// storeEvent persists a BGP event and adds it to the in-memory recent events cache.
func (m *Monitor) storeEvent(ctx context.Context, event models.BGPEvent) {
	m.logger.Info("bgp event",
		zap.String("type", event.EventType),
		zap.Int("asn", event.ASN),
		zap.String("provider", event.Provider),
		zap.String("prefix", event.Prefix),
		zap.String("collector", event.Collector))

	if err := m.db.InsertBGPEvent(ctx, event); err != nil {
		m.logger.Error("failed to store bgp event", zap.Error(err))
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.recentEvents = append(m.recentEvents, event)
	if len(m.recentEvents) > m.maxRecentCount {
		m.recentEvents = m.recentEvents[len(m.recentEvents)-m.maxRecentCount:]
	}
}
