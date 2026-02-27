package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/karimbayli/sentinel-v2/internal/api"
	"github.com/karimbayli/sentinel-v2/internal/buffer"
	"github.com/karimbayli/sentinel-v2/internal/config"
	"github.com/karimbayli/sentinel-v2/internal/models"
	"github.com/karimbayli/sentinel-v2/internal/probe"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"encoding/json"
)

const version = "1.0.0"

var (
	bufferDepthGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "sentinel_probe_buffer_depth",
		Help: "Number of probe batches currently buffered locally",
	})
	probeResultsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "sentinel_probe_results_total",
		Help: "Total probe results generated",
	})
	flushErrorsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "sentinel_probe_flush_errors_total",
		Help: "Total flush errors when sending to central",
	})
)

func main() {
	configPath := flag.String("config", "configs/probe.yaml", "path to config file")
	flag.Parse()

	cfg, err := config.LoadProbeAgentConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	logger := initLogger(cfg.LogLevel)
	defer logger.Sync()

	logger.Info("starting sentinel v2 probe agent",
		zap.String("version", version),
		zap.String("node_id", cfg.NodeID),
		zap.String("region", cfg.Region),
		zap.String("central_url", cfg.CentralURL))

	// Create context with signal handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Initialize local buffer queue
	bufferQueue, err := buffer.New(cfg.Buffer.DBPath, cfg.Buffer.MaxSize, logger)
	if err != nil {
		logger.Fatal("failed to initialize buffer queue", zap.Error(err))
	}
	defer bufferQueue.Close()

	// Initialize prober
	prober := probe.New(
		cfg.NodeID,
		cfg.Region,
		cfg.Targets,
		5*time.Second,  // TCP timeout
		10*time.Second, // HTTP timeout
		logger,
	)

	// Start background flush goroutine
	go flushLoop(ctx, cfg, bufferQueue, logger)

	// Main probe loop
	ticker := time.NewTicker(cfg.ProbeInterval)
	defer ticker.Stop()

	logger.Info("probe agent running",
		zap.Duration("interval", cfg.ProbeInterval),
		zap.Int("targets", len(cfg.Targets)))

	// Run immediately
	runProbeAndSend(ctx, cfg, prober, bufferQueue, logger)

	for {
		select {
		case <-ctx.Done():
			logger.Info("probe agent stopping")
			return
		case sig := <-sigCh:
			logger.Info("received shutdown signal", zap.String("signal", sig.String()))
			cancel()
			return
		case <-ticker.C:
			runProbeAndSend(ctx, cfg, prober, bufferQueue, logger)
		}
	}
}

// runProbeAndSend executes a probe cycle and attempts to send results to central.
func runProbeAndSend(ctx context.Context, cfg *config.ProbeAgentConfig, prober *probe.Prober, bufferQueue *buffer.Queue, logger *zap.Logger) {
	results := prober.RunCycle(ctx)
	probeResultsTotal.Add(float64(len(results)))

	batch := models.ProbeBatch{
		NodeID:    cfg.NodeID,
		Region:    cfg.Region,
		Timestamp: time.Now().UTC(),
		Version:   version,
		SentAt:    time.Now().Unix(), // FIX A-1: anti-replay timestamp
		Nonce:     generateNonce(),   // FIX A-1: unique nonce per request
		Results:   results,
	}

	logger.Info("probe cycle complete",
		zap.Int("results", len(results)),
		zap.String("node_id", cfg.NodeID))

	// Try to send directly to central
	if err := sendBatch(ctx, cfg, batch); err != nil {
		logger.Warn("failed to send batch to central, buffering locally",
			zap.Error(err))

		// Buffer locally
		if err := bufferQueue.Push(batch); err != nil {
			logger.Error("failed to buffer batch locally", zap.Error(err))
		}

		bufferDepthGauge.Set(float64(bufferQueue.Depth()))
	}
}

