package agents

import (
	"context"
	"fmt"
	"time"
)

// ════════════════════════════════════════════════════════════════════════
// DecisionAgent (Agent 5) — menghasilkan sinyal trading final
// ════════════════════════════════════════════════════════════════════════

// Compile-time check that DecisionAgent implements Agent.
var _ Agent = (*DecisionAgent)(nil)

// DecisionAgent (Agent 5) mengagregasi output dari TechnicalAgent, FundamentalAgent,
// dan RiskAgent untuk menghasilkan sinyal trading final (BUY/SELL/HOLD) dengan
// confidence scoring dan risk level assessment.
type DecisionAgent struct {
	config   SignalConfig
	mlClient MLPredictor // boleh nil jika ML service tidak tersedia
}

// NewDecisionAgent membuat instance DecisionAgent baru dengan konfigurasi yang telah divalidasi.
// mlClient boleh nil untuk menandakan ML service tidak tersedia.
func NewDecisionAgent(config SignalConfig, mlClient MLPredictor) *DecisionAgent {
	return &DecisionAgent{
		config:   validateConfig(config),
		mlClient: mlClient,
	}
}

// Name mengembalikan identifier agent.
func (a *DecisionAgent) Name() string {
	return "DecisionAgent"
}

// Run mengeksekusi pipeline decision untuk menghasilkan sinyal trading final.
// Steps:
// 1. Cek context cancellation
// 2. Hitung weighted score dari technical + fundamental
// 3. Tentukan sinyal (BUY/SELL/HOLD) berdasarkan threshold
// 4. Hitung base confidence
// 5. Terapkan ML boost (jika mlClient tersedia)
// 6. Tentukan risk level
// 7. Build entry/SL/TP/lot
// 8. Build upstream transparency fields
// 9. Return AgentOutput
func (a *DecisionAgent) Run(ctx context.Context, input AgentInput) AgentOutput {
	// 1. Context cancellation check
	if ctx.Err() != nil {
		return errorOutput(a.Name(), fmt.Errorf("context cancelled: %w", ctx.Err()))
	}

	// 2. Extract upstream outputs (nil-safe)
	tech := input.Technical
	fund := input.Fundamental
	risk := input.Risk

	// 3. Calculate weighted score
	weightedScore := calcWeightedScore(tech, fund, a.config)

	// 4. Determine signal from thresholds
	signal := determineSignal(weightedScore, a.config)

	// 5. Calculate base confidence
	confidence := calcConfidence(tech, fund, a.config)

	// 6. Optional ML boost
	var mlScore float64
	confidence, mlScore = applyMLBoost(ctx, confidence, a.mlClient, tech, input.Candles, a.config)

	// 7. Assess risk level
	riskLevel := assessRiskLevel(confidence)

	// 8. Build entry/SL/TP from risk output
	entry, sl, tp, lot := 0.0, 0.0, 0.0, 0.0
	if signal != "HOLD" {
		if risk != nil {
			sl = risk.StopLoss
			tp = risk.TakeProfit
			lot = risk.LotSize
		}
		if len(input.Candles) > 0 {
			entry = input.Candles[len(input.Candles)-1].Close
		}
	}

	// 9. Build upstream transparency fields (nil-safe defaults)
	techSignal, techConf, techReason := "HOLD", 0.0, ""
	if tech != nil {
		techSignal = tech.Signal
		techConf = tech.Confidence
		techReason = tech.Reason
	}

	fundSent, fundConf, fundReason := "neutral", 0.5, ""
	if fund != nil {
		fundSent = fund.Sentiment
		fundConf = fund.Confidence
		fundReason = fund.Reason
	}

	// 10. Return complete output
	return AgentOutput{
		AgentName: a.Name(),
		Success:   true,
		Timestamp: time.Now(),
		Decision: &DecisionOutput{
			Signal:        signal,
			Confidence:    confidence,
			ConfPct:       int(confidence * 100),
			Entry:         entry,
			StopLoss:      sl,
			TakeProfit:    tp,
			LotSize:       lot,
			RiskPct:       input.RiskPercent,
			TechSignal:    techSignal,
			TechConf:      techConf,
			TechReason:    techReason,
			FundSentiment: fundSent,
			FundConf:      fundConf,
			FundReason:    fundReason,
			MLScore:       mlScore,
			RiskLevel:     riskLevel,
			Pair:          input.Pair,
			Timestamp:     time.Now(),
		},
	}
}

// ════════════════════════════════════════════════════════════════════════
// DecisionAgent — helper functions untuk validasi dan komputasi sinyal
// ════════════════════════════════════════════════════════════════════════

// validateConfig menormalisasi SignalConfig, mengembalikan default untuk nilai yang tidak valid.
// Rules:
//   - TechWeight dan FundWeight harus masing-masing dalam [0.0, 1.0] DAN jumlahnya = 1.0; jika tidak, gunakan default
//   - SellThreshold harus < BuyThreshold; keduanya dalam [0.0, 1.0]; jika tidak, gunakan default
//   - MLBoostWeight harus dalam [0.0, 1.0]; jika tidak, gunakan default 0.20
func validateConfig(cfg SignalConfig) SignalConfig {
	defaults := DefaultSignalConfig()

	// Validasi weights: keduanya harus dalam [0.0, 1.0] dan jumlahnya = 1.0
	if cfg.TechWeight < 0.0 || cfg.TechWeight > 1.0 ||
		cfg.FundWeight < 0.0 || cfg.FundWeight > 1.0 ||
		cfg.TechWeight+cfg.FundWeight != 1.0 {
		cfg.TechWeight = defaults.TechWeight
		cfg.FundWeight = defaults.FundWeight
	}

	// Validasi thresholds: keduanya dalam [0.0, 1.0] dan SellThreshold < BuyThreshold
	if cfg.BuyThreshold < 0.0 || cfg.BuyThreshold > 1.0 ||
		cfg.SellThreshold < 0.0 || cfg.SellThreshold > 1.0 ||
		cfg.SellThreshold >= cfg.BuyThreshold {
		cfg.BuyThreshold = defaults.BuyThreshold
		cfg.SellThreshold = defaults.SellThreshold
	}

	// Validasi MLBoostWeight: harus dalam [0.0, 1.0]
	if cfg.MLBoostWeight < 0.0 || cfg.MLBoostWeight > 1.0 {
		cfg.MLBoostWeight = defaults.MLBoostWeight
	}

	return cfg
}

