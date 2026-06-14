package knowledge

import "time"

// ════════════════════════════════════════════════════════════════════════
// MarketRegime — kondisi pasar yang terdeteksi oleh RegimeDetectionAgent
// ════════════════════════════════════════════════════════════════════════

type MarketRegime string

const (
	RegimeTrending       MarketRegime = "trending"
	RegimeRanging        MarketRegime = "ranging"
	RegimeBreakout       MarketRegime = "breakout"
	RegimeHighVolatility MarketRegime = "high_vol"
	RegimeLowVolatility  MarketRegime = "low_vol"
	RegimeUnknown        MarketRegime = "unknown"
)

// ════════════════════════════════════════════════════════════════════════
// AgentMetrics — snapshot performa satu agen dalam window tertentu
// ════════════════════════════════════════════════════════════════════════

type AgentMetrics struct {
	AgentName    string       `json:"agent_name"`
	WinCount     int          `json:"win_count"`
	LossCount    int          `json:"loss_count"`
	LossStreak   int          `json:"loss_streak"`
	Accuracy     float64      `json:"accuracy"`
	AccuracyPrev float64      `json:"accuracy_prev"`
	ActiveRegime MarketRegime `json:"active_regime"`
	UpdatedAt    time.Time    `json:"updated_at"`
}

// AccuracyDelta menghitung penurunan akurasi.
func (m *AgentMetrics) AccuracyDelta() float64 {
	return m.Accuracy - m.AccuracyPrev
}

// ════════════════════════════════════════════════════════════════════════
// ExperienceReport — output MetaObserverAgent ketika mendeteksi kegagalan
// ════════════════════════════════════════════════════════════════════════

type ExperienceReport struct {
	AgentName      string       `json:"agent"`
	AccuracyBefore float64      `json:"accuracy_before"`
	AccuracyNow    float64      `json:"accuracy_now"`
	AccuracyDelta  float64      `json:"accuracy_delta"`
	LossStreak     int          `json:"loss_streak"`
	ActiveRegime   MarketRegime `json:"regime"`
	Pair           string       `json:"pair"`
	Cause          string       `json:"cause"`
	Reasoning      string       `json:"reasoning"`
	Timestamp      time.Time    `json:"timestamp"`
}

// ════════════════════════════════════════════════════════════════════════
// KnowledgeRule — output KnowledgeTransferAgent, disebar ke agen lain
// ════════════════════════════════════════════════════════════════════════

type RuleCondition struct {
	Regime   MarketRegime `json:"regime"`
	ADXBelow *float64     `json:"adx_below,omitempty"`
	ADXAbove *float64     `json:"adx_above,omitempty"`
	VolBelow *float64     `json:"vol_below,omitempty"`
	VolAbove *float64     `json:"vol_above,omitempty"`
}

type RuleAction struct {
	TargetAgent string  `json:"agent"`
	WeightDelta float64 `json:"weight_delta"`
	MinWeight   float64 `json:"min_weight"`
}

type KnowledgeRule struct {
	ID          string        `json:"id"`
	Condition   RuleCondition `json:"condition"`
	Action      RuleAction    `json:"action"`
	SourceAgent string        `json:"source_agent"`
	Confidence  float64       `json:"confidence"`
	Reasoning   string        `json:"reasoning"`
	CreatedAt   time.Time     `json:"created_at"`
	ExpiresAt   time.Time     `json:"expires_at"`
	ApplyCount  int           `json:"apply_count"`
}

// ════════════════════════════════════════════════════════════════════════
// RegimeContext — output RegimeDetectionAgent, dikirim ke seluruh pipeline
// ════════════════════════════════════════════════════════════════════════

type RegimeContext struct {
	Pair          string       `json:"pair"`
	Regime        MarketRegime `json:"regime"`
	ADX           float64      `json:"adx"`
	ATR           float64      `json:"atr"`
	Volatility    float64      `json:"volatility"`
	TrendStrength float64      `json:"trend_strength"`
	DetectedAt    time.Time    `json:"detected_at"`
}
