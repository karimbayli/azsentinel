package correlation

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/karimbayli/sentinel-v2/internal/bgp"
	"github.com/karimbayli/sentinel-v2/internal/models"
	"github.com/karimbayli/sentinel-v2/internal/social"
	"github.com/karimbayli/sentinel-v2/internal/storage"
	"go.uber.org/zap"
)

// StatusThresholds maps confidence scores to incident states.
const (
	ThresholdMajor    = 0.8
	ThresholdPartial  = 0.5
	ThresholdDegraded = 0.3
	HysteresisBand    = 0.05 // FIX A-13: prevent oscillation at boundaries
)

// Engine correlates multiple signals to assess target health.
type Engine struct {
	db            *storage.DB
	bgpMonitor    *bgp.Monitor
	socialMonitor *social.Monitor
	logger        *zap.Logger

	// Configuration
	intervalSeconds int
	windowMinutes   int
	weightNode      float64
	weightBGP       float64
	weightSocial    float64
	nodeFailRatio   float64
	socialSpikeX    float64
	watchASNs       []int
	calibration     bool

	// BGP withdrawal threshold: minimum number of distinct prefixes
	// withdrawn to fire the BGP signal (prevents single-flap false positives)
	bgpWithdrawalThreshold int

	// FIX R-21: Minimum reliable nodes needed to fire node signal
	minReliableNodes int

	// State change callback
	onStateChange func(target models.Target, prev, curr string, result models.CorrelationResult)
}

// Config holds correlation engine parameters.
type Config struct {
	IntervalSeconds        int
	WindowMinutes          int
	WeightNode             float64
	WeightBGP              float64
	WeightSocial           float64
	NodeFailRatio          float64
	SocialSpikeX           float64
	MinReliableNodes       int // FIX R-21: minimum reliable nodes to fire signal (0 = default 2)
	WatchASNs              []int
	Calibration            bool
	BGPWithdrawalThreshold int // 0 = default (3)
}

// New creates a new correlation engine.
func New(db *storage.DB, bgpMon *bgp.Monitor, socialMon *social.Monitor, cfg Config, logger *zap.Logger) *Engine {
	bgpThreshold := cfg.BGPWithdrawalThreshold
	if bgpThreshold <= 0 {
		bgpThreshold = 3 // Require at least 3 distinct prefix withdrawals
	}

	minReliableNodes := cfg.MinReliableNodes
	if minReliableNodes <= 0 {
		minReliableNodes = 2 // FIX R-21: Require at least 2 reliable nodes
	}

	return &Engine{
		db:                     db,
		bgpMonitor:             bgpMon,
		socialMonitor:          socialMon,
		logger:                 logger,
		intervalSeconds:        cfg.IntervalSeconds,
		windowMinutes:          cfg.WindowMinutes,
		weightNode:             cfg.WeightNode,
		weightBGP:              cfg.WeightBGP,
		weightSocial:           cfg.WeightSocial,
		nodeFailRatio:          cfg.NodeFailRatio,
		socialSpikeX:           cfg.SocialSpikeX,
		minReliableNodes:       minReliableNodes,
		watchASNs:              cfg.WatchASNs,
		calibration:            cfg.Calibration,
		bgpWithdrawalThreshold: bgpThreshold,
	}
}

// OnStateChange sets the callback for state transitions (used for alerting).
func (e *Engine) OnStateChange(fn func(target models.Target, prev, curr string, result models.CorrelationResult)) {
	e.onStateChange = fn
}

// Run starts the correlation engine loop.
func (e *Engine) Run(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(e.intervalSeconds) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			e.logger.Info("correlation engine shutting down")
			return
		case <-ticker.C:
			e.assess(ctx)
		}
	}
}

