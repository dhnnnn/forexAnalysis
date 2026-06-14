# 🤖 Forex Multi-Agent Analysis Bot

🌐 **Language / Bahasa / 言語:** [English](README.md) | [Bahasa Indonesia](docs/lang/README_ID.md) | [日本語](docs/lang/README_JA.md)

A real-time forex signal generation system built with Go, using a multi-agent architecture. Each agent operates autonomously with its own input/output contract, collectively producing trading signals delivered via WhatsApp.

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                     EXTERNAL DATA SOURCES                        │
│   OANDA WebSocket  │  Twelve Data REST  │  Alpha Vantage REST   │
└──────────┬─────────────────┬─────────────────────┬──────────────┘
           └─────────────────┴─────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│  Agent 1: MarketDataAgent                                       │
│  Fetch & normalize OHLCV candles → rolling buffer (200/pair)    │
└──────────────────────────┬──────────────────────────────────────┘
                           │
              ┌────────────┴────────────┐
              ▼ (concurrent)            ▼
┌──────────────────────┐  ┌──────────────────────────────────────┐
│  Agent 2: Technical  │  │  Agent 3: Fundamental                │
│  RSI, MACD, EMA,     │  │  News sentiment via Gemini API       │
│  Bollinger Bands     │  │  + Groq fallback, Redis cache        │
└──────────┬───────────┘  └───────────────┬──────────────────────┘
           └──────────────┬───────────────┘
                          ▼
┌─────────────────────────────────────────────────────────────────┐
│  Agent 4: RiskAgent                                             │
│  Position sizing, SL/TP calculation                             │
└──────────────────────────┬──────────────────────────────────────┘
                           ▼
┌─────────────────────────────────────────────────────────────────┐
│  Agent 5: DecisionAgent — "The Brain"                           │
│  Weighted scoring → BUY/SELL/HOLD with confidence & risk level  │
└──────────────────────────┬──────────────────────────────────────┘
                           ▼
┌─────────────────────────────────────────────────────────────────┐
│  Agent 6: WhatsAppAgent                                         │
│  Alert delivery (rate-limited, confidence-gated)                │
└─────────────────────────────────────────────────────────────────┘
```

## Features

- **Multi-Agent Pipeline** — 6 autonomous agents with clear contracts
- **Concurrent Execution** — TechnicalAgent + FundamentalAgent run in parallel per pair; multiple pairs processed concurrently
- **AI-Powered Sentiment** — Gemini 2.0 Flash for news analysis with Groq (Llama 3.3 70B) as fallback
- **Technical Indicators** — RSI(14), MACD(12,26,9), EMA(50,200), Bollinger Bands(20,2)
- **Risk Management** — Configurable position sizing with adjustable SL/TP
- **WhatsApp Integration** — Bidirectional: receive commands + send trading alerts
- **Interactive Chatbot** — Natural language forex Q&A via Gemini AI
- **Persistent Storage** — TimescaleDB for candle history and signal tracking
- **Graceful Degradation** — Missing data or failed agents don't crash the pipeline
- **Docker Ready** — Full stack in one `docker-compose up`

## Tech Stack

| Component | Technology |
|-----------|-----------|
| Core Engine | Go 1.25 |
| WhatsApp Service | Node.js + whatsapp-web.js |
| AI/NLP | Gemini API (primary) + Groq API (fallback) |
| Time-Series DB | TimescaleDB (PostgreSQL) |
| Cache | Redis 7 |
| Containerization | Docker Compose |

## Project Structure

```
├── cmd/
│   └── main.go                 # Entry point, pipeline orchestration
├── internal/
│   ├── agents/
│   │   ├── agent.go            # Agent interface, shared types
│   │   ├── market_data_agent.go
│   │   ├── technical_agent.go
│   │   ├── fundamental_agent.go
│   │   ├── risk_agent.go
│   │   ├── decision_agent.go
│   │   └── whatsapp_agent.go
│   ├── chatbot/
│   │   ├── handler.go          # Chat command routing
│   │   └── gemini_chat.go      # AI conversation
│   ├── config/
│   │   └── loader.go           # YAML config with env var expansion
│   ├── feed/
│   │   ├── websocket.go        # OANDA WebSocket feed
│   │   ├── rest_poller.go      # REST API fallback
│   │   └── normalizer.go       # Candle normalization
│   ├── indicators/
│   │   ├── rsi.go
│   │   ├── macd.go
│   │   ├── moving_average.go
│   │   ├── bollinger.go
│   │   └── scorer.go
│   ├── sentiment/
│   │   ├── gemini.go           # Gemini sentiment analysis
│   │   ├── news_fetcher.go     # Multi-source news aggregation
│   │   ├── cache.go            # Redis sentiment cache
│   │   └── interfaces.go
│   └── storage/
│       ├── postgres.go         # TimescaleDB persistence
│       └── batch.go            # Batch insert helper
├── whatsapp-service/
│   ├── index.js                # Node.js WhatsApp bridge
│   ├── Dockerfile
│   └── package.json
├── config/
│   └── config.yaml             # All configuration
├── migrations/
│   └── 001_init.sql            # TimescaleDB schema
├── docker-compose.yml
├── Dockerfile
└── .env.example
```

## Quick Start

### Prerequisites

- Docker & Docker Compose
- WhatsApp account for QR code scanning

### 1. Clone & Configure

```bash
git clone https://github.com/dhnnnn/forexAnalysis.git
cd forexAnalysis
cp .env.example .env
```

Edit `.env` with your API keys:

```env
# Data Sources
OANDA_API_KEY=your_oanda_key
OANDA_ACCOUNT_ID=your_account_id
TWELVE_DATA_KEY=your_twelve_data_key
ALPHA_VANTAGE_KEY=your_alpha_vantage_key

