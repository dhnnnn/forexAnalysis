# 🤖 Forex Multi-Agent Analysis Bot

🌐 **Language / Bahasa / 言語:** [English](../../README.md) | [Bahasa Indonesia](README_ID.md) | [日本語](README_JA.md)

Sistem real-time untuk menghasilkan sinyal trading forex menggunakan arsitektur multi-agent berbasis Go. Setiap agent beroperasi secara otonom dengan kontrak input/output sendiri, dan secara kolektif menghasilkan sinyal trading yang dikirim via WhatsApp.

## Arsitektur

```
┌─────────────────────────────────────────────────────────────────┐
│                     SUMBER DATA EKSTERNAL                        │
│   OANDA WebSocket  │  Twelve Data REST  │  Alpha Vantage REST   │
└──────────┬─────────────────┬─────────────────────┬──────────────┘
           └─────────────────┴─────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│  Agent 1: MarketDataAgent                                       │
│  Ambil & normalisasi candle OHLCV → rolling buffer (200/pair)   │
└──────────────────────────┬──────────────────────────────────────┘
                           │
              ┌────────────┴────────────┐
              ▼ (concurrent)            ▼
┌──────────────────────┐  ┌──────────────────────────────────────┐
│  Agent 2: Technical  │  │  Agent 3: Fundamental                │
│  RSI, MACD, EMA,     │  │  Sentimen berita via Gemini API      │
│  Bollinger Bands     │  │  + Groq fallback, Redis cache        │
└──────────┬───────────┘  └───────────────┬──────────────────────┘
           └──────────────┬───────────────┘
                          ▼
┌─────────────────────────────────────────────────────────────────┐
│  Agent 4: RiskAgent                                             │
│  Perhitungan posisi, SL/TP                                      │
└──────────────────────────┬──────────────────────────────────────┘
                           ▼
┌─────────────────────────────────────────────────────────────────┐
│  Agent 5: DecisionAgent — "Otak Utama"                          │
│  Weighted scoring → BUY/SELL/HOLD + confidence & risk level     │
└──────────────────────────┬──────────────────────────────────────┘
                           ▼
┌─────────────────────────────────────────────────────────────────┐
│  Agent 6: WhatsAppAgent                                         │
│  Pengiriman alert (rate-limited, confidence-gated)              │
└─────────────────────────────────────────────────────────────────┘
```

## Fitur

- **Pipeline Multi-Agent** — 6 agent otonom dengan kontrak yang jelas
- **Eksekusi Concurrent** — TechnicalAgent + FundamentalAgent jalan paralel per pair; beberapa pair diproses bersamaan
- **Sentimen Berbasis AI** — Gemini 2.0 Flash untuk analisis berita dengan Groq (Llama 3.3 70B) sebagai fallback
- **Indikator Teknikal** — RSI(14), MACD(12,26,9), EMA(50,200), Bollinger Bands(20,2)
- **Manajemen Risiko** — Position sizing yang bisa dikonfigurasi dengan SL/TP adjustable
- **Integrasi WhatsApp** — Bidireksional: terima perintah + kirim alert trading
- **Chatbot Interaktif** — Tanya jawab forex pakai bahasa natural via Gemini AI
- **Penyimpanan Persisten** — TimescaleDB untuk histori candle dan tracking sinyal
- **Graceful Degradation** — Data hilang atau agent gagal tidak menghentikan pipeline
- **Docker Ready** — Full stack dalam satu `docker-compose up`

## Tech Stack

| Komponen | Teknologi |
|----------|-----------|
| Engine Utama | Go 1.25 |
| Service WhatsApp | Node.js + whatsapp-web.js |
| AI/NLP | Gemini API (primer) + Groq API (fallback) |
| Database Time-Series | TimescaleDB (PostgreSQL) |
| Cache | Redis 7 |
| Containerisasi | Docker Compose |

## Struktur Project

```
├── cmd/
│   └── main.go                 # Entry point, orkestrasi pipeline
├── internal/
│   ├── agents/
│   │   ├── agent.go            # Interface Agent, tipe bersama
│   │   ├── market_data_agent.go
│   │   ├── technical_agent.go
│   │   ├── fundamental_agent.go
│   │   ├── risk_agent.go
│   │   ├── decision_agent.go
│   │   └── whatsapp_agent.go
│   ├── chatbot/
│   │   ├── handler.go          # Routing perintah chat
│   │   └── gemini_chat.go      # Percakapan AI
│   ├── config/
│   │   └── loader.go           # Config YAML + ekspansi env var
│   ├── feed/
│   │   ├── websocket.go        # OANDA WebSocket feed
│   │   ├── rest_poller.go      # Fallback REST API
│   │   └── normalizer.go       # Normalisasi candle
│   ├── indicators/
│   │   ├── rsi.go
│   │   ├── macd.go
│   │   ├── moving_average.go
│   │   ├── bollinger.go
│   │   └── scorer.go
│   ├── sentiment/
│   │   ├── gemini.go           # Analisis sentimen Gemini
│   │   ├── news_fetcher.go     # Agregasi berita multi-sumber
│   │   ├── cache.go            # Cache sentimen Redis
│   │   └── interfaces.go
│   └── storage/
│       ├── postgres.go         # Persistensi TimescaleDB
│       └── batch.go            # Helper batch insert
├── whatsapp-service/
│   ├── index.js                # Bridge WhatsApp Node.js
│   ├── Dockerfile
│   └── package.json
├── config/
│   └── config.yaml             # Semua konfigurasi
├── migrations/
│   └── 001_init.sql            # Skema TimescaleDB
├── docker-compose.yml
├── Dockerfile
└── .env.example
```

