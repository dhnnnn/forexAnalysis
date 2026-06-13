---
name: forex-bot-builder
description: >
  Use this skill to build, extend, or debug a Golang forex multi-agent analysis
  bot that produces real-time BUY/SELL signals with confidence scores, delivered
  via WhatsApp using Baileys (Node.js). This is a Multi-Agent System: setiap
  komponen adalah agent otonom dengan kontrak input/output sendiri.
  Trigger this skill whenever the user asks to: scaffold the project, implement
  a specific agent (MarketData, Technical, Fundamental, Risk, Decision,
  WhatsApp), implement a support layer (indicators, feed, ML service, Gemini
  sentiment, TimescaleDB storage), write or fix any file in the forex-bot
  codebase, or run/test the bot. Also trigger for tasks like "add a new pair",
  "tune the weights", "connect Gemini", "set up gRPC", "write the ML model",
  "setup Baileys QR login", or "deploy with Docker".
  Always read this skill before writing any code for this project.
---

# Forex Multi-Agent Bot — Skill Guide (Golang)

## Overview

Bot ini adalah **Multi-Agent System** berbasis Golang.

**Paradigma:** Setiap komponen adalah **agent otonom** dengan kontrak
input/output sendiri. Bukan pipeline fungsi biasa.

**Stack:**

- **Go (Golang)** — core engine: semua agent, indikator, orkestrasi
- **Node.js (Baileys)** — WhatsApp alert service, menerima sinyal dari Go via HTTP POST
- **Python** — ML microservice (Random Forest) via gRPC *(opsional, Phase 2)*
- **Gemini API** (gemini-1.5-flash) — news sentiment analysis di FundamentalAgent
- **TimescaleDB** — time-series storage (candles, signals)
- **Redis** — price cache + sentiment cache

---

## Arsitektur Multi-Agent (MVP)

```
┌──────────────────────────────────────┐
│         EXTERNAL DATA SOURCES        │
│  OANDA WS  │  Twelve Data  │  AlphaV │
└──────────────────┬───────────────────┘
                   │
                   ▼
        ┌─────────────────────┐
        │  Agent 1            │
        │  MarketDataAgent    │  ← Fetch OHLCV, rolling buffer 200 candle
        └──────────┬──────────┘
                   │
          ┌────────┴────────┐
          │   concurrent    │   ← goroutine paralel
          ▼                 ▼
┌──────────────────┐  ┌──────────────────────┐
│  Agent 2         │  │  Agent 3             │
│  TechnicalAgent  │  │  FundamentalAgent    │
│                  │  │                      │
│  RSI, MACD, EMA  │  │  News + Gemini NLP   │
│  → BUY/SELL/HOLD │  │  → bullish/bearish   │
└────────┬─────────┘  └──────────┬───────────┘
         └──────────┬────────────┘
                    │
                    ▼
         ┌──────────────────────┐
         │  Agent 4             │
         │  RiskAgent           │  ← Lot, SL, TP calculator
         └──────────┬───────────┘
                    │
                    ▼
         ┌──────────────────────┐
         │  Agent 5             │
         │  DecisionAgent       │  ← "Otak Utama" — sinyal final
         └──────────┬───────────┘
                    │
                    ▼
         ┌──────────────────────┐
         │  Agent 6             │
         │  WhatsAppAgent       │  ← Hanya kurir
         │                      │
         │  Go → HTTP POST      │
         │  → Node.js Baileys   │
         │  → WhatsApp          │
         └──────────────────────┘
```

**Kenapa dipisah Go + Node.js?**

```
AI Engine   = Go      → performa, concurrency, goroutine
WhatsApp    = Node.js → Baileys hanya tersedia di Node.js ekosistem
```

**Output pesan WA per sinyal:**

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

