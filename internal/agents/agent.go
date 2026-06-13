package agents

import (
	"context"
	"time"
)

// ════════════════════════════════════════════════════════════════════════
// Agent Interface — semua agent wajib implement ini
// ════════════════════════════════════════════════════════════════════════

// Agent adalah kontrak yang wajib diimplementasi oleh setiap agent dalam sistem.
// Setiap agent bersifat otonom: punya input, output, dan fallback sendiri.
type Agent interface {
	Name() string
	Run(ctx context.Context, input AgentInput) AgentOutput
}

// ════════════════════════════════════════════════════════════════════════
// Candle — representasi satu OHLCV candle
// ════════════════════════════════════════════════════════════════════════

// Candle merepresentasikan satu candle OHLCV dari data feed.
type Candle struct {
	Pair      string    `json:"pair"`
	Open      float64   `json:"open"`
	High      float64   `json:"high"`
	Low       float64   `json:"low"`
	Close     float64   `json:"close"`
	Volume    float64   `json:"volume"`
	Spread    float64   `json:"spread"`
	Timeframe string    `json:"timeframe"` // "5m" | "15m" | "1h" | "4h"
	Timestamp time.Time `json:"timestamp"`
}

// ════════════════════════════════════════════════════════════════════════
// AgentInput — container generik untuk input semua agent
// ════════════════════════════════════════════════════════════════════════

// AgentInput adalah container data yang mengalir antar agent.
// Setiap agent hanya membaca field yang relevan untuknya.
type AgentInput struct {
	Pair    string   // Currency pair, e.g. "EUR_USD"
	Candles []Candle // Rolling buffer candle dari MarketDataAgent

	// Output dari agent sebelumnya (diisi bertahap seiring pipeline berjalan)
	Technical   *TechnicalOutput   // dari TechnicalAgent (Agent 2)
	Fundamental *FundamentalOutput // dari FundamentalAgent (Agent 3)
	Risk        *RiskOutput        // dari RiskAgent (Agent 4)
	Decision    *DecisionOutput    // dari DecisionAgent (Agent 5)

	// Account parameters (untuk RiskAgent)
	AccountBalance float64
	RiskPercent    float64
}

// ════════════════════════════════════════════════════════════════════════
// AgentOutput — container generik untuk output semua agent
// ════════════════════════════════════════════════════════════════════════

// AgentOutput adalah hasil dari pemanggilan Agent.Run().
// Hanya satu field output yang terisi, sesuai agent yang menghasilkan.
type AgentOutput struct {
	AgentName string    `json:"agent_name"`
	Success   bool      `json:"success"`
	Error     error     `json:"error,omitempty"`
	Timestamp time.Time `json:"timestamp"`

	// Diisi oleh agent masing-masing (hanya satu yang non-nil per output)
	Technical   *TechnicalOutput   `json:"technical,omitempty"`
	Fundamental *FundamentalOutput `json:"fundamental,omitempty"`
	Risk        *RiskOutput        `json:"risk,omitempty"`
	Decision    *DecisionOutput    `json:"decision,omitempty"`
}

// ════════════════════════════════════════════════════════════════════════
// Output Structs per Agent
// ════════════════════════════════════════════════════════════════════════

// TechnicalOutput — hasil dari TechnicalAgent (Agent 2)
type TechnicalOutput struct {
	Signal     string  `json:"signal"`     // "BUY" | "SELL" | "HOLD"
	Confidence float64 `json:"confidence"` // 0.0–1.0

	RSI        float64 `json:"rsi"`
	MACDHist   float64 `json:"macd_hist"`
	EMA50      float64 `json:"ema50"`
	EMA200     float64 `json:"ema200"`
	BBPosition float64 `json:"bb_position"` // 0.0 = lower band, 1.0 = upper band

	TechScore float64 `json:"tech_score"` // weighted score final teknikal
	Reason    string  `json:"reason"`     // e.g. "RSI oversold + MACD bullish cross"
}

// FundamentalOutput — hasil dari FundamentalAgent (Agent 3)
type FundamentalOutput struct {
	Sentiment  string  `json:"sentiment"`  // "bullish" | "bearish" | "neutral"
	Confidence float64 `json:"confidence"` // 0.0–1.0
	Score      float64 `json:"score"`      // dinormalisasi: bullish>0.5, bearish<0.5
	Reason     string  `json:"reason"`     // max 15 kata
	FromCache  bool    `json:"from_cache"` // true jika dari Redis cache
}

// RiskOutput — hasil dari RiskAgent (Agent 4)
type RiskOutput struct {
	LotSize    float64 `json:"lot_size"`    // ukuran lot
	StopLoss   float64 `json:"stop_loss"`   // harga SL
	TakeProfit float64 `json:"take_profit"` // harga TP
	SLPips     float64 `json:"sl_pips"`     // SL dalam pip
	TPPips     float64 `json:"tp_pips"`     // TP dalam pip
	RiskAmount float64 `json:"risk_amount"` // nominal risk dalam USD
}

// DecisionOutput — hasil dari DecisionAgent (Agent 5) — sinyal final
type DecisionOutput struct {
	Signal     string  `json:"signal"`     // "BUY" | "SELL" | "HOLD"
	Confidence float64 `json:"confidence"` // 0.0–1.0
	ConfPct    int     `json:"conf_pct"`   // dalam persen (0–100)

	Entry      float64 `json:"entry"`
	StopLoss   float64 `json:"stop_loss"`
	TakeProfit float64 `json:"take_profit"`
	LotSize    float64 `json:"lot_size"`
	RiskPct    float64 `json:"risk_pct"`

	TechSignal    string  `json:"tech_signal"`
	TechConf      float64 `json:"tech_conf"`
	TechReason    string  `json:"tech_reason"`
	FundSentiment string  `json:"fund_sentiment"`
	FundConf      float64 `json:"fund_conf"`
	FundReason    string  `json:"fund_reason"`
	MLScore       float64 `json:"ml_score"` // opsional

	RiskLevel string    `json:"risk_level"` // "LOW" | "MEDIUM" | "HIGH"
	Pair      string    `json:"pair"`
	Timestamp time.Time `json:"timestamp"`
}