## Mulai Cepat

### Prasyarat

- Docker & Docker Compose
- Akun WhatsApp untuk scan QR code

### 1. Clone & Konfigurasi

```bash
git clone https://github.com/dhnnnn/forexAnalysis.git
cd forexAnalysis
cp .env.example .env
```

Edit `.env` dengan API key kamu:

```env
# Sumber Data
OANDA_API_KEY=oanda_key_kamu
OANDA_ACCOUNT_ID=account_id_kamu
TWELVE_DATA_KEY=twelve_data_key_kamu
ALPHA_VANTAGE_KEY=alpha_vantage_key_kamu

# AI
GEMINI_API_KEY=gemini_key_kamu
GROQ_API_KEY=groq_key_kamu              # opsional, fallback

# WhatsApp
WA_TARGET_PHONE=628xxxxxxxxxx           # nomor telepon kamu

# Database
DB_PASSWORD=password_db_kamu
REDIS_PASSWORD=password_redis_kamu
```

### 2. Jalankan Semuanya

```bash
docker-compose up --build
```

Ini akan menjalankan:
- **TimescaleDB** di port 5432
- **Redis** di port 6379
- **Go Agent** di port 8080
- **WhatsApp Service** di port 3001

### 3. Hubungkan WhatsApp

Perhatikan output konsol untuk QR code. Scan dengan:
**WhatsApp → Settings → Linked Devices → Link a Device**

### 4. Interaksi

Kirim pesan ke bot via WhatsApp:

| Perintah | Deskripsi |
|----------|-----------|
| `/help` | Tampilkan semua perintah |
| `/status` | Lihat status & pengaturan bot |
| `/set balance 500` | Set balance trading |
| `/set risk 2` | Set risk % per trade |
| `/risk` | Kalkulator manajemen risiko |
| `/analyze` | Force scan analisis |
| *(teks bebas)* | Tanya jawab forex via AI |

## Konfigurasi

Semua pengaturan ada di `config/config.yaml`. Bagian penting:

```yaml
# Pair yang dimonitor
pairs:
  - "EUR_USD"
  - "GBP_USD"

# Pengaturan akun
account:
  balance: 1000.0
  risk_percent: 1.0
  default_sl_pips: 20.0
  default_tp_pips: 40.0

# Threshold sinyal (DecisionAgent)
signal:
  buy_threshold: 0.60
  sell_threshold: 0.35
  weights:
    technical: 0.60
    fundamental: 0.40
```

## Perilaku Pipeline

- Berjalan setiap **5 menit** per pair
- Membutuhkan minimal **26 candle** sebelum analisis dimulai
- Alert hanya dikirim jika confidence ≥ 55%
- Rate-limited **1 alert per pair per 3 menit**
- Jika TechnicalAgent atau FundamentalAgent gagal, DecisionAgent tetap jalan menggunakan default yang aman

## Contoh Output Sinyal

```
🟢 BUY EUR_USD

📊 Confidence: 72% | Risk: MEDIUM

💰 Entry: 1.08450
🛑 SL: 1.08250
🎯 TP: 1.08850
📐 Lot: 0.05

📈 Tech: BUY (80%)
📰 Fund: bullish (65%)

⏰ 14:30:05 WIB
```

## Development

### Jalankan Lokal (tanpa Docker)

```bash
# Jalankan dependensi
docker-compose up timescaledb redis -d

# Jalankan Go agent
go run ./cmd/main.go

# Jalankan WhatsApp service (terminal terpisah)
cd whatsapp-service && npm install && node index.js
```

### Build Binary

```bash
go build -o forex-agent ./cmd/main.go
```

### Jalankan Test

```bash
go test ./...
```

## Roadmap

- [x] Pipeline multi-agent (6 agent)
- [x] Eksekusi agent concurrent
- [x] Analisis sentimen AI (Gemini + Groq fallback)
- [x] WhatsApp messaging bidireksional
- [x] Chatbot interaktif dengan perintah
- [x] Persistensi TimescaleDB
- [ ] Scheduling multi-timeframe (5m, 15m, 1h, 4h)
- [ ] ML prediction service (Python gRPC)
- [ ] Backtesting dengan sinyal historis
- [ ] Tracking win/loss & dashboard performa
- [ ] Web UI untuk monitoring

## Lisensi

MIT

## Disclaimer

Bot ini hanya untuk **tujuan edukasi dan riset**. Trading forex melibatkan risiko signifikan. Jangan trading dengan uang yang tidak sanggup kamu kehilangan. Sinyal masa lalu tidak menjamin performa di masa depan.