_Selalu gunakan money management yang baik!_
```

---

## Agent Rules

1. **Selalu cek agent mana yang sedang diimplementasi** sebelum menulis kode.
2. **Urutan eksekusi agent wajib dijaga:**
   `MarketData → (Technical ∥ Fundamental) → Risk → Decision → WhatsApp`
3. **Never skip validation** — setelah menulis Go file, verifikasi dengan `go build ./...`.
4. **Never hardcode secrets** — gunakan `config.yaml` + env vars.
5. **Gemini calls wajib timeout 2 detik** — fallback ke `SentimentResult{Score: 0.5}` (neutral).
6. **ML gRPC calls wajib timeout 500ms** — fallback ke `0.5` (neutral).
7. **Semua score adalah float64 range 0.0–1.0** — jangan return raw RSI/MACD sebagai score.
8. **Signal threshold:** BUY jika FinalScore >= 0.65, SELL jika <= 0.35, else HOLD.
9. **Alert hanya dikirim jika:** Confidence >= 0.60 AND Direction != HOLD AND RiskLevel != HIGH.
10. **Satu goroutine per currency pair** untuk main processing loop.
11. **Baileys service HARUS running** sebelum Go bot start — Go akan POST ke sana.
12. **WA session** disimpan di `wa_service/auth_info/` — JANGAN commit ke Git.
13. **QR Code** hanya scan sekali — session persists via `useMultiFileAuthState`.
14. Baca `docs/architecture.md` untuk full system diagram dan implementasi detail setiap agent.

---

## Project Structure

```
forex-agent/
│
├── cmd/
│   └── main.go                     # Entry point — init & orkestrasi semua agent
│
├── internal/
│   │
│   ├── agents/                     # ← CORE: semua agent di sini
│   │   ├── agent.go                # Agent interface + semua Output struct
│   │   ├── market_data_agent.go    # Agent 1: fetch OHLCV, rolling buffer
│   │   ├── technical_agent.go      # Agent 2: RSI, MACD, EMA, BB → BUY/SELL/HOLD
│   │   ├── fundamental_agent.go    # Agent 3: berita + Gemini NLP → sentiment
│   │   ├── risk_agent.go           # Agent 4: lot size, SL, TP calculator
│   │   ├── decision_agent.go       # Agent 5: otak utama → sinyal final
│   │   └── whatsapp_agent.go       # Agent 6: kurir → HTTP POST ke Baileys
│   │
│   ├── indicators/                 # Kalkulasi indikator teknikal (dipanggil Agent 2)
│   │   ├── rsi.go                  # RSI(14)
│   │   ├── macd.go                 # MACD(12,26,9)
│   │   ├── bollinger.go            # Bollinger Bands(20,2)
│   │   ├── moving_average.go       # EMA/SMA utilities
│   │   └── scorer.go               # TechnicalScore aggregator
│   │
│   ├── feed/                       # Data ingestion (dipanggil Agent 1)
│   │   ├── websocket.go            # WebSocket client (OANDA)
│   │   ├── rest_poller.go          # REST polling (Twelve Data / Alpha Vantage)
│   │   └── normalizer.go           # Validasi & normalisasi OHLCV
│   │
│   ├── ml/                         # ML gRPC client (dipanggil Agent 5, opsional)
│   │   ├── client.go               # gRPC client → Python ML service
│   │   └── proto/ml.proto          # gRPC contract
│   │
│   ├── sentiment/                  # Gemini + news (dipanggil Agent 3)
│   │   ├── gemini.go               # Gemini API client
│   │   ├── news_fetcher.go         # RSS / news API fetcher
│   │   └── cache.go                # Redis cache TTL 5 menit
│   │
│   └── storage/                    # Persistence layer
│       ├── timescale.go            # TimescaleDB queries
│       └── redis.go                # Redis client & helpers
│
├── wa_service/                     # Node.js + Baileys (service TERPISAH)
│   ├── index.js                    # Express server + Baileys init + QR handler
│   ├── sender.js                   # sendMessage wrapper
│   ├── auth_info/                  # Session WA (di .gitignore!)
│   ├── package.json
│   └── .env
│
├── ml_service/                     # Python gRPC microservice (Phase 2)
│   ├── main.py                     # gRPC server entry point (port 50051)
│   ├── model.py                    # RandomForest training & predict
│   ├── features.py                 # Feature engineering
│   ├── train.py                    # Script training model
│   └── requirements.txt
│
├── config/config.yaml              # Semua konfigurasi
├── migrations/001_init.sql         # TimescaleDB schema
├── docker-compose.yml              # TimescaleDB + Redis
├── go.mod
└── go.sum
```

---

## Agent Interface Contract (Ringkasan)

```go
// Semua agent wajib implement interface ini
type Agent interface {
    Name() string
    Run(ctx context.Context, input AgentInput) AgentOutput
}

