package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config is the root configuration structure matching config/config.yaml.
type Config struct {
	Oanda        OandaConfig        `yaml:"oanda"`
	TwelveData   TwelveDataConfig   `yaml:"twelve_data"`
	AlphaVantage AlphaVantageConfig `yaml:"alpha_vantage"`
	RSSFeeds     RSSFeedsConfig     `yaml:"rss_feeds"`
	Pairs        []string           `yaml:"pairs"`
	Scheduler    SchedulerConfig    `yaml:"scheduler"`
	Account      AccountConfig      `yaml:"account"`
	Gemini       GeminiConfig       `yaml:"gemini"`
	Groq         GroqConfig         `yaml:"groq"`
	MLService    MLServiceConfig    `yaml:"ml_service"`
	Signal       SignalConfig        `yaml:"signal"`
	TimescaleDB  TimescaleDBConfig  `yaml:"timescaledb"`
	Redis        RedisConfig        `yaml:"redis"`
	WhatsApp     WhatsAppConfig     `yaml:"whatsapp"`
}

type OandaConfig struct {
	WebSocketURL string `yaml:"websocket_url"`
	APIKey       string `yaml:"api_key"`
	AccountID    string `yaml:"account_id"`
}

type TwelveDataConfig struct {
	APIKey  string `yaml:"api_key"`
	BaseURL string `yaml:"base_url"`
}

type AlphaVantageConfig struct {
	APIKey  string `yaml:"api_key"`
	BaseURL string `yaml:"base_url"`
}

type RSSFeedsConfig struct {
	URLs []string `yaml:"urls"`
}

type SchedulerConfig struct {
	Timeframes []string `yaml:"timeframes"`
}

type AccountConfig struct {
	Balance       float64 `yaml:"balance"`
	RiskPercent   float64 `yaml:"risk_percent"`
	DefaultSLPips float64 `yaml:"default_sl_pips"`
	DefaultTPPips float64 `yaml:"default_tp_pips"`
}

type GeminiConfig struct {
	APIKey    string `yaml:"api_key"`
	Model     string `yaml:"model"`
	TimeoutMs int    `yaml:"timeout_ms"`
}

type GroqConfig struct {
	APIKey    string `yaml:"api_key"`
	Model     string `yaml:"model"`
	TimeoutMs int    `yaml:"timeout_ms"`
}

type MLServiceConfig struct {
	Enabled     bool   `yaml:"enabled"`
	GRPCAddress string `yaml:"grpc_address"`
	TimeoutMs   int    `yaml:"timeout_ms"`
}

type SignalConfig struct {
	BuyThreshold         float64       `yaml:"buy_threshold"`
	SellThreshold        float64       `yaml:"sell_threshold"`
	MinConfidenceToAlert float64       `yaml:"min_confidence_to_alert"`
	Weights              WeightsConfig `yaml:"weights"`
	MLBoostWeight        float64       `yaml:"ml_boost_weight"`
}

type WeightsConfig struct {
	Technical   float64 `yaml:"technical"`
	Fundamental float64 `yaml:"fundamental"`
}

type TimescaleDBConfig struct {
	DSN string `yaml:"dsn"`
}

type RedisConfig struct {
	Address         string `yaml:"address"`
	Password        string `yaml:"password"`
	SentimentTTLMin int    `yaml:"sentiment_ttl_minutes"`
	PriceTTLSec     int    `yaml:"price_ttl_seconds"`
}

type WhatsAppConfig struct {
	ServiceURL           string  `yaml:"service_url"`
	TargetPhone          string  `yaml:"target_phone"`
	MinConfidenceToAlert float64 `yaml:"min_confidence_to_alert"`
	RateLimitSeconds     int     `yaml:"rate_limit_seconds"`
}

// Load reads the YAML config file at the given path, expands environment
// variables in the content (e.g. ${OANDA_API_KEY}), and returns the parsed Config.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	// Expand environment variables in the YAML content
	expanded := os.ExpandEnv(string(data))

	var cfg Config
	if err := yaml.Unmarshal([]byte(expanded), &cfg); err != nil {
		return nil, fmt.Errorf("parse config yaml: %w", err)
	}

	return &cfg, nil
}