// assess runs a complete assessment cycle over all targets.
func (e *Engine) assess(ctx context.Context) {
	// FIX M-8: Add timeout to prevent blocking forever
	assessCtx, cancel := context.WithTimeout(ctx, time.Duration(e.intervalSeconds)*time.Second)
	defer cancel()

	targets, err := e.db.GetEnabledTargets(assessCtx)
	if err != nil {
		e.logger.Error("failed to get targets for correlation", zap.Error(err))
		return
	}

	for _, target := range targets {
		// NEVER include ANCHOR targets in correlation
		if target.Category == "ANCHOR" {
			continue
		}

		// FIX C-2: Query previous state BEFORE computing and storing new result
		var prevStatus string
		prev, err := e.db.GetLatestCorrelation(assessCtx, target.URL)
		if err != nil {
			e.logger.Error("failed to get previous correlation", zap.Error(err))
			prevStatus = "HEALTHY"
		} else if prev != nil {
			prevStatus = prev.Status
		} else {
			prevStatus = "HEALTHY"
		}

		result := e.assessTarget(assessCtx, target, prevStatus)
		if result == nil {
			continue
		}

		// Store the NEW correlation result (AFTER reading previous)
		if err := e.db.InsertCorrelationResult(assessCtx, *result); err != nil {
			e.logger.Error("failed to store correlation result",
				zap.String("target", target.URL),
				zap.Error(err))
		}

		// Check for state change using the PREVIOUSLY-read status
		if e.onStateChange != nil && !e.calibration {
			if prevStatus != result.Status {
				e.onStateChange(target, prevStatus, result.Status, *result)

				// Manage incidents
				e.manageIncident(assessCtx, target, prevStatus, result)
			}
		}
	}
}

// assessTarget computes the correlation result for a single target.
// FIX A-13: prevStatus is used for hysteresis — recovery requires dropping
// below the threshold by HysteresisBand to prevent oscillation.
func (e *Engine) assessTarget(ctx context.Context, target models.Target, prevStatus string) *models.CorrelationResult {
	now := time.Now().UTC()
	window := time.Duration(e.windowMinutes) * time.Minute

	// Signal 1: Multi-node failure agreement
	probes, err := e.db.GetRecentProbeResults(ctx, target.URL, window)
	if err != nil {
		e.logger.Error("failed to get probe results for correlation",
			zap.String("target", target.URL),
			zap.Error(err))
		return nil
	}

	// Group by node, only count reliable nodes, use latest probe per node
	nodeLatest := make(map[string]models.ProbeResult)
	for _, p := range probes {
		if existing, ok := nodeLatest[p.NodeID]; !ok || p.Time.After(existing.Time) {
			nodeLatest[p.NodeID] = p
		}
	}

	totalReliable := 0
	failingReliable := 0
	totalNodes := len(nodeLatest)
	failingNodes := 0
	for _, p := range nodeLatest {
		if !p.NodeReliable {
			continue
		}
		totalReliable++
		if !p.TCPSuccess || (p.HTTPStatus > 0 && (p.HTTPStatus >= 500 || p.HTTPStatus == 0)) {
			failingReliable++
			failingNodes++
		}
	}

	// Count all failing regardless of reliability for reporting
	for _, p := range nodeLatest {
		if !p.TCPSuccess || (p.HTTPStatus > 0 && (p.HTTPStatus >= 500 || p.HTTPStatus == 0)) {
			if p.NodeReliable {
				continue // Already counted
			}
			failingNodes++
		}
	}

	var nodeSignal float64
	if totalReliable >= e.minReliableNodes && totalReliable > 0 {
		// FIX R-21: Only fire node signal if we have enough reliable nodes
		ratio := float64(failingReliable) / float64(totalReliable)
		if ratio > e.nodeFailRatio {
			nodeSignal = e.weightNode
		}
	}

	// Signal 2: BGP route withdrawal (FIX C-4: threshold-based, not single-event)
	// A single route flap should NOT trigger all targets. We require a minimum
	// number of distinct prefix withdrawals to consider it significant.
	var bgpSignal float64
	if e.bgpMonitor != nil {
		withdrawalCount := e.bgpMonitor.CountRecentWithdrawals(window)
		if withdrawalCount >= e.bgpWithdrawalThreshold {
			bgpSignal = e.weightBGP
		}
	}

	// Signal 3: Social mention spike
	// FIX C-5: Social signal is inherently global (mentions about "internet outage"
	// are not per-target). We apply a discount factor: only 50% weight unless
	// the node signal also fires, confirming the social spike aligns with observed failures.
	var socialSignal float64
	if e.socialMonitor != nil {
		sig := e.socialMonitor.GetLatestSignal()
		if sig != nil && sig.Ratio > e.socialSpikeX {
			if nodeSignal > 0 {
				// Social + node agreement = full weight
				socialSignal = e.weightSocial
			} else {
				// Social alone = halved weight (unconfirmed)
				socialSignal = e.weightSocial * 0.5
			}
		}
	}

	confidence := nodeSignal + bgpSignal + socialSignal

	// FIX A-13: Determine status with hysteresis
	status := determineStatus(confidence, prevStatus)

	// Build signals list
	signals := []string{}
	if nodeSignal > 0 {
		signals = append(signals, "MULTI_NODE_FAILURE")
	}
	if bgpSignal > 0 {
		signals = append(signals, "BGP_WITHDRAWAL")
	}
	if socialSignal > 0 {
		signals = append(signals, "SOCIAL_SPIKE")
	}

	return &models.CorrelationResult{
		Time:          now,
		TargetURL:     target.URL,
		Status:        status,
		Confidence:    confidence,
		NodeSignal:    nodeSignal,
		BGPSignal:     bgpSignal,
		SocialSignal:  socialSignal,
		SignalsActive: signals,
		TotalNodes:    totalNodes,
		FailingNodes:  failingNodes,
	}
}

