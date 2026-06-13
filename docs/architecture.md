# 🤖 Forex Multi-Agent Analysis Bot — Architecture & Implementation Guide

> **Core Engine:** Go (Golang)
> **WhatsApp:** Node.js + Baileys (service terpisah, dihubungkan via HTTP POST)
> **AI/NLP:** Gemini API (FundamentalAgent) + Python gRPC ML Microservice (opsional, Phase 2)
> **Purpose:** Multi-agent real-time forex signal generation → alert ke WhatsApp
> **Paradigma:** Multi-Agent System — setiap komponen adalah **agent otonom** dengan kontrak input/output sendiri
> **Audience:** AI Agent / Developer untuk implementasi kode

---

## 📋 Daftar Isi

1. [Gambaran Multi-Agent Architecture](#1-gambaran-multi-agent-architecture)
2. [Agent Interface Contract](#2-agent-interface-contract)
3. [Struktur Direktori Project](#3-struktur-direktori-project)
4. [Agent 1 — MarketData Agent](#4-agent-1--marketdata-agent)
5. [Agent 2 — Technical Agent](#5-agent-2--technical-agent)
6. [Agent 3 — Fundamental Agent](#6-agent-3--fundamental-agent)
7. [Agent 4 — Risk Agent](#7-agent-4--risk-agent)
8. [Agent 5 — Decision Agent](#8-agent-5--decision-agent)
9. [Agent 6 — WhatsApp Agent](#9-agent-6--whatsapp-agent)
10. [ML Prediction Service (Python gRPC)](#10-ml-prediction-service-python-grpc)
11. [Storage Layer](#11-storage-layer)
12. [Konfigurasi & Environment](#12-konfigurasi--environment)
13. [Alur Data End-to-End](#13-alur-data-end-to-end)
14. [Startup Order & Dependencies](#14-startup-order--dependencies)
15. [Roadmap Implementasi](#15-roadmap-implementasi)
16. [Checklist Implementasi](#16-checklist-implementasi)

---

## 1. Gambaran Multi-Agent Architecture

### Alur Pipeline (MVP)

```
┌─────────────────────────────────────────────────────────────────────┐
│                     EXTERNAL DATA SOURCES                           │
│  OANDA WebSocket  │  Twelve Data REST  │  Alpha Vantage REST        │
│  (Real-time tick) │  (OHLCV + news)    │  (Historical + News)       │
└──────────┬────────────────┬────────────────────┬────────────────────┘
           │                │                    │
           └────────────────┴────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────────────┐
│  AGENT 1: MarketDataAgent   [Go]                                    │
│                                                                     │
│  • Fetch OHLCV multi-timeframe (5m, 15m, 1h, 4h)                   │
│  • Auto-reconnect + failover ke sumber alternatif                   │
│  • Validasi & normalisasi candle (pip precision 5 desimal)          │
│  • Rolling buffer 200 candles per pair                              │
│                                                                     │
│  Output: MarketDataOutput{ Symbol, OHLCV[], Timeframe, Timestamp }  │
└──────────────────────────┬──────────────────────────────────────────┘
                           │
              ┌────────────┴────────────┐
              │  (concurrent goroutine) │
              ▼                         ▼
┌──────────────────────┐  ┌──────────────────────────────────────────┐
│  AGENT 2:            │  │  AGENT 3:                                │
│  TechnicalAgent [Go] │  │  FundamentalAgent [Go + Gemini API]      │
│                      │  │                                          │
│  Membaca chart       │  │  Membaca berita ekonomi                  │
│                      │  │                                          │
│  Indikator:          │  │  Sumber:                                 │
│  • RSI(14)           │  │  • Alpha Vantage News API                │
│  • MACD(12,26,9)     │  │  • Twelve Data News                      │
│  • EMA(50, 200)      │  │  • RSS feeds ekonomi                     │
│  • Bollinger(20,2)   │  │                                          │
│                      │  │  NLP: Gemini API (timeout 2s)            │
│  Output:             │  │  Cache: Redis TTL 5 menit                │
│  { signal: "BUY",    │  │                                          │
│    confidence: 80% } │  │  Output:                                 │
│                      │  │  { sentiment: "bullish",                 │
│                      │  │    confidence: 75% }                     │
└──────────┬───────────┘  └───────────────┬──────────────────────────┘
           │                              │
           └──────────────┬───────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────────────┐
│  AGENT 4: RiskAgent   [Go]                                          │
│                                                                     │
│  Input:  { balance: 1000, risk_percent: 1, entry: 1.0845 }         │
│                                                                     │
│  Kalkulasi:                                                         │
│  • LotSize   = (Balance × Risk%) / (SL_pips × PipValue)            │
│  • Stop Loss = 20 pip default  (adjustable per pair)               │
│  • Take Profit = 2× SL = 40 pip (RR 1:2)                           │
│                                                                     │
│  Output: { lot: 0.05, sl: 1.08250, tp: 1.08850 }                   │
└──────────────────────────┬──────────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────────────┐
│  AGENT 5: DecisionAgent  [Go]  ← "Otak Utama"                      │
│                                                                     │
│  Input dari semua agent:                                            │
│  { technical: "BUY", fund_sentiment: "bullish", risk: {lot,sl,tp}} │
│                                                                     │
│  Logika Keputusan (rule-based MVP):                                 │
│  if technical == "BUY"  && fundamental == "bullish"  → BUY          │
│  if technical == "SELL" && fundamental == "bearish"  → SELL         │
│  if fundamental == "neutral"                         → ikuti tech   │
│  else (konflik)                                      → HOLD         │
│                                                                     │
│  Confidence = (tech_conf × 0.60) + (fund_conf × 0.40)              │
│  ML Score boost (opsional, Phase 2 via Python gRPC)                 │
│                                                                     │
│  Output: { signal: "BUY", confidence: 88%, entry, sl, tp, lot }    │
└──────────────────────────┬──────────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────────────┐
│  AGENT 6: WhatsAppAgent  [Go → Node.js]                            │
│                                                                     │
│  Hanya bertugas sebagai "kurir" — tidak membuat keputusan           │
│                                                                     │
│  Go (DecisionAgent output)                                          │
│       │                                                             │
│       │ HTTP POST /send-signal (JSON, timeout 10s)                  │
│       ▼                                                             │
│  Node.js + Baileys (port 3001)                                      │
│       │                                                             │
│       │ sock.sendMessage()                                          │
│       ▼                                                             │
│  WhatsApp (nomor tujuan)                                            │
└─────────────────────────────────────────────────────────────────────┘
```

### Kenapa Go + Node.js?

```
Go  = AI Engine, concurrency, goroutine per pair, performa tinggi
Node.js = Baileys hanya tersedia di Node.js ekosistem
→ Dipisah: Go POST ke Node.js via HTTP, Node.js kirim ke WhatsApp
```

### Kenapa Multi-Agent vs Pipeline Biasa?

| Pendekatan Lama (Pipeline)       | Pendekatan Baru (Multi-Agent)             |
|----------------------------------|-------------------------------------------|
| Fungsi dipanggil berurutan       | Agent 2 & 3 berjalan **concurrent**       |
| Coupling tinggi antar layer      | Setiap agent punya kontrak input/output   |
| Sulit di-test satu per satu      | Setiap agent bisa di-mock dan di-test isolasi |
| Sulit tambah agent baru          | Tinggal implement interface `Agent`       |
| Tidak ada fallback per agent     | Setiap agent punya fallback sendiri       |

---

## 2. Agent Interface Contract

### `internal/agents/agent.go`

```go
package agents

import (
	"context"
	"time"
)

// AgentInput adalah container generik untuk input semua agent
type AgentInput struct {
	Pair        string
	Candles     []Candle     // dari MarketDataAgent
	Technical   *TechnicalOutput   // dari TechnicalAgent
	Fundamental *FundamentalOutput // dari FundamentalAgent
	Risk        *RiskOutput        // dari RiskAgent
	AccountBalance float64
	RiskPercent    float64
}

// AgentOutput adalah container generik untuk output semua agent
type AgentOutput struct {
	AgentName  string
	Success    bool
	Error      error
	Timestamp  time.Time

	// Diisi sesuai agent yang menghasilkan
	Technical   *TechnicalOutput
	Fundamental *FundamentalOutput
	Risk        *RiskOutput
	Decision    *DecisionOutput
}

// Agent adalah interface yang wajib diimplementasi oleh semua agent
type Agent interface {
	Name() string
	Run(ctx context.Context, input AgentInput) AgentOutput
}

// ── Output structs per agent ─────────────────────────────────────────

// TechnicalOutput hasil dari TechnicalAgent
type TechnicalOutput struct {
	Signal     string  // "BUY" | "SELL" | "HOLD"
	Confidence float64 // 0.0–1.0

	RSI        float64
	MACDHist   float64
	EMA50      float64
	EMA200     float64
	BBPosition float64 // 0.0 = lower band, 1.0 = upper band

	TechScore  float64 // weighted score final teknikal
	Reason     string  // e.g. "RSI oversold + MACD bullish cross"
}

// FundamentalOutput hasil dari FundamentalAgent
type FundamentalOutput struct {
	Sentiment  string  // "bullish" | "bearish" | "neutral"
	Confidence float64 // 0.0–1.0
	Score      float64 // dinormalisasi: bullish>0.5, bearish<0.5
	Reason     string  // max 15 kata
	FromCache  bool    // true jika dari Redis cache
}

// RiskOutput hasil dari RiskAgent
type RiskOutput struct {
	LotSize    float64 // ukuran lot
	StopLoss   float64 // harga SL
	TakeProfit float64 // harga TP
	SLPips     float64 // SL dalam pip
	TPPips     float64 // TP dalam pip
	RiskAmount float64 // nominal risk dalam USD
}

// DecisionOutput hasil dari DecisionAgent (sinyal final)
type DecisionOutput struct {
	Signal     string  // "BUY" | "SELL" | "HOLD"
	Confidence float64 // 0.0–1.0
	ConfPct    int     // dalam persen

	Entry      float64
	StopLoss   float64
	TakeProfit float64
	LotSize    float64
	RiskPct    float64

	TechSignal  string
	TechConf    float64
	TechReason  string
	FundSentiment string
	FundConf    float64
	FundReason  string
	MLScore     float64 // opsional

	RiskLevel  string // "LOW" | "MEDIUM" | "HIGH"
	Pair       string
	Timestamp  time.Time
}
```

---

## 3. Struktur Direktori Project

```
forex-agent/
│
├── cmd/
│   └── main.go                         # Entry point — init & orkestrasi semua agent
│
├── internal/
│   │
│   ├── agents/                         # ← CORE: semua agent di sini
│   │   ├── agent.go                    # Agent interface + semua Output struct
│   │   ├── market_data_agent.go        # Agent 1: fetch OHLCV, rolling buffer
│   │   ├── technical_agent.go          # Agent 2: RSI, MACD, EMA, BB → BUY/SELL/HOLD
│   │   ├── fundamental_agent.go        # Agent 3: berita + Gemini NLP → sentiment
│   │   ├── risk_agent.go               # Agent 4: lot size, SL, TP calculator
│   │   ├── decision_agent.go           # Agent 5: otak utama → sinyal final
│   │   └── whatsapp_agent.go           # Agent 6: kurir → HTTP POST ke Baileys
│   │
│   ├── indicators/                     # Kalkulasi indikator (dipanggil Agent 2)
│   │   ├── rsi.go                      # RSI(14), Wilder's smoothing
│   │   ├── macd.go                     # MACD(12,26,9)
│   │   ├── bollinger.go                # Bollinger Bands(20,2)
│   │   ├── moving_average.go           # EMA/SMA utilities
│   │   └── scorer.go                   # TechnicalScore aggregator
│   │
│   ├── feed/                           # Data ingestion (dipanggil Agent 1)
│   │   ├── websocket.go                # WebSocket client (OANDA)
│   │   ├── rest_poller.go              # REST polling (Twelve Data / Alpha Vantage)
│   │   └── normalizer.go               # Validasi & normalisasi OHLCV
│   │
│   ├── ml/                             # ML gRPC client (dipanggil Agent 5, opsional)
│   │   ├── client.go                   # gRPC client → Python ML service
│   │   └── proto/
│   │       └── ml.proto                # gRPC contract
│   │
│   ├── sentiment/                      # Gemini + news fetcher (dipanggil Agent 3)
│   │   ├── gemini.go                   # Gemini API client (timeout 2s)
│   │   ├── news_fetcher.go             # RSS / news API fetcher
│   │   └── cache.go                    # Redis cache TTL 5 menit
│   │
│   └── storage/
│       ├── timescale.go                # TimescaleDB queries
│       └── redis.go                    # Redis client & helpers
│
├── wa_service/                         # Node.js + Baileys (service TERPISAH)
│   ├── index.js                        # Express server + Baileys init + QR handler
│   ├── sender.js                       # sendMessage wrapper
│   ├── auth_info/                      # Session WA (di .gitignore!)
│   ├── package.json
│   └── .env
│
├── ml_service/                         # Python gRPC microservice (Phase 2, opsional)
│   ├── main.py                         # gRPC server entry point (port 50051)
│   ├── model.py                        # RandomForest training & predict
│   ├── features.py                     # Feature engineering
│   ├── train.py                        # Script training model
│   └── requirements.txt
│
├── config/
│   └── config.yaml                     # Semua konfigurasi
│
├── migrations/
│   └── 001_init.sql                    # TimescaleDB schema
│
├── docker-compose.yml                  # TimescaleDB + Redis
├── go.mod
└── go.sum
```

---

## 4. Agent 1 — MarketData Agent

### Tugas
Mengambil data OHLCV dari sumber eksternal, memvalidasi, dan mempublikasikan ke pipeline.

### Sumber Data (priority order)
1. **OANDA WebSocket** — real-time tick, terbaik untuk latency
2. **Twelve Data REST** — fallback, support multi-timeframe
3. **Alpha Vantage REST** — fallback kedua, gratis tier tersedia

### `internal/agents/market_data_agent.go`

```go
package agents

import (
	"context"
	"log"
	"time"

	"github.com/yourusername/forex-agent/internal/feed"
)

// Candle merepresentasikan satu candle OHLCV
type Candle struct {
	Pair      string
	Open      float64
	High      float64
	Low       float64
	Close     float64
	Volume    float64
	Spread    float64
	Timeframe string    // "5m" | "15m" | "1h" | "4h"
	Timestamp time.Time
}

// MarketDataAgent mengambil data OHLCV dari berbagai sumber
type MarketDataAgent struct {
	wsFeeds    map[string]*feed.WebSocketFeed // pair → feed
	pairs      []string
	timeframes []string
	buffers    map[string][]Candle            // pair → rolling buffer 200 candles
}

func NewMarketDataAgent(pairs, timeframes []string, wsURL, apiKey string) *MarketDataAgent {
	a := &MarketDataAgent{
		pairs:      pairs,
		timeframes: timeframes,
		buffers:    make(map[string][]Candle),
		wsFeeds:    make(map[string]*feed.WebSocketFeed),
	}
	// Inisialisasi buffer kosong per pair
	for _, p := range pairs {
		a.buffers[p] = make([]Candle, 0, 200)
	}
	return a
}

func (a *MarketDataAgent) Name() string { return "MarketDataAgent" }

// Run mengambil data terbaru dan mengembalikan candle buffer
func (a *MarketDataAgent) Run(ctx context.Context, input AgentInput) AgentOutput {
	candles, ok := a.buffers[input.Pair]
	if !ok || len(candles) < 15 {
		// Belum cukup data — kembalikan output sukses tapi kosong
		return AgentOutput{
			AgentName: a.Name(),
			Success:   false,
			Error:     fmt.Errorf("insufficient candle data for %s: need min 15, have %d", input.Pair, len(candles)),
			Timestamp: time.Now(),
		}
	}

	return AgentOutput{
		AgentName: a.Name(),
		Success:   true,
		Timestamp: time.Now(),
	}
}

// AppendCandle menambahkan candle baru ke rolling buffer (max 200)
func (a *MarketDataAgent) AppendCandle(pair string, c Candle) {
	buf := a.buffers[pair]
	if len(buf) >= 200 {
		buf = buf[1:] // geser — buang candle tertua
	}
	a.buffers[pair] = append(buf, c)
}

// GetCandles mengembalikan rolling buffer untuk pair tertentu
func (a *MarketDataAgent) GetCandles(pair string) []Candle {
	return a.buffers[pair]
}
```

### Data OHLCV JSON Format

```json
{
  "symbol": "EURUSD",
  "timeframe": "1h",
  "open": 1.0840,
  "high": 1.0850,
  "low": 1.0830,
  "close": 1.0845,
  "volume": 12500,
  "spread": 1.2,
  "timestamp": "2025-01-01T20:00:00Z"
}
```

### Scheduler (multi-timeframe)

```go
// Di cmd/main.go
c := cron.New()
c.AddFunc("*/5  * * * *", func() { runPipeline("5m")  })
c.AddFunc("*/15 * * * *", func() { runPipeline("15m") })
c.AddFunc("0    * * * *", func() { runPipeline("1h")  })
c.AddFunc("0 */4 * * *", func() { runPipeline("4h")  })
c.Start()
```

---

## 5. Agent 2 — Technical Agent

### Tugas
Membaca chart dan menghasilkan sinyal BUY/SELL/HOLD berdasarkan indikator teknikal.

### Indikator yang Digunakan
- **RSI(14)** — Relative Strength Index
- **MACD(12,26,9)** — Moving Average Convergence Divergence
- **EMA(50, 200)** — Exponential Moving Average (trend filter)
- **Bollinger Bands(20,2)** — volatility bands

### `internal/agents/technical_agent.go`

```go
package agents

import (
	"context"
	"fmt"
	"time"

	"github.com/yourusername/forex-agent/internal/indicators"
)

type TechnicalAgent struct{}

func NewTechnicalAgent() *TechnicalAgent { return &TechnicalAgent{} }
func (a *TechnicalAgent) Name() string   { return "TechnicalAgent" }

func (a *TechnicalAgent) Run(ctx context.Context, input AgentInput) AgentOutput {
	if len(input.Candles) < 26 {
		return AgentOutput{
			AgentName: a.Name(),
			Success:   false,
			Error:     fmt.Errorf("need min 26 candles for MACD, got %d", len(input.Candles)),
			Timestamp: time.Now(),
		}
	}

	// Jalankan semua indikator
	result := indicators.Analyze(input.Candles)

	output := &TechnicalOutput{
		Signal:     result.TechnicalDir,
		Confidence: result.TechnicalScore,
		RSI:        result.RSIValue,
		MACDHist:   result.MACDResult.Histogram,
		BBPosition: result.BBPosition,
		TechScore:  result.TechnicalScore,
		Reason:     buildTechReason(result),
	}

	return AgentOutput{
		AgentName: a.Name(),
		Success:   true,
		Technical: output,
		Timestamp: time.Now(),
	}
}

func buildTechReason(r indicators.IndicatorResult) string {
	reasons := []string{}
	if r.RSIValue <= 30 {
		reasons = append(reasons, "RSI oversold")
	} else if r.RSIValue >= 70 {
		reasons = append(reasons, "RSI overbought")
	}
	if r.MACDDirection != "HOLD" {
		reasons = append(reasons, "MACD "+r.MACDDirection)
	}
	if r.BBPosition <= 0.10 {
		reasons = append(reasons, "harga di lower BB")
	} else if r.BBPosition >= 0.90 {
		reasons = append(reasons, "harga di upper BB")
	}
	if len(reasons) == 0 {
		return "No strong technical signal"
	}
	return strings.Join(reasons, " + ")
}
```

### Output Contoh

```json
{
  "signal": "BUY",
  "confidence": 0.80,
  "rsi": 28.5,
  "macd_hist": 0.00123,
  "bb_position": 0.08,
  "reason": "RSI oversold + MACD bullish cross + harga di lower BB"
}
```

### Scoring Rules

```
TechnicalScore = (RSIScore × 0.40) + (MACDScore × 0.40) + (BBScore × 0.20)

RSI:
  <= 30 → BUY,  0.85  (oversold kuat)
  <= 40 → BUY,  0.65  (oversold moderat)
  >= 70 → SELL, 0.85  (overbought kuat)
  >= 60 → SELL, 0.65  (overbought moderat)
  else  → HOLD, 0.50

MACD:
  crossover bullish → BUY,  0.80
  crossover bearish → SELL, 0.80
  histogram > 0    → BUY,  0.60
  histogram < 0    → SELL, 0.60

BB:
  position <= 0.10 → BUY,  0.80  (di lower band)
  position >= 0.90 → SELL, 0.80  (di upper band)
  else             → HOLD, 0.50
```

---

## 6. Agent 3 — Fundamental Agent

### Tugas
Membaca dan menganalisa berita ekonomi untuk menentukan sentimen fundamental pair.

### Sumber Berita
- Alpha Vantage News Sentiment API
- Twelve Data News
- RSS feeds: Reuters, Bloomberg Economic

### `internal/agents/fundamental_agent.go`

```go
package agents

import (
	"context"
	"fmt"
	"time"

	"github.com/yourusername/forex-agent/internal/sentiment"
)

type FundamentalAgent struct {
	gemini      *sentiment.GeminiClient
	newsFetcher *sentiment.NewsFetcher
}

func NewFundamentalAgent(gemini *sentiment.GeminiClient, fetcher *sentiment.NewsFetcher) *FundamentalAgent {
	return &FundamentalAgent{gemini: gemini, newsFetcher: fetcher}
}

func (a *FundamentalAgent) Name() string { return "FundamentalAgent" }

func (a *FundamentalAgent) Run(ctx context.Context, input AgentInput) AgentOutput {
	// Fetch headlines untuk pair ini
	headlines, err := a.newsFetcher.FetchForPair(ctx, input.Pair)
	if err != nil || len(headlines) == 0 {
		// Fallback: tidak ada berita → netral
		return AgentOutput{
			AgentName: a.Name(),
			Success:   true,
			Fundamental: &FundamentalOutput{
				Sentiment:  "neutral",
				Confidence: 0.5,
				Score:      0.5,
				Reason:     "no relevant news found",
			},
			Timestamp: time.Now(),
		}
	}

	// Analisa via Gemini (dengan Redis cache)
	sentResult := a.gemini.AnalyzeSentiment(input.Pair, headlines)

	return AgentOutput{
		AgentName: a.Name(),
		Success:   true,
		Fundamental: &FundamentalOutput{
			Sentiment:  sentResult.Sentiment,
			Confidence: sentResult.Confidence,
			Score:      sentResult.Score,
			Reason:     sentResult.Reason,
			FromCache:  sentResult.FromCache,
		},
		Timestamp: time.Now(),
	}
}
```

### Gemini Prompt Template

```
You are a professional forex market analyst.
Analyze the sentiment impact of these news headlines on the {PAIR} currency pair.

Headlines:
{HEADLINES_NEWLINE_SEPARATED}

Respond ONLY with a valid JSON object. No explanation, no markdown:
{
  "sentiment": "bullish" OR "bearish" OR "neutral",
  "confidence": 0.0 to 1.0,
  "reason": "max 15 words explanation"
}
```

### Score Normalization

```
bullish  → Score = 0.5 + (confidence × 0.5)   → range 0.5–1.0
bearish  → Score = 0.5 - (confidence × 0.5)   → range 0.0–0.5
neutral  → Score = 0.5
```

### Output Contoh

```json
{
  "sentiment": "bullish",
  "confidence": 0.75,
  "score": 0.875,
  "reason": "Fed rate hike expectations boost USD demand",
  "from_cache": false
}
```

---

## 7. Agent 4 — Risk Agent

### Tugas
Menghitung ukuran posisi, stop loss, dan take profit berdasarkan parameter risk management.

### Formula Kalkulasi

```
PipValue  = 10 USD per pip (untuk 1 lot standar, pair major)
LotSize   = (Balance × RiskPercent / 100) / (SL_pips × PipValue)
StopLoss  = EntryPrice - (SL_pips × 0.0001) [untuk BUY]
          = EntryPrice + (SL_pips × 0.0001) [untuk SELL]
TakeProfit = EntryPrice + (TP_pips × 0.0001) [untuk BUY]
           = EntryPrice - (TP_pips × 0.0001) [untuk SELL]
```

### `internal/agents/risk_agent.go`

```go
package agents

import (
	"context"
	"fmt"
	"math"
	"time"
)

const (
	DefaultSLPips     = 20.0  // Default stop loss 20 pip
	DefaultTPPips     = 40.0  // Default take profit 40 pip (RR 1:2)
	PipValuePerLot    = 10.0  // USD per pip untuk 1 lot standar
	PipSize           = 0.0001 // Ukuran 1 pip (pair major)
)

type RiskAgent struct {
	DefaultSLPips float64
	DefaultTPPips float64
}

func NewRiskAgent() *RiskAgent {
	return &RiskAgent{
		DefaultSLPips: DefaultSLPips,
		DefaultTPPips: DefaultTPPips,
	}
}

func (a *RiskAgent) Name() string { return "RiskAgent" }

func (a *RiskAgent) Run(ctx context.Context, input AgentInput) AgentOutput {
	if input.AccountBalance <= 0 {
		return AgentOutput{
			AgentName: a.Name(),
			Success:   false,
			Error:     fmt.Errorf("invalid balance: %.2f", input.AccountBalance),
			Timestamp: time.Now(),
		}
	}
	if input.Technical == nil {
		return AgentOutput{
			AgentName: a.Name(),
			Success:   false,
			Error:     fmt.Errorf("technical output required to determine direction"),
			Timestamp: time.Now(),
		}
	}

	direction := input.Technical.Signal
	entry := input.Candles[len(input.Candles)-1].Close
	riskPct := input.RiskPercent
	if riskPct <= 0 {
		riskPct = 1.0 // default risk 1%
	}

	slPips := a.DefaultSLPips
	tpPips := a.DefaultTPPips

	// Kalkulasi lot size
	riskAmount := input.AccountBalance * (riskPct / 100.0)
	lotSize    := riskAmount / (slPips * PipValuePerLot)
	lotSize     = math.Round(lotSize*100) / 100 // bulatkan ke 2 desimal

	// Kalkulasi harga SL dan TP
	var sl, tp float64
	switch direction {
	case "BUY":
		sl = entry - (slPips * PipSize)
		tp = entry + (tpPips * PipSize)
	case "SELL":
		sl = entry + (slPips * PipSize)
		tp = entry - (tpPips * PipSize)
	default:
		// HOLD — tidak perlu kalkulasi risk
		return AgentOutput{
			AgentName: a.Name(),
			Success:   true,
			Risk:      &RiskOutput{},
			Timestamp: time.Now(),
		}
	}

	// Bulatkan ke 5 desimal (pip precision)
	round5 := func(v float64) float64 { return math.Round(v*100000) / 100000 }

	return AgentOutput{
		AgentName: a.Name(),
		Success:   true,
		Risk: &RiskOutput{
			LotSize:    lotSize,
			StopLoss:   round5(sl),
			TakeProfit: round5(tp),
			SLPips:     slPips,
			TPPips:     tpPips,
			RiskAmount: riskAmount,
		},
		Timestamp: time.Now(),
	}
}
```

### Input/Output Contoh

```json
// Input
{
  "balance": 1000,
  "risk_percent": 1,
  "entry": 1.08450,
  "direction": "BUY"
}

// Output
{
  "lot_size": 0.05,
  "stop_loss": 1.08250,
  "take_profit": 1.08850,
  "sl_pips": 20,
  "tp_pips": 40,
  "risk_amount": 10.00
}
```

---

## 8. Agent 5 — Decision Agent

### Tugas
**"Otak utama"** — menggabungkan output dari semua agent dan menghasilkan sinyal final.

### Logika Keputusan

```
// Aturan dasar (rule-based MVP)
if technical.signal == "BUY" && fundamental.sentiment == "bullish":
    signal = "BUY"
elif technical.signal == "SELL" && fundamental.sentiment == "bearish":
    signal = "SELL"
elif technical.signal == fundamental.signal (keduanya sama):
    signal = technical.signal  // agreement tanpa fundamental
else:
    signal = "HOLD"  // konflik → tidak masuk posisi

// Confidence
confidence = (tech_confidence × 0.60) + (fund_confidence × 0.40)

// Optional: MLScore boost dari Python service
if ml_score > 0:
    confidence = (confidence × 0.80) + (ml_score × 0.20)
```

### `internal/agents/decision_agent.go`

```go
package agents

import (
	"context"
	"fmt"
	"time"

	"github.com/yourusername/forex-agent/internal/ml"
)

type DecisionAgent struct {
	mlClient *ml.Client // optional — bisa nil jika ML service tidak jalan
}

func NewDecisionAgent(mlClient *ml.Client) *DecisionAgent {
	return &DecisionAgent{mlClient: mlClient}
}

func (a *DecisionAgent) Name() string { return "DecisionAgent" }

func (a *DecisionAgent) Run(ctx context.Context, input AgentInput) AgentOutput {
	tech := input.Technical
	fund := input.Fundamental
	risk := input.Risk

	if tech == nil {
		return AgentOutput{
			AgentName: a.Name(),
			Success:   false,
			Error:     fmt.Errorf("TechnicalAgent output required"),
			Timestamp: time.Now(),
		}
	}

	// ── Logika sinyal utama ───────────────────────────────────────────
	signal := decideSignal(tech.Signal, fundSentiment(fund))

	// ── Confidence calculation ────────────────────────────────────────
	fundConf := 0.5 // default netral jika tidak ada fundamental
	fundSent := "neutral"
	fundReason := ""
	if fund != nil {
		fundConf   = fund.Confidence
		fundSent   = fund.Sentiment
		fundReason = fund.Reason
	}

	confidence := (tech.Confidence * 0.60) + (fundConf * 0.40)

	// ── Optional: ML Score boost ──────────────────────────────────────
	mlScore := 0.0
	if a.mlClient != nil && len(input.Candles) >= 26 {
		mlScore, _ = a.mlClient.Predict(ctx, tech, input.Candles)
		if mlScore > 0 {
			// Jika SELL, invert ML score (ML dilatih untuk probabilitas naik)
			adjustedML := mlScore
			if tech.Signal == "SELL" {
				adjustedML = 1.0 - mlScore
			}
			confidence = (confidence * 0.80) + (adjustedML * 0.20)
		}
	}

	// ── Risk assessment ───────────────────────────────────────────────
	riskLevel := assessRisk(confidence)

	// ── Entry, SL, TP dari RiskAgent ─────────────────────────────────
	var entry, sl, tp, lot float64
	if risk != nil && signal != "HOLD" {
		entry = input.Candles[len(input.Candles)-1].Close
		sl    = risk.StopLoss
		tp    = risk.TakeProfit
		lot   = risk.LotSize
	}

	return AgentOutput{
		AgentName: a.Name(),
		Success:   true,
		Decision: &DecisionOutput{
			Signal:        signal,
			Confidence:    confidence,
			ConfPct:       int(confidence * 100),
			Entry:         entry,
			StopLoss:      sl,
			TakeProfit:    tp,
			LotSize:       lot,
			RiskPct:       input.RiskPercent,
			TechSignal:    tech.Signal,
			TechConf:      tech.Confidence,
			TechReason:    tech.Reason,
			FundSentiment: fundSent,
			FundConf:      fundConf,
			FundReason:    fundReason,
			MLScore:       mlScore,
			RiskLevel:     riskLevel,
			Pair:          input.Pair,
			Timestamp:     time.Now(),
		},
		Timestamp: time.Now(),
	}
}

func decideSignal(techSignal, fundSentiment string) string {
	switch {
	case techSignal == "BUY" && fundSentiment == "bullish":
		return "BUY"
	case techSignal == "SELL" && fundSentiment == "bearish":
		return "SELL"
	case techSignal == fundSentiment && techSignal != "HOLD":
		return techSignal // agreement tanpa fundamental alignment
	case fundSentiment == "neutral":
		return techSignal // jika tidak ada berita, ikuti teknikal
	default:
		return "HOLD" // konflik antara technical dan fundamental
	}
}

func fundSentiment(fund *FundamentalOutput) string {
	if fund == nil {
		return "neutral"
	}
	return fund.Sentiment
}

func assessRisk(confidence float64) string {
	// Semakin jauh dari 0.5, semakin yakin (risk rendah)
	certainty := confidence
	if confidence < 0.5 {
		certainty = 1.0 - confidence
	}
	switch {
	case certainty >= 0.75:
		return "LOW"    // sinyal kuat
	case certainty >= 0.60:
		return "MEDIUM"
	default:
		return "HIGH"   // sinyal lemah → jangan kirim alert
	}
}
```

### Output Contoh

```json
{
  "signal": "BUY",
  "confidence": 0.88,
  "conf_pct": 88,
  "entry": 1.08450,
  "stop_loss": 1.08250,
  "take_profit": 1.08850,
  "lot_size": 0.05,
  "risk_pct": 1.0,
  "tech_signal": "BUY",
  "tech_conf": 0.80,
  "tech_reason": "RSI oversold + MACD bullish cross",
  "fund_sentiment": "bullish",
  "fund_conf": 0.75,
  "fund_reason": "Fed rate hike expectations boost USD",
  "ml_score": 0.72,
  "risk_level": "LOW",
  "pair": "EURUSD",
  "timestamp": "2025-01-01T20:00:00Z"
}
```

---

## 9. Agent 6 — WhatsApp Agent

### Tugas
Mengirim alert ke WhatsApp. **Hanya kurir** — tidak membuat keputusan apapun.

### Arsitektur

```
Go (DecisionAgent output)
     │
     │ HTTP POST /send-signal (JSON)
     │ timeout: 10 detik
     ▼
Node.js + Baileys (port 3001)
     │
     │ sock.sendMessage(jid, { text: message })
     ▼
WhatsApp (nomor tujuan)
```

### Kenapa Dipisah (Go + Node.js)?

```
AI Engine = Go/Python  →  performa, concurrency
WhatsApp   = Node.js   →  Baileys hanya tersedia di Node.js ekosistem
```

### `internal/agents/whatsapp_agent.go`

```go
package agents

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type WhatsAppAgent struct {
	serviceURL  string
	targetPhone string
	httpClient  *http.Client
}

func NewWhatsAppAgent(serviceURL, targetPhone string) *WhatsAppAgent {
	return &WhatsAppAgent{
		serviceURL:  serviceURL,
		targetPhone: targetPhone,
		httpClient:  &http.Client{Timeout: 10 * time.Second},
	}
}

func (a *WhatsAppAgent) Name() string { return "WhatsAppAgent" }

func (a *WhatsAppAgent) Run(ctx context.Context, input AgentInput) AgentOutput {
	if input.Decision == nil {
		return AgentOutput{
			AgentName: a.Name(),
			Success:   false,
			Error:     fmt.Errorf("DecisionAgent output required"),
			Timestamp: time.Now(),
		}
	}

	dec := input.Decision

	// Jangan kirim jika HOLD atau risk terlalu tinggi
	if dec.Signal == "HOLD" || dec.RiskLevel == "HIGH" {
		return AgentOutput{AgentName: a.Name(), Success: true, Timestamp: time.Now()}
	}

	// Jangan kirim jika confidence di bawah 60%
	if dec.Confidence < 0.60 {
		return AgentOutput{AgentName: a.Name(), Success: true, Timestamp: time.Now()}
	}

	msg := formatWhatsAppMessage(dec)

	body, _ := json.Marshal(map[string]string{
		"phone":   a.targetPhone,
		"message": msg,
	})

	req, err := http.NewRequestWithContext(ctx, "POST", a.serviceURL+"/send-signal", bytes.NewReader(body))
	if err != nil {
		return AgentOutput{AgentName: a.Name(), Success: false, Error: err, Timestamp: time.Now()}
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return AgentOutput{
			AgentName: a.Name(),
			Success:   false,
			Error:     fmt.Errorf("WA service unreachable: %w", err),
			Timestamp: time.Now(),
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return AgentOutput{
			AgentName: a.Name(),
			Success:   false,
			Error:     fmt.Errorf("WA service error: HTTP %d", resp.StatusCode),
			Timestamp: time.Now(),
		}
	}

	return AgentOutput{AgentName: a.Name(), Success: true, Timestamp: time.Now()}
}
```

### Format Pesan WhatsApp

```go
func formatWhatsAppMessage(d *DecisionOutput) string {
	dirEmoji := map[string]string{
		"BUY":  "🟢 BUY",
		"SELL": "🔴 SELL",
		"HOLD": "⚪ HOLD",
	}[d.Signal]

	riskEmoji := map[string]string{
		"LOW":    "✅",
		"MEDIUM": "⚠️",
		"HIGH":   "🚨",
	}[d.RiskLevel]

	return fmt.Sprintf(`🚀 *FOREX ALERT*

*Pair*   : %s
*Signal* : %s

*Entry*  : %.5f
*SL*     : %.5f (%.0f pip)
*TP*     : %.5f (%.0f pip)

📊 *Analysis:*
├ Technical   : %s (%.0f%%)
│  _%s_
└ Fundamental : %s (%.0f%%)
   _%s_

🎯 *Confidence* : %d%%
💰 *Lot Size*   : %.2f
⚠️ *Risk*       : %.1f%% | %s %s
⏰ %s WIB

_Selalu gunakan money management yang baik!_`,
		d.Pair, dirEmoji,
		d.Entry,
		d.StopLoss, d.RiskOutput.SLPips,
		d.TakeProfit, d.RiskOutput.TPPips,
		d.TechSignal, d.TechConf*100,
		d.TechReason,
		strings.ToUpper(d.FundSentiment), d.FundConf*100,
		d.FundReason,
		d.ConfPct,
		d.LotSize,
		d.RiskPct, riskEmoji, d.RiskLevel,
		d.Timestamp.In(time.FixedZone("WIB", 7*3600)).Format("15:04:05"),
	)
}
```

### Contoh Pesan WhatsApp Aktual

```
🚀 FOREX ALERT

Pair   : EURUSD
Signal : 🟢 BUY

Entry  : 1.08450
SL     : 1.08250 (20 pip)
TP     : 1.08850 (40 pip)

📊 Analysis:
├ Technical   : BUY (80%)
│  RSI oversold + MACD bullish cross
└ Fundamental : BULLISH (75%)
   Fed rate hike expectations boost USD

🎯 Confidence : 88%
💰 Lot Size   : 0.05
⚠️ Risk       : 1.0% | ✅ LOW
⏰ 20:00:05 WIB

Selalu gunakan money management yang baik!
```

### Baileys Node.js Service (`wa_service/index.js`)

```javascript
const {
  default: makeWASocket,
  useMultiFileAuthState,
  DisconnectReason,
} = require("@whiskeysockets/baileys");
const express = require("express");
const qrcode = require("qrcode-terminal");
require("dotenv").config();

const app = express();
app.use(express.json());
let sock = null;

async function connectToWhatsApp() {
  const { state, saveCreds } = await useMultiFileAuthState("auth_info");

  sock = makeWASocket({
    auth: state,
    printQRInTerminal: false,
    logger: require("pino")({ level: "silent" }),
  });

  sock.ev.on("connection.update", async ({ connection, lastDisconnect, qr }) => {
    if (qr) {
      console.log("\n📱 Scan QR ini dengan WhatsApp kamu:\n");
      qrcode.generate(qr, { small: true });
    }
    if (connection === "close") {
      const loggedOut =
        lastDisconnect?.error?.output?.statusCode === DisconnectReason.loggedOut;
      if (!loggedOut) connectToWhatsApp(); // auto-reconnect
    }
    if (connection === "open") console.log("[WA] ✅ WhatsApp Connected!");
  });

  sock.ev.on("creds.update", saveCreds);
}

// Endpoint dipanggil oleh Go WhatsAppAgent
app.post("/send-signal", async (req, res) => {
  const { phone, message } = req.body;
  if (!sock) return res.status(503).json({ error: "WA not connected yet" });
  try {
    const jid = phone.replace(/[^0-9]/g, "") + "@s.whatsapp.net";
    await sock.sendMessage(jid, { text: message });
    res.json({ ok: true });
  } catch (err) {
    res.status(500).json({ error: err.message });
  }
});

const PORT = process.env.WA_SERVICE_PORT || 3001;
app.listen(PORT, () => console.log(`[WA Service] Listening on :${PORT}`));
connectToWhatsApp();
```

---

## 10. ML Prediction Service (Python gRPC)

> ML Service bersifat **opsional** di MVP. Decision Agent akan fallback ke score 0.5 (neutral) jika ML service tidak tersedia.

### Peran dalam Multi-Agent

ML Service bukan agent otonom, melainkan **capability tambahan** yang dipanggil oleh Decision Agent untuk meningkatkan akurasi confidence score.

### `ml_service/model.py`

```python
import numpy as np
from sklearn.ensemble import RandomForestClassifier
from sklearn.preprocessing import StandardScaler
import joblib
import os

MODEL_PATH = "model.pkl"
SCALER_PATH = "scaler.pkl"

class ForexPredictor:
    """
    Features (6 kolom):
    [0] RSI value (0-100)
    [1] MACD histogram
    [2] Bollinger Band position (0.0-1.0)
    [3] Volume change %
    [4] Spread dalam pips
    [5] Arah candle sebelumnya (1=bullish, -1=bearish, 0=doji)

    Label:
    1 = harga naik >= 10 pip dalam 3 candle berikutnya
    0 = harga turun / flat
    """

    def __init__(self):
        self.model = None
        self.scaler = StandardScaler()

    def train(self, X: np.ndarray, y: np.ndarray):
        X_scaled = self.scaler.fit_transform(X)
        self.model = RandomForestClassifier(
            n_estimators=200,
            max_depth=10,
            min_samples_split=20,
            random_state=42,
            class_weight='balanced'
        )
        self.model.fit(X_scaled, y)
        joblib.dump(self.model, MODEL_PATH)
        joblib.dump(self.scaler, SCALER_PATH)

    def load(self):
        if os.path.exists(MODEL_PATH):
            self.model  = joblib.load(MODEL_PATH)
            self.scaler = joblib.load(SCALER_PATH)
            return True
        return False

    def predict_proba(self, features: list) -> float:
        """Return probabilitas harga naik (0.0 - 1.0)"""
        if self.model is None:
            return 0.5  # Fallback ke netral

        X = np.array(features).reshape(1, -1)
        X_scaled = self.scaler.transform(X)
        proba = self.model.predict_proba(X_scaled)[0]
        return float(proba[1])  # proba[1] = probabilitas kelas 1 (naik)
```

### Proto Contract

```protobuf
syntax = "proto3";
package ml;
option go_package = "./ml";

service MLService {
  rpc Predict(PredictRequest) returns (PredictResponse);
}

message PredictRequest {
  double rsi                   = 1;
  double macd_histogram        = 2;
  double bb_position           = 3;
  double volume_change         = 4;
  double spread                = 5;
  double prev_candle_direction = 6;
}

message PredictResponse {
  double score = 1;  // 0.0 - 1.0 probabilitas naik
}
```

Port gRPC: **50051** | Timeout client: **500ms** | Fallback: `0.5`

---

## 11. Storage Layer

### TimescaleDB Tables

```sql
-- Aktifkan TimescaleDB extension
CREATE EXTENSION IF NOT EXISTS timescaledb;

-- Tabel candles (data OHLCV)
CREATE TABLE IF NOT EXISTS candles (
    time        TIMESTAMPTZ NOT NULL,
    pair        VARCHAR(10)  NOT NULL,
    timeframe   VARCHAR(4)   NOT NULL DEFAULT '1h',
    open        DOUBLE PRECISION NOT NULL,
    high        DOUBLE PRECISION NOT NULL,
    low         DOUBLE PRECISION NOT NULL,
    close       DOUBLE PRECISION NOT NULL,
    volume      DOUBLE PRECISION,
    spread      DOUBLE PRECISION,
    PRIMARY KEY (time, pair, timeframe)
);
SELECT create_hypertable('candles', 'time', if_not_exists => TRUE);
CREATE INDEX ON candles (pair, timeframe, time DESC);

-- Tabel sinyal yang dihasilkan bot
CREATE TABLE IF NOT EXISTS signals (
    id            SERIAL PRIMARY KEY,
    time          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    pair          VARCHAR(10)  NOT NULL,
    direction     VARCHAR(4)   NOT NULL,
    confidence    DOUBLE PRECISION,
    tech_score    DOUBLE PRECISION,
    tech_signal   VARCHAR(4),
    fund_sentiment VARCHAR(10),
    fund_score    DOUBLE PRECISION,
    ml_score      DOUBLE PRECISION,
    risk_level    VARCHAR(6),
    lot_size      DOUBLE PRECISION,
    entry_price   DOUBLE PRECISION,
    stop_loss     DOUBLE PRECISION,
    take_profit   DOUBLE PRECISION,
    sl_pips       DOUBLE PRECISION,
    tp_pips       DOUBLE PRECISION,
    tech_reason   TEXT,
    fund_reason   TEXT
);
SELECT create_hypertable('signals', 'time', if_not_exists => TRUE);

-- Tabel cache berita
CREATE TABLE IF NOT EXISTS news_cache (
    hash        VARCHAR(64) PRIMARY KEY,
    pair        VARCHAR(10),
    headlines   TEXT,
    sentiment   VARCHAR(10),
    confidence  DOUBLE PRECISION,
    created_at  TIMESTAMPTZ DEFAULT NOW()
);
```

### Redis Keys

```
latest_price:{PAIR}          → float64, TTL 10s
latest_candle:{PAIR}:{TF}    → JSON Candle, TTL 10s
sentiment:{sha256_hash}      → JSON FundamentalOutput, TTL 5m
agent_status:{AGENT_NAME}    → "ok"|"error", TTL 60s
```

---

## 12. Konfigurasi & Environment

### `config/config.yaml`

```yaml
# Broker & Data Sources
oanda:
  websocket_url: "wss://stream-fxtrade.oanda.com/v3/accounts/{account_id}/pricing/stream"
  api_key: "${OANDA_API_KEY}"
  account_id: "${OANDA_ACCOUNT_ID}"

twelve_data:
  api_key: "${TWELVE_DATA_KEY}"
  base_url: "https://api.twelvedata.com"

alpha_vantage:
  api_key: "${ALPHA_VANTAGE_KEY}"
  base_url: "https://www.alphavantage.co/query"

# Currency pairs yang dipantau
pairs:
  - "EUR_USD"
  - "GBP_USD"
  - "USD_JPY"
  - "AUD_USD"

# Timeframe scheduler
scheduler:
  timeframes: ["5m", "15m", "1h", "4h"]

# Account settings
account:
  balance: 1000.0         # Modal awal dalam USD
  risk_percent: 1.0       # Risk per trade dalam %
  default_sl_pips: 20.0   # Default stop loss pip
  default_tp_pips: 40.0   # Default take profit pip (RR 1:2)

# AI & ML
gemini:
  api_key: "${GEMINI_API_KEY}"
  model: "gemini-1.5-flash"
  timeout_ms: 2000

ml_service:
  enabled: true
  grpc_address: "localhost:50051"
  timeout_ms: 500

# Signal thresholds (Decision Agent)
signal:
  buy_threshold: 0.65         # FinalScore >= ini → BUY
  sell_threshold: 0.35        # FinalScore <= ini → SELL
  min_confidence_to_alert: 0.60
  weights:
    technical: 0.60           # Lebih tinggi di MVP (rule-based)
    fundamental: 0.40
  ml_boost_weight: 0.20       # Diaktifkan hanya jika ml_service.enabled = true

# Storage
timescaledb:
  dsn: "postgres://forex_user:${DB_PASSWORD}@localhost:5432/forex_db"

redis:
  address: "localhost:6379"
  password: "${REDIS_PASSWORD}"
  sentiment_ttl_minutes: 5
  price_ttl_seconds: 10

# WhatsApp
whatsapp:
  service_url: "http://localhost:3001"
  target_phone: "${WA_TARGET_PHONE}"  # e.g. "628123456789"
  min_confidence_to_alert: 0.60       # Jangan kirim jika di bawah ini
  rate_limit_seconds: 180             # Min jarak antar pesan ke nomor yang sama
```

### Environment Variables Wajib

```
OANDA_API_KEY
OANDA_ACCOUNT_ID
TWELVE_DATA_KEY       (opsional — untuk fallback)
ALPHA_VANTAGE_KEY     (opsional — untuk fallback)
GEMINI_API_KEY
WA_TARGET_PHONE       (format: 628xxxxxxxxxx)
DB_PASSWORD
REDIS_PASSWORD
```

---

## 13. Alur Data End-to-End

### Alur Per Tick (setiap scheduler fire)

```
Jam 20:00 — Scheduler 1h fire untuk EURUSD
│
├─ [MarketDataAgent]
│    Ambil OHLCV EURUSD 1h → validasi → append ke buffer
│    Buffer: 150 candles tersedia
│
├─ CONCURRENT (goroutine):
│    ├─ [TechnicalAgent]
│    │    RSI=28.5 (oversold) + MACD crossover bullish + BB lower
│    │    → Signal: BUY, Confidence: 80%
│    │
│    └─ [FundamentalAgent]
│         Fetch headlines → Gemini analysis (atau Redis cache)
│         "Fed rate hike expectations boost USD demand"
│         → Sentiment: bullish, Confidence: 75%
│
├─ [RiskAgent]
│    Balance=1000, Risk=1%, Entry=1.08450
│    Lot=0.05, SL=1.08250 (20pip), TP=1.08850 (40pip)
│
├─ [DecisionAgent]
│    Technical=BUY + Fundamental=bullish → Signal: BUY ✅
│    Confidence = (0.80 × 0.60) + (0.75 × 0.40) = 78%
│    ML boost (opsional): score=0.72 → final = 80.4%
│    RiskLevel = MEDIUM (>60%)
│
├─ Filter: Signal=BUY, Confidence=80% >= 60%, RiskLevel=MEDIUM ✅
│
├─ [WhatsAppAgent]
│    POST http://localhost:3001/send-signal
│    → Pesan terkirim ke WhatsApp 628xxx
│
└─ [Storage]
     Simpan candle ke TimescaleDB
     Simpan signal ke TimescaleDB
     Update Redis: latest_price:EURUSD
```

### Alur Konflik (HOLD)

```
[TechnicalAgent]    → Signal: BUY
[FundamentalAgent]  → Sentiment: bearish
[DecisionAgent]     → KONFLIK → Signal: HOLD
[WhatsAppAgent]     → SKIP (tidak kirim)
```

---

## 14. Startup Order & Dependencies

```bash
# 1. Infrastructure
docker-compose up -d
# Tunggu TimescaleDB dan Redis ready

# 2. Jalankan migrasi DB
psql -U forex_user -d forex_db -f migrations/001_init.sql

# 3. WhatsApp Baileys Service (HARUS sebelum Go bot)
cd wa_service && node index.js
# → Scan QR Code dengan WhatsApp
# → Tunggu: "[WA] ✅ WhatsApp Connected!"

# 4. ML Service (opsional)
cd ml_service && python main.py
# → "[ML Service] Listening on :50051"

# 5. Go Bot (paling akhir)
go run cmd/main.go
```

### `docker-compose.yml`

```yaml
version: "3.8"
services:
  timescaledb:
    image: timescale/timescaledb:latest-pg16
    environment:
      POSTGRES_USER: forex_user
      POSTGRES_PASSWORD: ${DB_PASSWORD}
      POSTGRES_DB: forex_db
    ports:
      - "5432:5432"
    volumes:
      - timescale_data:/var/lib/postgresql/data
      - ./migrations:/docker-entrypoint-initdb.d

  redis:
    image: redis:7-alpine
    command: redis-server --requirepass ${REDIS_PASSWORD}
    ports:
      - "6379:6379"

volumes:
  timescale_data:
```

### Common Errors & Fixes

| Error | Penyebab | Fix |
|---|---|---|
| `WA service unreachable` | Baileys belum jalan | `node wa_service/index.js` dulu |
| HTTP `503` dari `/send-signal` | WA belum connect | Tunggu "✅ Connected" |
| `insufficient candle data` | Buffer belum terisi | Tunggu 15–200 candle masuk |
| `TechnicalAgent output required` | Urutan agent salah | Pastikan Technical selesai sebelum Risk/Decision |
| `context deadline exceeded` gRPC | ML service lambat | Timeout 500ms, fallback 0.5 |
| `json: cannot unmarshal` Gemini | Model return markdown | Strip backtick sebelum Unmarshal |
| `RSI always 50.0` | Kurang dari 15 candle | Normal — tunggu buffer terisi |
| `loggedOut` Baileys | WA logout paksa dari HP | Hapus `auth_info/`, scan QR ulang |

---

## 15. Roadmap Implementasi

> **Strategi:** Jangan langsung buat semuanya. Mulai dari yang menghasilkan output nyata,
> baru tambahkan kompleksitas secara bertahap.

### Minggu 1 — Fondasi Go

- Setup `go mod init github.com/<user>/forex-agent`
- Scaffold semua direktori `internal/agents/`
- Implementasi `Agent` interface + semua output struct di `agent.go`
- Implementasi `MarketDataAgent` dengan **mock data** dulu

### Minggu 2 — Ambil Data Forex Real

- Implementasi `internal/feed/websocket.go` (OANDA WebSocket)
- Implementasi `internal/feed/normalizer.go` (validasi OHLCV)
- Koneksi ke Twelve Data REST sebagai fallback
- Test: data EURUSD masuk ke rolling buffer ✅

### Minggu 3 — Technical Agent

- Implementasi semua indikator di `internal/indicators/`
  (RSI, MACD, Bollinger, EMA, TechnicalScore)
- Implementasi `TechnicalAgent` → output BUY/SELL/HOLD + confidence
- Unit test per indikator
- Target: `TechnicalAgent.Run()` menghasilkan sinyal valid ✅

### Minggu 4 — WhatsApp Integration

- Setup Baileys Node.js service (`wa_service/`)
- Scan QR Code → tunggu "✅ WhatsApp Connected!"
- Implementasi `WhatsAppAgent` (Go HTTP client ke Baileys)
- Implementasi `DecisionAgent` versi sederhana (technical-only dulu)
- Wire semua di `cmd/main.go`
- Test end-to-end: sinyal teknikal → pesan WA terkirim ✅

```
[Minggu 4 Target]
MarketDataAgent → TechnicalAgent → DecisionAgent → WhatsAppAgent → WA
```

### Minggu 5 — Risk Agent

- Implementasi `RiskAgent` (lot, SL, TP)
- Integrasi `RiskAgent` ke `DecisionAgent`
- Update format pesan WA dengan entry/SL/TP/lot
- Setup `docker-compose.yml` (TimescaleDB + Redis)
- Implementasi storage layer, jalankan `migrations/001_init.sql`

### Minggu 6 — Fundamental Agent

- Implementasi `internal/sentiment/news_fetcher.go`
- Implementasi `internal/sentiment/gemini.go` (Gemini API + timeout 2s)
- Implementasi `internal/sentiment/cache.go` (Redis TTL 5 menit)
- Implementasi `FundamentalAgent` (concurrent dengan TechnicalAgent via goroutine)
- Update `DecisionAgent` dengan logika fundamental konfirmasi ✅

### Minggu 7 — LLM / ML Boost (opsional)

- Train Random Forest dengan data historis CSV
- Setup Python gRPC service (`ml_service/`)
- Implementasi `internal/ml/client.go` (timeout 500ms)
- Integrasi ML score ke `DecisionAgent` sebagai boost opsional

### Minggu 8 — Backtesting & Evaluasi

- Replay data historis dari TimescaleDB
- Bandingkan: sinyal **rule-based** vs **dengan ML boost**
- Kumpulkan data untuk penelitian/paper
- Grafana dashboard (connect ke TimescaleDB)

> **Rekomendasi Paper:** Buat versi 1 **tanpa ML** dulu (rule-based yang stabil).
> Setelah baseline ada, tambahkan ML sebagai pembanding.
> Ini menghasilkan penelitian lebih kuat:
> *"apakah ML benar-benar meningkatkan kualitas sinyal dibanding metode tradisional?"*

---

## 16. Checklist Implementasi

### Phase 1 — Core MVP (Minggu 1–4)

**Fondasi & Data:**
- [ ] Setup `go mod init github.com/<user>/forex-agent`
- [ ] Scaffold `internal/agents/` dengan semua file
- [ ] Implementasi `Agent` interface + semua output struct (`agent.go`)
- [ ] Implementasi `MarketDataAgent` (mock data dulu → real data minggu 2)
- [ ] Implementasi `internal/feed/` (websocket, normalizer)

**Technical Agent:**
- [ ] Implementasi `internal/indicators/` (RSI, MACD, BB, EMA, scorer)
- [ ] Implementasi `TechnicalAgent` → BUY/SELL/HOLD + confidence
- [ ] Unit test per indikator

**WhatsApp Integration:**
- [ ] Setup `wa_service/` Node.js + Baileys (scan QR, tunggu connected)
- [ ] Implementasi `WhatsAppAgent` (Go HTTP client ke Baileys)
- [ ] Implementasi `DecisionAgent` (rule-based, technical-only dulu)
- [ ] Wire semua agent di `cmd/main.go` dengan scheduler cron
- [ ] Test: `go build ./...` harus bersih
- [ ] Test end-to-end: sinyal teknikal → pesan WA terkirim ✅

### Phase 2 — Risk + Fundamental (Minggu 5–6)

- [ ] Implementasi `RiskAgent` (lot, SL, TP)
- [ ] Setup `docker-compose.yml` (TimescaleDB + Redis)
- [ ] Jalankan `migrations/001_init.sql`
- [ ] Implementasi `internal/storage/` (timescale + redis)
- [ ] Implementasi `FundamentalAgent` + Gemini client + Redis cache
- [ ] Update `DecisionAgent` dengan logika fundamental konfirmasi
- [ ] Update format pesan WA dengan entry/SL/TP/lot ✅

### Phase 3 — ML + Polish & Research (Minggu 7–8)

- [ ] Train ML model (RandomForest) dengan data historis CSV
- [ ] Setup Python gRPC service `ml_service/` (port 50051)
- [ ] Implementasi `internal/ml/client.go` (timeout 500ms)
- [ ] Update `DecisionAgent` dengan ML score boost (opsional)
- [ ] Backtesting engine (replay TimescaleDB)
- [ ] Grafana dashboard (connect TimescaleDB)
- [ ] Structured logging (`log/slog` Go stdlib)
- [ ] Monitoring agent health di Redis
- [ ] Multi-pair concurrent (satu goroutine per pair)
- [ ] Bandingkan sinyal rule-based vs ML-boosted (untuk paper) ✅

---

> **⚠️ Disclaimer:** Bot ini untuk tujuan analisa dan edukasi.
> Win rate 60–65% sudah sangat baik di dunia forex profesional.
> Selalu gunakan risk management dan jangan mengandalkan sinyal otomatis sepenuhnya.