# AI
GEMINI_API_KEY=your_gemini_key
GROQ_API_KEY=your_groq_key          # optional fallback

# WhatsApp
WA_TARGET_PHONE=628xxxxxxxxxx       # your phone number

# Database
DB_PASSWORD=your_db_password
REDIS_PASSWORD=your_redis_password
```

### 2. Start Everything

```bash
docker-compose up --build
```

This starts:
- **TimescaleDB** on port 5432
- **Redis** on port 6379
- **Go Agent** on port 8080
- **WhatsApp Service** on port 3001

### 3. Link WhatsApp

Watch the console output for a QR code. Scan it with:
**WhatsApp → Settings → Linked Devices → Link a Device**

### 4. Interact

Send messages to the bot via WhatsApp:

| Command | Description |
|---------|-------------|
| `/help` | Show all commands |
| `/status` | View bot status & settings |
| `/set balance 500` | Set trading balance |
| `/set risk 2` | Set risk % per trade |
| `/risk` | Risk management calculator |
| `/analyze` | Force analysis scan |
| *(any text)* | AI-powered forex Q&A |

## Configuration

All settings are in `config/config.yaml`. Key sections:

```yaml
# Currency pairs to monitor
pairs:
  - "EUR_USD"
  - "GBP_USD"

# Account settings
account:
  balance: 1000.0
  risk_percent: 1.0
  default_sl_pips: 20.0
  default_tp_pips: 40.0

# Signal thresholds (DecisionAgent)
signal:
  buy_threshold: 0.60
  sell_threshold: 0.35
  weights:
    technical: 0.60
    fundamental: 0.40
```

## Pipeline Behavior

- Runs every **5 minutes** per pair
- Requires minimum **26 candles** before analysis starts
- Alerts only sent when confidence ≥ 55%
- Rate-limited to **1 alert per pair per 3 minutes**
- If TechnicalAgent or FundamentalAgent fails, DecisionAgent gracefully degrades using safe defaults

## Signal Output Example

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

### Run Locally (without Docker)

```bash
# Start dependencies
docker-compose up timescaledb redis -d

# Run Go agent
go run ./cmd/main.go

# Run WhatsApp service (separate terminal)
cd whatsapp-service && npm install && node index.js
```

### Build Binary

```bash
go build -o forex-agent ./cmd/main.go
```

### Run Tests

```bash
go test ./...
```

## Roadmap

- [x] Multi-agent pipeline (6 agents)
- [x] Concurrent agent execution
- [x] AI sentiment analysis (Gemini + Groq fallback)
- [x] WhatsApp bidirectional messaging
- [x] Interactive chatbot with commands
- [x] TimescaleDB persistence
- [ ] Multi-timeframe scheduling (5m, 15m, 1h, 4h)
- [ ] ML prediction service (Python gRPC)
- [ ] Backtesting with historical signals
- [ ] Win/loss tracking & performance dashboard
- [ ] Web UI for monitoring

## License

MIT

## Disclaimer

This bot is for **educational and research purposes only**. Forex trading involves significant risk. Do not trade with money you cannot afford to lose. Past signals do not guarantee future performance.