// sendBatch sends a probe batch to the central ingest endpoint with HMAC signature.
func sendBatch(ctx context.Context, cfg *config.ProbeAgentConfig, batch models.ProbeBatch) error {
	body, err := json.Marshal(batch)
	if err != nil {
		return fmt.Errorf("marshal batch: %w", err)
	}

	// Compute HMAC signature
	signature := api.ComputeHMAC(body, cfg.HMACSecret)

	url := cfg.CentralURL + "/api/v1/ingest/probe-batch"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Sentinel-Signature", signature)
	req.Header.Set("X-Sentinel-Node", cfg.NodeID)

	// FIX A-9: Reuse shared HTTP client instead of creating per-request
	resp, err := sharedHTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("post to central: %w", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("central returned %d", resp.StatusCode)
	}

	return nil
}

// FIX A-9: Shared HTTP client with connection pooling
var sharedHTTPClient = &http.Client{
	Timeout: 15 * time.Second,
	Transport: &http.Transport{
		MaxIdleConns:        10,
		MaxIdleConnsPerHost: 5,
		IdleConnTimeout:     60 * time.Second,
	},
}

// generateNonce creates a cryptographically random 16-byte hex nonce.
// FIX A-1: Each batch gets a unique nonce to prevent replay.
func generateNonce() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// flushLoop periodically attempts to flush buffered results to central.
func flushLoop(ctx context.Context, cfg *config.ProbeAgentConfig, bufferQueue *buffer.Queue, logger *zap.Logger) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	consecutiveFailures := 0

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			depth := bufferQueue.Depth()
			bufferDepthGauge.Set(float64(depth))

			if depth == 0 {
				consecutiveFailures = 0
				continue
			}

			logger.Info("attempting to flush buffer",
				zap.Int("depth", depth))

			items, err := bufferQueue.Peek(50) // Flush in batches of 50
			if err != nil {
				logger.Error("failed to peek buffer", zap.Error(err))
				continue
			}

			var flushedIDs []int64
			for _, item := range items {
				if err := sendBatch(ctx, cfg, item.Batch); err != nil {
					flushErrorsTotal.Inc()
					consecutiveFailures++

					// FIX H-7: Exponential backoff with context-aware wait
					backoff := time.Duration(math.Min(float64(consecutiveFailures)*5, 120)) * time.Second
					logger.Warn("flush failed, will retry with backoff",
						zap.Error(err),
						zap.Duration("backoff", backoff),
						zap.Int("consecutive_failures", consecutiveFailures))

					// Wait with context cancellation support
					select {
					case <-ctx.Done():
						return
					case <-time.After(backoff):
					}
					break
				}
				flushedIDs = append(flushedIDs, item.ID)
				consecutiveFailures = 0
			}

			if len(flushedIDs) > 0 {
				if err := bufferQueue.Remove(flushedIDs); err != nil {
					logger.Error("failed to remove flushed items from buffer", zap.Error(err))
				} else {
					logger.Info("flushed buffered results",
						zap.Int("flushed", len(flushedIDs)),
						zap.Int("remaining", bufferQueue.Depth()))
				}
			}
		}
	}
}

func initLogger(level string) *zap.Logger {
	var lvl zapcore.Level
	switch level {
	case "debug":
		lvl = zapcore.DebugLevel
	case "warn":
		lvl = zapcore.WarnLevel
	case "error":
		lvl = zapcore.ErrorLevel
	default:
		lvl = zapcore.InfoLevel
	}

	cfg := zap.Config{
		Level:            zap.NewAtomicLevelAt(lvl),
		Encoding:         "json",
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
		EncoderConfig: zapcore.EncoderConfig{
			TimeKey:        "ts",
			LevelKey:       "level",
			NameKey:        "logger",
			CallerKey:      "caller",
			MessageKey:     "msg",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.LowercaseLevelEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.SecondsDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		},
	}

	logger, err := cfg.Build()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to init logger: %v\n", err)
		os.Exit(1)
	}
	return logger
}
