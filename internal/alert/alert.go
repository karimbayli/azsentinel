package alert

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/karimbayli/sentinel-v2/internal/models"
	"go.uber.org/zap"
)

// Dispatcher sends Telegram alerts for state transitions.
type Dispatcher struct {
	botToken     string
	chatID       int64
	dedupMinutes int
	logger       *zap.Logger
	httpClient   *http.Client

	mu         sync.Mutex
	lastAlerts map[string]time.Time // target_url → last alert time
}

// New creates a new alert dispatcher.
func New(botToken string, chatID int64, dedupMinutes int, logger *zap.Logger) *Dispatcher {
	return &Dispatcher{
		botToken:     botToken,
		chatID:       chatID,
		dedupMinutes: dedupMinutes,
		logger:       logger,
		httpClient:   &http.Client{Timeout: 10 * time.Second},
		lastAlerts:   make(map[string]time.Time),
	}
}

// SendAlert sends an alert for a target state transition.
func (d *Dispatcher) SendAlert(ctx context.Context, target models.Target, prevStatus, newStatus string, result models.CorrelationResult) {
	// De-duplication check
	d.mu.Lock()
	if last, ok := d.lastAlerts[target.URL]; ok {
		if time.Since(last) < time.Duration(d.dedupMinutes)*time.Minute {
			d.mu.Unlock()
			d.logger.Debug("alert deduplicated",
				zap.String("target", target.URL),
				zap.String("status", newStatus))
			return
		}
	}
	d.lastAlerts[target.URL] = time.Now()

	// FIX A-8: Clean up stale entries (older than 3x dedup window)
	maxAge := time.Duration(d.dedupMinutes*3) * time.Minute
	for url, ts := range d.lastAlerts {
		if time.Since(ts) > maxAge {
			delete(d.lastAlerts, url)
		}
	}
	d.mu.Unlock()

	// Build alert message
	msg := d.formatAlert(target, prevStatus, newStatus, result)

	if err := d.sendTelegram(ctx, msg); err != nil {
		d.logger.Error("failed to send telegram alert",
			zap.String("target", target.URL),
			zap.Error(err))
	} else {
		d.logger.Info("alert sent",
			zap.String("target", target.URL),
			zap.String("new_status", newStatus))
	}
}

// SendStatusOverview sends a summary of all target statuses.
func (d *Dispatcher) SendStatusOverview(ctx context.Context, statuses []models.TargetStatus) error {
	var sb strings.Builder
	sb.WriteString("🔍 *Sentinel V2 — Status Overview*\n\n")

	for _, s := range statuses {
		emoji := statusEmoji(s.Status)
		sb.WriteString(fmt.Sprintf("%s *%s*\n", emoji, s.Target.DisplayName))
		sb.WriteString(fmt.Sprintf("   Status: `%s` | Confidence: `%.1f%%`\n", s.Status, s.Confidence*100))
		sb.WriteString(fmt.Sprintf("   URL: %s\n\n", s.Target.URL))
	}

	return d.sendTelegram(ctx, sb.String())
}

// formatAlert builds a human-readable Telegram alert message.
func (d *Dispatcher) formatAlert(target models.Target, prevStatus, newStatus string, result models.CorrelationResult) string {
	var sb strings.Builder

	emoji := statusEmoji(newStatus)
	direction := "⬆️ ESCALATED"
	if isRecovery(prevStatus, newStatus) {
		direction = "⬇️ RECOVERED"
		emoji = "✅"
	}

	sb.WriteString(fmt.Sprintf("%s *%s — %s*\n\n", emoji, target.DisplayName, direction))
	sb.WriteString(fmt.Sprintf("🎯 *Target:* `%s`\n", target.URL))
	sb.WriteString(fmt.Sprintf("📊 *Status:* `%s` → `%s`\n", prevStatus, newStatus))
	sb.WriteString(fmt.Sprintf("🔢 *Confidence:* `%.1f%%`\n", result.Confidence*100))
	sb.WriteString(fmt.Sprintf("📡 *Category:* `%s` | Criticality: `%d/10`\n\n", target.Category, target.Criticality))

	// Signal breakdown
	sb.WriteString("*Signals:*\n")
	if result.NodeSignal > 0 {
		sb.WriteString(fmt.Sprintf("  🖥️ Node failure: %d/%d nodes failing\n",
			result.FailingNodes, result.TotalNodes))
	}
	if result.BGPSignal > 0 {
		sb.WriteString("  🌐 BGP: Route withdrawal detected\n")
	}
	if result.SocialSignal > 0 {
		sb.WriteString("  📱 Social: Mention spike detected\n")
	}
	if len(result.SignalsActive) == 0 {
		sb.WriteString("  ✅ All signals normal\n")
	}

	sb.WriteString(fmt.Sprintf("\n🕐 *Time:* `%s`", result.Time.Format(time.RFC3339)))

	return sb.String()
}

// sendTelegram sends a message via the Telegram Bot API.
func (d *Dispatcher) sendTelegram(ctx context.Context, text string) error {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", d.botToken)

	payload := map[string]interface{}{
		"chat_id":    d.chatID,
		"text":       text,
		"parse_mode": "Markdown",
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send message: %w", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram api returned %d", resp.StatusCode)
	}

	return nil
}

// HandleStatusCommand processes the /status bot command.
func (d *Dispatcher) HandleStatusCommand(ctx context.Context, statuses []models.TargetStatus) {
	if err := d.SendStatusOverview(ctx, statuses); err != nil {
		d.logger.Error("failed to send status overview", zap.Error(err))
	}
}

func statusEmoji(status string) string {
	switch status {
	case "MAJOR_OUTAGE":
		return "🔴"
	case "PARTIAL_OUTAGE":
		return "🟠"
	case "DEGRADED":
		return "🟡"
	default:
		return "🟢"
	}
}

func isRecovery(prev, curr string) bool {
	order := map[string]int{"HEALTHY": 0, "DEGRADED": 1, "PARTIAL_OUTAGE": 2, "MAJOR_OUTAGE": 3}
	return order[curr] < order[prev]
}
