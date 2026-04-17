package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/karimbayli/sentinel-v2/internal/alert"
	"github.com/karimbayli/sentinel-v2/internal/api"
	"github.com/karimbayli/sentinel-v2/internal/bgp"
	"github.com/karimbayli/sentinel-v2/internal/config"
	"github.com/karimbayli/sentinel-v2/internal/correlation"
	"github.com/karimbayli/sentinel-v2/internal/logger"
	"github.com/karimbayli/sentinel-v2/internal/models"
	"github.com/karimbayli/sentinel-v2/internal/probe"
	"github.com/karimbayli/sentinel-v2/internal/social"
	"github.com/karimbayli/sentinel-v2/internal/storage"
	"go.uber.org/zap"
)

const version = "1.0.0"

func main() {
	configPath := flag.String("config", "configs/central.yaml", "path to config file")
	calibration := flag.Bool("calibration", false, "enable calibration mode (no alerts, no incidents)")
	flag.Parse()

	// Check env override for calibration
	if os.Getenv("SENTINEL_CALIBRATION") == "true" {
		*calibration = true
	}

	// Load config
	cfg, err := config.LoadCentralConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	if *calibration {
		cfg.Calibration = true
	}

	// Initialize logger
	logger := logger.New(cfg.LogLevel)
	defer logger.Sync()

	logger.Info("starting sentinel v2 central server",
		zap.String("version", version),
		zap.Bool("calibration", cfg.Calibration))

	// Create context with signal handling for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Connect to TimescaleDB
	db, err := storage.New(ctx, cfg.Database.DSN(), logger)
	if err != nil {
		logger.Fatal("failed to connect to database", zap.Error(err))
	}
	defer db.Close()
	logger.Info("connected to TimescaleDB")

	// Sync targets and nodes from config
	if err := db.SyncTargets(ctx, cfg.Targets); err != nil {
		logger.Fatal("failed to sync targets", zap.Error(err))
	}
	if err := db.SyncNodes(ctx, cfg.Nodes); err != nil {
		logger.Fatal("failed to sync nodes", zap.Error(err))
	}
	logger.Info("synced config to database",
		zap.Int("targets", len(cfg.Targets)),
		zap.Int("nodes", len(cfg.Nodes)))

	// Start BGP Monitor
	var bgpMonitor *bgp.Monitor
	if cfg.BGP.Enabled {
		bgpMonitor = bgp.New(cfg.BGP.WSURL, cfg.BGP.Collectors, cfg.BGP.WatchASNs, db, logger)
		go bgpMonitor.Run(ctx)
		logger.Info("bgp monitor started", zap.Strings("collectors", cfg.BGP.Collectors))
	}

	// Start Social Monitor
	var socialMonitor *social.Monitor
	if cfg.Social.Enabled {
		socialMonitor = social.New(
			cfg.Social.BotToken,
			cfg.Social.ChannelIDs,
			cfg.Social.WindowMinutes,
			cfg.Social.BaselineDays,
			cfg.Social.BotFilterRate,
			db, logger,
		)
		go socialMonitor.Run(ctx)
		logger.Info("social monitor started")
	}

	// Start Alert Dispatcher
	var alertDispatcher *alert.Dispatcher
	if cfg.Alert.Enabled && !cfg.Calibration {
		alertDispatcher = alert.New(
			cfg.Alert.BotToken,
			cfg.Alert.ChatID,
			cfg.Alert.DedupMinutes,
			logger,
		)
		logger.Info("alert dispatcher initialized")
	}

	// Start Correlation Engine
	engine := correlation.New(db, bgpMonitor, socialMonitor, correlation.Config{
		IntervalSeconds:  cfg.Correlation.IntervalSeconds,
		WindowMinutes:    cfg.Correlation.WindowMinutes,
		WeightNode:       cfg.Correlation.WeightNode,
		WeightBGP:        cfg.Correlation.WeightBGP,
		WeightSocial:     cfg.Correlation.WeightSocial,
		NodeFailRatio:    cfg.Correlation.NodeFailRatio,
		SocialSpikeX:     cfg.Correlation.SocialSpikeX,
		MinReliableNodes: cfg.Correlation.MinReliableNodes,
		WatchASNs:        cfg.BGP.WatchASNs,
		Calibration:      cfg.Calibration,
	}, logger)

	if alertDispatcher != nil {
		engine.OnStateChange(func(target models.Target, prev, curr string, result models.CorrelationResult) {
			// NEVER alert on ANCHOR targets
			if target.Category == "ANCHOR" {
				return
			}
			alertDispatcher.SendAlert(ctx, target, prev, curr, result)
		})
	}

	go engine.Run(ctx)
	logger.Info("correlation engine started",
		zap.Int("interval_sec", cfg.Correlation.IntervalSeconds),
		zap.Int("window_min", cfg.Correlation.WindowMinutes))

	// Start Local Probe Agent (node-eu-central)
	localNodeID := "node-eu-central"
	for _, n := range cfg.Nodes {
		if n.Country == "AZ" {
			localNodeID = n.NodeID
			break
		}
	}

	prober := probe.New(
		localNodeID,
		"eu-frankfurt",
		cfg.Targets,
		cfg.Probe.TCPTimeout,
		cfg.Probe.HTTPTimeout,
		logger,
	)

	go runLocalProber(ctx, prober, db, cfg.Probe.IntervalSeconds, localNodeID, logger)

	// Start REST API
	apiServer := api.New(db, cfg.Server.HMACSecret, cfg.Nodes, cfg.Targets, logger)
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)

	httpServer := &http.Server{
		Addr:         addr,
		Handler:      apiServer.Handler(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.Info("http server starting", zap.String("addr", addr))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("http server error", zap.Error(err))
		}
	}()

	if cfg.Calibration {
		logger.Warn("⚠️  CALIBRATION MODE ACTIVE — no alerts or incidents will be generated")
	}

	// Wait for shutdown signal
	sig := <-sigCh
	logger.Info("received shutdown signal", zap.String("signal", sig.String()))

	// Graceful shutdown
	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("http server shutdown error", zap.Error(err))
	}

	logger.Info("sentinel v2 central server stopped")
}

