# Frontend Design Document
## Real-Time Candlestick Chart + Agent Debate Dashboard

> Dokumen ini menjelaskan kebutuhan, arsitektur, design system, dan layout
> untuk frontend dashboard yang menampilkan candlestick chart real-time
> beserta "debat" antar agent di samping chart.

---

## Daftar Isi

1. [Tujuan & Filosofi](#1-tujuan--filosofi)
2. [Tech Stack](#2-tech-stack)
3. [Arsitektur & Data Flow](#3-arsitektur--data-flow)
4. [Design System](#4-design-system)
5. [Layout & Wireframe](#5-layout--wireframe)
6. [Komponen Utama](#6-komponen-utama)
7. [WebSocket Protocol](#7-websocket-protocol)
8. [State Management](#8-state-management)
9. [Responsive Behavior](#9-responsive-behavior)
10. [File Structure](#10-file-structure)

---

## 1. Tujuan & Filosofi

### Tujuan
- Menampilkan pergerakan candlestick **real-time** dari pipeline backend
- Menampilkan "debat" antar agent (Technical, Fundamental, Decision, MetaObserver, KTA) di samping chart — layaknya percakapan grup
- Memberikan visibility ke regime detection, knowledge rules aktif, dan adaptive weights
- Berfungsi sebagai **demo tool** untuk paper/presentasi

### Filosofi Desain
- **Dark theme first** — standar trading platform, enak di mata untuk waktu lama
- **Information density tinggi** — trader terbiasa banyak data sekaligus
- **Real-time feedback** — setiap perubahan state langsung terlihat tanpa refresh
- **Glanceable** — status penting (signal, regime, confidence) terlihat dalam 1 detik

---

## 2. Tech Stack

| Layer | Teknologi | Alasan |
|-------|-----------|--------|
| Framework | React 18 + TypeScript | Ecosystem matang, component-based |
| Build Tool | Vite | Cepat, HMR instan |
| Chart | `lightweight-charts` (TradingView) | Open-source, ringan (40KB), profesional |
| Styling | Tailwind CSS | Utility-first, cepat iterasi, dark mode built-in |
| Real-time | Native WebSocket | Simpel, low overhead, langsung dari Go backend |
| State | Zustand | Minimal boilerplate, cocok untuk real-time updates |
| Icons | Lucide React | Konsisten, tree-shakeable |
| Fonts | Inter (UI) + JetBrains Mono (data) | Readable, profesional |

---

## 3. Arsitektur & Data Flow

```
┌─────────────────────────────────────────────────────────────┐
│                        BROWSER                               │
│                                                              │
│  ┌────────────┐    ┌───────────────┐    ┌────────────────┐  │
│  │ Chart Store│    │ Agent Store   │    │ Regime Store   │  │
│  │ (candles)  │    │ (debates)     │    │ (context)      │  │
│  └─────┬──────┘    └───────┬───────┘    └───────┬────────┘  │
│        │                   │                    │            │
│  ┌─────┴──────────────────┴────────────────────┴─────────┐  │
│  │              WebSocket Connection Manager               │  │
│  └─────────────────────────┬───────────────────────────────┘  │
└────────────────────────────┼─────────────────────────────────┘
                             │ ws://localhost:8080/ws
                             │
┌────────────────────────────┼─────────────────────────────────┐
│  GO BACKEND (:8080)        │                                  │
│                            │                                  │
│  WebSocket Hub ────────────┘                                  │
│       ↑                                                       │
│  Pipeline Loop (setiap 5 menit)                               │
│       │                                                       │
│  ┌────┴────┐  ┌────────┐  ┌──────────┐  ┌────────────────┐  │
│  │ Candles │  │ Agents │  │ Regime   │  │ Knowledge Base │  │
│  └─────────┘  └────────┘  └──────────┘  └────────────────┘  │
└───────────────────────────────────────────────────────────────┘
```

### WebSocket Messages (Backend → Frontend)

```typescript
// Union type untuk semua WS messages
type WSMessage =
  | { type: "candle"; data: CandleData }
  | { type: "agent_output"; data: AgentDebateEntry }
  | { type: "regime"; data: RegimeData }
  | { type: "signal"; data: SignalData }
  | { type: "knowledge_rule"; data: KnowledgeRuleData }
  | { type: "metrics"; data: AgentMetricsData }
```

---

## 4. Design System

### 4.1 Color Palette

```
Background & Surfaces
─────────────────────
bg-primary:      #0d1117    (darkest — main bg)
bg-secondary:    #161b22    (card/panel bg)
bg-tertiary:     #1c2128    (hover states)
bg-elevated:     #21262d    (modals, dropdowns)
border:          #30363d    (subtle borders)

Text
─────
text-primary:    #e6edf3    (headings, important)
text-secondary:  #8b949e    (body text, descriptions)
text-muted:      #484f58    (timestamps, labels)

Accent — Trading Signals
─────────────────────────
buy-green:       #2ea043    (BUY signal)
buy-green-bg:    #2ea04320  (BUY background tint)
sell-red:        #f85149    (SELL signal)
sell-red-bg:     #f8514920  (SELL background tint)
hold-amber:      #d29922    (HOLD signal)
hold-amber-bg:   #d2992220  (HOLD background tint)

Regime Colors
─────────────
trending:        #58a6ff    (blue)
ranging:         #8b949e    (gray)
breakout:        #bc8cff    (purple)
high-vol:        #f85149    (red)
low-vol:         #3fb950    (green)

Agent Colors (untuk avatar/badge)
──────────────────────────────────
technical:       #58a6ff    (blue)
fundamental:     #d2a8ff    (purple)
decision:        #ffa657    (orange)
regime:          #79c0ff    (light blue)
meta-observer:   #f0883e    (amber)
kta:             #56d364    (green)
risk:            #ff7b72    (coral)
```

### 4.2 Typography

```
Font Families:
  UI/Labels:    'Inter', system-ui, sans-serif
  Data/Numbers: 'JetBrains Mono', monospace

Font Sizes (rem):
  xs:    0.75rem   (timestamps, badges)
  sm:    0.875rem  (labels, secondary)
  base:  1rem      (body text)
  lg:    1.125rem  (headings)
  xl:    1.25rem   (pair name, signal)
  2xl:   1.5rem    (hero numbers like confidence %)

Font Weights:
  normal: 400  (body)
  medium: 500  (labels)
  semibold: 600 (headings)
  bold:   700  (signals, numbers)
```

### 4.3 Spacing & Radius

```
Spacing Scale (px):
  1: 4px    2: 8px    3: 12px   4: 16px
  5: 20px   6: 24px   8: 32px   10: 40px

Border Radius:
  sm: 4px   (badges, small elements)
  md: 8px   (cards, panels)
  lg: 12px  (modals)
  full: 9999px (avatars, pills)
```

### 4.4 Shadows & Effects

```
shadow-sm:    0 1px 2px rgba(0,0,0,0.3)
shadow-md:    0 4px 8px rgba(0,0,0,0.4)
shadow-glow:  0 0 12px rgba(46,160,67,0.3)  (for active buy signals)

Transitions:
  default: 150ms ease
  data:    300ms ease (number changes, progress bars)
```

---

## 5. Layout & Wireframe

### 5.1 Desktop Layout (>= 1280px)

```
┌──────────────────────────────────────────────────────────────────┐
│  HEADER BAR (h: 48px)                                            │
│  [Logo] [EUR_USD ▼] [GBP_USD]  [1h ▼]    [●Connected] [⚙️]    │
├──────────────────────────────────────────┬───────────────────────┤
│                                          │                       │
│  CANDLESTICK CHART (flex: 2)             │  AGENT DEBATE PANEL   │
│                                          │  (w: 400px, fixed)    │
│  ┌────────────────────────────────────┐  │                       │
│  │                                    │  │  ┌─────────────────┐  │
│  │     Candlestick + Volume           │  │  │ TechnicalAgent  │  │
│  │     + Signal markers               │  │  │ 🟢 BUY 72%     │  │
│  │     + Regime bands                 │  │  │ RSI oversold... │  │
│  │                                    │  │  └─────────────────┘  │
│  │                                    │  │  ┌─────────────────┐  │
│  │                                    │  │  │ FundamentalAgent│  │
│  │                                    │  │  │ 🔴 SELL 60%    │  │
│  │                                    │  │  │ USD strength... │  │
│  │                                    │  │  └─────────────────┘  │
│  │                                    │  │  ┌─────────────────┐  │
│  └────────────────────────────────────┘  │  │ DecisionAgent   │  │
│                                          │  │ 🟡 HOLD 52%    │  │
│  ┌────────────────────────────────────┐  │  │ Conflict...     │  │
│  │  INDICATORS BAR                    │  │  └─────────────────┘  │
│  │  RSI: 28 | MACD: +0.002 | ADX: 18 │  │                       │
│  └────────────────────────────────────┘  │  ── Knowledge ──────  │
│                                          │  ┌─────────────────┐  │
├──────────────────────────────────────────┤  │ Rule: -0.15     │  │
│  STATUS BAR (h: 36px)                    │  │ tech in ranging  │  │
│  Regime: RANGING | Tech: 0.45 Fund: 0.55│  └─────────────────┘  │
│  | Rules: 2 active | Last eval: 14:30   │                       │
├──────────────────────────────────────────┴───────────────────────┤
│                                                                    │
│  HISTORY & LOG PANEL (h: 280px, resizable)                        │
│  [Signals] [Performance] [Rules] [Regime] [System Log]            │
│                                                                    │
│  ┌──────────────────────────────────────────────────────────────┐ │
│  │ #  Time       Pair     Signal  Conf  Entry    SL       TP    │ │
│  │ 1  14:05:00   EUR_USD  BUY     72%   1.08490  1.08290  1.08890│
│  │ 2  14:05:01   GBP_USD  SELL    65%   1.27120  1.27320  1.26720│
│  │ 3  14:00:00   EUR_USD  HOLD    48%   —        —        —     │ │
│  │ 4  13:55:00   EUR_USD  BUY     70%   1.08430  1.08230  1.08830│
│  │    ↳ eval: ✅ correct (+18 pips)                              │ │
│  │ 5  13:50:00   GBP_USD  SELL    62%   1.27250  1.27450  1.26850│
│  │    ↳ eval: ❌ incorrect (-5 pips)                             │ │
│  └──────────────────────────────────────────────────────────────┘ │
│                                                                    │
└────────────────────────────────────────────────────────────────────┘
```

### 5.2 Panel Breakdown

#### Header Bar
- Pair selector (dropdown, highlight active pair)
- Timeframe tabs (1h, 5m, 15m, 4h)
- Connection status indicator (green dot = connected)
- Settings gear icon

#### Chart Area (kiri, flex grow)
- TradingView Lightweight Charts candlestick
- Volume bars di bawah candles
- BUY/SELL markers pada chart (triangle up/down)
- Regime background bands (subtle color tint per regime period)
- Bollinger Bands overlay (optional toggle)

#### Agent Debate Panel (kanan, fixed 400px)
- Scroll vertikal, terbaru di bawah (chat-style)
- Setiap entry = 1 card per agent per cycle
- Grouped per timestamp (14:00, 14:05, 14:10...)
- Badge warna per agent
- Signal pill (BUY/SELL/HOLD) + confidence bar
- Expandable reasoning text
- Knowledge section di bawah (rules aktif, weights)

#### Indicators Bar
- Horizontal strip di bawah chart
- Key numbers: RSI, MACD Histogram, ADX, ATR, BB Position
- Color-coded (oversold = green, overbought = red)

#### Status Bar (footer)
- Current regime + badge color
- Active adaptive weights (tech: 0.45, fund: 0.55)
- Active rules count
- Last evaluation timestamp
- Pipeline cycle countdown

---

## 6. Komponen Utama

### 6.1 `<CandlestickChart />`

```typescript
interface CandlestickChartProps {
  pair: string;
  timeframe: string;
  candles: CandleData[];
  signals: SignalMarker[];   // BUY/SELL markers on chart
  regimeBands: RegimeBand[]; // background color bands
}
```

- Menggunakan `lightweight-charts` createChart API
- Auto-resize on container resize
- Crosshair sync dengan indicator bar
- Signal markers sebagai custom series markers

### 6.2 `<AgentDebatePanel />`

```typescript
interface AgentDebateEntry {
  id: string;
  timestamp: string;
  agent: AgentName;
  signal: "BUY" | "SELL" | "HOLD";
  confidence: number;       // 0-1
  reasoning: string;        // max 50 words
  regime?: string;
  weightInfo?: string;      // "tech: 0.45, fund: 0.55"
  isKnowledge?: boolean;    // true for KTA entries
}
```

- Chat-style layout (newest at bottom, auto-scroll)
- Grouped by pipeline cycle timestamp
- Agent avatar (colored circle + initial)
- Signal pill badge (colored background)
- Confidence progress bar
- Collapsible reasoning (show first line, expand on click)

### 6.3 `<AgentCard />`

```
┌──────────────────────────────────────┐
│ 🔵 TechnicalAgent        14:05:00   │
│                                      │
│ ┌──────────┐  Confidence: ████░ 72% │
│ │  🟢 BUY  │                        │
│ └──────────┘  RSI oversold (28) +   │
│               MACD bullish cross     │
│                                      │
│ [▼ more details]                     │
└──────────────────────────────────────┘
```

- Compact by default (signal + 1 line reason)
- Expandable untuk full detail (RSI, MACD, BB, EMA values)
- Subtle animation on new entry (fade-in slide-up)

### 6.4 `<KnowledgePanel />`

```
┌──────────────────────────────────────┐
│ 🧠 Active Knowledge Rules            │
│                                      │
│ Rule #1 (expires in 18h)             │
│ "Reduce TechnicalAgent weight in     │
│  ranging market"                     │
│ Δ weight: -0.15 | applied: 3x       │
│ confidence: 78%                      │
│                                      │
│ ── Adaptive Weights ──               │
│ Technical:    ████████░░  0.45       │
│ Fundamental:  ███████████  0.55      │
│ (base: 0.60 / 0.40)                 │
└──────────────────────────────────────┘
```

### 6.5 `<RegimeBadge />`

```typescript
interface RegimeBadgeProps {
  regime: "trending" | "ranging" | "breakout" | "high_vol" | "low_vol";
  adx: number;
  trendStrength: number;
}
```

- Pill badge dengan warna sesuai regime
- Icon (📈 trending, ↔️ ranging, 💥 breakout, 🌋 high_vol, 🧊 low_vol)
- Tooltip dengan detail ADX, ATR, volatility

### 6.6 `<ConnectionStatus />`

- Green dot: connected
- Yellow dot: reconnecting
- Red dot: disconnected
- Auto reconnect with exponential backoff

### 6.7 `<HistoryPanel />`

Panel di bagian bawah layar yang menampilkan semua histori dan log aktivitas sistem.
Menggunakan tab navigation untuk berbagai jenis data.

```typescript
interface HistoryPanelProps {
  activeTab: HistoryTab;
  onTabChange: (tab: HistoryTab) => void;
}

type HistoryTab = "signals" | "performance" | "rules" | "regime" | "system";
```

**Fitur utama:**
- Resizable height (drag border atas)
- Collapsible (double-click border = minimize)
- Setiap tab punya tabel data sendiri
- Auto-refresh dari WebSocket + polling REST untuk historical

#### Tab 1: Signals (Riwayat Sinyal)

```
┌─────────────────────────────────────────────────────────────────────┐
│ #  Time       Pair      Signal  Conf   Regime    Entry     Status   │
│ 1  14:05:00   EUR_USD   BUY     72%    ranging   1.08490   ⏳ pending│
│ 2  14:05:01   GBP_USD   SELL    65%    trending  1.27120   ⏳ pending│
│ 3  14:00:00   EUR_USD   HOLD    48%    ranging   —         — skip   │
│ 4  13:30:00   EUR_USD   BUY     70%    trending  1.08430   ✅ +18pip│
│ 5  13:30:01   GBP_USD   SELL    62%    ranging   1.27250   ❌ -5pip │
│ 6  13:00:00   EUR_USD   BUY     68%    trending  1.08380   ✅ +22pip│
└─────────────────────────────────────────────────────────────────────┘
```

- Status: ⏳ pending eval, ✅ correct, ❌ incorrect, — skipped (HOLD)
- Klik row = expand detail (SL, TP, lot, tech reason, fund reason)
- Filter: pair, signal direction, status
- Color row: hijau muda (correct), merah muda (incorrect)

#### Tab 2: Performance (Performa Agent)

```
┌──────────────────────────────────────────────────────────────────┐
│ Agent            Accuracy (20)  Win  Loss  Streak  Regime        │
│ TechnicalAgent   ████████░░ 65%  13    7    0      ranging       │
│ FundamentalAgent ██████░░░░ 55%  11    9    2      ranging       │
│                                                                   │
│ ── Accuracy Over Time (mini sparkline chart) ──                   │
│ Technical:  ▁▂▃▄▅▆▅▄▃▄▅▆▇▆▅▄▃▃▄▅                               │
│ Fundamental: ▅▅▄▃▃▂▃▄▃▂▂▃▃▄▃▂▂▁▂▃                              │
│                                                                   │
│ ── Recent Evaluations ──                                          │
│ Time       Agent            Pair     Correct  Pips    Regime      │
│ 14:30:00   TechnicalAgent   EUR_USD  ✅       +18     trending   │
│ 14:30:00   FundamentalAgent EUR_USD  ✅       +18     trending   │
│ 14:25:00   TechnicalAgent   GBP_USD  ❌       -5      ranging    │
│ 14:25:00   FundamentalAgent GBP_USD  ❌       -5      ranging    │
└──────────────────────────────────────────────────────────────────┘
```

- Summary cards per agent (accuracy, win/loss, streak)
- Mini sparkline accuracy history
- Tabel evaluasi terbaru
- Alert badge jika ada agent degradasi

#### Tab 3: Rules (Knowledge Rules)

```
┌──────────────────────────────────────────────────────────────────┐
│ ── Active Rules (2) ──                                            │
│                                                                   │
│ 🟢 Rule #a3f2..  Created: 13:30  Expires in: 18h 30m            │
│    Source: TechnicalAgent failed in ranging                       │
│    Action: TechnicalAgent weight -0.15 (min: 0.05)               │
│    Confidence: 78%  |  Applied: 3x                               │
│    Reasoning: "Technical indicators unreliable in low-ADX..."    │
│                                                                   │
│ 🟢 Rule #b8c1..  Created: 12:00  Expires in: 12h 00m            │
│    Source: FundamentalAgent failed in breakout                    │
│    Action: FundamentalAgent weight -0.20 (min: 0.05)             │
│    Confidence: 72%  |  Applied: 5x                               │
│    Reasoning: "Sentiment analysis lags during sudden breakout"   │
│                                                                   │
│ ── Expired Rules (last 24h) ──                                    │
│ ⚫ Rule #c4d5..  Lived: 24h  Applied: 12x  Impact: +8% accuracy │
│ ⚫ Rule #e6f7..  Lived: 18h  Applied: 7x   Impact: +3% accuracy │
└──────────────────────────────────────────────────────────────────┘
```

- Active rules dengan countdown timer (expires in...)
- Applied count incrementing real-time
- Expired rules history dengan impact assessment
- Visual: progress bar untuk remaining TTL

#### Tab 4: Regime (Regime History)

```
┌──────────────────────────────────────────────────────────────────┐
│ ── Current ──                                                     │
│ EUR_USD: 📊 RANGING (ADX: 18.5, Vol: 0.42%, Trend: 0.37)        │
│ GBP_USD: 📈 TRENDING (ADX: 32.1, Vol: 1.2%, Trend: 0.64)       │
│                                                                   │
│ ── Regime Timeline (horizontal bar) ──                            │
│ EUR_USD: [trending ██][ranging ████████][breakout █][ranging ███] │
│ GBP_USD: [ranging ███][trending ███████████][high_vol ██]        │
│                                                                   │
│ ── Regime Change Log ──                                           │
│ Time       Pair      From        → To          ADX   Vol         │
│ 14:05:00   EUR_USD   trending    → ranging     18.5  0.42%      │
│ 13:20:00   GBP_USD   ranging     → trending    32.1  1.20%      │
│ 12:45:00   EUR_USD   breakout    → trending    28.3  1.80%      │
│ 12:00:00   EUR_USD   ranging     → breakout    22.0  2.70%      │
└──────────────────────────────────────────────────────────────────┘
```

- Current regime per pair dengan detail numbers
- Timeline bar (horizontal stacked bar chart) menunjukkan durasi setiap regime
- Change log tabel (kapan regime berubah)
- Color-coded sesuai regime palette

#### Tab 5: System Log

```
┌──────────────────────────────────────────────────────────────────┐
│ [DEBUG] [INFO] [WARN] [ERROR]  Filter: [________]  Auto-scroll ☑│
│                                                                   │
│ 14:05:04 INFO  📊 Pipeline completed pair=EUR_USD signal=HOLD    │
│ 14:05:03 INFO  ✨ KTA generated 1 new rule                       │
│ 14:05:03 WARN  🚨 MetaObserver: TechnicalAgent accuracy drop 20% │
│ 14:05:02 DEBUG ✅ FundamentalAgent completed sentiment=bearish    │
│ 14:05:02 DEBUG ✅ TechnicalAgent completed signal=BUY conf=0.72  │
│ 14:05:01 DEBUG 🔍 RegimeDetection regime=ranging ADX=18.5        │
│ 14:05:00 DEBUG 🔊 Broadcaster: 2 rules distributed               │
│ 14:05:00 INFO  ── Pipeline start EUR_USD ──                       │
│ 14:00:04 INFO  📊 Pipeline completed pair=GBP_USD signal=SELL    │
│ ...                                                               │
└──────────────────────────────────────────────────────────────────┘
```

- Level filter buttons (DEBUG, INFO, WARN, ERROR)
- Text search filter
- Auto-scroll toggle
- Timestamp + colored level badge
- Monospace font untuk log lines
- Klik log entry = expand full context

---

## 7. WebSocket Protocol

### Endpoint
```
ws://localhost:8080/ws
```

### Message Format (JSON)
```typescript
// Server → Client
interface ServerMessage {
  type: "candle" | "agent_output" | "regime" | "signal" | 
        "knowledge_rule" | "metrics" | "pipeline_start" | "pipeline_end";
  pair: string;
  timestamp: string; // ISO 8601
  data: any;
}

// Candle update
{
  type: "candle",
  pair: "EUR_USD",
  timestamp: "2026-06-14T14:05:00Z",
  data: {
    open: 1.08450,
    high: 1.08520,
    low: 1.08410,
    close: 1.08490,
    volume: 1250.5
  }
}

// Agent debate entry
{
  type: "agent_output",
  pair: "EUR_USD",
  timestamp: "2026-06-14T14:05:02Z",
  data: {
    agent: "TechnicalAgent",
    signal: "BUY",
    confidence: 0.72,
    reasoning: "RSI oversold (28) + MACD bullish cross",
    details: {
      rsi: 28.4,
      macd_hist: 0.00023,
      bb_position: 0.12,
      ema50: 1.08320,
      ema200: 1.08150
    }
  }
}

// Regime change
{
  type: "regime",
  pair: "EUR_USD",
  timestamp: "2026-06-14T14:05:01Z",
  data: {
    regime: "ranging",
    adx: 18.5,
    atr: 0.00045,
    volatility: 0.0042,
    trend_strength: 0.37
  }
}

// Final signal (after Decision)
{
  type: "signal",
  pair: "EUR_USD",
  timestamp: "2026-06-14T14:05:03Z",
  data: {
    signal: "HOLD",
    confidence: 0.52,
    entry: 1.08490,
    stop_loss: 1.08290,
    take_profit: 1.08890,
    lot_size: 0.05,
    tech_weight: 0.45,
    fund_weight: 0.55,
    regime: "ranging"
  }
}

// Knowledge rule generated
{
  type: "knowledge_rule",
  pair: "EUR_USD",
  timestamp: "2026-06-14T14:05:04Z",
  data: {
    source_agent: "TechnicalAgent",
    target_agent: "TechnicalAgent",
    regime: "ranging",
    weight_delta: -0.15,
    confidence: 0.78,
    reasoning: "Technical indicators unreliable in low-ADX ranging market",
    expires_in_hours: 24
  }
}

// Pipeline lifecycle (untuk UI loading state)
{
  type: "pipeline_start",
  pair: "EUR_USD",
  timestamp: "2026-06-14T14:05:00Z",
  data: {}
}
{
  type: "pipeline_end",
  pair: "EUR_USD", 
  timestamp: "2026-06-14T14:05:04Z",
  data: { duration_ms: 4200 }
}
```

### Client → Server (optional)
```typescript
// Subscribe ke pair tertentu
{ type: "subscribe", pairs: ["EUR_USD", "GBP_USD"] }

// Request historical data
{ type: "history", pair: "EUR_USD", limit: 200 }
```

### REST Endpoints (untuk History Panel)

```
GET /api/signals?pair=EUR_USD&limit=50
  → SignalHistoryEntry[]

GET /api/performance?agent=TechnicalAgent&limit=50
  → PerformanceLogEntry[]

GET /api/rules?status=active
GET /api/rules?status=expired&limit=20
  → KnowledgeRuleEntry[]

GET /api/regime/history?pair=EUR_USD&limit=100
  → RegimeLogEntry[]

GET /api/regime/changes?pair=EUR_USD&limit=20
  → RegimeChangeEntry[]

GET /api/logs?level=INFO&limit=200
  → SystemLogEntry[]
```

### History Data Types

```typescript
interface SignalHistoryEntry {
  id: number;
  timestamp: string;
  pair: string;
  signal: "BUY" | "SELL" | "HOLD";
  confidence: number;
  regime: string;
  entry: number;
  stopLoss: number;
  takeProfit: number;
  lotSize: number;
  techSignal: string;
  techConf: number;
  techReason: string;
  fundSentiment: string;
  fundConf: number;
  fundReason: string;
  // Evaluation (null jika belum dievaluasi)
  evalStatus: "pending" | "correct" | "incorrect" | "skipped" | null;
  evalPrice: number | null;
  pipsMove: number | null;
  evalTime: string | null;
}

interface PerformanceLogEntry {
  agentName: string;
  pair: string;
  regime: string;
  signal: string;
  entryPrice: number;
  evalPrice: number;
  correct: boolean;
  pipsMove: number;
  signalTime: string;
  evalTime: string;
}

interface AgentPerformanceSummary {
  agentName: string;
  accuracy: number;       // current rolling window
  accuracyPrev: number;   // previous window
  winCount: number;
  lossCount: number;
  lossStreak: number;
  dominantRegime: string;
  // Sparkline data (last 40 outcomes)
  history: boolean[];     // true=win, false=loss
}

interface KnowledgeRuleEntry {
  id: string;
  sourceAgent: string;
  targetAgent: string;
  regime: string;
  weightDelta: number;
  minWeight: number;
  confidence: number;
  reasoning: string;
  applyCount: number;
  createdAt: string;
  expiresAt: string;
  status: "active" | "expired";
  // Impact (calculated for expired rules)
  impactAccuracyDelta?: number;
}

interface RegimeLogEntry {
  pair: string;
  regime: string;
  adx: number;
  atr: number;
  volatility: number;
  trendStrength: number;
  detectedAt: string;
}

interface RegimeChangeEntry {
  pair: string;
  fromRegime: string;
  toRegime: string;
  adx: number;
  volatility: number;
  changedAt: string;
}

interface SystemLogEntry {
  timestamp: string;
  level: "DEBUG" | "INFO" | "WARN" | "ERROR";
  message: string;
  agent?: string;
  pair?: string;
  details?: Record<string, any>;
}
```

---

## 8. State Management (Zustand)

```typescript
// stores/chartStore.ts
interface ChartStore {
  candles: Map<string, CandleData[]>;  // pair → candles
  addCandle: (pair: string, candle: CandleData) => void;
}

// stores/agentStore.ts
interface AgentStore {
  debates: AgentDebateEntry[];        // sorted by timestamp
  addDebateEntry: (entry: AgentDebateEntry) => void;
  clearOlderThan: (hours: number) => void;
}

// stores/regimeStore.ts  
interface RegimeStore {
  currentRegime: Map<string, RegimeData>;  // pair → regime
  regimeHistory: RegimeBand[];
  setRegime: (pair: string, regime: RegimeData) => void;
}

// stores/knowledgeStore.ts
interface KnowledgeStore {
  activeRules: KnowledgeRuleData[];
  adaptiveWeights: { tech: number; fund: number };
  addRule: (rule: KnowledgeRuleData) => void;
  setWeights: (tech: number, fund: number) => void;
}

// stores/historyStore.ts
interface HistoryStore {
  // Signals tab
  signals: SignalHistoryEntry[];
  addSignal: (signal: SignalHistoryEntry) => void;
  updateSignalEval: (id: number, eval: EvalResult) => void;

  // Performance tab
  agentSummaries: AgentPerformanceSummary[];
  performanceLogs: PerformanceLogEntry[];
  addPerformanceLog: (log: PerformanceLogEntry) => void;

  // Rules tab
  activeRules: KnowledgeRuleEntry[];
  expiredRules: KnowledgeRuleEntry[];

  // Regime tab
  regimeChanges: RegimeChangeEntry[];
  regimeTimeline: Map<string, RegimeLogEntry[]>;  // pair → entries

  // System log tab
  logs: SystemLogEntry[];
  logFilter: { level: string; search: string };
  addLog: (log: SystemLogEntry) => void;
  setLogFilter: (filter: Partial<{ level: string; search: string }>) => void;

  // Active tab state
  activeTab: HistoryTab;
  setActiveTab: (tab: HistoryTab) => void;

  // Panel state
  panelHeight: number;
  isCollapsed: boolean;
  setPanelHeight: (h: number) => void;
  toggleCollapse: () => void;
}

// stores/connectionStore.ts
interface ConnectionStore {
  status: "connected" | "reconnecting" | "disconnected";
  lastMessage: string;  // timestamp
}
```

---

## 9. Responsive Behavior

| Breakpoint | Layout |
|-----------|--------|
| >= 1280px (xl) | Side-by-side: Chart + Debate Panel, History panel di bawah full-width |
| 768–1279px (md) | Chart full width, Debate sebagai drawer kanan, History collapsible |
| < 768px (sm) | Stacked: Chart → Debate → History (semua tabs jadi bottom nav) |

### Panel Resize Behavior
- History panel: default 280px height, min 150px, max 50vh
- Drag handle di border atas history panel
- Double-click handle = collapse/expand
- State disimpan di localStorage

### Mobile-specific
- Chart height: 50vh
- Debate panel: bottom sheet (drag up)
- History panel: full-screen modal dengan tab navigation
- Pinch-to-zoom pada chart
- Signal notifications sebagai toast

---

## 10. File Structure

```
web/
├── index.html
├── package.json
├── tsconfig.json
├── tailwind.config.ts
├── vite.config.ts
├── public/
│   └── favicon.svg
└── src/
    ├── main.tsx
    ├── App.tsx
    ├── index.css                  (Tailwind imports + custom vars)
    │
    ├── components/
    │   ├── layout/
    │   │   ├── Header.tsx
    │   │   ├── StatusBar.tsx
    │   │   └── MainLayout.tsx
    │   │
    │   ├── chart/
    │   │   ├── CandlestickChart.tsx
    │   │   ├── SignalMarkers.tsx
    │   │   ├── RegimeBands.tsx
    │   │   ├── IndicatorBar.tsx
    │   │   └── VolumeOverlay.tsx
    │   │
    │   ├── agents/
    │   │   ├── AgentDebatePanel.tsx
    │   │   ├── AgentCard.tsx
    │   │   ├── AgentAvatar.tsx
    │   │   ├── SignalBadge.tsx
    │   │   ├── ConfidenceBar.tsx
    │   │   └── DebateTimestamp.tsx
    │   │
    │   ├── knowledge/
    │   │   ├── KnowledgePanel.tsx
    │   │   ├── RuleCard.tsx
    │   │   └── WeightsDisplay.tsx
    │   │
    │   ├── history/
    │   │   ├── HistoryPanel.tsx       (tab container + resize handle)
    │   │   ├── SignalsTab.tsx          (riwayat sinyal + status eval)
    │   │   ├── PerformanceTab.tsx     (agent accuracy + sparkline)
    │   │   ├── RulesTab.tsx           (active + expired rules)
    │   │   ├── RegimeTab.tsx          (timeline + change log)
    │   │   ├── SystemLogTab.tsx       (filtered log stream)
    │   │   └── shared/
    │   │       ├── DataTable.tsx       (reusable sortable table)
    │   │       ├── StatusBadge.tsx     (✅ ❌ ⏳)
    │   │       ├── Sparkline.tsx       (mini inline chart)
    │   │       └── TabButton.tsx
    │   │
    │   └── common/
    │       ├── RegimeBadge.tsx
    │       ├── ConnectionStatus.tsx
    │       ├── PairSelector.tsx
    │       ├── TimeframeSelector.tsx
    │       └── ResizeHandle.tsx
    │
    ├── stores/
    │   ├── chartStore.ts
    │   ├── agentStore.ts
    │   ├── regimeStore.ts
    │   ├── knowledgeStore.ts
    │   ├── historyStore.ts           (signals, performance, logs)
    │   └── connectionStore.ts
    │
    ├── hooks/
    │   ├── useWebSocket.ts           (connection manager + reconnect)
    │   ├── useChart.ts               (chart instance lifecycle)
    │   ├── useAutoScroll.ts          (debate panel auto-scroll)
    │   ├── useResizable.ts           (panel resize drag)
    │   └── useHistoryPolling.ts      (REST polling for historical data)
    │
    ├── types/
    │   ├── candle.ts
    │   ├── agent.ts
    │   ├── regime.ts
    │   ├── knowledge.ts
    │   ├── history.ts
    │   └── websocket.ts
    │
    └── utils/
        ├── formatters.ts             (price, percentage, time, pips)
        ├── colors.ts                 (agent colors, regime colors)
        └── constants.ts              (API URLs, timeframes)
```

---

## Catatan Implementasi

### Prioritas Build

```
Phase 1 — Skeleton (1 hari)
  [ ] Vite + React + Tailwind setup
  [ ] Dark theme base
  [ ] MainLayout (header + chart area + debate panel)
  [ ] Connection status (mock)

Phase 2 — Chart (1-2 hari)
  [ ] lightweight-charts integration
  [ ] Candle rendering dari mock data
  [ ] Signal markers
  [ ] Indicator bar

Phase 3 — Agent Debate (1-2 hari)
  [ ] AgentDebatePanel + AgentCard
  [ ] Chat-style scroll behavior
  [ ] Agent avatars + colors
  [ ] Signal badges + confidence bars

Phase 4 — WebSocket Integration (1 hari)
  [ ] useWebSocket hook
  [ ] Zustand stores wired to WS messages
  [ ] Real-time candle updates
  [ ] Real-time debate entries

Phase 5 — Knowledge & Polish (1 hari)
  [ ] Knowledge panel
  [ ] Regime bands on chart
  [ ] Responsive breakpoints
  [ ] Animations (fade-in, slide-up)

Phase 6 — Backend WS Hub (Go) (1 hari)
  [ ] WebSocket hub di Go backend
  [ ] Broadcast candle dari pipeline
  [ ] Broadcast agent output per cycle
  [ ] Historical data REST endpoint
```

### Performa

- Candle buffer di frontend: max 500 candles (trim oldest)
- Debate entries: max 100 entries (trim oldest)
- Chart rendering: requestAnimationFrame throttled
- WebSocket reconnect: exponential backoff (1s, 2s, 4s, 8s, max 30s)

### Accessibility

- Semantic HTML (header, main, aside, article)
- ARIA labels pada interactive elements
- Keyboard navigation pada pair selector dan timeframe
- Color coding SELALU disertai text/icon (tidak rely purely pada warna)
- Sufficient contrast ratio (WCAG AA minimum)

---

*Dokumen ini akan di-update seiring development berjalan.*
