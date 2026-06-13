# 🤖 Forex Market Analysis Bot — Architecture & Implementation Guide

> **Target Stack:** Go (Golang) + Python ML Microservice + Gemini API  
> **Purpose:** Real-time forex signal generation dengan confidence score  
> **Audience:** AI Agent / Developer untuk implementasi kode

---

## 📋 Daftar Isi

1. [Gambaran Arsitektur](#1-gambaran-arsitektur)
2. [Struktur Direktori Project](#2-struktur-direktori-project)
3. [Layer 1 — Data Ingestion](#3-layer-1--data-ingestion)
4. [Layer 2 — Technical Indicator Engine](#4-layer-2--technical-indicator-engine)
5. [Layer 3 — ML Prediction Service](#5-layer-3--ml-prediction-service)
6. [Layer 4 — Sentiment Analysis (Gemini)](#6-layer-4--sentiment-analysis-gemini)
7. [Layer 5 — Signal Aggregator](#7-layer-5--signal-aggregator)
8. [Layer 6 — Alert & Output](#8-layer-6--alert--output)
9. [Layer 7 — Storage](#9-layer-7--storage)
10. [Konfigurasi & Environment](#10-konfigurasi--environment)
11. [gRPC Contract ML Service](#11-grpc-contract-ml-service)
12. [Alur Data End-to-End](#12-alur-data-end-to-end)
13. [Dependencies & Setup](#13-dependencies--setup)
14. [Checklist Implementasi](#14-checklist-implementasi)

---

## 1. Gambaran Arsitektur

```
┌─────────────────────────────────────────────────────────────────┐
│                     EXTERNAL DATA SOURCES                       │
│   OANDA WebSocket Feed    │    Alpha Vantage REST API           │
│   (Real-time OHLCV tick)  │    (Historical + News)             │
└──────────────┬────────────────────────┬────────────────────────┘
               │                        │
               ▼                        ▼
┌─────────────────────────────────────────────────────────────────┐
│              LAYER 1: DATA INGESTION (Go)                       │
│                                                                 │
│  ┌──────────────────┐      ┌──────────────────────────────┐    │
│  │  WebSocket Client│      │     REST Poller              │    │
│  │  (goroutine)     │      │  (scheduled via cron)        │    │
│  │  - reconnect     │      │  - historical OHLCV          │    │
│  │  - heartbeat     │      │  - news headlines            │    │
│  └────────┬─────────┘      └──────────────┬───────────────┘    │
│           │                               │                     │
│           └─────────────┬─────────────────┘                    │
│                         ▼                                       │
│              ┌─────────────────────┐                           │
│              │   Price Normalizer  │  ← Validates & formats    │
│              │   (OHLCV struct)    │     raw tick data         │
│              └──────────┬──────────┘                           │
└─────────────────────────┼───────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────────┐
│              LAYER 2: TECHNICAL INDICATOR ENGINE (Go)           │
│                                                                 │
│  Input: Candle OHLCV slice (rolling window 200 candles)        │
│                                                                 │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌───────────────────┐ │
│  │  RSI(14) │ │ MACD     │ │Bollinger │ │  MA Cross         │ │
│  │  0-100   │ │(12,26,9) │ │Bands(20) │ │  (MA50 vs MA200)  │ │
│  └──────────┘ └──────────┘ └──────────┘ └───────────────────┘ │
│                                                                 │
│  Output: IndicatorResult{RSI, MACDHist, BBPosition, MASignal}  │
│  TechnicalScore: float64 (0.0 - 1.0)                          │
└─────────────────────────┬───────────────────────────────────────┘
                          │
              ┌───────────┴───────────┐
              ▼                       ▼
┌─────────────────────┐   ┌──────────────────────────────────────┐
│  LAYER 3: ML ENGINE │   │  LAYER 4: SENTIMENT ENGINE           │
│  (Python gRPC svc)  │   │  (Go → Gemini API)                   │
│                     │   │                                      │
│  Random Forest      │   │  Input: News headlines (RSS/API)     │
│  Features: 6 cols   │   │  Prompt: structured JSON request     │
│  - RSI              │   │  Model: gemini-1.5-flash             │
│  - MACD histogram   │   │  Timeout: 2s (fallback if exceeded)  │
│  - BB position      │   │                                      │
│  - Volume change    │   │  Output:                             │
│  - Spread           │   │  {sentiment, confidence, reason}     │
│  - Prev candle dir  │   │                                      │
│                     │   │  Cache: Redis TTL 5 menit            │
│  Output: float64    │   │  (same news → skip API call)         │
│  MLScore (0.0-1.0)  │   │                                      │
└────────┬────────────┘   └──────────────┬───────────────────────┘
         │                               │
         └───────────────┬───────────────┘
                         ▼
┌─────────────────────────────────────────────────────────────────┐
│              LAYER 5: SIGNAL AGGREGATOR (Go)                    │
│                                                                 │
│  Weights (configurable via config.yaml):                       │
│    Technical : 0.50 (most reliable for entry timing)           │
│    ML Model  : 0.30 (pattern recognition)                      │
│    Sentiment : 0.20 (macro context)                            │
│                                                                 │
│  FinalScore = (Tech × 0.50) + (ML × 0.30) + (Sent × 0.20)    │
│                                                                 │
│  Direction Logic:                                               │
│    FinalScore >= 0.65  → BUY  🟢                               │
│    FinalScore <= 0.35  → SELL 🔴                               │
│    0.35 < Score < 0.65 → HOLD ⚪                               │
│                                                                 │
│  Signal{Pair, Direction, Confidence%, TechScore,               │
│          MLScore, SentScore, Reason, Timestamp}                │
└─────────────────────────┬───────────────────────────────────────┘
                          │
              ┌───────────┴───────────┐
              ▼                       ▼
┌─────────────────────┐   ┌──────────────────────────────────────┐
│  LAYER 6A: TELEGRAM │   │  LAYER 6B: STORAGE (TimescaleDB)     │
│  Alert Bot          │   │                                      │
│                     │   │  Tables:                             │
│  Format:            │   │  - candles (OHLCV time-series)       │
│  🔔 SIGNAL DETECTED │   │  - indicators (RSI, MACD, BB)        │
│  Pair: EUR/USD      │   │  - signals (semua sinyal + score)    │
│  Action: 🟢 BUY     │   │  - news_cache (deduplikasi berita)   │
│  Score: 71.6%       │   │                                      │
│  Tech: 78%          │   │  Redis:                              │
│  ML:   65%          │   │  - latest_price:{pair}               │
│  Sent: 60%          │   │  - sentiment_cache:{hash}            │
└─────────────────────┘   └──────────────────────────────────────┘
```

---

## 2. Struktur Direktori Project

```
forex-bot/
│
├── cmd/
│   └── main.go                    # Entry point — init semua service
│
├── internal/
│   ├── feed/
│   │   ├── websocket.go           # Koneksi WebSocket ke OANDA
│   │   ├── rest_poller.go         # REST polling Alpha Vantage
│   │   └── normalizer.go          # Validasi & normalisasi OHLCV
│   │
│   ├── indicators/
│   │   ├── rsi.go                 # RSI(14) calculation
│   │   ├── macd.go                # MACD(12,26,9) calculation
│   │   ├── bollinger.go           # Bollinger Bands(20,2)
│   │   ├── moving_average.go      # SMA / EMA utilities
│   │   └── scorer.go              # TechnicalScore aggregator
│   │
│   ├── ml/
│   │   ├── client.go              # gRPC client ke Python ML service
│   │   └── proto/
│   │       └── ml.proto           # gRPC contract definition
│   │
│   ├── sentiment/
│   │   ├── gemini.go              # Gemini API client
│   │   ├── news_fetcher.go        # RSS / news API fetcher
│   │   └── cache.go               # Redis cache untuk sentiment
│   │
│   ├── strategy/
│   │   ├── aggregator.go          # Weighted score aggregation
│   │   ├── signal.go              # Signal struct & direction logic
│   │   └── risk.go                # Risk level assessment
│   │
│   ├── alert/
│   │   ├── telegram.go            # Telegram Bot API integration
│   │   └── formatter.go           # Format pesan alert
│   │
│   └── storage/
│       ├── timescale.go           # TimescaleDB queries
│       └── redis.go               # Redis client & helpers
│
├── ml_service/                    # Python microservice (terpisah)
│   ├── main.py                    # gRPC server entry point
│   ├── model.py                   # RandomForest training & predict
│   ├── features.py                # Feature engineering
│   ├── train.py                   # Script training model
│   └── requirements.txt
│
├── config/
│   └── config.yaml                # Semua konfigurasi
│
├── migrations/
│   └── 001_init.sql               # TimescaleDB schema
│
├── docker-compose.yml             # TimescaleDB + Redis
├── go.mod
└── go.sum
```

---

## 3. Layer 1 — Data Ingestion

### `internal/feed/websocket.go`

```go
package feed

import (
	"encoding/json"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

// OHLCVCandle merepresentasikan satu candle data
type OHLCVCandle struct {
	Pair      string    `json:"pair"`
	Open      float64   `json:"open"`
	High      float64   `json:"high"`
	Low       float64   `json:"low"`
	Close     float64   `json:"close"`
	Volume    float64   `json:"volume"`
	Spread    float64   `json:"spread"`
	Timestamp time.Time `json:"timestamp"`
}

type WebSocketFeed struct {
	url     string
	apiKey  string
	pairs   []string
	Output  chan OHLCVCandle  // channel output ke layer berikutnya
	done    chan struct{}
}

func NewWebSocketFeed(url, apiKey string, pairs []string) *WebSocketFeed {
	return &WebSocketFeed{
		url:    url,
		apiKey: apiKey,
		pairs:  pairs,
		Output: make(chan OHLCVCandle, 100),
		done:   make(chan struct{}),
	}
}

// Connect menghubungkan ke WebSocket dengan auto-reconnect
func (f *WebSocketFeed) Connect() {
	go func() {
		for {
			select {
			case <-f.done:
				return
			default:
				if err := f.connectOnce(); err != nil {
					log.Printf("[WS] Disconnected: %v — reconnecting in 5s", err)
					time.Sleep(5 * time.Second)
				}
			}
		}
	}()
}

func (f *WebSocketFeed) connectOnce() error {
	conn, _, err := websocket.DefaultDialer.Dial(f.url, nil)
	if err != nil {
		return err
	}
	defer conn.Close()

	// Subscribe ke pair yang diinginkan
	subscribeMsg := map[string]interface{}{
		"type":        "subscribe",
		"instruments": f.pairs,
	}
	if err := conn.WriteJSON(subscribeMsg); err != nil {
		return err
	}

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			return err
		}

		var candle OHLCVCandle
		if err := json.Unmarshal(msg, &candle); err != nil {
			log.Printf("[WS] Parse error: %v", err)
			continue
		}

		f.Output <- candle
	}
}

func (f *WebSocketFeed) Stop() {
	close(f.done)
}
```

### `internal/feed/normalizer.go`

```go
package feed

import (
	"errors"
	"math"
)

// Normalize memvalidasi dan membersihkan data candle mentah
func Normalize(raw OHLCVCandle) (OHLCVCandle, error) {
	// Validasi nilai tidak boleh negatif atau NaN
	if raw.Close <= 0 || math.IsNaN(raw.Close) {
		return OHLCVCandle{}, errors.New("invalid close price")
	}
	if raw.High < raw.Low {
		return OHLCVCandle{}, errors.New("high < low: invalid candle")
	}
	if raw.Open <= 0 || raw.High <= 0 || raw.Low <= 0 {
		return OHLCVCandle{}, errors.New("invalid OHLC values")
	}

	// Bulatkan ke 5 desimal (standar forex pip)
	raw.Open  = math.Round(raw.Open*100000) / 100000
	raw.High  = math.Round(raw.High*100000) / 100000
	raw.Low   = math.Round(raw.Low*100000) / 100000
	raw.Close = math.Round(raw.Close*100000) / 100000

	return raw, nil
}
```

---

## 4. Layer 2 — Technical Indicator Engine

### `internal/indicators/rsi.go`

```go
package indicators

import "math"

// RSI menghitung Relative Strength Index (periode default: 14)
// Input: slice harga close, minimal length = period + 1
// Output: nilai RSI (0.0 - 100.0)
func RSI(closes []float64, period int) float64 {
	if len(closes) < period+1 {
		return 50.0 // nilai netral jika data tidak cukup
	}

	var gains, losses float64
	for i := 1; i <= period; i++ {
		change := closes[i] - closes[i-1]
		if change > 0 {
			gains += change
		} else {
			losses += math.Abs(change)
		}
	}

	avgGain := gains / float64(period)
	avgLoss := losses / float64(period)

	// Smoothing untuk periode selanjutnya
	for i := period + 1; i < len(closes); i++ {
		change := closes[i] - closes[i-1]
		if change > 0 {
			avgGain = (avgGain*float64(period-1) + change) / float64(period)
			avgLoss = (avgLoss * float64(period-1)) / float64(period)
		} else {
			avgGain = (avgGain * float64(period-1)) / float64(period)
			avgLoss = (avgLoss*float64(period-1) + math.Abs(change)) / float64(period)
		}
	}

	if avgLoss == 0 {
		return 100.0
	}

	rs := avgGain / avgLoss
	return 100.0 - (100.0 / (1.0 + rs))
}

// RSIScore mengkonversi nilai RSI ke confidence score (0.0 - 1.0)
// Untuk sinyal BUY: RSI rendah (oversold) = score tinggi
// Untuk sinyal SELL: RSI tinggi (overbought) = score tinggi
func RSIScore(rsi float64) (direction string, score float64) {
	switch {
	case rsi <= 30: // Oversold kuat → BUY signal
		return "BUY", 0.85
	case rsi <= 40: // Oversold moderat → BUY lemah
		return "BUY", 0.65
	case rsi >= 70: // Overbought kuat → SELL signal
		return "SELL", 0.85
	case rsi >= 60: // Overbought moderat → SELL lemah
		return "SELL", 0.65
	default: // Netral
		return "HOLD", 0.50
	}
}
```

### `internal/indicators/macd.go`

```go
package indicators

// MACDResult menyimpan semua komponen MACD
type MACDResult struct {
	MACDLine   float64
	SignalLine float64
	Histogram  float64
}

// MACD menghitung Moving Average Convergence Divergence
// Standard forex: fast=12, slow=26, signal=9
func MACD(closes []float64, fast, slow, signal int) MACDResult {
	if len(closes) < slow+signal {
		return MACDResult{}
	}

	emaFast   := EMA(closes, fast)
	emaSlow   := EMA(closes, slow)
	macdLine  := emaFast - emaSlow

	// Signal line = EMA dari MACD line (simplified)
	signalLine := macdLine * 0.9 // approximation; idealnya EMA dari MACD series
	histogram  := macdLine - signalLine

	return MACDResult{
		MACDLine:   macdLine,
		SignalLine: signalLine,
		Histogram:  histogram,
	}
}

// MACDScore menghasilkan score dari posisi histogram dan crossover
func MACDScore(result MACDResult, prevHistogram float64) (direction string, score float64) {
	// Bullish crossover: histogram berubah dari negatif ke positif
	if prevHistogram < 0 && result.Histogram > 0 {
		return "BUY", 0.80
	}
	// Bearish crossover: histogram berubah dari positif ke negatif
	if prevHistogram > 0 && result.Histogram < 0 {
		return "SELL", 0.80
	}
	// Momentum berlanjut
	if result.Histogram > 0 {
		return "BUY", 0.60
	}
	if result.Histogram < 0 {
		return "SELL", 0.60
	}
	return "HOLD", 0.50
}
```

### `internal/indicators/scorer.go`

```go
package indicators

import "github.com/yourusername/forex-bot/internal/feed"

// IndicatorResult menyimpan semua hasil indikator
type IndicatorResult struct {
	RSIValue      float64
	RSIDirection  string
	RSIScore      float64

	MACDResult    MACDResult
	MACDDirection string
	MACDScore     float64

	BBPosition    float64 // 0.0 = di lower band, 1.0 = di upper band
	BBDirection   string
	BBScore       float64

	TechnicalScore float64 // weighted average semua indikator
	TechnicalDir   string  // arah mayoritas
}

// Analyze menjalankan semua indikator dan mengembalikan score gabungan
func Analyze(candles []feed.OHLCVCandle) IndicatorResult {
	closes := extractCloses(candles)

	rsiVal := RSI(closes, 14)
	rsiDir, rsiScore := RSIScore(rsiVal)

	macdResult := MACD(closes, 12, 26, 9)
	var prevHist float64
	if len(candles) > 1 {
		prevCloses := extractCloses(candles[:len(candles)-1])
		prevMACD := MACD(prevCloses, 12, 26, 9)
		prevHist = prevMACD.Histogram
	}
	macdDir, macdScore := MACDScore(macdResult, prevHist)

	bbPos, bbDir, bbScore := BollingerScore(candles, 20, 2.0)

	// Weighted technical score: RSI 40%, MACD 40%, BB 20%
	techScore := (rsiScore * 0.40) + (macdScore * 0.40) + (bbScore * 0.20)
	techDir   := majorityDirection(rsiDir, macdDir, bbDir)

	return IndicatorResult{
		RSIValue: rsiVal, RSIDirection: rsiDir, RSIScore: rsiScore,
		MACDResult: macdResult, MACDDirection: macdDir, MACDScore: macdScore,
		BBPosition: bbPos, BBDirection: bbDir, BBScore: bbScore,
		TechnicalScore: techScore, TechnicalDir: techDir,
	}
}

func extractCloses(candles []feed.OHLCVCandle) []float64 {
	closes := make([]float64, len(candles))
	for i, c := range candles {
		closes[i] = c.Close
	}
	return closes
}

func majorityDirection(dirs ...string) string {
	count := map[string]int{}
	for _, d := range dirs {
		count[d]++
	}
	best, max := "HOLD", 0
	for dir, n := range count {
		if n > max {
			best, max = dir, n
		}
	}
	return best
}
```

---

## 5. Layer 3 — ML Prediction Service

### `ml_service/model.py`

```python
# ml_service/model.py
import numpy as np
from sklearn.ensemble import RandomForestClassifier
from sklearn.preprocessing import StandardScaler
import joblib
import os

MODEL_PATH = "model.pkl"
SCALER_PATH = "scaler.pkl"

class ForexPredictor:
    """
    Memprediksi arah harga berikutnya berdasarkan fitur teknikal.

    Features (6 kolom):
    [0] RSI value (0-100)
    [1] MACD histogram value
    [2] Bollinger Band position (0.0-1.0)
    [3] Volume change % dari candle sebelumnya
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
        """Latih model dengan data historis"""
        X_scaled = self.scaler.fit_transform(X)
        self.model = RandomForestClassifier(
            n_estimators=200,
            max_depth=10,
            min_samples_split=20,   # Hindari overfit
            random_state=42,
            class_weight='balanced' # Tangani imbalanced dataset
        )
        self.model.fit(X_scaled, y)
        joblib.dump(self.model, MODEL_PATH)
        joblib.dump(self.scaler, SCALER_PATH)

    def load(self):
        """Load model yang sudah dilatih"""
        if os.path.exists(MODEL_PATH):
            self.model  = joblib.load(MODEL_PATH)
            self.scaler = joblib.load(SCALER_PATH)
            return True
        return False

    def predict_proba(self, features: list) -> float:
        """
        Return probabilitas harga naik (0.0 - 1.0)
        Contoh: 0.72 = 72% kemungkinan naik
        """
        if self.model is None:
            return 0.5  # Fallback ke netral

        X = np.array(features).reshape(1, -1)
        X_scaled = self.scaler.transform(X)
        proba = self.model.predict_proba(X_scaled)[0]

        # proba[1] = probabilitas kelas 1 (naik)
        return float(proba[1])
```

### `ml_service/main.py` — gRPC Server

```python
# ml_service/main.py
import grpc
from concurrent import futures
import ml_pb2
import ml_pb2_grpc
from model import ForexPredictor

predictor = ForexPredictor()

class MLServicer(ml_pb2_grpc.MLServiceServicer):
    def Predict(self, request, context):
        features = [
            request.rsi,
            request.macd_histogram,
            request.bb_position,
            request.volume_change,
            request.spread,
            request.prev_candle_direction
        ]
        score = predictor.predict_proba(features)
        return ml_pb2.PredictResponse(score=score)

def serve():
    predictor.load()
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    ml_pb2_grpc.add_MLServiceServicer_to_server(MLServicer(), server)
    server.add_insecure_port('[::]:50051')
    server.start()
    print("[ML Service] Listening on :50051")
    server.wait_for_termination()

if __name__ == '__main__':
    serve()
```

### `internal/ml/proto/ml.proto`

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
  double prev_candle_direction = 6;  // 1=bullish, -1=bearish, 0=doji
}

message PredictResponse {
  double score = 1;  // 0.0 - 1.0 probabilitas naik
}
```

---

## 6. Layer 4 — Sentiment Analysis (Gemini)

### `internal/sentiment/gemini.go`

```go
package sentiment

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type GeminiClient struct {
	apiKey     string
	httpClient *http.Client
	cache      *RedisCache
}

type SentimentResult struct {
	Sentiment  string  // "bullish" | "bearish" | "neutral"
	Confidence float64 // 0.0 - 1.0
	Reason     string
	Score      float64 // dinormalisasi: bullish>0.5, bearish<0.5
}

func NewGeminiClient(apiKey string, cache *RedisCache) *GeminiClient {
	return &GeminiClient{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 2 * time.Second, // HARD LIMIT: jangan lebih dari 2 detik
		},
		cache: cache,
	}
}

// AnalyzeSentiment menganalisa sentimen berita untuk currency pair
// Mengembalikan score netral (0.5) jika API timeout atau error
func (g *GeminiClient) AnalyzeSentiment(pair string, headlines []string) SentimentResult {
	// Hash headlines sebagai cache key
	cacheKey := fmt.Sprintf("sentiment:%x", sha256.Sum256([]byte(strings.Join(headlines, "|"))))

	// Cek cache Redis dulu (TTL 5 menit)
	if cached, ok := g.cache.Get(cacheKey); ok {
		var result SentimentResult
		if err := json.Unmarshal([]byte(cached), &result); err == nil {
			return result
		}
	}

	// Buat prompt yang strict — output hanya JSON
	prompt := fmt.Sprintf(`You are a professional forex market analyst.
Analyze the sentiment impact of these news headlines on the %s currency pair.

Headlines:
%s

Respond ONLY with a valid JSON object. No explanation, no markdown:
{
  "sentiment": "bullish" OR "bearish" OR "neutral",
  "confidence": 0.0 to 1.0,
  "reason": "max 15 words explanation"
}`, pair, strings.Join(headlines, "\n"))

	result, err := g.callAPI(prompt)
	if err != nil {
		// Fallback ke netral jika API gagal/timeout
		return SentimentResult{Sentiment: "neutral", Confidence: 0.5, Score: 0.5}
	}

	// Normalisasi ke score 0.0-1.0
	result.Score = sentimentToScore(result.Sentiment, result.Confidence)

	// Simpan ke cache
	if data, err := json.Marshal(result); err == nil {
		g.cache.Set(cacheKey, string(data), 5*time.Minute)
	}

	return result
}

func (g *GeminiClient) callAPI(prompt string) (SentimentResult, error) {
	url := "https://generativelanguage.googleapis.com/v1beta/models/gemini-1.5-flash:generateContent?key=" + g.apiKey

	body := map[string]interface{}{
		"contents": []map[string]interface{}{
			{"parts": []map[string]string{{"text": prompt}}},
		},
	}

	bodyBytes, _ := json.Marshal(body)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return SentimentResult{}, err
	}
	defer resp.Body.Close()

	var geminiResp struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&geminiResp); err != nil {
		return SentimentResult{}, err
	}

	if len(geminiResp.Candidates) == 0 {
		return SentimentResult{}, fmt.Errorf("no candidates in response")
	}

	text := geminiResp.Candidates[0].Content.Parts[0].Text

	var result SentimentResult
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		return SentimentResult{}, fmt.Errorf("failed to parse JSON: %v", err)
	}

	return result, nil
}

func sentimentToScore(sentiment string, confidence float64) float64 {
	switch sentiment {
	case "bullish":
		// Score di atas 0.5, semakin tinggi confidence semakin tinggi score
		return 0.5 + (confidence * 0.5)
	case "bearish":
		// Score di bawah 0.5
		return 0.5 - (confidence * 0.5)
	default:
		return 0.5
	}
}
```

---

## 7. Layer 5 — Signal Aggregator

### `internal/strategy/aggregator.go`

```go
package strategy

import (
	"time"

	"github.com/yourusername/forex-bot/internal/indicators"
	"github.com/yourusername/forex-bot/internal/ml"
	"github.com/yourusername/forex-bot/internal/sentiment"
)

// ScoreWeights mengatur bobot masing-masing komponen
// Jumlah harus = 1.0
type ScoreWeights struct {
	Technical float64 // Indikator teknikal (RSI, MACD, BB)
	ML        float64 // Random Forest prediction
	Sentiment float64 // Gemini sentiment analysis
}

// DefaultWeights adalah bobot default yang direkomendasikan
var DefaultWeights = ScoreWeights{
	Technical: 0.50,
	ML:        0.30,
	Sentiment: 0.20,
}

// Signal merepresentasikan sinyal trading final
type Signal struct {
	Pair           string
	Direction      string    // "BUY" | "SELL" | "HOLD"
	Confidence     float64   // Final score 0.0-1.0
	ConfidencePct  int       // Confidence dalam persen
	RiskLevel      string    // "LOW" | "MEDIUM" | "HIGH"

	// Breakdown per komponen
	TechScore  float64
	MLScore    float64
	SentScore  float64
	TechReason string
	SentReason string

	Timestamp  time.Time
}

// Aggregate menggabungkan semua score menjadi sinyal final
func Aggregate(
	techResult  indicators.IndicatorResult,
	mlScore     float64,
	sentResult  sentiment.SentimentResult,
	pair        string,
	weights     ScoreWeights,
) Signal {
	// Normalisasi ML score berdasarkan arah teknikal
	// Jika teknikal BUY, ML score tinggi = bagus
	// Jika teknikal SELL, kita perlu invert ML score
	adjustedML := mlScore
	if techResult.TechnicalDir == "SELL" {
		adjustedML = 1.0 - mlScore
	}

	// Weighted average
	final := (techResult.TechnicalScore * weights.Technical) +
		(adjustedML * weights.ML) +
		(sentResult.Score * weights.Sentiment)

	direction := scoreToDirection(final)
	risk      := assessRisk(final)

	return Signal{
		Pair:          pair,
		Direction:     direction,
		Confidence:    final,
		ConfidencePct: int(final * 100),
		RiskLevel:     risk,
		TechScore:     techResult.TechnicalScore,
		MLScore:       mlScore,
		SentScore:     sentResult.Score,
		TechReason:    buildTechReason(techResult),
		SentReason:    sentResult.Reason,
		Timestamp:     time.Now(),
	}
}

func scoreToDirection(score float64) string {
	switch {
	case score >= 0.65:
		return "BUY"
	case score <= 0.35:
		return "SELL"
	default:
		return "HOLD"
	}
}

func assessRisk(score float64) string {
	// Semakin jauh dari 0.5, semakin rendah risk (semakin yakin)
	confidence := score
	if score < 0.5 {
		confidence = 1.0 - score
	}
	switch {
	case confidence >= 0.75:
		return "LOW"    // Sinyal kuat, risk rendah
	case confidence >= 0.60:
		return "MEDIUM"
	default:
		return "HIGH"   // Sinyal lemah, jangan masuk
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
	if len(reasons) == 0 {
		return "No strong technical signal"
	}
	return strings.Join(reasons, " + ")
}
```

---

## 8. Layer 6 — Alert & Output

### `internal/alert/telegram.go`

```go
package alert

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/yourusername/forex-bot/internal/strategy"
)

type TelegramBot struct {
	token  string
	chatID string
}

func NewTelegramBot(token, chatID string) *TelegramBot {
	return &TelegramBot{token: token, chatID: chatID}
}

// SendSignal mengirim alert sinyal trading ke Telegram
func (t *TelegramBot) SendSignal(sig strategy.Signal, currentPrice float64) error {
	msg := formatSignalMessage(sig, currentPrice)
	return t.sendMessage(msg)
}

func formatSignalMessage(sig strategy.Signal, price float64) string {
	dirEmoji := map[string]string{
		"BUY":  "🟢",
		"SELL": "🔴",
		"HOLD": "⚪",
	}[sig.Direction]

	riskEmoji := map[string]string{
		"LOW":    "✅",
		"MEDIUM": "⚠️",
		"HIGH":   "🚨",
	}[sig.RiskLevel]

	return fmt.Sprintf(`🔔 *SIGNAL DETECTED*

*Pair*    : %s
*Action*  : %s %s
*Price*   : %.5f

📊 *Confidence Breakdown:*
├ Technical : %d%% (%s)
├ ML Model  : %d%%
└ Sentiment : %d%% (%s)

🎯 *Final Score* : %d%%
%s *Risk Level*  : %s
⏰ *Time*        : %s

_Selalu gunakan money management yang baik!_`,
		sig.Pair,
		dirEmoji, sig.Direction,
		price,
		int(sig.TechScore*100), sig.TechReason,
		int(sig.MLScore*100),
		int(sig.SentScore*100), sig.SentReason,
		sig.ConfidencePct,
		riskEmoji, sig.RiskLevel,
		sig.Timestamp.Format("15:04:05 WIB"),
	)
}

func (t *TelegramBot) sendMessage(text string) error {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", t.token)
	body, _ := json.Marshal(map[string]interface{}{
		"chat_id":    t.chatID,
		"text":       text,
		"parse_mode": "Markdown",
	})
	_, err := http.Post(url, "application/json", bytes.NewReader(body))
	return err
}
```

---

## 9. Layer 7 — Storage

### `migrations/001_init.sql`

```sql
-- Aktifkan TimescaleDB extension
CREATE EXTENSION IF NOT EXISTS timescaledb;

-- Tabel candles (data OHLCV)
CREATE TABLE IF NOT EXISTS candles (
    time        TIMESTAMPTZ NOT NULL,
    pair        VARCHAR(10)  NOT NULL,
    open        DOUBLE PRECISION NOT NULL,
    high        DOUBLE PRECISION NOT NULL,
    low         DOUBLE PRECISION NOT NULL,
    close       DOUBLE PRECISION NOT NULL,
    volume      DOUBLE PRECISION,
    spread      DOUBLE PRECISION,
    PRIMARY KEY (time, pair)
);

-- Konversi ke hypertable (time-series optimization)
SELECT create_hypertable('candles', 'time', if_not_exists => TRUE);
CREATE INDEX ON candles (pair, time DESC);

-- Tabel sinyal yang dihasilkan bot
CREATE TABLE IF NOT EXISTS signals (
    id          SERIAL PRIMARY KEY,
    time        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    pair        VARCHAR(10)  NOT NULL,
    direction   VARCHAR(4)   NOT NULL,   -- BUY/SELL/HOLD
    confidence  DOUBLE PRECISION,
    tech_score  DOUBLE PRECISION,
    ml_score    DOUBLE PRECISION,
    sent_score  DOUBLE PRECISION,
    risk_level  VARCHAR(6),
    price       DOUBLE PRECISION,
    tech_reason TEXT,
    sent_reason TEXT
);

SELECT create_hypertable('signals', 'time', if_not_exists => TRUE);

-- Tabel cache berita (hindari re-fetch berita sama)
CREATE TABLE IF NOT EXISTS news_cache (
    hash        VARCHAR(64) PRIMARY KEY,
    pair        VARCHAR(10),
    headlines   TEXT,
    sentiment   VARCHAR(10),
    confidence  DOUBLE PRECISION,
    created_at  TIMESTAMPTZ DEFAULT NOW()
);
```

---

## 10. Konfigurasi & Environment

### `config/config.yaml`

```yaml
# Broker & Data Source
oanda:
  websocket_url: "wss://stream-fxtrade.oanda.com/v3/accounts/{account_id}/pricing/stream"
  api_key: "${OANDA_API_KEY}"
  account_id: "${OANDA_ACCOUNT_ID}"

alpha_vantage:
  api_key: "${ALPHA_VANTAGE_KEY}"
  base_url: "https://www.alphavantage.co/query"

# Currency pairs yang dipantau
pairs:
  - "EUR_USD"
  - "GBP_USD"
  - "USD_JPY"
  - "AUD_USD"

# AI & ML
gemini:
  api_key: "${GEMINI_API_KEY}"
  model: "gemini-1.5-flash" # Gunakan Flash untuk latency rendah
  timeout_ms: 2000 # Hard limit 2 detik

ml_service:
  grpc_address: "localhost:50051"
  timeout_ms: 500

# Signal thresholds
signal:
  buy_threshold: 0.65 # Final score >= ini → BUY
  sell_threshold: 0.35 # Final score <= ini → SELL
  weights:
    technical: 0.50
    ml: 0.30
    sentiment: 0.20

# Storage
timescaledb:
  dsn: "postgres://forex_user:${DB_PASSWORD}@localhost:5432/forex_db"

redis:
  address: "localhost:6379"
  password: "${REDIS_PASSWORD}"
  sentiment_ttl_minutes: 5
  price_ttl_seconds: 10

# Alert
telegram:
  bot_token: "${TELEGRAM_BOT_TOKEN}"
  chat_id: "${TELEGRAM_CHAT_ID}"
  min_confidence_to_alert: 60 # Hanya alert jika >= 60%
```

---

## 11. gRPC Contract ML Service

### Flow komunikasi Go ↔ Python

```
Go (ML Client)                     Python (ML Service gRPC)
     │                                        │
     │  PredictRequest {                      │
     │    rsi: 28.5,                          │
     │    macd_histogram: 0.00234,            │
     │    bb_position: 0.12,                  │
     │    volume_change: 0.15,                │
     │──────────────────────────────────────>│
     │    spread: 1.2,                        │
     │    prev_candle_direction: 1.0          │
     │  }                                     │
     │                                        │  feature extraction
     │                                        │  scaler.transform()
     │                                        │  model.predict_proba()
     │                                        │
     │  PredictResponse { score: 0.72 }      │
     │<──────────────────────────────────────│
     │                                        │
     │  Timeout jika > 500ms → pakai 0.5     │
```

---

## 12. Alur Data End-to-End

```
1. WebSocket OANDA menerima tick baru EUR/USD
   │
2. Normalizer memvalidasi & format candle
   │
3. Candle disimpan ke buffer rolling window (200 candles)
   │
4. Secara CONCURRENT (goroutine):
   ├── 4a. Indicator Engine menghitung RSI, MACD, BB → TechScore
   ├── 4b. ML Client gRPC memanggil Python service → MLScore
   └── 4c. Sentiment Client memanggil Gemini API → SentScore
            (dengan Redis cache — skip API jika berita sama)
   │
5. Aggregator menggabungkan ketiga score:
   FinalScore = (0.50 × Tech) + (0.30 × ML) + (0.20 × Sent)
   │
6. Direction ditentukan berdasarkan threshold:
   >= 0.65 → BUY | <= 0.35 → SELL | lainnya → HOLD
   │
7. Jika Direction != HOLD DAN Confidence >= 60%:
   ├── Kirim alert Telegram
   └── Simpan ke TimescaleDB (tabel signals)
   │
8. Candle & indikator selalu disimpan ke TimescaleDB
   dan latest price diupdate di Redis
```

---

## 13. Dependencies & Setup

### `go.mod` utama

```
github.com/gorilla/websocket v1.5.1
github.com/redis/go-redis/v9 v9.5.1
github.com/jackc/pgx/v5 v5.5.5
google.golang.org/grpc v1.63.2
google.golang.org/protobuf v1.33.0
github.com/robfig/cron/v3 v3.0.1
gopkg.in/yaml.v3 v3.0.1
```

### `ml_service/requirements.txt`

```
scikit-learn==1.4.2
numpy==1.26.4
grpcio==1.63.0
grpcio-tools==1.63.0
joblib==1.4.0
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

---

## 14. Checklist Implementasi

### Phase 1 — MVP (Target: 1-2 minggu)

- [ ] Setup project Go dengan struktur direktori
- [ ] Implementasi WebSocket client ke OANDA (dengan reconnect)
- [ ] Implementasi RSI, MACD, Bollinger Bands
- [ ] TechnicalScore aggregator
- [ ] Telegram bot basic
- [ ] Docker Compose untuk TimescaleDB + Redis
- [ ] Simpan candle ke TimescaleDB

### Phase 2 — ML Integration (Target: minggu 3-4)

- [ ] Kumpulkan data historis & buat dataset training
- [ ] Train Random Forest model
- [ ] Setup Python gRPC service
- [ ] Integrasi ML Client di Go
- [ ] Weighted score aggregator dengan ML

### Phase 3 — Sentiment & Polish (Target: bulan 2)

- [ ] Integrasi Gemini API dengan fallback & timeout
- [ ] Redis cache untuk sentiment
- [ ] Backtesting engine (uji di data historis)
- [ ] Grafana dashboard dari TimescaleDB
- [ ] Multi-pair monitoring concurrent
- [ ] Logging & monitoring (structured logs)

---

> **⚠️ Disclaimer:** Bot ini untuk tujuan analisa dan edukasi.  
> Selalu gunakan risk management yang baik dan jangan mengandalkan sinyal otomatis sepenuhnya.  
> Win rate 60-65% sudah sangat baik di dunia forex profesional.
