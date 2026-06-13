---
name: forex-bot-builder
description: >
  Use this skill to build, extend, or debug a Golang forex market analysis bot
  that produces real-time BUY/SELL signals with confidence scores, delivered via
  WhatsApp using Baileys (Node.js). Trigger this skill whenever the user asks to:
  scaffold the project, implement a specific layer (data feed, indicators, ML
  service, Gemini sentiment, signal aggregator, WhatsApp/Baileys alerts,
  TimescaleDB storage), write or fix any file in the forex-bot codebase, or
  run/test the bot. Also trigger for tasks like "add a new pair", "tune the
  weights", "connect Gemini", "set up gRPC", "write the ML model", "setup
  Baileys QR login", or "deploy with Docker". Always read this skill before
  writing any code for this project.
---

# Forex Bot Builder Skill

## Overview

This skill guides an AI agent to build a modular forex market analysis bot.

**Stack:**

- **Go (Golang)** — core engine: data ingestion, indicators, signal aggregation
- **Node.js (Baileys)** — WhatsApp alert service, menerima sinyal dari Go via HTTP POST
- **Python** — ML microservice (Random Forest) via gRPC
- **Gemini API** (gemini-1.5-flash) — news sentiment analysis
- **TimescaleDB** — time-series storage (candles, signals)
- **Redis** — price cache + sentiment cache

**Arsitektur alert:**

```
Go Signal Aggregator
        │
        │ HTTP POST /send-signal (JSON)
        ▼
Node.js Baileys Service (port 3001)
        │
        │ sock.sendMessage(...)
        ▼
WhatsApp (nomor tujuan)
```

**Output pesan WA per sinyal:**

```
🔔 SIGNAL DETECTED

Pair    : EUR/USD
Action  : 🟢 BUY
Price   : 1.08432

📊 Confidence Breakdown:
├ Technical : 78% (RSI oversold + MACD bullish cross)
├ ML Model  : 65%
└ Sentiment : 60% (bullish news)

🎯 Final Score : 71%
⚠️ Risk Level  : MEDIUM
⏰ 14:32:05 WIB
```

---

## Agent Rules

1. **Always check which phase** the user is in before writing code (see Phases below).
2. **Never skip validation** — after writing a Go file, verify it compiles with `go build ./...`.
3. **Never hardcode secrets** — use `config.yaml` + env vars only.
4. **Gemini calls must have a 2-second timeout** — if exceeded, return `SentimentResult{Score: 0.5}` (neutral fallback), never block the signal pipeline.
5. **ML gRPC calls must have a 500ms timeout** — fallback to `0.5` (neutral).
6. **All scores are float64 in range 0.0–1.0** — never return raw RSI/MACD values as scores.
7. **Signal threshold**: BUY if FinalScore >= 0.65, SELL if <= 0.35, else HOLD.
8. **Alert only if** Confidence >= 0.60 AND Direction != HOLD AND RiskLevel != HIGH.
9. **One goroutine per currency pair** for the main processing loop.
10. **Baileys service MUST be running** before Go bot starts — Go will POST to it.
11. **WA session** disimpan di `wa_service/auth_info/` — JANGAN commit ke Git.
12. **QR Code** hanya scan sekali — session persists via `useMultiFileAuthState`.
13. Read `references/architecture.md` for full system diagram.
14. Read `references/baileys.md` for Baileys implementation details.

---

## Project Structure

```
forex-bot/
├── cmd/main.go
├── internal/
│   ├── feed/          websocket.go · rest_poller.go · normalizer.go
│   ├── indicators/    rsi.go · macd.go · bollinger.go · moving_average.go · scorer.go
│   ├── ml/            client.go · proto/ml.proto
│   ├── sentiment/     gemini.go · news_fetcher.go · cache.go
│   ├── strategy/      aggregator.go · signal.go · risk.go
│   ├── alert/         whatsapp.go · formatter.go     ← HTTP client ke Baileys
│   └── storage/       timescale.go · redis.go
├── wa_service/                                        ← Node.js Baileys service
│   ├── index.js                                      ← Express server + Baileys
│   ├── sender.js                                     ← sendMessage wrapper
│   ├── auth_info/                                    ← session WA (di .gitignore!)
│   ├── package.json
│   └── .env
├── ml_service/        main.py · model.py · features.py · train.py · requirements.txt
├── config/config.yaml
├── migrations/001_init.sql
├── docker-compose.yml
├── go.mod
└── go.sum
```