// manageIncident creates or resolves incidents based on state changes.
func (e *Engine) manageIncident(ctx context.Context, target models.Target, prevStatus string, result *models.CorrelationResult) {
	if result.Status != "HEALTHY" && prevStatus == "HEALTHY" {
		// New incident
		inc := models.Incident{
			ID:             uuid.New().String(),
			TargetURL:      target.URL,
			StartedAt:      result.Time,
			PeakConfidence: result.Confidence,
			PeakStatus:     result.Status,
			SignalsFired:   result.SignalsActive,
		}
		if err := e.db.UpsertIncident(ctx, inc); err != nil {
			e.logger.Error("failed to create incident", zap.Error(err))
		}
	} else if result.Status == "HEALTHY" && prevStatus != "HEALTHY" {
		// Resolve incident
		openInc, err := e.db.GetOpenIncident(ctx, target.URL)
		if err != nil {
			e.logger.Error("failed to get open incident", zap.Error(err))
			return
		}
		if openInc != nil {
			resolvedAt := result.Time
			openInc.ResolvedAt = &resolvedAt
			if err := e.db.UpsertIncident(ctx, *openInc); err != nil {
				e.logger.Error("failed to resolve incident", zap.Error(err))
			}
		}
	}
}

// statusOrder maps status strings to numeric severity for comparison.
var statusOrder = map[string]int{
	"HEALTHY":        0,
	"DEGRADED":       1,
	"PARTIAL_OUTAGE": 2,
	"MAJOR_OUTAGE":   3,
}

// determineStatus maps confidence to status with hysteresis to prevent oscillation.
// FIX A-13: When recovering (going to lower severity), confidence must drop below
// the threshold by HysteresisBand before the status actually changes.
func determineStatus(confidence float64, prevStatus string) string {
	// Calculate what the raw thresholds would give us
	rawStatus := "HEALTHY"
	switch {
	case confidence > ThresholdMajor:
		rawStatus = "MAJOR_OUTAGE"
	case confidence > ThresholdPartial:
		rawStatus = "PARTIAL_OUTAGE"
	case confidence > ThresholdDegraded:
		rawStatus = "DEGRADED"
	}

	prevOrder := statusOrder[prevStatus]
	rawOrder := statusOrder[rawStatus]

	if rawOrder >= prevOrder {
		// Escalating or same — use raw thresholds (no hysteresis for escalation)
		return rawStatus
	}

	// Recovering — apply hysteresis: require dropping below threshold - band
	switch prevStatus {
	case "MAJOR_OUTAGE":
		if confidence > ThresholdMajor-HysteresisBand {
			return "MAJOR_OUTAGE" // Stay at MAJOR until clearly below
		}
	case "PARTIAL_OUTAGE":
		if confidence > ThresholdPartial-HysteresisBand {
			return "PARTIAL_OUTAGE" // Stay at PARTIAL until clearly below
		}
	case "DEGRADED":
		if confidence > ThresholdDegraded-HysteresisBand {
			return "DEGRADED" // Stay at DEGRADED until clearly below
		}
	}

	return rawStatus
}
