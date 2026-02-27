package config

import (
	"fmt"
	"os"
	"time"

	"github.com/karimbayli/sentinel-v2/internal/models"
	"gopkg.in/yaml.v3"
)

// CentralConfig is the full configuration for the central server.
type CentralConfig struct {
	Server      ServerConfig      `yaml:"server"`
	Database    DatabaseConfig    `yaml:"database"`
	Probe       ProbeConfig       `yaml:"probe"`
	BGP         BGPConfig         `yaml:"bgp"`
	Social      SocialConfig      `yaml:"social"`
	Correlation CorrelationConfig `yaml:"correlation"`
	Alert       AlertConfig       `yaml:"alert"`
	Nodes       []models.Node     `yaml:"nodes"`
	Targets     []models.Target   `yaml:"targets"`
	Calibration bool              `yaml:"calibration"`
	LogLevel    string            `yaml:"log_level"`
}

// ProbeAgentConfig is the configuration for a vantage node probe agent.
type ProbeAgentConfig struct {
	NodeID        string          `yaml:"node_id"`
	Region        string          `yaml:"region"`
	Country       string          `yaml:"country"`
	CentralURL    string          `yaml:"central_url"`
	HMACSecret    string          `yaml:"hmac_secret"`
	ProbeInterval time.Duration   `yaml:"probe_interval"`
	Targets       []models.Target `yaml:"targets"`
	Buffer        BufferConfig    `yaml:"buffer"`
	LogLevel      string          `yaml:"log_level"`
}

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Host       string `yaml:"host"`
	Port       int    `yaml:"port"`
	HMACSecret string `yaml:"hmac_secret"`
}

// DatabaseConfig holds TimescaleDB connection params.
type DatabaseConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	DBName   string `yaml:"dbname"`
	SSLMode  string `yaml:"sslmode"`
}

func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		d.User, d.Password, d.Host, d.Port, d.DBName, d.SSLMode)
}

// ProbeConfig controls probing behavior.
type ProbeConfig struct {
	IntervalSeconds int           `yaml:"interval_seconds"`
	TCPTimeout      time.Duration `yaml:"tcp_timeout"`
	HTTPTimeout     time.Duration `yaml:"http_timeout"`
}

// BGPConfig controls the BGP monitor.
type BGPConfig struct {
	Enabled    bool     `yaml:"enabled"`
	WSURL      string   `yaml:"ws_url"`
	Collectors []string `yaml:"collectors"` // FIX R-2: multiple RIS collectors
	WatchASNs  []int    `yaml:"watch_asns"`
}

// SocialConfig controls the Telegram social monitor.
type SocialConfig struct {
	Enabled       bool    `yaml:"enabled"`
	BotToken      string  `yaml:"bot_token"`
	ChannelIDs    []int64 `yaml:"channel_ids"`
	WindowMinutes int     `yaml:"window_minutes"`
	BaselineDays  int     `yaml:"baseline_days"`
	BotFilterRate int     `yaml:"bot_filter_rate"`
}

// CorrelationConfig controls the correlation engine.
type CorrelationConfig struct {
	IntervalSeconds  int     `yaml:"interval_seconds"`
	WindowMinutes    int     `yaml:"window_minutes"`
	WeightNode       float64 `yaml:"weight_node"`
	WeightBGP        float64 `yaml:"weight_bgp"`
	WeightSocial     float64 `yaml:"weight_social"`
	NodeFailRatio    float64 `yaml:"node_fail_ratio"`
	SocialSpikeX     float64 `yaml:"social_spike_x"`
	MinReliableNodes int     `yaml:"min_reliable_nodes"` // FIX R-21: min nodes for signal
}

// AlertConfig controls Telegram alerting.
type AlertConfig struct {
	Enabled      bool   `yaml:"enabled"`
	BotToken     string `yaml:"bot_token"`
	ChatID       int64  `yaml:"chat_id"`
	DedupMinutes int    `yaml:"dedup_minutes"`
}

// BufferConfig controls the local disk-backed buffer.
type BufferConfig struct {
	DBPath  string `yaml:"db_path"`
	MaxSize int    `yaml:"max_size"`
}

// LoadCentralConfig reads and parses a central YAML config file.
func LoadCentralConfig(path string) (*CentralConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}

	// Expand environment variables
	expanded := os.ExpandEnv(string(data))

	cfg := &CentralConfig{
		Server: ServerConfig{
			Host: "0.0.0.0",
			Port: 8080,
		},
		Database: DatabaseConfig{
			Host:    "localhost",
			Port:    5432,
			User:    "sentinel",
			DBName:  "sentinel",
			SSLMode: "disable",
		},
		Probe: ProbeConfig{
			IntervalSeconds: 60,
			TCPTimeout:      5 * time.Second,
			HTTPTimeout:     10 * time.Second,
		},
		BGP: BGPConfig{
			WSURL:      "wss://ris-live.ripe.net/v1/ws/",
			Collectors: []string{"rrc00", "rrc03", "rrc13", "rrc20", "rrc21"},
			WatchASNs:  []int{29049, 31721, 39232, 34377, 57021},
		},
		Social: SocialConfig{
			WindowMinutes: 15,
			BaselineDays:  30,
			BotFilterRate: 50,
		},
		Correlation: CorrelationConfig{
			IntervalSeconds: 30,
			WindowMinutes:   5,
			WeightNode:      0.5,
			WeightBGP:       0.3,
			WeightSocial:    0.2,
			NodeFailRatio:   0.8,
			SocialSpikeX:    3.0,
		},
		Alert: AlertConfig{
			DedupMinutes: 5,
		},
		LogLevel: "info",
	}

	if err := yaml.Unmarshal([]byte(expanded), cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	return cfg, nil
}

// LoadProbeAgentConfig reads and parses a probe agent YAML config file.
func LoadProbeAgentConfig(path string) (*ProbeAgentConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}

	expanded := os.ExpandEnv(string(data))

	cfg := &ProbeAgentConfig{
		ProbeInterval: 60 * time.Second,
		Buffer: BufferConfig{
			DBPath:  "/var/lib/sentinel/buffer.db",
			MaxSize: 10000,
		},
		LogLevel: "info",
	}

	if err := yaml.Unmarshal([]byte(expanded), cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	return cfg, nil
}
