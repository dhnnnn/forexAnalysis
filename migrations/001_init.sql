-- ════════════════════════════════════════════════════════════════
-- Forex Multi-Agent Bot — Database Schema
-- TimescaleDB Migration #001
-- ════════════════════════════════════════════════════════════════

-- Aktifkan TimescaleDB extension
CREATE EXTENSION IF NOT EXISTS timescaledb;

-- ────────────────────────────────────────────────────────────────
-- Tabel candles — Data OHLCV (hypertable, partisi per hari)
-- ────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS candles (
    time        TIMESTAMPTZ      NOT NULL,
    pair        TEXT             NOT NULL,
    timeframe   TEXT             NOT NULL DEFAULT '1h',
    open        DOUBLE PRECISION NOT NULL,
    high        DOUBLE PRECISION NOT NULL,
    low         DOUBLE PRECISION NOT NULL,
    close       DOUBLE PRECISION NOT NULL,
    volume      DOUBLE PRECISION,
    spread      DOUBLE PRECISION,
    PRIMARY KEY (time, pair, timeframe)
);

SELECT create_hypertable('candles', 'time', if_not_exists => TRUE);
CREATE INDEX IF NOT EXISTS idx_candles_pair_tf ON candles (pair, timeframe, time DESC);

-- ────────────────────────────────────────────────────────────────
-- Tabel signals — Sinyal yang dihasilkan Decision Agent
-- ────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS signals (
    id              SERIAL,
    time            TIMESTAMPTZ      NOT NULL DEFAULT NOW(),
    pair            TEXT             NOT NULL,
    direction       TEXT             NOT NULL,
    confidence      DOUBLE PRECISION,
    tech_score      DOUBLE PRECISION,
    tech_signal     TEXT,
    fund_sentiment  TEXT,
    fund_score      DOUBLE PRECISION,
    ml_score        DOUBLE PRECISION,
    risk_level      TEXT,
    lot_size        DOUBLE PRECISION,
    entry_price     DOUBLE PRECISION,
    stop_loss       DOUBLE PRECISION,
    take_profit     DOUBLE PRECISION,
    sl_pips         DOUBLE PRECISION,
    tp_pips         DOUBLE PRECISION,
    tech_reason     TEXT,
    fund_reason     TEXT,
    PRIMARY KEY (time, id)
);

SELECT create_hypertable('signals', 'time', if_not_exists => TRUE);

-- ────────────────────────────────────────────────────────────────
-- Tabel news_cache — Cache berita (opsional, Phase 2)
-- ────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS news_cache (
    hash        VARCHAR(64)  PRIMARY KEY,
    pair        VARCHAR(10),
    headlines   TEXT,
    sentiment   VARCHAR(10),
    confidence  DOUBLE PRECISION,
    created_at  TIMESTAMPTZ  DEFAULT NOW()
);