// ════════════════════════════════════════════════════════════════════════
// clamp — membatasi nilai ke range [0.0, 1.0]
// ════════════════════════════════════════════════════════════════════════

// clamp membatasi nilai v ke dalam rentang [0.0, 1.0].
func clamp(v float64) float64 {
	if v < 0.0 {
		return 0.0
	}
	if v > 1.0 {
		return 1.0
	}
	return v
}

// ════════════════════════════════════════════════════════════════════════
// calcWeightedScore — menghitung weighted score dari technical dan fundamental
// ════════════════════════════════════════════════════════════════════════

// calcWeightedScore menghitung skor tertimbang dari output teknikal dan fundamental.
// Jika tech nil, gunakan default 0.5 untuk TechScore.
// Jika fund nil, gunakan default 0.5 untuk Score.
// Input score di-clamp ke [0.0, 1.0] sebelum perhitungan.
func calcWeightedScore(tech *TechnicalOutput, fund *FundamentalOutput, cfg SignalConfig) float64 {
	techScore := 0.5
	if tech != nil {
		techScore = clamp(tech.TechScore)
	}

	fundScore := 0.5
	if fund != nil {
		fundScore = clamp(fund.Score)
	}

	return (techScore * cfg.TechWeight) + (fundScore * cfg.FundWeight)
}

// ════════════════════════════════════════════════════════════════════════
// determineSignal — menentukan sinyal BUY/SELL/HOLD berdasarkan weighted score
// ════════════════════════════════════════════════════════════════════════

// determineSignal mengembalikan sinyal trading berdasarkan weighted score dan threshold.
// Return "BUY" jika score >= BuyThreshold, "SELL" jika score <= SellThreshold, "HOLD" otherwise.
func determineSignal(weightedScore float64, cfg SignalConfig) string {
	if weightedScore >= cfg.BuyThreshold {
		return "BUY"
	}
	if weightedScore <= cfg.SellThreshold {
		return "SELL"
	}
	return "HOLD"
}

// ════════════════════════════════════════════════════════════════════════
// calcConfidence — menghitung confidence tertimbang dari technical dan fundamental
// ════════════════════════════════════════════════════════════════════════

// calcConfidence menghitung confidence level tertimbang.
// Jika tech nil, gunakan default 0.5 untuk Confidence.
// Jika fund nil, gunakan default 0.5 untuk Confidence.
// Input confidence di-clamp ke [0.0, 1.0] sebelum perhitungan.
func calcConfidence(tech *TechnicalOutput, fund *FundamentalOutput, cfg SignalConfig) float64 {
	techConf := 0.5
	if tech != nil {
		techConf = clamp(tech.Confidence)
	}

	fundConf := 0.5
	if fund != nil {
		fundConf = clamp(fund.Confidence)
	}

	return (techConf * cfg.TechWeight) + (fundConf * cfg.FundWeight)
}

// ════════════════════════════════════════════════════════════════════════
// applyMLBoost — menerapkan ML confidence boost jika tersedia
// ════════════════════════════════════════════════════════════════════════

// applyMLBoost memanggil MLPredictor untuk mendapatkan confidence boost.
// Jika mlClient nil, return baseConfidence dan mlScore=0.0.
// Menggunakan child context dengan timeout 500ms.
// Jika error atau score <= 0, return baseConfidence dan mlScore=0.0.
// Jika berhasil: adjusted = (baseConf × (1 - MLBoostWeight)) + (mlScore × MLBoostWeight).
func applyMLBoost(ctx context.Context, baseConfidence float64, mlClient MLPredictor, tech *TechnicalOutput, candles []Candle, cfg SignalConfig) (float64, float64) {
	if mlClient == nil {
		return baseConfidence, 0.0
	}

	mlCtx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	defer cancel()

	score, err := mlClient.Predict(mlCtx, tech, candles)
	if err != nil || score <= 0 {
		return baseConfidence, 0.0
	}

	adjusted := (baseConfidence * (1.0 - cfg.MLBoostWeight)) + (score * cfg.MLBoostWeight)
	return adjusted, score
}

// ════════════════════════════════════════════════════════════════════════
// assessRiskLevel — menentukan level risiko berdasarkan confidence
// ════════════════════════════════════════════════════════════════════════

// assessRiskLevel mengembalikan level risiko berdasarkan confidence.
// Return "LOW" jika confidence >= 0.75, "MEDIUM" jika >= 0.50, "HIGH" jika < 0.50.
func assessRiskLevel(confidence float64) string {
	if confidence >= 0.75 {
		return "LOW"
	}
	if confidence >= 0.50 {
		return "MEDIUM"
	}
	return "HIGH"
}
