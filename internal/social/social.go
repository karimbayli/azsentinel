package social

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/karimbayli/sentinel-v2/internal/models"
	"github.com/karimbayli/sentinel-v2/internal/storage"
	"go.uber.org/zap"
)

// FIX A-6: English keywords require AZ context to avoid false positives from generic "down".
var keywordsAZ = []string{"kəsildi", "işləmir", "internet yoxdur", "bağlandı", "xəta", "olmur"}
var keywordsRU = []string{"не работает", "интернет пропал", "отключили", "недоступен"}
var keywordsEN = []string{"internet outage", "internet down", "not working in baku", "Azerbaijan internet", "Azerbaijan outage"}

// Location hints for Azerbaijan.
var locationHints = []string{"baku", "ganja", "sumqayit", "bakı", "gəncə", "sumqayıt", "azerbaijan", "azərbaycan"}

// AllKeywords returns all keywords across languages.
func AllKeywords() []string {
	all := make([]string, 0, len(keywordsAZ)+len(keywordsRU)+len(keywordsEN))
	all = append(all, keywordsAZ...)
	all = append(all, keywordsRU...)
	all = append(all, keywordsEN...)
	return all
}

// Monitor watches public Telegram channels for outage-related mentions.
type Monitor struct {
	botToken      string
	channelIDs    []int64
	windowMinutes int
	baselineDays  int
	botFilterRate int
	db            *storage.DB
	logger        *zap.Logger
	httpClient    *http.Client

	mu           sync.RWMutex
	lastSignal   *models.SocialSignal
	lastUpdateID int64
}

// New creates a new Telegram social signal monitor.
func New(botToken string, channelIDs []int64, windowMinutes, baselineDays, botFilterRate int, db *storage.DB, logger *zap.Logger) *Monitor {
	return &Monitor{
		botToken:      botToken,
		channelIDs:    channelIDs,
		windowMinutes: windowMinutes,
		baselineDays:  baselineDays,
		botFilterRate: botFilterRate,
		db:            db,
		logger:        logger,
		httpClient:    &http.Client{Timeout: 10 * time.Second},
	}
}

// GetLatestSignal returns the most recent social signal (in-memory).
func (m *Monitor) GetLatestSignal() *models.SocialSignal {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastSignal
}

// Run starts the social monitoring loop.
func (m *Monitor) Run(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(m.windowMinutes) * time.Minute)
	defer ticker.Stop()

	// Run immediately on start
	m.collectAndAnalyze(ctx)

	for {
		select {
		case <-ctx.Done():
			m.logger.Info("social monitor shutting down")
			return
		case <-ticker.C:
			m.collectAndAnalyze(ctx)
		}
	}
}

// collectAndAnalyze performs one cycle of collecting and analyzing social signals.
func (m *Monitor) collectAndAnalyze(ctx context.Context) {
	totalMentions := 0
	matchedKeywords := make(map[string]bool)

	for _, channelID := range m.channelIDs {
		mentions, keywords, err := m.scanChannel(ctx, channelID)
		if err != nil {
			m.logger.Error("failed to scan channel",
				zap.Int64("channel_id", channelID),
				zap.Error(err))
			continue
		}
		totalMentions += mentions
		for _, kw := range keywords {
			matchedKeywords[kw] = true
		}
	}

	// Get baseline from DB
	baseline, err := m.db.GetSocialBaseline(ctx, m.baselineDays)
	if err != nil {
		m.logger.Error("failed to get social baseline", zap.Error(err))
		baseline = 0
	}

	var ratio float64
	if baseline > 0 {
		ratio = float64(totalMentions) / baseline
	}

	sampleKW := make([]string, 0, len(matchedKeywords))
	for kw := range matchedKeywords {
		sampleKW = append(sampleKW, kw)
	}

	signal := models.SocialSignal{
		Time:           time.Now().UTC(),
		WindowMinutes:  m.windowMinutes,
		MentionCount:   totalMentions,
		BaselineRate:   baseline,
		Ratio:          ratio,
		SampleKeywords: sampleKW,
	}

	if err := m.db.InsertSocialSignal(ctx, signal); err != nil {
		m.logger.Error("failed to store social signal", zap.Error(err))
	}

	m.mu.Lock()
	m.lastSignal = &signal
	m.mu.Unlock()

	m.logger.Info("social signal collected",
		zap.Int("mentions", totalMentions),
		zap.Float64("baseline", baseline),
		zap.Float64("ratio", ratio))
}

// FIX A-7: telegramUpdate reads BOTH channel_post and message fields.
type telegramUpdate struct {
	UpdateID    int64        `json:"update_id"`
	ChannelPost *telegramMsg `json:"channel_post"` // For channel posts
	Message     *telegramMsg `json:"message"`      // For group/private messages
}

type telegramMsg struct {
	MessageID int64  `json:"message_id"`
	Date      int64  `json:"date"`
	Text      string `json:"text"`
	From      *struct {
		ID    int64 `json:"id"`
		IsBot bool  `json:"is_bot"`
	} `json:"from"`
	Chat *struct {
		ID int64 `json:"id"`
	} `json:"chat"`
}

type telegramResponse struct {
	OK     bool             `json:"ok"`
	Result []telegramUpdate `json:"result"`
}

// scanChannel scans a Telegram channel for keyword mentions.
func (m *Monitor) scanChannel(ctx context.Context, channelID int64) (int, []string, error) {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/getUpdates?chat_id=%d&offset=%d&limit=100",
		m.botToken, channelID, m.lastUpdateID+1)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return 0, nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return 0, nil, fmt.Errorf("telegram api: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024))
	if err != nil {
		return 0, nil, fmt.Errorf("read response: %w", err)
	}

	var tgResp telegramResponse
	if err := json.Unmarshal(body, &tgResp); err != nil {
		return 0, nil, fmt.Errorf("unmarshal response: %w", err)
	}

	if !tgResp.OK {
		return 0, nil, fmt.Errorf("telegram api returned not ok")
	}

	windowStart := time.Now().Add(-time.Duration(m.windowMinutes) * time.Minute)
	mentions := 0
	var matchedKeywords []string

	for _, u := range tgResp.Result {
		if u.UpdateID > m.lastUpdateID {
			m.lastUpdateID = u.UpdateID
		}

		// FIX A-7: Check both channel_post and message fields
		msg := u.ChannelPost
		if msg == nil {
			msg = u.Message
		}
		if msg == nil || msg.Text == "" {
			continue
		}

		// Filter bot accounts
		if msg.From != nil && msg.From.IsBot {
			continue
		}

		msgTime := time.Unix(msg.Date, 0)
		if msgTime.Before(windowStart) {
			continue
		}

		text := strings.ToLower(msg.Text)
		for _, kw := range AllKeywords() {
			if strings.Contains(text, strings.ToLower(kw)) {
				// Check if this is an English keyword
				isEN := false
				for _, enKw := range keywordsEN {
					if strings.ToLower(kw) == strings.ToLower(enKw) {
						isEN = true
						break
					}
				}

				// If it's an English keyword, require an AZ location hint
				if isEN {
					hasContext := false
					for _, hint := range locationHints {
						if strings.Contains(text, strings.ToLower(hint)) {
							hasContext = true
							break
						}
					}
					if !hasContext {
						continue // Skip if no AZ context found
					}
				}

				mentions++
				matchedKeywords = append(matchedKeywords, kw)
				break // Count each message only once
			}
		}
	}

	return mentions, matchedKeywords, nil
}