// runLocalProber runs the local Europe fallback probe node.
// Also reports node health so node-eu-central appears in /api/v1/nodes.
func runLocalProber(ctx context.Context, prober *probe.Prober, db *storage.DB, intervalSec int, nodeID string, logger *zap.Logger) {
	ticker := time.NewTicker(time.Duration(intervalSec) * time.Second)
	defer ticker.Stop()

	logger.Info("local prober started", zap.Int("interval_sec", intervalSec), zap.String("node_id", nodeID))

	// Run immediately
	runLocalProbeAndReport(ctx, prober, db, nodeID, logger)

	for {
		select {
		case <-ctx.Done():
			logger.Info("local prober stopping")
			return
		case <-ticker.C:
			runLocalProbeAndReport(ctx, prober, db, nodeID, logger)
		}
	}
}

// runLocalProbeAndReport probes and stores both results and node health.
func runLocalProbeAndReport(ctx context.Context, prober *probe.Prober, db *storage.DB, nodeID string, logger *zap.Logger) {
	results := prober.RunCycle(ctx)
	if err := db.InsertProbeResults(ctx, results); err != nil {
		logger.Error("failed to store local probe results", zap.Error(err))
	}

	// FIX M-7: Report node health for the local node
	health := models.NodeHealth{
		Time:       time.Now().UTC(),
		NodeID:     nodeID,
		IsAlive:    true,
		BaselineOK: true,
		ProbeCount: len(results),
		Version:    version,
	}
	for _, r := range results {
		if !r.NodeReliable {
			health.BaselineOK = false
			break
		}
	}
	totalLatency, count := 0, 0
	for _, r := range results {
		if r.TotalMs > 0 {
			totalLatency += r.TotalMs
			count++
		}
	}
	if count > 0 {
		health.AvgLatencyMs = totalLatency / count
	}
	if err := db.InsertNodeHealth(ctx, health); err != nil {
		logger.Error("failed to store local node health", zap.Error(err))
	}

	logger.Debug("local probe cycle complete", zap.Int("results", len(results)))
}