// AgentInput: container data yang mengalir antar agent
type AgentInput struct {
    Pair           string
    Candles        []Candle           // dari MarketDataAgent
    Technical      *TechnicalOutput   // dari TechnicalAgent
    Fundamental    *FundamentalOutput // dari FundamentalAgent
    Risk           *RiskOutput        // dari RiskAgent
    Decision       *DecisionOutput    // dari DecisionAgent
    AccountBalance float64
    RiskPercent    float64
}
```

Detail lengkap semua struct ada di `internal/agents/agent.go` — lihat `docs/architecture.md` Section 2.

---

## Phases & Checklist

Kerjakan per fase secara berurutan. Tanya user sedang di fase mana jika tidak jelas.

### Phase 1 — Core MVP (Minggu 1–4)

**Target akhir fase:** sinyal teknikal terkirim ke WhatsApp.

**Minggu 1 — Fondasi Go:**

- [ ] `go mod init github.com/<user>/forex-agent`
- [ ] Scaffold semua direktori `internal/`
- [ ] Implement `internal/agents/agent.go` (interface + semua output struct)
- [ ] Implement `MarketDataAgent` dengan mock data dulu

**Minggu 2 — Data Real:**

- [ ] Implement `internal/feed/websocket.go` (OANDA WebSocket)
- [ ] Implement `internal/feed/normalizer.go` (validasi OHLCV)
- [ ] Koneksi ke Twelve Data REST sebagai fallback
- [ ] `MarketDataAgent` pakai feed nyata → buffer terisi candle EURUSD

**Minggu 3 — Technical Agent:**

- [ ] Implement `internal/indicators/rsi.go` (RSI 14, Wilder's smoothing)
- [ ] Implement `internal/indicators/macd.go` (fast=12, slow=26, signal=9)
- [ ] Implement `internal/indicators/bollinger.go` (period=20, std=2)
- [ ] Implement `internal/indicators/moving_average.go` (EMA/SMA utils)
- [ ] Implement `internal/indicators/scorer.go` (TechnicalScore aggregator)
- [ ] Implement `TechnicalAgent` → output BUY/SELL/HOLD + confidence
- [ ] Unit test tiap indikator

**Minggu 4 — WhatsApp Integration:**

- [ ] `mkdir wa_service && cd wa_service && npm init -y`
- [ ] `npm install @whiskeysockets/baileys express dotenv qrcode-terminal pino`
- [ ] Implement `wa_service/index.js` (Express + Baileys init + QR handler)
- [ ] Implement `wa_service/sender.js`
- [ ] Buat `wa_service/.env` dengan `WA_TARGET_PHONE`
- [ ] Test: `node index.js` → scan QR → tunggu "✅ Connected"
- [ ] Implement `WhatsAppAgent` (Go HTTP client ke Baileys)
- [ ] Implement `DecisionAgent` versi sederhana (technical-only dulu)
- [ ] Wire semua di `cmd/main.go`
- [ ] Test end-to-end: mock signal → pesan WA terkirim ✅

```
[Minggu 4 Target]
TechnicalAgent → DecisionAgent (technical-only) → WhatsAppAgent → WA
```

### Phase 2 — Risk + Fundamental (Minggu 5–6)

**Minggu 5 — Risk Agent:**

- [ ] Implement `RiskAgent` (lot, SL, TP berdasarkan balance & risk %)
- [ ] Integrasi RiskAgent ke DecisionAgent
- [ ] Update format pesan WA dengan SL/TP/lot
- [ ] Setup `docker-compose.yml` (TimescaleDB + Redis)
- [ ] Implement `internal/storage/timescale.go` + `redis.go`
- [ ] Jalankan `migrations/001_init.sql`

**Minggu 6 — Fundamental Agent:**

- [ ] Implement `internal/sentiment/news_fetcher.go`
- [ ] Implement `internal/sentiment/cache.go` (Redis, TTL 5 menit)
- [ ] Implement `internal/sentiment/gemini.go` (timeout 2s, fallback neutral)
- [ ] Implement `FundamentalAgent` (concurrent dengan TechnicalAgent)
- [ ] Update `DecisionAgent` dengan logika fundamental konfirmasi

### Phase 3 — ML + Research (Minggu 7–8)

**Minggu 7 — ML Boost (opsional):**

- [ ] Tulis `ml_service/proto/ml.proto`
- [ ] Implement `ml_service/model.py` (RandomForest training)
- [ ] Implement `ml_service/main.py` (gRPC server, port 50051)
- [ ] Implement `internal/ml/client.go` (timeout 500ms)
- [ ] Update `DecisionAgent` dengan ML score boost

**Minggu 8 — Backtesting & Evaluasi:**

- [ ] Backtesting engine (replay data historis dari TimescaleDB)
- [ ] Bandingkan: sinyal rule-based vs dengan ML boost
- [ ] Grafana dashboard (connect ke TimescaleDB)
- [ ] Structured logging (`log/slog` Go stdlib)
- [ ] Kumpulkan data untuk paper/penelitian

---

## Decision Agent Logic

```go
// Aturan dasar (Phase 1 — rule-based MVP)
if technical == "BUY" && fundamental == "bullish"  → BUY
if technical == "SELL" && fundamental == "bearish" → SELL
if fundamental == "neutral"                        → ikuti technical
else                                               → HOLD (konflik)

