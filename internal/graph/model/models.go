package model

// ════════════════════════════════════════════════════════════════════════
// GraphQL Models — hand-written untuk full control
// ════════════════════════════════════════════════════════════════════════

type Candle struct {
	Pair      string  `json:"pair"`
	Open      float64 `json:"open"`
	High      float64 `json:"high"`
	Low       float64 `json:"low"`
	Close     float64 `json:"close"`
	Volume    float64 `json:"volume"`
	Spread    float64 `json:"spread"`
	Timeframe string  `json:"timeframe"`
	Timestamp string  `json:"timestamp"`
}

type SignalEntry struct {
	ID            int        `json:"id"`
	Timestamp     string     `json:"timestamp"`
	Pair          string     `json:"pair"`
	Signal        string     `json:"signal"`
	Confidence    float64    `json:"confidence"`
	Regime        string     `json:"regime"`
	Entry         float64    `json:"entry"`
	StopLoss      float64    `json:"stopLoss"`
	TakeProfit    float64    `json:"takeProfit"`
	LotSize       float64    `json:"lotSize"`
	TechSignal    string     `json:"techSignal"`
	TechConf      float64    `json:"techConf"`
	TechReason    string     `json:"techReason"`
	FundSentiment string     `json:"fundSentiment"`
	FundConf      float64    `json:"fundConf"`
	FundReason    string     `json:"fundReason"`
	EvalStatus    *string    `json:"evalStatus"`
	EvalPrice     *float64   `json:"evalPrice"`
	PipsMove      *float64   `json:"pipsMove"`
	EvalTime      *string    `json:"evalTime"`
}

type AgentDebateEntry struct {
	ID         string        `json:"id"`
	Timestamp  string        `json:"timestamp"`
	Pair       string        `json:"pair"`
	Agent      string        `json:"agent"`
	Signal     string        `json:"signal"`
	Confidence float64       `json:"confidence"`
	Reasoning  string        `json:"reasoning"`
	Details    *AgentDetails `json:"details"`
}

type AgentDetails struct {
	RSI        *float64 `json:"rsi"`
	MACDHist   *float64 `json:"macdHist"`
	BBPosition *float64 `json:"bbPosition"`
	EMA50      *float64 `json:"ema50"`
	EMA200     *float64 `json:"ema200"`
	Sentiment  *string  `json:"sentiment"`
	Score      *float64 `json:"score"`
	Regime     *string  `json:"regime"`
	TechWeight *float64 `json:"techWeight"`
	FundWeight *float64 `json:"fundWeight"`
}

type RegimeContext struct {
	Pair          string  `json:"pair"`
	Regime        string  `json:"regime"`
	ADX           float64 `json:"adx"`
	ATR           float64 `json:"atr"`
	Volatility    float64 `json:"volatility"`
	TrendStrength float64 `json:"trendStrength"`
	DetectedAt    string  `json:"detectedAt"`
}

type RegimeChange struct {
	Pair       string  `json:"pair"`
	FromRegime string  `json:"fromRegime"`
	ToRegime   string  `json:"toRegime"`
	ADX        float64 `json:"adx"`
	Volatility float64 `json:"volatility"`
	ChangedAt  string  `json:"changedAt"`
}

type KnowledgeRule struct {
	ID          string  `json:"id"`
	SourceAgent string  `json:"sourceAgent"`
	TargetAgent string  `json:"targetAgent"`
	Regime      string  `json:"regime"`
	WeightDelta float64 `json:"weightDelta"`
	MinWeight   float64 `json:"minWeight"`
	Confidence  float64 `json:"confidence"`
	Reasoning   string  `json:"reasoning"`
	ApplyCount  int     `json:"applyCount"`
	CreatedAt   string  `json:"createdAt"`
	ExpiresAt   string  `json:"expiresAt"`
	Status      string  `json:"status"`
}

type AgentSummary struct {
	AgentName     string  `json:"agentName"`
	Accuracy      float64 `json:"accuracy"`
	AccuracyPrev  float64 `json:"accuracyPrev"`
	WinCount      int     `json:"winCount"`
	LossCount     int     `json:"lossCount"`
	LossStreak    int     `json:"lossStreak"`
	DominantRegime string `json:"dominantRegime"`
	History       []bool  `json:"history"`
}

type PerformanceLog struct {
	AgentName  string  `json:"agentName"`
	Pair       string  `json:"pair"`
	Regime     string  `json:"regime"`
	Signal     string  `json:"signal"`
	EntryPrice float64 `json:"entryPrice"`
	EvalPrice  float64 `json:"evalPrice"`
	Correct    bool    `json:"correct"`
	PipsMove   float64 `json:"pipsMove"`
	SignalTime string  `json:"signalTime"`
	EvalTime   string  `json:"evalTime"`
}

type SystemLog struct {
	Timestamp string  `json:"timestamp"`
	Level     string  `json:"level"`
	Message   string  `json:"message"`
	Agent     *string `json:"agent"`
	Pair      *string `json:"pair"`
}

type PipelineEvent struct {
	Type       string `json:"type"`
	Pair       string `json:"pair"`
	Timestamp  string `json:"timestamp"`
	DurationMs *int   `json:"durationMs"`
}

type AdaptiveWeights struct {
	TechWeight   float64 `json:"techWeight"`
	FundWeight   float64 `json:"fundWeight"`
	RulesApplied int     `json:"rulesApplied"`
	Regime       string  `json:"regime"`
}
