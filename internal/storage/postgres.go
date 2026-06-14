package storage

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dhnnnn/forex-agent/internal/agents"
)

// ════════════════════════════════════════════════════════════════════════
// PostgreSQL Storage Layer — persist candles & signals ke TimescaleDB
// ════════════════════════════════════════════════════════════════════════

// Store menyediakan akses ke TimescaleDB untuk persist data forex.
type Store struct {
	pool *pgxpool.Pool
}

// New membuat koneksi pool ke TimescaleDB dan mengembalikan Store.
// DSN format: "postgres://user:pass@host:port/db?sslmode=disable"
func New(ctx context.Context, dsn string) (*Store, error) {
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse dsn: %w", err)
	}

	// Pool settings optimal untuk forex bot (low connection count)
	config.MaxConns = 5
	config.MinConns = 1
	config.MaxConnLifetime = 30 * time.Minute
	config.MaxConnIdleTime = 5 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}

	// Verify connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping db: %w", err)
	}

	slog.Info("✅ TimescaleDB connected", "dsn_host", config.ConnConfig.Host)
	return &Store{pool: pool}, nil
}

// Close menutup semua koneksi database.
func (s *Store) Close() {
	s.pool.Close()
}

// ════════════════════════════════════════════════════════════════════════
// Candle Operations
// ════════════════════════════════════════════════════════════════════════

// InsertCandles menyimpan batch candle ke tabel candles.
// Menggunakan ON CONFLICT untuk upsert (update jika sudah ada).
func (s *Store) InsertCandles(ctx context.Context, candles []agents.Candle) error {
	if len(candles) == 0 {
		return nil
	}

	const query = `
		INSERT INTO candles (time, pair, timeframe, open, high, low, close, volume, spread)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (time, pair, timeframe) DO UPDATE SET
			open = EXCLUDED.open,
			high = EXCLUDED.high,
			low = EXCLUDED.low,
			close = EXCLUDED.close,
			volume = EXCLUDED.volume,
			spread = EXCLUDED.spread
	`

	batch := &pgxBatch{}
	for _, c := range candles {
		batch.Queue(query, c.Timestamp, c.Pair, c.Timeframe, c.Open, c.High, c.Low, c.Close, c.Volume, c.Spread)
	}

	br := s.pool.SendBatch(ctx, batch.batch())
	defer br.Close()

	for i := 0; i < len(candles); i++ {
		if _, err := br.Exec(); err != nil {
			return fmt.Errorf("insert candle %d: %w", i, err)
		}
	}

	slog.Debug("💾 Candles persisted", "count", len(candles), "pair", candles[0].Pair)
	return nil
}

// GetCandles membaca candles dari database untuk pair dan timeframe tertentu.
// Mengembalikan candles terbaru (ordered by time DESC, limit n).
func (s *Store) GetCandles(ctx context.Context, pair, timeframe string, limit int) ([]agents.Candle, error) {
	const query = `
		SELECT time, pair, timeframe, open, high, low, close, volume, spread
		FROM candles
		WHERE pair = $1 AND timeframe = $2
		ORDER BY time DESC
		LIMIT $3
	`

	rows, err := s.pool.Query(ctx, query, pair, timeframe, limit)
	if err != nil {
		return nil, fmt.Errorf("query candles: %w", err)
	}
	defer rows.Close()

	var candles []agents.Candle
	for rows.Next() {
		var c agents.Candle
		if err := rows.Scan(&c.Timestamp, &c.Pair, &c.Timeframe, &c.Open, &c.High, &c.Low, &c.Close, &c.Volume, &c.Spread); err != nil {
			return nil, fmt.Errorf("scan candle: %w", err)
		}
		candles = append(candles, c)
	}

	// Reverse agar oldest-first (untuk indikator)
	for i, j := 0, len(candles)-1; i < j; i, j = i+1, j-1 {
		candles[i], candles[j] = candles[j], candles[i]
	}

	return candles, rows.Err()
}

// ════════════════════════════════════════════════════════════════════════
// Signal Operations
// ════════════════════════════════════════════════════════════════════════

// InsertSignal menyimpan trading signal dari DecisionAgent ke tabel signals.
func (s *Store) InsertSignal(ctx context.Context, d *agents.DecisionOutput) error {
	if d == nil {
		return nil
	}

	const query = `
		INSERT INTO signals (
			time, pair, direction, confidence, tech_score, tech_signal,
			fund_sentiment, fund_score, ml_score, risk_level,
			lot_size, entry_price, stop_loss, take_profit,
			sl_pips, tp_pips, tech_reason, fund_reason
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10,
			$11, $12, $13, $14,
			$15, $16, $17, $18
		)
	`

	_, err := s.pool.Exec(ctx, query,
		d.Timestamp, d.Pair, d.Signal, d.Confidence, d.TechConf, d.TechSignal,
		d.FundSentiment, d.FundConf, d.MLScore, d.RiskLevel,
		d.LotSize, d.Entry, d.StopLoss, d.TakeProfit,
		0.0, 0.0, // sl_pips, tp_pips — tidak ada di DecisionOutput, set 0
		d.TechReason, d.FundReason,
	)
	if err != nil {
		return fmt.Errorf("insert signal: %w", err)
	}

	slog.Debug("💾 Signal persisted", "pair", d.Pair, "signal", d.Signal, "confidence", fmt.Sprintf("%.0f%%", d.Confidence*100))
	return nil
}

// GetRecentSignals membaca N signal terbaru untuk pair tertentu.
func (s *Store) GetRecentSignals(ctx context.Context, pair string, limit int) ([]agents.DecisionOutput, error) {
	const query = `
		SELECT time, pair, direction, confidence, tech_score, tech_signal,
			   fund_sentiment, fund_score, ml_score, risk_level,
			   lot_size, entry_price, stop_loss, take_profit,
			   tech_reason, fund_reason
		FROM signals
		WHERE pair = $1
		ORDER BY time DESC
		LIMIT $2
	`

	rows, err := s.pool.Query(ctx, query, pair, limit)
	if err != nil {
		return nil, fmt.Errorf("query signals: %w", err)
	}
	defer rows.Close()

	var signals []agents.DecisionOutput
	for rows.Next() {
		var d agents.DecisionOutput
		if err := rows.Scan(
			&d.Timestamp, &d.Pair, &d.Signal, &d.Confidence, &d.TechConf, &d.TechSignal,
			&d.FundSentiment, &d.FundConf, &d.MLScore, &d.RiskLevel,
			&d.LotSize, &d.Entry, &d.StopLoss, &d.TakeProfit,
			&d.TechReason, &d.FundReason,
		); err != nil {
			return nil, fmt.Errorf("scan signal: %w", err)
		}
		d.ConfPct = int(d.Confidence * 100)
		signals = append(signals, d)
	}

	return signals, rows.Err()
}

// GetSignalStats mengembalikan statistik singkat: total signals, buy count, sell count.
func (s *Store) GetSignalStats(ctx context.Context, pair string, since time.Duration) (total, buys, sells int, err error) {
	const query = `
		SELECT
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE direction = 'BUY') as buys,
			COUNT(*) FILTER (WHERE direction = 'SELL') as sells
		FROM signals
		WHERE pair = $1 AND time > NOW() - $2::interval
	`

	sinceStr := fmt.Sprintf("%d hours", int(since.Hours()))
	err = s.pool.QueryRow(ctx, query, pair, sinceStr).Scan(&total, &buys, &sells)
	if err != nil {
		err = fmt.Errorf("query signal stats: %w", err)
	}
	return
}