// Confidence
confidence = (tech_confidence × 0.60) + (fund_confidence × 0.40)

// Phase 3: Optional ML boost
if ml_score > 0:
    confidence = (confidence × 0.80) + (ml_score × 0.20)
```

**Risk Level:**

```
certainty := max(confidence, 1-confidence)
>= 0.75 → "LOW"    (sinyal kuat → kirim alert)
>= 0.60 → "MEDIUM" (sinyal cukup → kirim alert)
<  0.60 → "HIGH"   (sinyal lemah → JANGAN kirim)
```

---

## Indicator Scoring Rules

```
TechnicalScore = (RSIScore × 0.40) + (MACDScore × 0.40) + (BBScore × 0.20)

RSI(14):
  <= 30 → BUY,  0.85  (oversold kuat)
  <= 40 → BUY,  0.65  (oversold moderat)
  >= 70 → SELL, 0.85  (overbought kuat)
  >= 60 → SELL, 0.65  (overbought moderat)
  else  → HOLD, 0.50

MACD(12,26,9):
  crossover bullish → BUY,  0.80
  crossover bearish → SELL, 0.80
  histogram > 0    → BUY,  0.60
  histogram < 0    → SELL, 0.60

Bollinger(20,2):
  position <= 0.10 → BUY,  0.80  (harga di lower band)
  position >= 0.90 → SELL, 0.80  (harga di upper band)
  else             → HOLD, 0.50
```

---

## Risk Agent Formula

```
PipValue  = 10 USD per pip (1 lot standar, pair major)
LotSize   = (Balance × RiskPercent / 100) / (SL_pips × PipValue)
            → dibulatkan ke 2 desimal

StopLoss  = Entry - (SL_pips × 0.0001)   [BUY]
          = Entry + (SL_pips × 0.0001)   [SELL]
TakeProfit = Entry + (TP_pips × 0.0001)  [BUY]
           = Entry - (TP_pips × 0.0001)  [SELL]

Default: SL = 20 pip, TP = 40 pip (RR 1:2)
```

**Contoh (Balance=$1000, Risk=1%, Entry=1.08450, BUY):**

```json
{ "lot": 0.05, "sl": 1.08250, "tp": 1.08850 }
```

---

## WhatsApp Message Format

```
🚀 *FOREX ALERT*

*Pair*   : EURUSD
*Signal* : 🟢 BUY

*Entry*  : 1.08450
*SL*     : 1.08250 (20 pip)
*TP*     : 1.08850 (40 pip)

📊 *Analysis:*
├ Technical   : BUY (80%)
│  _RSI oversold + MACD bullish cross_
└ Fundamental : BULLISH (75%)
   _Fed rate hike expectations boost USD_

🎯 *Confidence* : 88%
💰 *Lot Size*   : 0.05
⚠️ *Risk*       : 1.0% | ✅ LOW
⏰ 20:00:05 WIB

