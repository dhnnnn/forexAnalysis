# 🤖 Forex マルチエージェント分析ボット

🌐 **Language / Bahasa / 言語:** [English](../../README.md) | [Bahasa Indonesia](README_ID.md) | [日本語](README_JA.md)

Goで構築されたリアルタイムFXシグナル生成システム。マルチエージェントアーキテクチャを採用し、各エージェントが独自の入出力契約で自律的に動作します。全エージェントが協調してトレーディングシグナルを生成し、WhatsApp経由で配信します。

## アーキテクチャ

```
┌─────────────────────────────────────────────────────────────────┐
│                     外部データソース                               │
│   OANDA WebSocket  │  Twelve Data REST  │  Alpha Vantage REST   │
└──────────┬─────────────────┬─────────────────────┬──────────────┘
           └─────────────────┴─────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│  Agent 1: MarketDataAgent                                       │
│  OHLCVキャンドルの取得＆正規化 → ローリングバッファ (200/ペア)      │
└──────────────────────────┬──────────────────────────────────────┘
                           │
              ┌────────────┴────────────┐
              ▼ (並行処理)              ▼
┌──────────────────────┐  ┌──────────────────────────────────────┐
│  Agent 2: Technical  │  │  Agent 3: Fundamental                │
│  RSI, MACD, EMA,     │  │  Gemini APIによるニュースセンチメント   │
│  Bollinger Bands     │  │  + Groqフォールバック、Redisキャッシュ  │
└──────────┬───────────┘  └───────────────┬──────────────────────┘
           └──────────────┬───────────────┘
                          ▼
┌─────────────────────────────────────────────────────────────────┐
│  Agent 4: RiskAgent                                             │
│  ポジションサイジング、SL/TP計算                                   │
└──────────────────────────┬──────────────────────────────────────┘
                           ▼
┌─────────────────────────────────────────────────────────────────┐
│  Agent 5: DecisionAgent —「メインブレイン」                       │
│  加重スコアリング → BUY/SELL/HOLD + 信頼度 & リスクレベル         │
└──────────────────────────┬──────────────────────────────────────┘
                           ▼
┌─────────────────────────────────────────────────────────────────┐
│  Agent 6: WhatsAppAgent                                         │
│  アラート配信（レート制限、信頼度ゲート付き）                       │
└─────────────────────────────────────────────────────────────────┘
```

## 機能

- **マルチエージェントパイプライン** — 明確な契約を持つ6つの自律エージェント
- **並行実行** — TechnicalAgent + FundamentalAgentがペアごとに並列実行、複数ペアも同時処理
- **AI駆動センチメント分析** — Gemini 2.0 Flashによるニュース分析、Groq (Llama 3.3 70B) をフォールバックとして使用
- **テクニカル指標** — RSI(14)、MACD(12,26,9)、EMA(50,200)、Bollinger Bands(20,2)
- **リスク管理** — 設定可能なポジションサイジング、調整可能なSL/TP
- **WhatsApp統合** — 双方向：コマンド受信 + トレーディングアラート送信
- **インタラクティブチャットボット** — Gemini AIによる自然言語FX Q&A
- **永続ストレージ** — TimescaleDBによるキャンドル履歴とシグナル追跡
- **グレースフルデグラデーション** — データ欠損やエージェント障害でもパイプラインは停止しない
- **Docker対応** — `docker-compose up` 一発でフルスタック起動

## 技術スタック

| コンポーネント | 技術 |
|--------------|------|
| コアエンジン | Go 1.25 |
| WhatsAppサービス | Node.js + whatsapp-web.js |
| AI/NLP | Gemini API（プライマリ）+ Groq API（フォールバック）|
| 時系列DB | TimescaleDB (PostgreSQL) |
| キャッシュ | Redis 7 |
| コンテナ化 | Docker Compose |

## プロジェクト構成

```
├── cmd/
│   └── main.go                 # エントリポイント、パイプラインオーケストレーション
├── internal/
│   ├── agents/
│   │   ├── agent.go            # Agentインターフェース、共有型
│   │   ├── market_data_agent.go
│   │   ├── technical_agent.go
│   │   ├── fundamental_agent.go
│   │   ├── risk_agent.go
│   │   ├── decision_agent.go
│   │   └── whatsapp_agent.go
│   ├── chatbot/
│   │   ├── handler.go          # チャットコマンドルーティング
│   │   └── gemini_chat.go      # AI会話
│   ├── config/
│   │   └── loader.go           # YAML設定 + 環境変数展開
│   ├── feed/
│   │   ├── websocket.go        # OANDA WebSocketフィード
│   │   ├── rest_poller.go      # REST APIフォールバック
│   │   └── normalizer.go       # キャンドル正規化
│   ├── indicators/
│   │   ├── rsi.go
│   │   ├── macd.go
│   │   ├── moving_average.go
│   │   ├── bollinger.go
│   │   └── scorer.go
│   ├── sentiment/
│   │   ├── gemini.go           # Geminiセンチメント分析
│   │   ├── news_fetcher.go     # マルチソースニュース集約
│   │   ├── cache.go            # Redisセンチメントキャッシュ
│   │   └── interfaces.go
│   └── storage/
│       ├── postgres.go         # TimescaleDB永続化
│       └── batch.go            # バッチ挿入ヘルパー
├── whatsapp-service/
│   ├── index.js                # Node.js WhatsAppブリッジ
│   ├── Dockerfile
│   └── package.json
├── config/
│   └── config.yaml             # 全設定
├── migrations/
│   └── 001_init.sql            # TimescaleDBスキーマ
├── docker-compose.yml
├── Dockerfile
└── .env.example
```

