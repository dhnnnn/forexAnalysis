# Technical Implementation Guide
## Self-Adaptive Multi-Agent Forex — MetaObserver + Knowledge Transfer Extension

> Dokumen ini adalah panduan teknis untuk menambahkan tiga agen baru ke project `forex-agent` yang sudah ada.
> Tidak ada kode existing yang dihapus. Semua penambahan bersifat additive.

---

## Daftar Isi

1. [Gambaran Perubahan](#1-gambaran-perubahan)
2. [Struktur File Baru](#2-struktur-file-baru)
3. [Knowledge Types & Structs](#3-knowledge-types--structs)
4. [RegimeDetectionAgent](#4-regimedetectionagent)
5. [MetaObserverAgent](#5-metaobserveragent)
6. [KnowledgeTransferAgent](#6-knowledgetransferagent)
7. [Modifikasi Agent Existing](#7-modifikasi-agent-existing)
8. [Modifikasi DecisionAgent](#8-modifikasi-decisionagent)
9. [Modifikasi Pipeline di main.go](#9-modifikasi-pipeline-di-maingo)
10. [Skema Database Tambahan](#10-skema-database-tambahan)
11. [Konfigurasi Baru di config.yaml](#11-konfigurasi-baru-di-configyaml)
12. [Checklist Implementasi](#12-checklist-implementasi)

---

## 1. Gambaran Perubahan

### Yang DITAMBAH (tidak ada yang dihapus)

```
Agen baru:
  ✦ RegimeDetectionAgent   → deteksi kondisi pasar (Trending/Ranging/Breakout/dll)
  ✦ MetaObserverAgent      → pantau performa setiap agen, buat ExperienceReport
  ✦ KnowledgeTransferAgent → ubah ExperienceReport jadi KnowledgeRule via LLM

File baru:
  internal/knowledge/types.go
  internal/knowledge/store.go
  internal/knowledge/broadcaster.go
  internal/agents/regime_detection_agent.go
  internal/agents/meta_observer_agent.go
  internal/agents/knowledge_transfer_agent.go
  migrations/002_knowledge.sql

Yang dimodifikasi (minor):
  internal/agents/agent.go          → tambah interface baru
  internal/agents/technical_agent.go → terima KnowledgeRule
  internal/agents/decision_agent.go  → adaptive weight dari KB
  cmd/main.go                        → orkestrasi pipeline baru
  config/config.yaml                 → section baru
```

### Alur pipeline baru

```
MarketData → RegimeDetection → [TechnicalAgent + FundamentalAgent + TrendAgent + RangeAgent]
                                              ↓
                                    MetaObserverAgent
                                              ↓
                                  KnowledgeTransferAgent ──→ Redis KB
                                              ↓        ↖ (feedback loop)
                                  AdaptiveDecisionAgent
                                              ↓
                                         RiskAgent
                                              ↓
                                       WhatsAppAgent
```

---

## 2. Struktur File Baru

Buat direktori dan file kosong dulu:

```bash
mkdir -p internal/knowledge

touch internal/knowledge/types.go
touch internal/knowledge/store.go
touch internal/knowledge/broadcaster.go

touch internal/agents/regime_detection_agent.go
touch internal/agents/meta_observer_agent.go
touch internal/agents/knowledge_transfer_agent.go
```

---

## 3. Knowledge Types & Structs

**File: `internal/knowledge/types.go`**

```go
package knowledge

import "time"

// MarketRegime mendefinisikan kondisi pasar yang terdeteksi
type MarketRegime string

const (
	RegimeTrending       MarketRegime = "trending"
	RegimeRanging        MarketRegime = "ranging"
	RegimeBreakout       MarketRegime = "breakout"
	RegimeHighVolatility MarketRegime = "high_vol"
	RegimeLowVolatility  MarketRegime = "low_vol"
	RegimeUnknown        MarketRegime = "unknown"
)

// AgentMetrics adalah snapshot performa satu agen dalam window tertentu
type AgentMetrics struct {
	AgentName    string       `json:"agent_name"`
	WinCount     int          `json:"win_count"`
	LossCount    int          `json:"loss_count"`
	LossStreak   int          `json:"loss_streak"`
	Accuracy     float64      `json:"accuracy"`      // rolling 20 sinyal terakhir
	AccuracyPrev float64      `json:"accuracy_prev"` // accuracy 20 sinyal sebelumnya
	ActiveRegime MarketRegime `json:"active_regime"`
	UpdatedAt    time.Time    `json:"updated_at"`
}

// AccuracyDelta menghitung penurunan akurasi
func (m *AgentMetrics) AccuracyDelta() float64 {
	return m.Accuracy - m.AccuracyPrev
}

// ExperienceReport adalah output MetaObserverAgent ketika mendeteksi kegagalan
type ExperienceReport struct {
	AgentName      string       `json:"agent"`
	AccuracyBefore float64      `json:"accuracy_before"`
	AccuracyNow    float64      `json:"accuracy_now"`
	AccuracyDelta  float64      `json:"accuracy_delta"`
	LossStreak     int          `json:"loss_streak"`
	ActiveRegime   MarketRegime `json:"regime"`
	Pair           string       `json:"pair"`
	Cause          string       `json:"cause"`      // diisi LLM
	Reasoning      string       `json:"reasoning"`  // penjelasan LLM
	Timestamp      time.Time    `json:"timestamp"`
}

// RuleCondition mendefinisikan kapan sebuah rule berlaku
type RuleCondition struct {
	Regime    MarketRegime `json:"regime"`
	ADXBelow  *float64     `json:"adx_below,omitempty"`
	ADXAbove  *float64     `json:"adx_above,omitempty"`
	VolBelow  *float64     `json:"vol_below,omitempty"`
	VolAbove  *float64     `json:"vol_above,omitempty"`
}

// RuleAction mendefinisikan apa yang dilakukan ketika kondisi terpenuhi
type RuleAction struct {
	TargetAgent string  `json:"agent"`
	WeightDelta float64 `json:"weight_delta"` // negatif = kurangi bobot
	MinWeight   float64 `json:"min_weight"`   // batas bawah agar tidak nol
}

// KnowledgeRule adalah output KnowledgeTransferAgent — disebar ke agen lain
type KnowledgeRule struct {
	ID          string        `json:"id"`
	Condition   RuleCondition `json:"condition"`
	Action      RuleAction    `json:"action"`
	SourceAgent string        `json:"source_agent"` // agen yang gagal
	Confidence  float64       `json:"confidence"`   // 0.0–1.0
	Reasoning   string        `json:"reasoning"`    // dari LLM
	CreatedAt   time.Time     `json:"created_at"`
	ExpiresAt   time.Time     `json:"expires_at"`   // rule tidak abadi, default 24 jam
	ApplyCount  int           `json:"apply_count"`  // berapa kali sudah diterapkan
}

// RegimeContext adalah output RegimeDetectionAgent, dikirim ke seluruh pipeline
type RegimeContext struct {
	Pair          string       `json:"pair"`
	Regime        MarketRegime `json:"regime"`
	ADX           float64      `json:"adx"`
	ATR           float64      `json:"atr"`
	Volatility    float64      `json:"volatility"`
	TrendStrength float64      `json:"trend_strength"` // 0–1
	DetectedAt    time.Time    `json:"detected_at"`
}
```

---

**File: `internal/knowledge/store.go`**

```go
package knowledge

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	keyPrefixRule    = "kb:rule:"
	keyPrefixMetrics = "kb:metrics:"
	keyRuleIndex     = "kb:rule:index"
	defaultRuleTTL   = 24 * time.Hour
)

type Store struct {
	rdb *redis.Client
}

func NewStore(rdb *redis.Client) *Store {
	return &Store{rdb: rdb}
}

// SaveRule menyimpan KnowledgeRule ke Redis dengan TTL
func (s *Store) SaveRule(ctx context.Context, rule KnowledgeRule) error {
	data, err := json.Marshal(rule)
	if err != nil {
		return err
	}
	key := keyPrefixRule + rule.ID
	ttl := time.Until(rule.ExpiresAt)
	if ttl <= 0 {
		ttl = defaultRuleTTL
	}
	pipe := s.rdb.Pipeline()
	pipe.Set(ctx, key, data, ttl)
	pipe.SAdd(ctx, keyRuleIndex, rule.ID)
	_, err = pipe.Exec(ctx)
	return err
}

// GetActiveRules mengambil semua rule yang masih berlaku
func (s *Store) GetActiveRules(ctx context.Context) ([]KnowledgeRule, error) {
	ids, err := s.rdb.SMembers(ctx, keyRuleIndex).Result()
	if err != nil {
		return nil, err
	}
	rules := []KnowledgeRule{}
	for _, id := range ids {
		key := keyPrefixRule + id
		data, err := s.rdb.Get(ctx, key).Result()
		if err == redis.Nil {
			// Rule sudah expired, hapus dari index
			s.rdb.SRem(ctx, keyRuleIndex, id)
			continue
		}
		if err != nil {
			continue
		}
		var rule KnowledgeRule
		if err := json.Unmarshal([]byte(data), &rule); err != nil {
			continue
		}
		rules = append(rules, rule)
	}
	return rules, nil
}

// GetRulesForRegime mengambil rules yang berlaku untuk regime tertentu
func (s *Store) GetRulesForRegime(ctx context.Context, regime MarketRegime) ([]KnowledgeRule, error) {
	all, err := s.GetActiveRules(ctx)
	if err != nil {
		return nil, err
	}
	var filtered []KnowledgeRule
	for _, r := range all {
		if r.Condition.Regime == regime {
			filtered = append(filtered, r)
		}
	}
	return filtered, nil
}

// SaveMetrics menyimpan AgentMetrics (untuk keperluan logging dan paper)
func (s *Store) SaveMetrics(ctx context.Context, m AgentMetrics) error {
	data, err := json.Marshal(m)
	if err != nil {
		return err
	}
	key := fmt.Sprintf("%s%s", keyPrefixMetrics, m.AgentName)
	return s.rdb.Set(ctx, key, data, 0).Err()
}

// GetMetrics mengambil metrics sebuah agen
func (s *Store) GetMetrics(ctx context.Context, agentName string) (*AgentMetrics, error) {
	key := keyPrefixMetrics + agentName
	data, err := s.rdb.Get(ctx, key).Result()
	if err != nil {
		return nil, err
	}
	var m AgentMetrics
	if err := json.Unmarshal([]byte(data), &m); err != nil {
		return nil, err
	}
	return &m, nil
}
```

---

## 4. RegimeDetectionAgent

**File: `internal/agents/regime_detection_agent.go`**

```go
package agents

import (
	"context"
	"math"

	"github.com/dhnnnn/forex-agent/internal/knowledge"
)

// RegimeDetectionAgent menganalisis candle buffer dan mengklasifikasikan kondisi pasar.
// Menggunakan ADX (kekuatan trend), ATR (volatilitas), dan Bollinger Band width.
type RegimeDetectionAgent struct {
	adxPeriod    int
	atrPeriod    int
	adxThreshold float64 // di atas ini = trending
	volThreshold float64 // di atas ini = high volatility
}

func NewRegimeDetectionAgent() *RegimeDetectionAgent {
	return &RegimeDetectionAgent{
		adxPeriod:    14,
		atrPeriod:    14,
		adxThreshold: 25.0, // ADX > 25 = trending (standar industri)
		volThreshold: 0.015, // ATR/price > 1.5% = high volatility
	}
}

// Detect mengklasifikasikan regime dari slice candle OHLCV
// Input: candles []Candle dari MarketDataAgent (minimal 30 candle)
// Output: RegimeContext yang dikirim ke seluruh pipeline
func (r *RegimeDetectionAgent) Detect(ctx context.Context, pair string, candles []Candle) knowledge.RegimeContext {
	if len(candles) < 30 {
		return knowledge.RegimeContext{
			Pair:   pair,
			Regime: knowledge.RegimeUnknown,
		}
	}

	adx := r.calculateADX(candles)
	atr := r.calculateATR(candles)
	lastPrice := candles[len(candles)-1].Close
	relVol := atr / lastPrice

	bbWidth := r.calculateBBWidth(candles, 20, 2.0)
	trendStrength := math.Min(adx/50.0, 1.0) // normalisasi 0–1

	regime := r.classify(adx, relVol, bbWidth)

	return knowledge.RegimeContext{
		Pair:          pair,
		Regime:        regime,
		ADX:           adx,
		ATR:           atr,
		Volatility:    relVol,
		TrendStrength: trendStrength,
		DetectedAt:    time.Now(),
	}
}

func (r *RegimeDetectionAgent) classify(adx, relVol, bbWidth float64) knowledge.MarketRegime {
	switch {
	case relVol > r.volThreshold*1.8 && bbWidth > 0.03:
		// Volatilitas sangat tinggi dan band melebar = breakout
		return knowledge.RegimeBreakout
	case adx > r.adxThreshold && relVol > r.volThreshold:
		return knowledge.RegimeTrending
	case adx > r.adxThreshold && relVol <= r.volThreshold:
		// Trending tapi volatilitas rendah = low vol trending
		return knowledge.RegimeLowVolatility
	case adx <= r.adxThreshold && relVol > r.volThreshold*1.5:
		return knowledge.RegimeHighVolatility
	default:
		return knowledge.RegimeRanging
	}
}

// calculateADX menghitung Average Directional Index (14 periode)
// Menggunakan formula Wilder's smoothed DX
func (r *RegimeDetectionAgent) calculateADX(candles []Candle) float64 {
	period := r.adxPeriod
	if len(candles) < period*2 {
		return 0
	}

	dxValues := make([]float64, 0, len(candles)-1)
	var prevATR, prevPlusDI, prevMinusDI float64
	smoothed := false

	for i := 1; i < len(candles); i++ {
		high := candles[i].High
		low := candles[i].Low
		prevHigh := candles[i-1].High
		prevLow := candles[i-1].Low
		prevClose := candles[i-1].Close

		// True Range
		tr := math.Max(high-low, math.Max(
			math.Abs(high-prevClose),
			math.Abs(low-prevClose),
		))

		// Directional Movement
		plusDM := 0.0
		if high-prevHigh > prevLow-low && high-prevHigh > 0 {
			plusDM = high - prevHigh
		}
		minusDM := 0.0
		if prevLow-low > high-prevHigh && prevLow-low > 0 {
			minusDM = prevLow - low
		}

		// Wilder smoothing setelah period pertama
		if !smoothed && i >= period {
			prevATR = tr
			prevPlusDI = plusDM
			prevMinusDI = minusDM
			smoothed = true
			continue
		}
		if !smoothed {
			continue
		}

		prevATR = prevATR - (prevATR / float64(period)) + tr
		prevPlusDI = prevPlusDI - (prevPlusDI / float64(period)) + plusDM
		prevMinusDI = prevMinusDI - (prevMinusDI / float64(period)) + minusDM

		if prevATR == 0 {
			continue
		}
		diPlus := (prevPlusDI / prevATR) * 100
		diMinus := (prevMinusDI / prevATR) * 100
		diSum := diPlus + diMinus
		if diSum == 0 {
			continue
		}
		dx := (math.Abs(diPlus-diMinus) / diSum) * 100
		dxValues = append(dxValues, dx)
	}

	if len(dxValues) == 0 {
		return 0
	}
	// ADX = rata-rata DX
	sum := 0.0
	start := len(dxValues) - period
	if start < 0 {
		start = 0
	}
	for _, v := range dxValues[start:] {
		sum += v
	}
	return sum / float64(len(dxValues[start:]))
}

// calculateATR menghitung Average True Range
func (r *RegimeDetectionAgent) calculateATR(candles []Candle) float64 {
	if len(candles) < 2 {
		return 0
	}
	period := r.atrPeriod
	sum := 0.0
	count := 0
	start := len(candles) - period
	if start < 1 {
		start = 1
	}
	for i := start; i < len(candles); i++ {
		tr := math.Max(candles[i].High-candles[i].Low, math.Max(
			math.Abs(candles[i].High-candles[i-1].Close),
			math.Abs(candles[i].Low-candles[i-1].Close),
		))
		sum += tr
		count++
	}
	if count == 0 {
		return 0
	}
	return sum / float64(count)
}

// calculateBBWidth menghitung lebar Bollinger Band relatif terhadap harga
func (r *RegimeDetectionAgent) calculateBBWidth(candles []Candle, period int, multiplier float64) float64 {
	if len(candles) < period {
		return 0
	}
	recent := candles[len(candles)-period:]
	sum := 0.0
	for _, c := range recent {
		sum += c.Close
	}
	mean := sum / float64(period)

	variance := 0.0
	for _, c := range recent {
		diff := c.Close - mean
		variance += diff * diff
	}
	stddev := math.Sqrt(variance / float64(period))

	upper := mean + multiplier*stddev
	lower := mean - multiplier*stddev

	if mean == 0 {
		return 0
	}
	return (upper - lower) / mean // width relatif
}
```

---

## 5. MetaObserverAgent

**File: `internal/agents/meta_observer_agent.go`**

```go
package agents

import (
	"context"
	"sync"
	"time"

	"github.com/dhnnnn/forex-agent/internal/knowledge"
)

const (
	rollingWindow      = 20   // jumlah sinyal untuk hitung accuracy
	dropThreshold      = 0.20 // penurunan akurasi 20% trigger report
	lossStreakTrigger  = 4    // 4 loss berturut = trigger report
)

// SignalOutcome adalah hasil evaluasi sinyal setelah harga bergerak
type SignalOutcome struct {
	AgentName string
	Pair      string
	Correct   bool
	Regime    knowledge.MarketRegime
	Timestamp time.Time
}

// agentTracker melacak performa satu agen
type agentTracker struct {
	name        string
	outcomes    []SignalOutcome // rolling window
	lossStreak  int
	mu          sync.Mutex
}

func (t *agentTracker) record(outcome SignalOutcome) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.outcomes = append(t.outcomes, outcome)
	if len(t.outcomes) > rollingWindow*2 {
		t.outcomes = t.outcomes[len(t.outcomes)-rollingWindow*2:]
	}
	if !outcome.Correct {
		t.lossStreak++
	} else {
		t.lossStreak = 0
	}
}

func (t *agentTracker) accuracy(last int) float64 {
	t.mu.Lock()
	defer t.mu.Unlock()
	if len(t.outcomes) == 0 {
		return 0.5 // default neutral
	}
	start := len(t.outcomes) - last
	if start < 0 {
		start = 0
	}
	window := t.outcomes[start:]
	wins := 0
	for _, o := range window {
		if o.Correct {
			wins++
		}
	}
	return float64(wins) / float64(len(window))
}

func (t *agentTracker) dominantRegime(last int) knowledge.MarketRegime {
	t.mu.Lock()
	defer t.mu.Unlock()
	counts := map[knowledge.MarketRegime]int{}
	start := len(t.outcomes) - last
	if start < 0 {
		start = 0
	}
	for _, o := range t.outcomes[start:] {
		counts[o.Regime]++
	}
	var best knowledge.MarketRegime
	bestN := 0
	for r, n := range counts {
		if n > bestN {
			best = r
			bestN = n
		}
	}
	return best
}

// MetaObserverAgent memantau semua agen dan menghasilkan ExperienceReport
// ketika mendeteksi penurunan performa signifikan.
type MetaObserverAgent struct {
	trackers map[string]*agentTracker
	store    *knowledge.Store
	mu       sync.RWMutex
}

func NewMetaObserverAgent(store *knowledge.Store) *MetaObserverAgent {
	return &MetaObserverAgent{
		trackers: make(map[string]*agentTracker),
		store:    store,
	}
}

// RegisterAgent mendaftarkan agen ke dalam pengawasan MetaObserver
func (m *MetaObserverAgent) RegisterAgent(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.trackers[name]; !exists {
		m.trackers[name] = &agentTracker{name: name}
	}
}

// RecordOutcome dipanggil setelah setiap sinyal dievaluasi
// Dipanggil dari evaluator loop di main.go setelah harga move 20 pip
func (m *MetaObserverAgent) RecordOutcome(outcome SignalOutcome) {
	m.mu.RLock()
	tracker, exists := m.trackers[outcome.AgentName]
	m.mu.RUnlock()
	if !exists {
		return
	}
	tracker.record(outcome)

	// Simpan metrics ke Redis untuk logging
	ctx := context.Background()
	metrics := knowledge.AgentMetrics{
		AgentName:    outcome.AgentName,
		LossStreak:   tracker.lossStreak,
		Accuracy:     tracker.accuracy(rollingWindow),
		AccuracyPrev: tracker.accuracy(rollingWindow * 2),
		ActiveRegime: outcome.Regime,
		UpdatedAt:    time.Now(),
	}
	_ = m.store.SaveMetrics(ctx, metrics)
}

// Observe memindai semua tracker dan menghasilkan ExperienceReport
// jika ditemukan degradasi performa. Dipanggil setiap siklus pipeline.
func (m *MetaObserverAgent) Observe() []knowledge.ExperienceReport {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var reports []knowledge.ExperienceReport

	for _, tracker := range m.trackers {
		accNow := tracker.accuracy(rollingWindow)
		accPrev := tracker.accuracy(rollingWindow * 2)
		delta := accNow - accPrev
		regime := tracker.dominantRegime(rollingWindow)

		shouldReport := delta < -dropThreshold || tracker.lossStreak >= lossStreakTrigger

		if shouldReport {
			report := knowledge.ExperienceReport{
				AgentName:      tracker.name,
				AccuracyBefore: accPrev,
				AccuracyNow:    accNow,
				AccuracyDelta:  delta,
				LossStreak:     tracker.lossStreak,
				ActiveRegime:   regime,
				Timestamp:      time.Now(),
			}
			reports = append(reports, report)
		}
	}
	return reports
}
```

---

## 6. KnowledgeTransferAgent

**File: `internal/agents/knowledge_transfer_agent.go`**

```go
package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/dhnnnn/forex-agent/internal/knowledge"
	"github.com/google/uuid"
)

// KnowledgeTransferAgent mengubah ExperienceReport menjadi KnowledgeRule
// menggunakan LLM (Gemini primary, Groq fallback) untuk reasoning.
// Ini adalah komponen novelty utama — LLM digunakan bukan untuk prediksi harga,
// melainkan untuk mengekstrak "mengapa sebuah agen gagal" dan
// menyusunnya menjadi aturan yang dapat diterapkan oleh agen lain.
type KnowledgeTransferAgent struct {
	geminiClient interface{ Complete(ctx context.Context, prompt string) (string, error) }
	groqClient   interface{ Complete(ctx context.Context, prompt string) (string, error) }
	store        *knowledge.Store
	ruleTTL      time.Duration
	minConf      float64 // confidence minimum agar rule disimpan
}

func NewKnowledgeTransferAgent(
	gemini interface{ Complete(ctx context.Context, prompt string) (string, error) },
	groq interface{ Complete(ctx context.Context, prompt string) (string, error) },
	store *knowledge.Store,
) *KnowledgeTransferAgent {
	return &KnowledgeTransferAgent{
		geminiClient: gemini,
		groqClient:   groq,
		store:        store,
		ruleTTL:      24 * time.Hour,
		minConf:      0.60,
	}
}

// Process menerima slice ExperienceReport dan menghasilkan KnowledgeRule
// Ini dipanggil setelah MetaObserverAgent.Observe() menghasilkan report
func (k *KnowledgeTransferAgent) Process(ctx context.Context, reports []knowledge.ExperienceReport) []knowledge.KnowledgeRule {
	var rules []knowledge.KnowledgeRule

	for _, report := range reports {
		rule, err := k.processOne(ctx, report)
		if err != nil {
			// Log error, lanjut ke report berikutnya
			fmt.Printf("[KTA] error processing report for %s: %v\n", report.AgentName, err)
			continue
		}
		if rule.Confidence < k.minConf {
			fmt.Printf("[KTA] rule confidence too low (%.2f), skipping\n", rule.Confidence)
			continue
		}
		if err := k.store.SaveRule(ctx, *rule); err != nil {
			fmt.Printf("[KTA] failed to save rule: %v\n", err)
		}
		rules = append(rules, *rule)
	}
	return rules
}

func (k *KnowledgeTransferAgent) processOne(ctx context.Context, report knowledge.ExperienceReport) (*knowledge.KnowledgeRule, error) {
	prompt := k.buildPrompt(report)

	// Coba Gemini dulu, fallback ke Groq
	raw, err := k.geminiClient.Complete(ctx, prompt)
	if err != nil {
		raw, err = k.groqClient.Complete(ctx, prompt)
		if err != nil {
			return nil, fmt.Errorf("both LLM clients failed: %w", err)
		}
	}

	return k.parseResponse(raw, report)
}

// buildPrompt membangun prompt yang dikirim ke LLM.
// Prompt dirancang untuk menghasilkan JSON terstruktur, bukan teks bebas.
func (k *KnowledgeTransferAgent) buildPrompt(r knowledge.ExperienceReport) string {
	return fmt.Sprintf(`You are an expert forex trading system analyst specializing in multi-agent systems.

An agent in our forex trading pipeline has experienced a significant performance drop:

Agent: %s
Accuracy before: %.1f%%
Accuracy now: %.1f%%  
Accuracy drop: %.1f%%
Loss streak: %d consecutive incorrect signals
Market regime during failure: %s

Your task: Analyze WHY this agent is failing and generate exactly ONE knowledge rule that can be broadcast to other agents to prevent cascading failures.

Respond ONLY with valid JSON in this exact format (no markdown, no explanation outside JSON):
{
  "condition": {
    "regime": "%s",
    "adx_below": <number if ADX constraint matters, else null>,
    "adx_above": <number if ADX constraint matters, else null>,
    "vol_below": <number if volatility constraint matters, else null>
  },
  "action": {
    "agent": "%s",
    "weight_delta": <negative float between -0.5 and -0.1>,
    "min_weight": 0.05
  },
  "confidence": <float between 0.0 and 1.0>,
  "reasoning": "<one clear sentence explaining the failure cause>"
}`,
		r.AgentName,
		r.AccuracyBefore*100,
		r.AccuracyNow*100,
		r.AccuracyDelta*100,
		r.LossStreak,
		string(r.ActiveRegime),
		string(r.ActiveRegime),
		r.AgentName,
	)
}

// parseResponse mengubah teks JSON dari LLM menjadi KnowledgeRule struct
func (k *KnowledgeTransferAgent) parseResponse(raw string, report knowledge.ExperienceReport) (*knowledge.KnowledgeRule, error) {
	// Bersihkan response jika ada markdown wrapper
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "```json")
	raw = strings.TrimPrefix(raw, "```")
	raw = strings.TrimSuffix(raw, "```")
	raw = strings.TrimSpace(raw)

	// Parse ke struct sementara
	var parsed struct {
		Condition struct {
			Regime   string   `json:"regime"`
			ADXBelow *float64 `json:"adx_below"`
			ADXAbove *float64 `json:"adx_above"`
			VolBelow *float64 `json:"vol_below"`
		} `json:"condition"`
		Action struct {
			Agent       string  `json:"agent"`
			WeightDelta float64 `json:"weight_delta"`
			MinWeight   float64 `json:"min_weight"`
		} `json:"action"`
		Confidence float64 `json:"confidence"`
		Reasoning  string  `json:"reasoning"`
	}

	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return nil, fmt.Errorf("invalid JSON from LLM: %w\nraw: %s", err, raw)
	}

	rule := &knowledge.KnowledgeRule{
		ID: uuid.New().String(),
		Condition: knowledge.RuleCondition{
			Regime:   knowledge.MarketRegime(parsed.Condition.Regime),
			ADXBelow: parsed.Condition.ADXBelow,
			ADXAbove: parsed.Condition.ADXAbove,
			VolBelow: parsed.Condition.VolBelow,
		},
		Action: knowledge.RuleAction{
			TargetAgent: parsed.Action.Agent,
			WeightDelta: parsed.Action.WeightDelta,
			MinWeight:   parsed.Action.MinWeight,
		},
		SourceAgent: report.AgentName,
		Confidence:  parsed.Confidence,
		Reasoning:   parsed.Reasoning,
		CreatedAt:   time.Now(),
		ExpiresAt:   time.Now().Add(k.ruleTTL),
		ApplyCount:  0,
	}

	// Validasi minimal
	if rule.Action.WeightDelta > 0 {
		// KTA hanya boleh mengurangi bobot agen yang gagal
		rule.Action.WeightDelta = -0.2
	}

	return rule, nil
}
```

---

## 7. Modifikasi Agent Existing

### Tambah interface di `internal/agents/agent.go`

Tambahkan di bawah interface `Agent` yang sudah ada:

```go
// KnowledgeAware adalah interface untuk agen yang bisa menerima KnowledgeRule
type KnowledgeAware interface {
	ApplyKnowledge(rules []knowledge.KnowledgeRule, regime knowledge.RegimeContext)
}
```

### Modifikasi `TechnicalAgent`

Tambahkan field dan method baru ke struct yang sudah ada:

```go
// Tambahkan field ke struct TechnicalAgent yang sudah ada:
type TechnicalAgent struct {
	// ... field existing ...
	
	// TAMBAHAN BARU:
	baseWeight    float64
	currentWeight float64 // dimodifikasi oleh KnowledgeRule
	regimeCtx     knowledge.RegimeContext
}

// ApplyKnowledge mengimplementasikan interface KnowledgeAware
// Dipanggil sebelum setiap siklus analisis
func (t *TechnicalAgent) ApplyKnowledge(rules []knowledge.KnowledgeRule, regime knowledge.RegimeContext) {
	t.regimeCtx = regime
	t.currentWeight = t.baseWeight // reset ke base dulu
	
	for _, rule := range rules {
		if rule.Condition.Regime != regime.Regime {
			continue
		}
		// Cek kondisi ADX jika ada
		if rule.Condition.ADXBelow != nil && regime.ADX > *rule.Condition.ADXBelow {
			continue
		}
		// Terapkan weight delta
		if rule.Action.TargetAgent == "TechnicalAgent" {
			t.currentWeight += rule.Action.WeightDelta
			// Jangan kurang dari minimum
			if t.currentWeight < rule.Action.MinWeight {
				t.currentWeight = rule.Action.MinWeight
			}
		}
	}
}
```

---

## 8. Modifikasi DecisionAgent

Modifikasi `Process()` di `decision_agent.go` untuk membaca bobot dinamis:

```go
// Di dalam DecisionAgent, tambahkan method baru:

// GetAdaptiveWeights mengambil bobot dari KnowledgeBase dan regime context
// Menggantikan bobot static dari config
func (d *DecisionAgent) GetAdaptiveWeights(
	ctx context.Context,
	store *knowledge.Store,
	regime knowledge.RegimeContext,
) (techWeight, fundWeight float64) {
	// Mulai dari base weight di config
	techWeight = d.config.Signal.Weights.Technical  // 0.60
	fundWeight = d.config.Signal.Weights.Fundamental // 0.40

	// Ambil rules aktif untuk regime ini
	rules, err := store.GetRulesForRegime(ctx, regime.Regime)
	if err != nil || len(rules) == 0 {
		return
	}

	// Terapkan setiap rule
	for _, rule := range rules {
		switch rule.Action.TargetAgent {
		case "TechnicalAgent":
			techWeight += rule.Action.WeightDelta
		case "FundamentalAgent":
			fundWeight += rule.Action.WeightDelta
		}
		// Increment apply count
		rule.ApplyCount++
		_ = store.SaveRule(ctx, rule)
	}

	// Normalisasi: pastikan jumlah = 1.0
	total := techWeight + fundWeight
	if total > 0 {
		techWeight /= total
		fundWeight /= total
	}

	// Clamp ke range yang aman
	techWeight = clamp(techWeight, 0.15, 0.85)
	fundWeight = 1.0 - techWeight

	return
}

func clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
```

---

## 9. Modifikasi Pipeline di `main.go`

Tambahkan inisialisasi dan orkestrasi tiga agen baru ke fungsi `main()`:

```go
// Di dalam main() setelah inisialisasi existing:

// === INISIALISASI KOMPONEN BARU ===
kbStore := knowledge.NewStore(redisClient)

regimeAgent := agents.NewRegimeDetectionAgent()
metaObserver := agents.NewMetaObserverAgent(kbStore)
kta := agents.NewKnowledgeTransferAgent(geminiClient, groqClient, kbStore)

// Daftarkan semua agen ke MetaObserver
metaObserver.RegisterAgent("TechnicalAgent")
metaObserver.RegisterAgent("FundamentalAgent")
metaObserver.RegisterAgent("TrendAgent")
metaObserver.RegisterAgent("RangeAgent")

// === PIPELINE LOOP (tiap 5 menit, per pair) ===
for _, pair := range cfg.Pairs {
	go func(pair string) {
		ticker := time.NewTicker(5 * time.Minute)
		for range ticker.C {
			candles := marketDataAgent.GetCandles(pair)
			if len(candles) < 30 {
				continue
			}

			// Step 1: Deteksi regime
			regime := regimeAgent.Detect(ctx, pair, candles)

			// Step 2: Ambil rules aktif dari KB
			rules, _ := kbStore.GetRulesForRegime(ctx, regime.Regime)

			// Step 3: Jalankan agen pool dengan regime context
			techResult  := technicalAgent.ProcessWithRegime(candles, regime, rules)
			fundResult  := fundamentalAgent.Process(ctx, pair)

			// Step 4: MetaObserver catat outcome (setelah evaluasi sinyal sebelumnya)
			// Ini dilakukan oleh goroutine evaluator terpisah — lihat bagian di bawah

			// Step 5: Cek apakah MetaObserver punya report baru
			reports := metaObserver.Observe()
			if len(reports) > 0 {
				// Step 6: KTA proses report dan simpan rules baru ke KB
				newRules := kta.Process(ctx, reports)
				log.Printf("[Pipeline] KTA generated %d new rules\n", len(newRules))
			}

			// Step 7: Decision dengan adaptive weights
			techW, fundW := decisionAgent.GetAdaptiveWeights(ctx, kbStore, regime)
			signal := decisionAgent.Decide(techResult, fundResult, techW, fundW, regime)

			// Step 8: Risk + WhatsApp
			if signal.Confidence >= cfg.Signal.MinConfidence {
				sized := riskAgent.Size(signal, cfg.Account)
				waAgent.Send(ctx, sized, regime)
			}
		}
	}(pair)
}

// === EVALUATOR GOROUTINE ===
// Setelah N menit, evaluasi apakah sinyal sebelumnya benar
// (harga bergerak ke arah yang diprediksi >= threshold pip)
go func() {
	evalTicker := time.NewTicker(30 * time.Minute)
	for range evalTicker.C {
		pendingSignals := signalStore.GetPendingEvaluation()
		for _, sig := range pendingSignals {
			correct := evaluateSignal(sig) // bandingkan dengan harga aktual
			metaObserver.RecordOutcome(agents.SignalOutcome{
				AgentName: sig.AgentName,
				Pair:      sig.Pair,
				Correct:   correct,
				Regime:    sig.Regime,
				Timestamp: time.Now(),
			})
		}
	}
}()
```

---

## 10. Skema Database Tambahan

**File: `migrations/002_knowledge.sql`**

```sql
-- Tabel untuk menyimpan ExperienceReport (untuk paper: data eksperimen)
CREATE TABLE IF NOT EXISTS experience_reports (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_name  VARCHAR(50) NOT NULL,
    pair        VARCHAR(10),
    accuracy_before DECIMAL(5,4),
    accuracy_now    DECIMAL(5,4),
    accuracy_delta  DECIMAL(5,4),
    loss_streak     INTEGER,
    active_regime   VARCHAR(20),
    reasoning       TEXT,
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

-- Tabel untuk menyimpan KnowledgeRule history (untuk analisis paper)
CREATE TABLE IF NOT EXISTS knowledge_rules (
    id           UUID PRIMARY KEY,
    source_agent VARCHAR(50),
    target_agent VARCHAR(50),
    regime       VARCHAR(20),
    condition    JSONB,
    action       JSONB,
    confidence   DECIMAL(4,3),
    reasoning    TEXT,
    apply_count  INTEGER DEFAULT 0,
    created_at   TIMESTAMPTZ DEFAULT NOW(),
    expires_at   TIMESTAMPTZ
);

-- Tabel untuk tracking performa agen per regime (untuk paper: results table)
CREATE TABLE IF NOT EXISTS agent_performance_log (
    id          BIGSERIAL PRIMARY KEY,
    agent_name  VARCHAR(50),
    pair        VARCHAR(10),
    regime      VARCHAR(20),
    correct     BOOLEAN,
    signal_time TIMESTAMPTZ,
    eval_time   TIMESTAMPTZ DEFAULT NOW()
);

-- Hypertable TimescaleDB untuk performance log
SELECT create_hypertable('agent_performance_log', 'eval_time', if_not_exists => TRUE);

-- Index untuk query cepat per regime (untuk eksperimen paper)
CREATE INDEX idx_perf_regime ON agent_performance_log(regime, agent_name);
CREATE INDEX idx_perf_pair   ON agent_performance_log(pair, signal_time);
```

---

## 11. Konfigurasi Baru di `config.yaml`

Tambahkan section baru di bawah section `signal` yang sudah ada:

```yaml
# Knowledge Transfer System
knowledge:
  enabled: true
  
  regime_detection:
    adx_period: 14
    atr_period: 14
    adx_threshold: 25.0      # ADX > 25 = trending
    vol_threshold: 0.015     # ATR/price > 1.5% = high vol

  meta_observer:
    rolling_window: 20       # evaluasi 20 sinyal terakhir
    drop_threshold: 0.20     # penurunan akurasi 20% trigger report
    loss_streak_trigger: 4   # 4 loss berturut = trigger report
    observe_interval: "5m"   # seberapa sering MetaObserver cek

  knowledge_transfer:
    rule_ttl: "24h"          # rule expired setelah 24 jam
    min_confidence: 0.60     # confidence minimum untuk simpan rule
    max_rules_active: 20     # maksimum rule aktif di KB

  evaluation:
    eval_delay: "30m"        # tunggu 30 menit sebelum evaluasi sinyal
    pip_threshold: 15        # sinyal dianggap benar jika harga gerak >= 15 pip
```

---

## 12. Checklist Implementasi

Urutan pengerjaan yang disarankan agar bisa ditest bertahap:

```
Phase 1 — Fondasi (1–2 hari)
  [ ] Buat internal/knowledge/types.go
  [ ] Buat internal/knowledge/store.go
  [ ] Jalankan migration 002_knowledge.sql
  [ ] Test: simpan dan baca KnowledgeRule dari Redis

Phase 2 — RegimeDetectionAgent (1 hari)
  [ ] Implementasi regime_detection_agent.go
  [ ] Unit test: classifikasi dengan data candle dummy
  [ ] Integrasi ke main.go (read-only, belum ubah pipeline)
  [ ] Log regime per pair selama 1 hari, verifikasi hasilnya masuk akal

Phase 3 — MetaObserverAgent (1–2 hari)
  [ ] Implementasi meta_observer_agent.go
  [ ] Buat evaluator goroutine di main.go
  [ ] Test: simulasi 5 loss streak, pastikan ExperienceReport terbuat
  [ ] Log reports ke Postgres untuk keperluan paper

Phase 4 — KnowledgeTransferAgent (2–3 hari)
  [ ] Implementasi knowledge_transfer_agent.go
  [ ] Test prompt dengan Gemini API langsung (gunakan curl/Postman dulu)
  [ ] Test parseResponse dengan berbagai format output LLM
  [ ] Integrasi ke pipeline

Phase 5 — Adaptive Decision (1 hari)
  [ ] Modifikasi DecisionAgent.GetAdaptiveWeights()
  [ ] Test: dengan rule aktif vs tanpa rule, bandingkan bobot yang dihasilkan

Phase 6 — End-to-end test (2–3 hari)
  [ ] Jalankan full pipeline dengan paper trading
  [ ] Verifikasi rules disimpan dan diterapkan dengan benar
  [ ] Mulai logging untuk data eksperimen paper
```

---

*Dokumen ini adalah living document — update setiap kali ada perubahan arsitektur.*