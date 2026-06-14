-- ════════════════════════════════════════════════════════════════════════
-- Migration 002: Knowledge Transfer System Tables
-- ════════════════════════════════════════════════════════════════════════
-- Tabel-tabel ini menyimpan data untuk:
-- 1. ExperienceReport (kegagalan agen yang terdeteksi MetaObserver)
-- 2. KnowledgeRule history (rules yang dihasilkan KTA)
-- 3. Agent performance log (tracking accuracy per regime)
-- ════════════════════════════════════════════════════════════════════════

-- ── Tabel 1: Experience Reports ───────────────────────────────────────
-- Menyimpan setiap kali MetaObserver mendeteksi degradasi performa agen.
-- Digunakan untuk analisis dan paper.
CREATE TABLE IF NOT EXISTS experience_reports (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_name      VARCHAR(50) NOT NULL,
    pair            VARCHAR(10),
    accuracy_before DECIMAL(5,4),
    accuracy_now    DECIMAL(5,4),
    accuracy_delta  DECIMAL(5,4),
    loss_streak     INTEGER,
    active_regime   VARCHAR(20),
    cause           TEXT,
    reasoning       TEXT,
    created_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_exp_reports_agent ON experience_reports(agent_name, created_at);
CREATE INDEX IF NOT EXISTS idx_exp_reports_regime ON experience_reports(active_regime);

-- ── Tabel 2: Knowledge Rules ──────────────────────────────────────────
-- History semua KnowledgeRule yang pernah dihasilkan oleh KTA.
-- Primary store tetap Redis (untuk real-time), tabel ini untuk logging/paper.
CREATE TABLE IF NOT EXISTS knowledge_rules (
    id              UUID PRIMARY KEY,
    source_agent    VARCHAR(50) NOT NULL,
    target_agent    VARCHAR(50) NOT NULL,
    regime          VARCHAR(20) NOT NULL,
    condition_json  JSONB,
    action_json     JSONB,
    confidence      DECIMAL(4,3),
    reasoning       TEXT,
    apply_count     INTEGER DEFAULT 0,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    expires_at      TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_rules_regime ON knowledge_rules(regime);
CREATE INDEX IF NOT EXISTS idx_rules_source ON knowledge_rules(source_agent, created_at);
CREATE INDEX IF NOT EXISTS idx_rules_target ON knowledge_rules(target_agent);

-- ── Tabel 3: Agent Performance Log ───────────────────────────────────
-- Tracking setiap evaluasi sinyal: apakah prediksi benar atau salah.
-- Hypertable TimescaleDB untuk query time-series yang efisien.
CREATE TABLE IF NOT EXISTS agent_performance_log (
    id          BIGSERIAL,
    agent_name  VARCHAR(50) NOT NULL,
    pair        VARCHAR(10) NOT NULL,
    regime      VARCHAR(20),
    signal      VARCHAR(10),
    entry_price DECIMAL(10,5),
    eval_price  DECIMAL(10,5),
    correct     BOOLEAN NOT NULL,
    pips_move   DECIMAL(8,2),
    signal_time TIMESTAMPTZ NOT NULL,
    eval_time   TIMESTAMPTZ DEFAULT NOW()
);

-- Buat hypertable TimescaleDB (jika TimescaleDB extension tersedia)
-- Jika tidak ada TimescaleDB, tabel tetap bisa dipakai sebagai tabel biasa.
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'timescaledb') THEN
        PERFORM create_hypertable('agent_performance_log', 'eval_time', if_not_exists => TRUE);
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_perf_regime ON agent_performance_log(regime, agent_name);
CREATE INDEX IF NOT EXISTS idx_perf_pair ON agent_performance_log(pair, signal_time);
CREATE INDEX IF NOT EXISTS idx_perf_agent ON agent_performance_log(agent_name, eval_time);

-- ── Tabel 4: Regime Detection Log ────────────────────────────────────
-- Log regime yang terdeteksi per pair per waktu (untuk validasi dan paper).
CREATE TABLE IF NOT EXISTS regime_log (
    id              BIGSERIAL,
    pair            VARCHAR(10) NOT NULL,
    regime          VARCHAR(20) NOT NULL,
    adx             DECIMAL(6,2),
    atr             DECIMAL(10,6),
    volatility      DECIMAL(8,5),
    trend_strength  DECIMAL(4,3),
    detected_at     TIMESTAMPTZ DEFAULT NOW()
);

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'timescaledb') THEN
        PERFORM create_hypertable('regime_log', 'detected_at', if_not_exists => TRUE);
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_regime_log_pair ON regime_log(pair, detected_at);
CREATE INDEX IF NOT EXISTS idx_regime_log_regime ON regime_log(regime);