## クイックスタート

### 前提条件

- Docker & Docker Compose
- QRコードスキャン用のWhatsAppアカウント

### 1. クローン＆設定

```bash
git clone https://github.com/dhnnnn/forexAnalysis.git
cd forexAnalysis
cp .env.example .env
```

`.env` にAPIキーを設定：

```env
# データソース
OANDA_API_KEY=あなたのOANDAキー
OANDA_ACCOUNT_ID=あなたのアカウントID
TWELVE_DATA_KEY=あなたのTwelveDataキー
ALPHA_VANTAGE_KEY=あなたのAlphaVantageキー

# AI
GEMINI_API_KEY=あなたのGeminiキー
GROQ_API_KEY=あなたのGroqキー              # オプション、フォールバック

# WhatsApp
WA_TARGET_PHONE=628xxxxxxxxxx             # あなたの電話番号

# データベース
DB_PASSWORD=あなたのDBパスワード
REDIS_PASSWORD=あなたのRedisパスワード
```

### 2. 全サービス起動

```bash
docker-compose up --build
```

起動されるサービス：
- **TimescaleDB** ポート 5432
- **Redis** ポート 6379
- **Go Agent** ポート 8080
- **WhatsApp Service** ポート 3001

### 3. WhatsApp接続

コンソール出力にQRコードが表示されます。以下でスキャン：
**WhatsApp → 設定 → リンクされたデバイス → デバイスをリンク**

### 4. 操作

WhatsApp経由でボットにメッセージを送信：

| コマンド | 説明 |
|---------|------|
| `/help` | 全コマンド表示 |
| `/status` | ボットのステータスと設定を表示 |
| `/set balance 500` | トレーディング残高を設定 |
| `/set risk 2` | トレードごとのリスク%を設定 |
| `/risk` | リスク管理計算機 |
| `/analyze` | 分析スキャンを強制実行 |
| *(任意のテキスト)* | AI搭載FX Q&A |

## 設定

全設定は `config/config.yaml` にあります。主要セクション：

```yaml
# 監視する通貨ペア
pairs:
  - "EUR_USD"
  - "GBP_USD"

# アカウント設定
account:
  balance: 1000.0
  risk_percent: 1.0
  default_sl_pips: 20.0
  default_tp_pips: 40.0

# シグナル閾値 (DecisionAgent)
signal:
  buy_threshold: 0.60
  sell_threshold: 0.35
  weights:
    technical: 0.60
    fundamental: 0.40
```

## パイプラインの動作

- ペアごとに**5分間隔**で実行
- 分析開始前に最低**26本のキャンドル**が必要
- 信頼度 ≥ 55% の場合のみアラート送信
- **ペアごとに3分に1回**のレート制限
- TechnicalAgentまたはFundamentalAgentが失敗した場合、DecisionAgentは安全なデフォルト値で処理を継続

## シグナル出力例

```
🟢 BUY EUR_USD

📊 信頼度: 72% | リスク: MEDIUM

💰 エントリー: 1.08450
🛑 SL: 1.08250
🎯 TP: 1.08850
📐 ロット: 0.05

📈 テクニカル: BUY (80%)
📰 ファンダメンタル: bullish (65%)

⏰ 14:30:05 JST
```

## 開発

### ローカル実行（Docker不要）

```bash
# 依存サービスを起動
docker-compose up timescaledb redis -d

# Go agentを実行
go run ./cmd/main.go

# WhatsAppサービスを実行（別ターミナル）
cd whatsapp-service && npm install && node index.js
```

### バイナリビルド

```bash
go build -o forex-agent ./cmd/main.go
```

### テスト実行

```bash
go test ./...
```

## ロードマップ

- [x] マルチエージェントパイプライン（6エージェント）
- [x] エージェントの並行実行
- [x] AIセンチメント分析（Gemini + Groqフォールバック）
- [x] WhatsApp双方向メッセージング
- [x] コマンド付きインタラクティブチャットボット
- [x] TimescaleDB永続化
- [ ] マルチタイムフレームスケジューリング（5m, 15m, 1h, 4h）
- [ ] ML予測サービス（Python gRPC）
- [ ] 過去シグナルによるバックテスト
- [ ] 勝敗追跡＆パフォーマンスダッシュボード
- [ ] モニタリング用Web UI

## ライセンス

MIT

## 免責事項

このボットは**教育・研究目的のみ**です。FX取引には重大なリスクが伴います。失っても良い金額以上の資金で取引しないでください。過去のシグナルは将来のパフォーマンスを保証するものではありません。
