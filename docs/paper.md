# Panduan Penulisan Paper & Desain Eksperimen
## "A Self-Adaptive Multi-Agent Framework for Forex Trading Using Regime-Aware Knowledge Transfer and Meta-Observation"

> Target: Jurnal SINTA 2 (JNTETI / IJAIN / Lontar Komputer)
> Estimasi halaman: 10–14 halaman (format dua kolom IEEE)

---

## Daftar Isi

1. [Judul & Abstrak](#1-judul--abstrak)
2. [Struktur Paper Lengkap](#2-struktur-paper-lengkap)
3. [Section 1 — Introduction](#3-section-1--introduction)
4. [Section 2 — Related Work](#4-section-2--related-work)
5. [Section 3 — Methodology](#5-section-3--methodology)
6. [Section 4 — Experimental Setup](#6-section-4--experimental-setup)
7. [Section 5 — Results & Discussion](#7-section-5--results--discussion)
8. [Section 6 — Conclusion](#8-section-6--conclusion)
9. [Desain Eksperimen Detail](#9-desain-eksperimen-detail)
10. [Cara Mengumpulkan Data Eksperimen](#10-cara-mengumpulkan-data-eksperimen)
11. [Template Tabel & Grafik](#11-template-tabel--grafik)
12. [Referensi Wajib](#12-referensi-wajib)
13. [Timeline Penulisan](#13-timeline-penulisan)

---

## 1. Judul & Abstrak

### Judul Utama

```
A Self-Adaptive Multi-Agent Framework for Forex Trading Using
Regime-Aware Knowledge Transfer and Meta-Observation
```

### Judul Alternatif (jika target jurnal lebih ke sistem)

```
MetaKnowledge-MAS: A Collective Learning Framework for Multi-Agent
Forex Signal Generation with Dynamic Knowledge Transfer
```

### Draft Abstrak (250 kata)

```
Multi-agent systems (MAS) for financial trading have demonstrated
promising performance in signal generation. However, existing frameworks
suffer from a fundamental limitation: when an agent fails due to a
shifting market regime, the system merely reduces its influence weight
without understanding *why* the failure occurred or transferring
that understanding to other agents. This paper proposes a self-adaptive
multi-agent framework that introduces three novel components:
(1) a RegimeDetectionAgent that classifies market conditions into
five regimes — Trending, Ranging, Breakout, High Volatility, and
Low Volatility — using ADX, ATR, and Bollinger Band width;
(2) a MetaObserverAgent that monitors agent performance within each
regime, detects degradation events via rolling accuracy windows, and
generates structured ExperienceReports; and
(3) a KnowledgeTransferAgent that leverages a Large Language Model
(Gemini 2.0 Flash with Groq Llama 3.3 fallback) to extract causal
reasoning from ExperienceReports and broadcast KnowledgeRules to
the agent pool, enabling collective adaptation.

We evaluate the framework on EUR/USD, GBP/USD, and USD/JPY using
six months of M5 candle data (January–June 2024). Experiments compare
three baselines: (A) static-weight MAS without regime awareness,
(B) regime-aware MAS with static weights, and (C) our proposed system
with full knowledge transfer. Results show that the proposed framework
achieves a win rate of 67.3% (+14.2% over baseline A), a Sharpe ratio
of 1.84, and a maximum drawdown of 8.7%, demonstrating that collective
inter-agent learning — specifically knowing *why* an agent fails —
produces measurably superior outcomes compared to systems that only
track *who* fails.
```

> **Catatan:** Isi angka hasil eksperimen setelah backtest selesai.
> Angka di atas adalah ilustrasi target yang realistis untuk paper Sinta 2.

---

## 2. Struktur Paper Lengkap

```
Paper Structure (IEEE two-column format, ~12 halaman)

1. Introduction                          ~1.0 hal
   1.1 Problem Statement
   1.2 Research Questions
   1.3 Contributions

2. Related Work                          ~1.5 hal
   2.1 Multi-Agent Systems for Trading
   2.2 Regime Detection in Financial Markets
   2.3 Knowledge Transfer in MAS
   2.4 Gap Analysis (tabel komparasi)

3. Proposed Framework                    ~3.5 hal
   3.1 System Overview
   3.2 Market Regime Formalization
   3.3 RegimeDetectionAgent
   3.4 MetaObserverAgent
   3.5 KnowledgeTransferAgent
   3.6 Adaptive Decision Engine
   3.7 Knowledge Rule Lifecycle

4. Experimental Setup                    ~1.5 hal
   4.1 Dataset
   4.2 Baseline Configurations
   4.3 Evaluation Metrics
   4.4 Implementation Details

5. Results and Discussion                ~3.0 hal
   5.1 Overall Performance Comparison
   5.2 Per-Regime Analysis
   5.3 Knowledge Rule Analysis
   5.4 Ablation Study
   5.5 Qualitative Example
   5.6 Threats to Validity

6. Conclusion                            ~0.5 hal

References                               ~1.0 hal (20–25 referensi)
```

---

## 3. Section 1 — Introduction

### Hook Pembuka (paragraf 1)

Mulai dengan kontradiksi yang belum pernah diangkat di paper lain:

```
Existing multi-agent frameworks for financial trading — including
TradingAgents [Xiao et al., 2024], FinCon [Yu et al., 2024], and
ATLAS [2025] — have demonstrated that decomposing trading decisions
across specialized agents improves performance over single-agent
approaches. However, these frameworks share a critical blind spot:
when an agent fails, the system responds by reducing that agent's
influence weight. It does not ask *why* the agent failed. It does
not extract the failure's cause. And it does not share that knowledge
with other agents that may be heading toward the same failure.
```

### Problem Statement (paragraf 2)

```
Consider the following scenario. In January, EUR/USD is trending
strongly and a TrendAgent achieves 78% accuracy. In February,
the market transitions to a ranging regime. The same TrendAgent's
accuracy collapses to 38%. A conventional system reduces the
TrendAgent's weight — a punishment without understanding. The
RangeAgent receives no information explaining *why* the TrendAgent
failed, what market conditions triggered the failure, or what
constraints it should apply to avoid similar errors. This represents
a fundamental learning gap in existing multi-agent trading systems.
```

### Research Questions

```
This paper addresses three research questions:

RQ1: How can a multi-agent forex system formally detect market regime
     transitions and use them to condition agent behavior?

RQ2: How can agent performance degradation be monitored, diagnosed,
     and represented as structured knowledge (ExperienceReports)?

RQ3: Can LLM-based reasoning extract causal knowledge from agent
     failures and transfer it as actionable rules (KnowledgeRules)
     to improve collective system performance?
```

### Contributions (bullet 3 poin)

```
The main contributions of this paper are:

(1) A RegimeDetectionAgent that classifies forex market conditions
    into five formal regimes using a combination of ADX, ATR, and
    Bollinger Band width, enabling regime-conditioned agent activation.

(2) A MetaObserverAgent that continuously monitors per-agent accuracy
    within each market regime, detects degradation events using rolling
    windows, and generates structured ExperienceReports capturing
    failure context.

(3) A KnowledgeTransferAgent that employs LLM-based causal reasoning
    to transform ExperienceReports into KnowledgeRules broadcast across
    the agent pool — enabling collective adaptation beyond simple
    weight adjustment. To our knowledge, this is the first application
    of LLM-driven inter-agent knowledge transfer in forex trading MAS.
```

---

## 4. Section 2 — Related Work

### 2.1 Multi-Agent Systems for Trading

Tulis 2–3 paragraf merangkum:
- TradingAgents (Xiao et al., 2024) — debate mechanism
- FinCon (Yu et al., 2024) — verbal reinforcement
- ATLAS (2025) — dynamic prompt optimization
- HedgeAgents (Li et al., 2025) — balanced portfolio

### 2.2 Regime Detection in Financial Markets

- ADX-based regime classification (standar industri)
- Hidden Markov Models untuk regime switching
- DRL-based adaptive agents (Sarani et al., 2024)

### 2.3 Knowledge Transfer in MAS

- Verbal feedback untuk LLM adaptation (Bitcoin trading paper, 2025)
- Multi-agent reinforcement learning knowledge sharing
- Memory-augmented agents (FinMem, 2024)

### 2.4 Gap Analysis — TABEL WAJIB ADA

```
Tabel 1. Perbandingan framework yang ada dengan proposed system

| Framework        | MAS | LLM | Forex | Regime | MetaObs | KnowTransfer |
|-----------------|-----|-----|-------|--------|---------|--------------|
| TradingAgents   |  ✓  |  ✓  |  ✗   |   ✗    |    ✗    |      ✗       |
| FinCon          |  ✓  |  ✓  |  ✗   |   ✗    |    ✗    |      ✗       |
| ATLAS           |  ✓  |  ✓  |  ✗   |   ✗    |    ✗    |      ✗       |
| HedgeAgents     |  ✓  |  ✓  |  ✗   |   ✗    |    ✗    |      ✗       |
| QuantAgent      |  ✓  |  ✓  |  ✗   |   partial|  ✗    |      ✗       |
| DRL-Forex [29]  |  ✓  |  ✗  |  ✓   |   ✗    |    ✗    |      ✗       |
| **Proposed**    |  ✓  |  ✓  |  ✓   |   ✓    |    ✓    |      ✓       |

Keterangan:
MAS = multi-agent architecture
LLM = large language model integration
Forex = tested on forex market (bukan saham)
Regime = formal regime detection
MetaObs = agent performance monitoring
KnowTransfer = inter-agent knowledge transfer
```

Tutup Related Work dengan:

```
As shown in Table 1, no existing framework combines formal regime
detection, agent meta-observation, and LLM-driven knowledge transfer
in the context of forex trading. This paper addresses this gap.
```

---

## 5. Section 3 — Methodology

### 3.1 System Overview

Sertakan diagram arsitektur lengkap. Caption:

```
Figure 1. Proposed self-adaptive multi-agent framework architecture.
Gray nodes represent existing components; purple nodes represent
novel contributions (★). Dashed arrows indicate the knowledge
feedback loop from KnowledgeTransferAgent back to the agent pool.
```

### 3.2 Market Regime Formalization

Definisikan regime sebagai fungsi matematis:

```
Definition 1 (Market Regime). A market regime R ∈ {Trending, Ranging,
Breakout, HighVol, LowVol} is a discrete state determined by:

R = classify(ADX_14, ATR_14, BBW_20)

where:
  ADX_14   = Average Directional Index (14 periods)
  ATR_14   = Average True Range (14 periods)
  BBW_20   = Bollinger Band Width (20 periods, σ=2)
            = (Upper - Lower) / Middle

Classification rules:
  IF ATR/price > θ_vol × 1.8 AND BBW > 0.03  → Breakout
  IF ADX > θ_adx AND ATR/price > θ_vol        → Trending
  IF ADX > θ_adx AND ATR/price ≤ θ_vol        → LowVol
  IF ADX ≤ θ_adx AND ATR/price > θ_vol × 1.5 → HighVol
  ELSE                                          → Ranging

dengan θ_adx = 25, θ_vol = 0.015 (dikalibrasi dari data training)
```

### 3.3 ExperienceReport Formalization

```
Definition 2 (ExperienceReport). For agent α and time window W:

E(α, W) = {
  agent:   α,
  acc_t:   Accuracy(α, W),
  acc_t-1: Accuracy(α, W-1),
  Δacc:    acc_t - acc_t-1,
  streak:  ConsecutiveLoss(α),
  regime:  R_dominant(W),
  cause:   LLM_reason(α, R, Δacc)
}

E(α, W) is generated when:
  Δacc < -τ_drop (default: -0.20)
  OR streak ≥ τ_streak (default: 4)
```

### 3.4 KnowledgeRule Formalization

```
Definition 3 (KnowledgeRule). A rule κ generated from E(α, W):

κ = {
  condition: (R, C_opt),
  action:    (α_target, Δw, w_min),
  conf:      LLM_confidence ∈ [0,1],
  ttl:       24 hours
}

where:
  C_opt  = optional constraints (ADX < threshold, vol < threshold)
  α_target = agent whose weight should be modified
  Δw     = weight delta (always negative: Δw ∈ [-0.5, -0.1])
  w_min  = minimum weight floor (default: 0.05)
```

### 3.5 Adaptive Weight Function

Ini formula kunci yang membedakan dari sistem static:

```
Adaptive weight untuk agen α pada siklus t:

w_α(t) = clip(w_base(α) + Σ Δw_κ(α, R_t), w_min, w_max)

dimana:
  w_base(α) = bobot default dari konfigurasi
  Δw_κ       = weight delta dari rule κ yang berlaku
  R_t         = regime aktif pada waktu t
  clip(x, a, b) = max(a, min(x, b))

Setelah semua bobot dihitung, normalisasi:
  w_norm(α) = w_α(t) / Σ_β w_β(t)
```

### 3.6 LLM Role dalam KTA

Penting untuk dijelaskan secara eksplisit agar reviewer tidak salah paham:

```
Penting: LLM dalam framework ini TIDAK digunakan untuk prediksi harga.
LLM digunakan semata-mata sebagai reasoning engine untuk:

(1) Menganalisis *mengapa* sebuah agen gagal berdasarkan konteks regime
(2) Mengekstrak hubungan kausal: "IF regime = X THEN agent Y likely fails"
(3) Menghasilkan KnowledgeRule dalam format JSON terstruktur

Input ke LLM: ExperienceReport (JSON)
Output dari LLM: KnowledgeRule (JSON)

Pendekatan ini memisahkan prediction dari reasoning — sebuah
pembagian tanggung jawab yang lebih tepat secara arsitektural.
```

---

## 6. Section 4 — Experimental Setup

### 4.1 Dataset

```
Tabel 2. Konfigurasi dataset eksperimen

| Properti         | Detail                              |
|-----------------|-------------------------------------|
| Currency pairs   | EUR/USD, GBP/USD, USD/JPY           |
| Timeframe        | M5 (5-menit candle)                 |
| Periode          | Januari 2024 – Juni 2024 (6 bulan) |
| Jumlah candle    | ~51,840 per pair (8,640/bulan)      |
| Sumber data      | OANDA REST API + Twelve Data        |
| Train period     | Jan–Mar 2024 (knowledge building)   |
| Test period      | Apr–Jun 2024 (evaluation)           |
| Normalisasi      | Min-max per window 200 candle       |
```

> **Alasan pemilihan 2024:** Data 2024 mencakup periode market yang variatif —
> ada trending kuat (Q1), ranging (Feb), dan beberapa breakout event (Mar-Apr).
> Ini memastikan semua lima regime terwakili.

### 4.2 Tiga Baseline

```
Baseline A — Static MAS (sistem existing kamu):
  - Agen: MarketData + Technical + Fundamental + Risk + Decision + WhatsApp
  - Bobot: static (Technical=0.60, Fundamental=0.40)
  - Tanpa regime detection
  - Tanpa MetaObserver
  - Tanpa knowledge transfer

Baseline B — Regime-Aware Static MAS:
  - Tambah RegimeDetectionAgent di atas Baseline A
  - Bobot agen disesuaikan per regime (hard-coded, bukan dari KB)
  - Tanpa MetaObserver
  - Tanpa knowledge transfer

Proposed — Full Self-Adaptive MAS:
  - Semua agen Baseline B
  - Tambah MetaObserverAgent
  - Tambah KnowledgeTransferAgent
  - Adaptive weights dari KnowledgeBase
```

Tiga baseline ini penting untuk:
- Baseline A vs B: buktikan regime detection berkontribusi
- Baseline B vs Proposed: buktikan knowledge transfer berkontribusi
- Ablation study yang clean dan tidak bisa dibantah reviewer

### 4.3 Metrik Evaluasi

```
Tabel 3. Metrik evaluasi dan definisinya

| Metrik          | Formula / Definisi                          | Arah  |
|----------------|----------------------------------------------|-------|
| Win Rate (WR)  | TP / (TP + FP), signal dianggap benar        | ↑     |
|                | jika harga bergerak ≥ 15 pip ke arah sinyal  |       |
| Profit Factor  | Gross Profit / Gross Loss                    | ↑     |
| Sharpe Ratio   | (R_p - R_f) / σ_p, R_f = 0 (paper trading)  | ↑     |
| Max Drawdown   | Penurunan equity terbesar dari puncak        | ↓     |
| Calmar Ratio   | Annualized Return / Max Drawdown             | ↑     |
| Rule Coverage  | % sinyal yang dipengaruhi ≥1 KnowledgeRule   | info  |
| Rule Accuracy  | % KnowledgeRule yang terbukti meningkatkan WR| ↑     |
```

### 4.4 Implementation Details

```
Tabel 4. Detail implementasi sistem

| Komponen           | Teknologi                        |
|-------------------|----------------------------------|
| Core framework     | Go 1.25                          |
| LLM (primary)      | Gemini 2.0 Flash                 |
| LLM (fallback)     | Groq Llama 3.3 70B               |
| Knowledge store    | Redis 7 (rule cache)             |
| Time-series DB     | TimescaleDB (PostgreSQL 16)      |
| Technical indicators| RSI(14), MACD(12,26,9), EMA(50,200), BB(20,2) |
| Regime detection   | ADX(14), ATR(14), BB Width(20)   |
| Signal evaluation  | 30-menit delay, pip threshold=15 |
| Hardware           | [isi dengan spec kamu]           |
| Backtesting period | Apr–Jun 2024 (3 bulan test)      |
```

---

## 7. Section 5 — Results & Discussion

### 5.1 Overall Performance — TABEL UTAMA

```
Tabel 5. Perbandingan performa keseluruhan (Apr–Jun 2024)

| Metrik          | Baseline A | Baseline B | Proposed | Δ (A→P) |
|----------------|-----------|-----------|---------|---------|
| Win Rate (%)   |   53.1    |   58.4    |  67.3   |  +14.2% |
| Profit Factor  |   1.12    |   1.31    |  1.68   |  +0.56  |
| Sharpe Ratio   |   0.84    |   1.21    |  1.84   |  +1.00  |
| Max Drawdown   |  14.2%    |  11.8%    |   8.7%  |  -5.5%  |
| Calmar Ratio   |   0.91    |   1.15    |  1.74   |  +0.83  |
| Total signals  |   847     |   712     |   689   |   —     |

* Angka ini adalah ilustrasi. Isi dengan hasil eksperimen aktual.
* Proposed menghasilkan lebih sedikit sinyal karena filtering yang lebih ketat.
```

### 5.2 Per-Regime Analysis — TABEL PALING UNIK DI PAPER

```
Tabel 6. Win rate per market regime (Proposed vs Baseline A)

| Regime       | Freq (%) | Baseline A | Proposed | Delta  |
|-------------|---------|-----------|---------|--------|
| Trending    |  31.2   |   61.4%   |  74.2%  | +12.8% |
| Ranging     |  28.7   |   44.1%   |  63.8%  | +19.7% |
| Breakout    |  12.4   |   49.3%   |  58.1%  |  +8.8% |
| High Vol    |  18.9   |   51.2%   |  64.7%  | +13.5% |
| Low Vol     |   8.8   |   55.6%   |  71.3%  | +15.7% |

Insight untuk diskusi:
- Delta terbesar di Ranging (+19.7%) — ini membuktikan hypothesis utama:
  KnowledgeRule paling efektif menghukum TrendAgent saat pasar ranging.
- Breakout delta paling kecil (+8.8%) — regime ini paling sulit diprediksi,
  bahkan dengan knowledge transfer. Ini honest dan memperkuat credibility.
```

### 5.3 Knowledge Rule Analysis

```
Tabel 7. Analisis KnowledgeRule yang dihasilkan selama eksperimen

| Statistik                          | Nilai  |
|-----------------------------------|--------|
| Total rules generated             |   47   |
| Rules applied (confidence ≥ 0.60) |   38   |
| Rules expired without use         |    9   |
| Average rule confidence           |  0.74  |
| Rules that improved win rate      |   31   |
| Rule accuracy (31/38)             | 81.6%  |
| Most common condition regime      | Ranging|
| Most common target agent          | TrendAgent |
| Average weight delta applied      |  -0.23 |
```

Tambahkan 3–5 contoh rule nyata dari eksperimen. Ini sangat meyakinkan reviewer:

```
Tabel 8. Contoh KnowledgeRules yang dihasilkan sistem

| Rule ID | Condition              | Action               | Conf | Reasoning (ringkas)              |
|--------|------------------------|---------------------|------|----------------------------------|
| R-001  | Ranging, ADX < 20      | TrendAgent: -0.35   | 0.88 | Trend assumptions invalid in range |
| R-014  | LowVol, ATR < 0.0012   | TrendAgent: -0.28   | 0.79 | Low momentum, trend signals noise  |
| R-022  | HighVol, BBW > 0.04    | FundAgent: -0.15    | 0.71 | Sentiment lag during volatility    |
| R-033  | Ranging                | RangeAgent: +0.0*   | 0.82 | Range conditions favor oscillators |
| R-041  | Breakout, ADX > 30     | TechnicalAgent: +0  | 0.66 | Technical confirmation sufficient  |

* KTA hanya mengurangi bobot agen gagal. Weight positif terjadi karena
  normalisasi: saat TrendAgent dikurangi, RangeAgent otomatis naik relatif.
```

### 5.4 Ablation Study

```
Tabel 9. Ablation study — kontribusi setiap komponen

| Konfigurasi                           | Win Rate | Sharpe |
|--------------------------------------|---------|--------|
| Full system (proposed)               |  67.3%  |  1.84  |
| Tanpa KnowledgeTransferAgent         |  61.2%  |  1.34  |
| Tanpa MetaObserverAgent              |  59.7%  |  1.22  |
| Tanpa RegimeDetectionAgent           |  58.4%  |  1.21  |
| Baseline A (tanpa semua 3 komponen)  |  53.1%  |  0.84  |

Cara baca: hapus satu komponen, ukur dampaknya.
Ini membuktikan setiap komponen berkontribusi secara independen.
```

### 5.5 Qualitative Example — BUAT PEMBACA BISA "MERASAKAN" SISTEMNYA

Ini bagian yang membuat paper berkesan dan dibaca sampai habis:

```
Figure 2. Contoh knowledge transfer event pada 15 Februari 2024

Konteks: EUR/USD masuk regime Ranging (ADX = 18.3, ATR/price = 0.009)

08:00 WIB — TrendAgent menghasilkan sinyal BUY (confidence 72%)
           berdasarkan EMA crossover.
           
08:30 WIB — Sinyal dievaluasi: INCORRECT. Harga bergerak sideways.
           MetaObserver mencatat: TrendAgent loss_streak = 4

08:35 WIB — MetaObserver menghasilkan ExperienceReport:
           { agent: "TrendAgent", acc_delta: -0.31, regime: "Ranging",
             loss_streak: 4 }

08:36 WIB — KnowledgeTransferAgent mengirim ke Gemini:
           "TrendAgent accuracy dropped 31% during Ranging regime..."
           
08:36 WIB — Gemini menghasilkan KnowledgeRule R-001:
           { condition: {regime: "Ranging", adx_below: 20},
             action: {agent: "TrendAgent", weight_delta: -0.35},
             confidence: 0.88,
             reasoning: "Trend-following assumptions invalid when 
                          directional movement is absent (ADX < 20)" }

09:00 WIB — Sinyal berikutnya: TechnicalAgent bobot = 0.60 → 0.39
           (setelah normalisasi). RangeAgent relatif naik.
           DecisionAgent memilih HOLD (tidak cukup confidence).
           
10:00 WIB — Pasar masih ranging. Keputusan HOLD terbukti benar.
           Tanpa knowledge transfer: sistem mungkin membeli lagi
           berdasarkan sinyal TrendAgent yang masih aktif.
```

### 5.6 Threats to Validity

Ini penting untuk credibility — jangan disembunyikan:

```
Internal validity:
- Evaluasi sinyal menggunakan pip threshold fixed (15 pip) yang
  mungkin tidak optimal untuk semua regime dan pair.
- MetaObserver menggunakan rolling window 20 sinyal — angka ini
  dikalibrasi secara empiris, bukan dari teori formal.

External validity:
- Eksperimen terbatas pada 3 major pairs (EUR/USD, GBP/USD, USD/JPY).
  Generalisasi ke exotic pairs atau cryptocurrency belum diuji.
- Data 6 bulan mungkin tidak cukup untuk menangkap semua siklus regime.

Construct validity:
- "Sinyal benar" didefinisikan sebagai harga bergerak ≥ 15 pip dalam
  30 menit. Definisi ini mungkin tidak selaras dengan profitabilitas nyata
  karena tidak memperhitungkan spread dan slippage.
```

---

## 8. Section 6 — Conclusion

```
Paragraf 1 (restatement kontribusi):
This paper presented a self-adaptive multi-agent framework for forex
trading that addresses the fundamental limitation of existing systems:
the inability to learn *why* an agent fails and transfer that
knowledge collectively. Through three novel components — 
RegimeDetectionAgent, MetaObserverAgent, and KnowledgeTransferAgent —
the proposed framework achieves [angka win rate] win rate, a [angka]
Sharpe ratio, and [angka]% maximum drawdown on a six-month backtest
across three major currency pairs, outperforming all baselines.

Paragraf 2 (implikasi praktis):
The results demonstrate that LLM-based causal reasoning, when applied
to agent failure analysis rather than price prediction, provides a
qualitatively different and measurably superior adaptation mechanism.
The system is deployable in real-time environments, as demonstrated
by the Go-based implementation with sub-second signal generation
latency and WhatsApp delivery integration.

Paragraf 3 (future work):
Future work includes: (1) extending the evaluation to exotic currency
pairs and cryptocurrency markets; (2) incorporating multi-timeframe
regime detection (M5, M15, H1) for hierarchical regime classification;
(3) exploring online learning for KnowledgeRule refinement beyond
TTL-based expiration; and (4) developing a formal theoretical framework
for KnowledgeRule convergence under non-stationary market conditions.
```

---

## 9. Desain Eksperimen Detail

### Timeline Backtest

```
Phase 1 — Knowledge Building (Jan–Mar 2024):
  - Jalankan full pipeline
  - Biarkan MetaObserver membangun baseline accuracy per agent
  - KTA mulai menghasilkan rules setelah minggu ke-3
  - Simpan semua ExperienceReport dan KnowledgeRule ke Postgres
  - Jangan evaluasi performa di sini — ini "warmup"

Phase 2 — Evaluation (Apr–Jun 2024):
  - Jalankan tiga konfigurasi secara PARALEL di tiga instance terpisah
  - Baseline A: existing system kamu
  - Baseline B: tambah RegimeDetectionAgent saja
  - Proposed: full system
  - Setiap instance catat sinyal ke tabel berbeda di Postgres

Phase 3 — Analysis (setelah backtest selesai):
  - Query SQL untuk hitung semua metrik
  - Export ke CSV untuk grafik
```

### Query SQL untuk Metrik Utama

```sql
-- 1. Win Rate per sistem
SELECT
  system_name,
  COUNT(*) AS total_signals,
  SUM(CASE WHEN correct THEN 1 ELSE 0 END) AS wins,
  ROUND(AVG(CASE WHEN correct THEN 1.0 ELSE 0.0 END) * 100, 2) AS win_rate_pct
FROM backtest_signals
WHERE eval_time BETWEEN '2024-04-01' AND '2024-07-01'
GROUP BY system_name;

-- 2. Win Rate per Regime (proposed system)
SELECT
  regime,
  COUNT(*) AS total,
  ROUND(AVG(CASE WHEN correct THEN 1.0 ELSE 0.0 END) * 100, 2) AS win_rate,
  COUNT(*) * 100.0 / SUM(COUNT(*)) OVER () AS regime_freq_pct
FROM backtest_signals
WHERE system_name = 'proposed'
  AND eval_time BETWEEN '2024-04-01' AND '2024-07-01'
GROUP BY regime
ORDER BY regime_freq_pct DESC;

-- 3. Knowledge Rule statistics
SELECT
  COUNT(*) AS total_rules,
  AVG(confidence) AS avg_confidence,
  SUM(apply_count) AS total_applications,
  condition->>'regime' AS dominant_regime
FROM knowledge_rules
WHERE created_at BETWEEN '2024-04-01' AND '2024-07-01'
GROUP BY condition->>'regime'
ORDER BY total_rules DESC;

-- 4. Ablation: accuracy sebelum dan sesudah rule diterapkan
SELECT
  source_agent,
  AVG(accuracy_before) AS avg_acc_before,
  AVG(accuracy_now) AS avg_acc_now,
  AVG(accuracy_delta) AS avg_delta,
  COUNT(*) AS n_reports
FROM experience_reports
WHERE created_at BETWEEN '2024-04-01' AND '2024-07-01'
GROUP BY source_agent;

-- 5. Sharpe Ratio (hitung equity curve dulu)
WITH daily_returns AS (
  SELECT
    DATE(eval_time) AS day,
    system_name,
    SUM(CASE WHEN correct THEN 0.01 ELSE -0.01 END) AS daily_return
  FROM backtest_signals
  WHERE eval_time BETWEEN '2024-04-01' AND '2024-07-01'
  GROUP BY DATE(eval_time), system_name
)
SELECT
  system_name,
  AVG(daily_return) / NULLIF(STDDEV(daily_return), 0) * SQRT(252) AS sharpe_ratio
FROM daily_returns
GROUP BY system_name;
```

### Cara Jalankan 3 Baseline Paralel

Gunakan Docker Compose dengan 3 profile:

```yaml
# docker-compose.backtest.yml

services:
  baseline-a:
    build: .
    environment:
      - SYSTEM_NAME=baseline_a
      - ENABLE_REGIME=false
      - ENABLE_META_OBSERVER=false
      - ENABLE_KTA=false
      - DB_TABLE_PREFIX=bla_
    
  baseline-b:
    build: .
    environment:
      - SYSTEM_NAME=baseline_b
      - ENABLE_REGIME=true
      - ENABLE_META_OBSERVER=false
      - ENABLE_KTA=false
      - DB_TABLE_PREFIX=blb_
    
  proposed:
    build: .
    environment:
      - SYSTEM_NAME=proposed
      - ENABLE_REGIME=true
      - ENABLE_META_OBSERVER=true
      - ENABLE_KTA=true
      - DB_TABLE_PREFIX=prop_
```

Tambahkan flag di `config.yaml`:

```yaml
experiment:
  system_name: "proposed"     # ganti per instance
  enable_regime: true
  enable_meta_observer: true
  enable_kta: true
  table_prefix: "prop_"
```

---

## 10. Cara Mengumpulkan Data Eksperimen

### Langkah 1: Siapkan historical data

```bash
# Download M5 candles untuk 3 pair, Jan–Jun 2024
# Gunakan OANDA Historical Data API

curl -H "Authorization: Bearer $OANDA_API_KEY" \
  "https://api-fxtrade.oanda.com/v3/instruments/EUR_USD/candles?
   count=51840&granularity=M5&from=2024-01-01T00:00:00Z" \
  > data/eurusd_m5_2024.json

# Repeat untuk GBP_USD dan USD_JPY
```

### Langkah 2: Buat backtesting mode

Tambahkan flag `--backtest` ke `main.go`:

```go
// Di main.go
var backtestMode = flag.Bool("backtest", false, "run in backtest mode")
var backtestFile = flag.String("data", "", "path to historical data JSON")

// Di mode backtest, ganti WebSocket feed dengan file reader
// Semua logika agent tetap sama — hanya sumber datanya berbeda
```

### Langkah 3: Catat semua sinyal ke Postgres

```go
// Setiap kali ada sinyal, simpan ke DB:
type BacktestSignal struct {
    SystemName  string
    Pair        string
    Direction   string  // BUY / SELL / HOLD
    EntryPrice  float64
    Regime      string
    TechWeight  float64
    FundWeight  float64
    Confidence  float64
    SignalTime  time.Time
    // Diisi setelah evaluasi:
    ExitPrice   float64
    Correct     bool
    EvalTime    time.Time
}
```

### Langkah 4: Evaluasi sinyal setelah 30 menit

```go
// Goroutine evaluator: setiap sinyal dievaluasi 30 menit kemudian
// Sinyal BUY dianggap benar jika harga naik ≥ 15 pip dalam 30 menit
// Sinyal SELL dianggap benar jika harga turun ≥ 15 pip dalam 30 menit

func evaluateSignal(signal BacktestSignal, currentPrice float64) bool {
    pipValue := 0.0001 // untuk 4-digit pairs (EUR/USD, GBP/USD)
    if signal.Pair == "USD_JPY" {
        pipValue = 0.01
    }
    threshold := 15 * pipValue
    
    switch signal.Direction {
    case "BUY":
        return currentPrice - signal.EntryPrice >= threshold
    case "SELL":
        return signal.EntryPrice - currentPrice >= threshold
    default:
        return false
    }
}
```

---

## 11. Template Tabel & Grafik

### Grafik 1: Win Rate per Regime (Bar Chart)

```
Sumbu X: Regime (Trending, Ranging, Breakout, HighVol, LowVol)
Sumbu Y: Win Rate (%)
Series: Baseline A (abu-abu), Baseline B (biru muda), Proposed (biru tua)

Tool: matplotlib Python atau Excel
Simpan sebagai: results/fig_winrate_per_regime.png (300 DPI)
```

### Grafik 2: Equity Curve (Line Chart)

```
Sumbu X: Waktu (April–Juni 2024)
Sumbu Y: Cumulative return (%)
Series: 3 garis untuk Baseline A, B, dan Proposed
Highlight: titik di mana knowledge rule pertama kali diterapkan

Tool: matplotlib
Simpan sebagai: results/fig_equity_curve.png (300 DPI)
```

### Grafik 3: Knowledge Rule Timeline

```
Sumbu X: Waktu
Sumbu Y: Jumlah active rules
Color: per regime (warna berbeda per regime)

Ini menunjukkan bagaimana KB tumbuh dan berkontraksi seiring waktu.
```

### Python Script untuk Generate Grafik

```python
# analysis/plot_results.py
import pandas as pd
import matplotlib.pyplot as plt
import psycopg2

conn = psycopg2.connect("postgresql://user:pass@localhost/forex")

# Win rate per regime
df = pd.read_sql("""
    SELECT system_name, regime,
           AVG(CASE WHEN correct THEN 1.0 ELSE 0.0 END) * 100 as win_rate
    FROM backtest_signals
    WHERE eval_time BETWEEN '2024-04-01' AND '2024-07-01'
    GROUP BY system_name, regime
""", conn)

pivot = df.pivot(index='regime', columns='system_name', values='win_rate')
pivot.plot(kind='bar', figsize=(10, 6), color=['#aaaaaa', '#6699cc', '#003366'])
plt.ylabel('Win Rate (%)')
plt.title('Win Rate per Market Regime')
plt.legend(['Baseline A', 'Baseline B', 'Proposed'])
plt.tight_layout()
plt.savefig('results/fig_winrate_per_regime.png', dpi=300)
```

---

## 12. Referensi Wajib

Minimal 20 referensi. Yang wajib ada:

```
[1]  Xiao, Y., et al. "TradingAgents: Multi-Agents LLM Financial 
     Trading Framework." AAAI 2025. arXiv:2412.20138

[2]  Yu, Y., et al. "FinCon: A Synthesized LLM Multi-Agent System 
     with Conceptual Verbal Reinforcement." NeurIPS 2024.

[3]  Li, X., et al. "HedgeAgents: A Balanced-Aware Multi-Agent 
     Financial Trading System." WWW 2025.

[4]  Wu, S., et al. "MountainLion: A Multi-Modal LLM-Based Agent 
     System for Financial Trading." arXiv 2025.

[5]  Sarani, D., et al. "A Deep Reinforcement Learning Approach for 
     Trading Optimization in the Forex Market." arXiv:2405.19982, 2024.

[6]  Fatemi, S., Hu, Y. "FinVision: A Multi-Agent Framework for 
     Stock Market Prediction." ICAIF 2024.

[7]  Yu, Y., et al. "FinMem: A Performance-Enhanced LLM Trading Agent 
     with Layered Memory." IEEE Transactions on Big Data, 2025.

[8]  [Bitcoin Adaptive MAS] "An Adaptive Multi Agent Bitcoin 
     Trading System." arXiv:2510.08068, 2025.

[9]  Wawer, M., Chudziak, B. "Integrating Traditional Technical 
     Analysis with AI: A Multi-Agent LLM Approach." ICAART 2025.

[10] Hernandez-Aguila, A., et al. "Using Fuzzy Inference Systems 
     for the Creation of Forex Market Predictive Models." 
     IEEE Access, 2021.

[11] Wilder, J.W. "New Concepts in Technical Trading Systems." 
     Trend Research, 1978. (referensi ADX)

[12] Bollinger, J. "Bollinger on Bollinger Bands." 
     McGraw-Hill, 2002. (referensi Bollinger Band)

[13] Zhang, H., et al. "A Comprehensive Survey on Multi-Agent 
     Reinforcement Learning." IEEE TNNLS, 2024.

[14] Guo, T., et al. "Large Language Model Based Multi-Agents: 
     A Survey of Progress and Challenges." IJCAI 2024.

[15] Google. "Gemini: A Family of Highly Capable Multimodal Models." 
     Technical Report, 2024.

[16] [Groq/Llama 3.3 reference]
     Meta AI. "Llama 3: Open Foundation and Fine-Tuned Chat Models."
     arXiv, 2024.

[17] TimescaleDB. "Time-Series Database for PostgreSQL." 
     https://www.timescale.com, 2024.

[18] [Tambahkan referensi teknis Go/Redis sesuai kebutuhan]

[19] Sharpe, W.F. "The Sharpe Ratio." 
     Journal of Portfolio Management, 1994.

[20] [Satu paper tentang knowledge transfer in MAS — cari di Scholar]
```

---

## 13. Timeline Penulisan

```
Minggu 1–2: Implementasi & Backtest
  [ ] Implementasi 3 agen baru (lihat TECHNICAL_IMPLEMENTATION.md)
  [ ] Siapkan historical data Jan–Jun 2024
  [ ] Jalankan backtest 3 konfigurasi paralel
  [ ] Kumpulkan semua data ke Postgres

Minggu 3: Analisis Data
  [ ] Jalankan query SQL untuk semua metrik
  [ ] Buat grafik dengan Python/matplotlib
  [ ] Identifikasi 3–5 KnowledgeRule terbaik untuk contoh di paper
  [ ] Draft tabel hasil lengkap

Minggu 4–5: Penulisan Draft
  [ ] Section 1 Introduction (1–2 hari)
  [ ] Section 2 Related Work + gap table (2–3 hari)
  [ ] Section 3 Methodology + diagram + formula (3–4 hari)
  [ ] Section 4 Experimental Setup (1 hari)

Minggu 6: Penulisan Draft (lanjutan)
  [ ] Section 5 Results + semua tabel + grafik (3–4 hari)
  [ ] Section 6 Conclusion (0.5 hari)
  [ ] Abstrak final (0.5 hari)
  [ ] Referensi lengkap

Minggu 7: Review & Polish
  [ ] Cek seluruh formula matematika
  [ ] Cek konsistensi notasi di seluruh paper
  [ ] Minta peer review dari rekan
  [ ] Perbaiki berdasarkan feedback

Minggu 8: Submission
  [ ] Format sesuai template jurnal target (IEEE/JNTETI/dll)
  [ ] Cek plagiarism (target < 15%)
  [ ] Tulis cover letter
  [ ] Submit
```

---

*Dokumen ini adalah panduan hidup — update setiap kali ada perubahan metodologi atau hasil eksperimen.*

*Versi: 1.0 | Dibuat untuk target jurnal Sinta 2*