---

## Phases & Checklist

Work through phases in order. Ask the user which phase they are on if unclear.

### Phase 1 — Core MVP

**Goal:** Bot bisa terima data real-time → hitung indikator → kirim alert WA.

**1a. Setup Go core:**

- [ ] `go mod init github.com/<user>/forex-bot`
- [ ] Scaffold semua direktori `internal/`
- [ ] Implement `internal/feed/websocket.go` (lihat: [Feed Implementation](#feed-implementation))
- [ ] Implement `internal/feed/normalizer.go`
- [ ] Implement `internal/indicators/rsi.go`
- [ ] Implement `internal/indicators/macd.go`
- [ ] Implement `internal/indicators/bollinger.go`
- [ ] Implement `internal/indicators/moving_average.go` (EMA/SMA utils)
- [ ] Implement `internal/indicators/scorer.go` (TechnicalScore aggregator)
- [ ] Implement `internal/strategy/signal.go` + `aggregator.go` (technical-only dulu)
- [ ] Implement `internal/alert/whatsapp.go` (HTTP client ke Baileys service)
- [ ] Implement `internal/alert/formatter.go`
- [ ] Setup `docker-compose.yml` (TimescaleDB + Redis)
- [ ] Run `migrations/001_init.sql`
- [ ] Implement `internal/storage/timescale.go` + `redis.go`
- [ ] Wire semua di `cmd/main.go`
- [ ] Test: `go build ./...` harus bersih

**1b. Setup Baileys (Node.js) — kerjakan PARALEL dengan 1a:**

- [ ] `mkdir wa_service && cd wa_service && npm init -y`
- [ ] `npm install @whiskeysockets/baileys express dotenv qrcode-terminal pino`
- [ ] Implement `wa_service/index.js` (Express + Baileys init + QR handler)
- [ ] Implement `wa_service/sender.js`
- [ ] Buat `wa_service/.env` dengan `WA_TARGET_PHONE`
- [ ] Test: `node index.js` → scan QR Code di terminal → tunggu "✅ Connected"
- [ ] Test kirim pesan manual: `curl -X POST localhost:3001/send-signal -d '{"phone":"628xxx","message":"test"}'`

### Phase 2 — ML Integration

**Goal:** Tambah ML confidence layer dari Python gRPC service.

- [ ] Tulis `ml_service/proto/ml.proto`
- [ ] Generate Go & Python stubs: `protoc --go_out=. --go-grpc_out=. ml.proto`
- [ ] Implement `ml_service/model.py` (RandomForest training)
- [ ] Implement `ml_service/features.py` (feature extraction)
- [ ] Implement `ml_service/train.py` (training script, butuh data historis CSV)
- [ ] Implement `ml_service/main.py` (gRPC server, port 50051)
- [ ] Implement `internal/ml/client.go` (Go gRPC client, timeout 500ms)
- [ ] Update `internal/strategy/aggregator.go` untuk include MLScore
- [ ] Test gRPC: jalankan Python service, panggil dari Go

### Phase 3 — Sentiment & Polish

**Goal:** Tambah Gemini sentiment, backtesting, monitoring.

- [ ] Implement `internal/sentiment/news_fetcher.go`
- [ ] Implement `internal/sentiment/cache.go` (Redis, TTL 5 menit)
- [ ] Implement `internal/sentiment/gemini.go` (timeout 2s, fallback neutral)
- [ ] Update `aggregator.go` untuk include SentimentScore
- [ ] Backtesting engine (replay data historis dari TimescaleDB)
- [ ] Grafana dashboard (connect ke TimescaleDB)
- [ ] Structured logging (`log/slog` Go stdlib)

---

## Feed Implementation

### WebSocket Client (`internal/feed/websocket.go`)

Key behaviors the agent MUST implement:

- Auto-reconnect dengan exponential backoff (max 30 detik)
- Heartbeat / ping setiap 30 detik
- Output via buffered `chan OHLCVCandle` (kapasitas 100)
- Goroutine-safe: hanya satu goroutine menulis ke channel

```go
type OHLCVCandle struct {
    Pair      string
    Open, High, Low, Close, Volume, Spread float64
    Timestamp time.Time
}

type WebSocketFeed struct {
    Output chan OHLCVCandle
    // internal fields: url, apiKey, pairs, done chan struct{}
}
```

### Normalizer Rules (`internal/feed/normalizer.go`)

Validasi wajib sebelum candle diteruskan ke pipeline:

- Close, Open, High, Low > 0 dan tidak NaN/Inf
- High >= Low, High >= Open, High >= Close
- Low <= Open, Low <= Close
- Bulatkan ke 5 desimal (pip precision)
- Return `error` jika invalid — jangan panic

---

## Indicator Implementation

### RSI (`internal/indicators/rsi.go`)

- Period default: **14**
- Butuh minimal `period + 1` candle — return `50.0` jika data kurang
- Gunakan Wilder's smoothing (bukan simple average)
- `RSIScore(rsi float64) (direction string, score float64)`:
  - rsi <= 30 → BUY, 0.85
  - rsi <= 40 → BUY, 0.65
  - rsi >= 70 → SELL, 0.85
  - rsi >= 60 → SELL, 0.65
  - else → HOLD, 0.50

### MACD (`internal/indicators/macd.go`)

- Standard: fast=12, slow=26, signal=9
- Butuh EMA dari `internal/indicators/moving_average.go`
- `MACDScore` deteksi crossover: histogram berubah tanda dari candle sebelumnya = 0.80
- Histogram positif/negatif tanpa crossover = 0.60

### Bollinger Bands (`internal/indicators/bollinger.go`)

- Period: 20, StdDev: 2.0
- `BBPosition` = (Close - LowerBand) / (UpperBand - LowerBand) → range 0.0–1.0
- Score: BBPosition <= 0.10 → BUY 0.80 | BBPosition >= 0.90 → SELL 0.80

### TechnicalScore Aggregator (`internal/indicators/scorer.go`)

```
TechnicalScore = (RSIScore × 0.40) + (MACDScore × 0.40) + (BBScore × 0.20)
TechnicalDir   = majority vote dari RSIDir, MACDDir, BBDir
```

Input: `[]OHLCVCandle` (rolling window, max 200 candles)
Output: `IndicatorResult` struct berisi semua nilai dan score

---

## Signal Aggregation

### Weights (`internal/strategy/aggregator.go`)

```
FinalScore = (TechScore × 0.50) + (MLScore × 0.30) + (SentScore × 0.20)
```

**ML score adjustment:** Jika TechnicalDir == "SELL", gunakan `1.0 - MLScore` karena
ML model dilatih untuk memprediksi probabilitas naik — perlu di-invert untuk SELL signal.

**Direction rules:**

```
FinalScore >= 0.65 → "BUY"
FinalScore <= 0.35 → "SELL"
else               → "HOLD"
```

**Risk assessment:**

```
confidence := max(FinalScore, 1-FinalScore)
>= 0.75 → "LOW" risk (signal kuat)
>= 0.60 → "MEDIUM" risk
<  0.60 → "HIGH" risk (jangan alert)
```

### Signal Struct

```go
type Signal struct {
    Pair, Direction, RiskLevel string
    Confidence    float64
    ConfidencePct int
    TechScore, MLScore, SentScore float64
    TechReason, SentReason string
    Timestamp time.Time
}
```

---

## ML Service (Python gRPC)

### Feature Vector (urutan HARUS persis ini)

```
Index 0: RSI value          (0–100, dinormalisasi oleh scaler)
Index 1: MACD histogram     (float, bisa negatif)
Index 2: BB position        (0.0–1.0)
Index 3: Volume change %    (dari candle sebelumnya, e.g. 0.15 = +15%)
Index 4: Spread dalam pips  (e.g. 1.2)
Index 5: Prev candle dir    (1.0=bullish, -1.0=bearish, 0.0=doji)
```

### Label untuk Training

```
1 = harga naik >= 10 pip dalam 3 candle berikutnya
0 = turun atau flat
```

### Model Config

```python
RandomForestClassifier(
    n_estimators=200,
    max_depth=10,
    min_samples_split=20,
    class_weight='balanced',
    random_state=42
)
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

Port gRPC: **50051**

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
bullish  → Score = 0.5 + (confidence × 0.5)
bearish  → Score = 0.5 - (confidence × 0.5)
neutral  → Score = 0.5
```

### Cache Strategy (Redis)

- Key: `sentiment:{sha256(headlines_joined)}`
- TTL: **5 menit**
- Cek cache sebelum panggil API — skip API jika hit
- Fallback ke `Score: 0.5` (neutral) jika: timeout, API error, JSON parse error

---

## Storage Schema

### TimescaleDB Tables

```sql
-- Candles (hypertable, partisi per hari)
CREATE TABLE candles (
    time  TIMESTAMPTZ NOT NULL, pair VARCHAR(10) NOT NULL,
    open  DOUBLE PRECISION, high DOUBLE PRECISION,
    low   DOUBLE PRECISION, close DOUBLE PRECISION,
    volume DOUBLE PRECISION, spread DOUBLE PRECISION,
    PRIMARY KEY (time, pair)
);
SELECT create_hypertable('candles', 'time');

-- Signals
CREATE TABLE signals (
    id SERIAL PRIMARY KEY, time TIMESTAMPTZ DEFAULT NOW(),
    pair VARCHAR(10), direction VARCHAR(4), confidence DOUBLE PRECISION,
    tech_score DOUBLE PRECISION, ml_score DOUBLE PRECISION,
    sent_score DOUBLE PRECISION, risk_level VARCHAR(6),
    price DOUBLE PRECISION, tech_reason TEXT, sent_reason TEXT
);
SELECT create_hypertable('signals', 'time');
```

### Redis Keys

```
latest_price:{PAIR}         → float64, TTL 10s
sentiment:{sha256_hash}     → JSON SentimentResult, TTL 5m
```

---

## Config & Secrets

### `config/config.yaml` Required Fields

```yaml
oanda:
  websocket_url: "wss://stream-fxtrade.oanda.com/v3/..."
  api_key: "${OANDA_API_KEY}"

pairs: [EUR_USD, GBP_USD, USD_JPY, AUD_USD]

gemini:
  api_key: "${GEMINI_API_KEY}"
  model: "gemini-1.5-flash"
  timeout_ms: 2000

ml_service:
  grpc_address: "localhost:50051"
  timeout_ms: 500

signal:
  buy_threshold: 0.65
  sell_threshold: 0.35
  min_confidence_to_alert: 0.60
  weights:
    technical: 0.50
    ml: 0.30
    sentiment: 0.20

whatsapp:
  service_url: "http://localhost:3001" # Baileys Node.js service
  target_phone: "${WA_TARGET_PHONE}" # e.g. "628123456789"
```

**Env vars wajib di `.env`:**
`OANDA_API_KEY`, `OANDA_ACCOUNT_ID`, `GEMINI_API_KEY`,
`WA_TARGET_PHONE`, `DB_PASSWORD`, `REDIS_PASSWORD`

---

## Baileys WhatsApp Service

> Detail lengkap: baca `references/baileys.md`

### Cara Kerja

1. Go bot proses sinyal → HTTP POST ke `http://localhost:3001/send-signal`
2. Node.js Express menerima payload → Baileys `sock.sendMessage()` ke nomor tujuan
3. Session WA disimpan di `auth_info/` — QR hanya scan sekali

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

  sock.ev.on(
    "connection.update",
    async ({ connection, lastDisconnect, qr }) => {
      if (qr) {
        console.log("\n📱 Scan QR ini dengan WhatsApp kamu:\n");
        qrcode.generate(qr, { small: true });
      }
      if (connection === "close") {
        const loggedOut =
          lastDisconnect?.error?.output?.statusCode ===
          DisconnectReason.loggedOut;
        console.log("[WA] Disconnected. LoggedOut:", loggedOut);
        if (!loggedOut) connectToWhatsApp(); // auto-reconnect kecuali logout
      }
      if (connection === "open") console.log("[WA] ✅ WhatsApp Connected!");
    },
  );

  sock.ev.on("creds.update", saveCreds);
}

// Endpoint dipanggil oleh Go bot
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

### `wa_service/.env`

```
WA_TARGET_PHONE=628xxxxxxxxxx
WA_SERVICE_PORT=3001
```

### Rules Baileys WAJIB Diikuti Agent

- **JANGAN** commit `auth_info/` ke Git → tambah ke `.gitignore`
- **Format JID:** `{nomor}@s.whatsapp.net` (personal) | `{id}@g.us` (grup)
- **Nomor:** hapus semua non-digit, prefix kode negara (62 Indonesia)
- **Rate limit:** jangan kirim > 1 pesan per 3 detik ke nomor yang sama
- **Auto-reconnect** kecuali `DisconnectReason.loggedOut`
- Selalu gunakan `useMultiFileAuthState` (bukan `useSingleFileAuthState`)

---

## Go → Baileys HTTP Client (`internal/alert/whatsapp.go`)

```go
package alert

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
    "time"

    "github.com/yourusername/forex-bot/internal/strategy"
)

type WhatsAppClient struct {
    serviceURL  string
    targetPhone string
    httpClient  *http.Client
}

func NewWhatsAppClient(serviceURL, targetPhone string) *WhatsAppClient {
    return &WhatsAppClient{
        serviceURL:  serviceURL,
        targetPhone: targetPhone,
        httpClient:  &http.Client{Timeout: 10 * time.Second},
    }
}

func (w *WhatsAppClient) SendSignal(sig strategy.Signal, currentPrice float64) error {
    if sig.RiskLevel == "HIGH" {
        return nil  // jangan kirim sinyal lemah
    }
    msg := FormatSignalMessage(sig, currentPrice)
    body, _ := json.Marshal(map[string]string{
        "phone":   w.targetPhone,
        "message": msg,
    })
    resp, err := w.httpClient.Post(w.serviceURL+"/send-signal", "application/json", bytes.NewReader(body))
    if err != nil {
        return fmt.Errorf("WA service unreachable: %w", err)
    }
    defer resp.Body.Close()
    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("WA service error: HTTP %d", resp.StatusCode)
    }
    return nil
}
```

---

## WhatsApp Message Format

```
🔔 SIGNAL DETECTED

Pair    : EUR/USD
Action  : 🟢 BUY
Price   : 1.08432

📊 Confidence Breakdown:
├ Technical : 78% (RSI oversold + MACD bullish cross)
├ ML Model  : 65%
└ Sentiment : 60% (bullish USD news)

🎯 Final Score : 71%
⚠️ Risk Level  : MEDIUM
⏰ 14:32:05 WIB

_Selalu gunakan money management yang baik!_
```

Format WA: bold=`*teks*` | italic=`_teks_` | monospace=`` `teks` ``
Emoji: BUY=🟢 SELL=🔴 HOLD=⚪ | LOW=✅ MEDIUM=⚠️ HIGH=🚨

---

## Startup Order (WAJIB URUT)

```bash
# 1. Infrastructure
docker-compose up -d

# 2. Baileys WA Service (HARUS PERTAMA sebelum Go bot)
cd wa_service && node index.js
# → Scan QR Code → tunggu "✅ WhatsApp Connected!"

# 3. ML Service
python ml_service/main.py

# 4. Go Bot (terakhir)
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

```json
{
  "dependencies": {
    "@whiskeysockets/baileys": "^6.7.0",
    "express": "^4.18.2",
    "dotenv": "^16.3.1",
    "qrcode-terminal": "^0.12.0",
    "pino": "^8.15.0"
  }
}
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
| Pesan WA tidak terkirim          | Format JID salah / rate limit    | Cek nomor format `628xxx`, jangan spam <3 detik   |
| `loggedOut` tidak reconnect      | WA logout paksa dari HP          | Normal — scan QR ulang, tidak bisa auto-reconnect |
| `context deadline exceeded` gRPC | ML service lambat                | Pastikan timeout 500ms, fallback ke 0.5           |
| `json: cannot unmarshal` Gemini  | Model return markdown            | Strip backtick/markdown sebelum Unmarshal         |
| `hypertable already exists`      | Migration 2x                     | Tambah `IF NOT EXISTS` di migration               |
| `RSI always 50.0`                | Kurang dari 15 candle            | Normal — tunggu rolling window terisi             |

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

- `references/architecture.md` — diagram arsitektur lengkap + alur data end-to-end
- `references/oanda-api.md` — format WebSocket message OANDA dan auth header
- `references/backtesting.md` — panduan backtesting dengan data historis TimescaleDB

> ⚠️ **Disclaimer:** Bot ini untuk analisa dan edukasi.
> Win rate 60–65% sudah sangat baik secara profesional.
> Selalu gunakan risk management dan jangan rely 100% pada sinyal otomatis.