_Selalu gunakan money management yang baik!_
```

Format WA: bold=`*teks*` | italic=`_teks_` | monospace=`` `teks` ``
Emoji: BUY=🟢 SELL=🔴 HOLD=⚪ | LOW=✅ MEDIUM=⚠️ HIGH=🚨

---

## Baileys WhatsApp Service

### `wa_service/index.js` — Wajib Implement Ini

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
      if (!loggedOut) connectToWhatsApp(); // auto-reconnect kecuali logout
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

### Rules Baileys WAJIB Diikuti

- **JANGAN** commit `auth_info/` ke Git → tambah ke `.gitignore`
- **Format JID:** `{nomor}@s.whatsapp.net` (personal) | `{id}@g.us` (grup)
- **Nomor:** hapus semua non-digit, prefix kode negara (62 Indonesia)
- **Rate limit:** jangan kirim > 1 pesan per 3 detik ke nomor yang sama
- **Auto-reconnect** kecuali `DisconnectReason.loggedOut`
- Selalu gunakan `useMultiFileAuthState` (bukan `useSingleFileAuthState`)

---

## ML Service (Python gRPC) — Phase 2

### Feature Vector (urutan HARUS persis ini)

```
Index 0: RSI value          (0–100)
Index 1: MACD histogram     (float, bisa negatif)
Index 2: BB position        (0.0–1.0)
Index 3: Volume change %    (e.g. 0.15 = +15%)
Index 4: Spread dalam pips  (e.g. 1.2)
Index 5: Prev candle dir    (1.0=bullish, -1.0=bearish, 0.0=doji)
```

### Proto Definition

```protobuf
service MLService {
  rpc Predict(PredictRequest) returns (PredictResponse);
}
message PredictRequest {
  double rsi = 1; double macd_histogram = 2; double bb_position = 3;
  double volume_change = 4; double spread = 5; double prev_candle_direction = 6;
}
message PredictResponse { double score = 1; }
```

Port gRPC: **50051** | Timeout client: **500ms** | Fallback: `0.5`

---

## Sentiment (Gemini)

### Prompt Template

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

**Cache Strategy (Redis):**

- Key: `sentiment:{sha256(headlines_joined)}`
- TTL: **5 menit**
- Fallback ke `Score: 0.5` jika: timeout, API error, JSON parse error

---

## Storage Schema

### TimescaleDB Tables

```sql
-- Candles
CREATE TABLE IF NOT EXISTS candles (
    time      TIMESTAMPTZ NOT NULL,
    pair      VARCHAR(10) NOT NULL,
    timeframe VARCHAR(4)  NOT NULL DEFAULT '1h',
    open      DOUBLE PRECISION, high DOUBLE PRECISION,
    low       DOUBLE PRECISION, close DOUBLE PRECISION,
    volume    DOUBLE PRECISION, spread DOUBLE PRECISION,
    PRIMARY KEY (time, pair, timeframe)
);
SELECT create_hypertable('candles', 'time', if_not_exists => TRUE);

-- Signals
CREATE TABLE IF NOT EXISTS signals (
    id             SERIAL PRIMARY KEY,
    time           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    pair           VARCHAR(10), direction VARCHAR(4),
    confidence     DOUBLE PRECISION, tech_score DOUBLE PRECISION,
    fund_sentiment VARCHAR(10), fund_score DOUBLE PRECISION,
    ml_score       DOUBLE PRECISION, risk_level VARCHAR(6),
    lot_size       DOUBLE PRECISION, entry_price DOUBLE PRECISION,
    stop_loss      DOUBLE PRECISION, take_profit DOUBLE PRECISION,
    sl_pips        DOUBLE PRECISION, tp_pips DOUBLE PRECISION,
    tech_reason    TEXT, fund_reason TEXT
);
SELECT create_hypertable('signals', 'time', if_not_exists => TRUE);
```

### Redis Keys

```
latest_price:{PAIR}          → float64, TTL 10s
latest_candle:{PAIR}:{TF}    → JSON Candle, TTL 10s
sentiment:{sha256_hash}      → JSON FundamentalOutput, TTL 5m
agent_status:{AGENT_NAME}    → "ok"|"error", TTL 60s
```

---

## Config & Secrets

### `config/config.yaml` Required Fields

```yaml
oanda:
  websocket_url: "wss://stream-fxtrade.oanda.com/v3/..."
  api_key: "${OANDA_API_KEY}"

pairs: [EUR_USD, GBP_USD, USD_JPY, AUD_USD]

account:
  balance: 1000.0
  risk_percent: 1.0
  default_sl_pips: 20.0
  default_tp_pips: 40.0

gemini:
  api_key: "${GEMINI_API_KEY}"
  model: "gemini-1.5-flash"
  timeout_ms: 2000

ml_service:
  enabled: false   # aktifkan di Phase 3
  grpc_address: "localhost:50051"
  timeout_ms: 500

signal:
  buy_threshold: 0.65
  sell_threshold: 0.35
  min_confidence_to_alert: 0.60
  weights:
    technical: 0.60
    fundamental: 0.40

whatsapp:
  service_url: "http://localhost:3001"
  target_phone: "${WA_TARGET_PHONE}"
  rate_limit_seconds: 180
```

**Env vars wajib di `.env`:**
`OANDA_API_KEY`, `OANDA_ACCOUNT_ID`, `GEMINI_API_KEY`,
`WA_TARGET_PHONE`, `DB_PASSWORD`, `REDIS_PASSWORD`

---

## Startup Order (WAJIB URUT)

```bash
# 1. Infrastructure
docker-compose up -d

# 2. Baileys WA Service (HARUS PERTAMA sebelum Go bot)
cd wa_service && node index.js
# → Scan QR Code → tunggu "✅ WhatsApp Connected!"

# 3. ML Service (Phase 3, opsional)
python ml_service/main.py

# 4. Go Bot (paling akhir)
go run cmd/main.go
```

---

## Go Dependencies

```
go get github.com/gorilla/websocket@v1.5.1
go get github.com/redis/go-redis/v9@v9.5.1
go get github.com/jackc/pgx/v5@v5.5.5
go get google.golang.org/grpc@v1.63.2
go get google.golang.org/protobuf@v1.33.0
go get github.com/robfig/cron/v3@v3.0.1
go get gopkg.in/yaml.v3@v3.0.1
```

## Node.js Dependencies (`wa_service/`)

```bash
npm install @whiskeysockets/baileys express dotenv qrcode-terminal pino
```

## Python Dependencies (`ml_service/requirements.txt`)

```
scikit-learn==1.4.2
numpy==1.26.4
grpcio==1.63.0
grpcio-tools==1.63.0
joblib==1.4.0
```

---

## Common Errors & Fixes

| Error                            | Penyebab                         | Fix                                               |
| -------------------------------- | -------------------------------- | ------------------------------------------------- |
| `WA service unreachable`         | Baileys service belum jalan      | Jalankan `node wa_service/index.js` dulu          |
| HTTP `503` dari `/send-signal`   | WA belum connect (QR belum scan) | Tunggu "✅ Connected" di terminal                 |
| `Connection Failure` Baileys     | Session expired atau logout      | Hapus `auth_info/`, restart, scan QR ulang        |
| `loggedOut` tidak reconnect      | WA logout paksa dari HP          | Normal — scan QR ulang                            |
| `context deadline exceeded` gRPC | ML service lambat                | Pastikan timeout 500ms, fallback ke 0.5           |
| `json: cannot unmarshal` Gemini  | Model return markdown            | Strip backtick/markdown sebelum Unmarshal         |
| `insufficient candle data`       | Buffer belum terisi              | Normal — tunggu rolling window terisi (min 26)    |
| `TechnicalAgent output required` | Urutan agent salah               | Pastikan Technical selesai sebelum Risk/Decision  |
| `RSI always 50.0`                | Kurang dari 15 candle            | Normal — tunggu buffer terisi                     |
| `hypertable already exists`      | Migration 2x                     | Tambah `IF NOT EXISTS` di migration               |

## .gitignore Wajib

```
wa_service/auth_info/
wa_service/node_modules/
wa_service/.env
.env
*.env
```

---

## Reference Files

- `docs/architecture.md` — full system diagram, implementasi kode lengkap setiap agent,
  alur data end-to-end, storage schema, docker setup

> ⚠️ **Disclaimer:** Bot ini untuk analisa dan edukasi.
> Win rate 60–65% sudah sangat baik secara profesional.
> Selalu gunakan risk management dan jangan rely 100% pada sinyal otomatis.
