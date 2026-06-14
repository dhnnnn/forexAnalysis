package agents

import (
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/dhnnnn/forexAnalysis/internal/knowledge"
)

// ════════════════════════════════════════════════════════════════════════
// MetaObserverAgent — memantau performa agen dan menghasilkan ExperienceReport
// ════════════════════════════════════════════════════════════════════════

const (
	// DefaultRollingWindow adalah jumlah sinyal terakhir untuk hitung accuracy.
	DefaultRollingWindow = 20

	// DefaultDropThreshold adalah penurunan akurasi yang trigger report (20%).
	DefaultDropThreshold = 0.20

	// DefaultLossStreakTrigger adalah jumlah loss berturut yang trigger report.
	DefaultLossStreakTrigger = 4
)

// MetaObserverConfig menyimpan konfigurasi dari config.yaml.
type MetaObserverConfig struct {
	RollingWindow    int
	DropThreshold    float64
	LossStreakTrigger int
}

// SignalOutcome adalah hasil evaluasi sinyal setelah harga bergerak.
type SignalOutcome struct {
	AgentName string
	Pair      string
	Correct   bool
	Regime    knowledge.MarketRegime
	Timestamp time.Time
}

// agentTracker melacak performa satu agen secara internal.
type agentTracker struct {
	name       string
	outcomes   []SignalOutcome
	lossStreak int
	mu         sync.Mutex
}

func (t *agentTracker) record(outcome SignalOutcome, maxWindow int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.outcomes = append(t.outcomes, outcome)

	// Trim buffer agar tidak membengkak (simpan 2x window)
	if len(t.outcomes) > maxWindow*2 {
		t.outcomes = t.outcomes[len(t.outcomes)-maxWindow*2:]
	}

	if !outcome.Correct {
		t.lossStreak++
	} else {
		t.lossStreak = 0
	}
}

func (t *agentTracker) accuracy(last int) float64 {
	t.mu.Lock()
	defer t.mu.Unlock()

	if len(t.outcomes) == 0 {
		return 0.5 // default neutral
	}

	start := len(t.outcomes) - last
	if start < 0 {
		start = 0
	}
	window := t.outcomes[start:]

	wins := 0
	for _, o := range window {
		if o.Correct {
			wins++
		}
	}
	return float64(wins) / float64(len(window))
}

func (t *agentTracker) dominantRegime(last int) knowledge.MarketRegime {
	t.mu.Lock()
	defer t.mu.Unlock()

	counts := map[knowledge.MarketRegime]int{}
	start := len(t.outcomes) - last
	if start < 0 {
		start = 0
	}
	for _, o := range t.outcomes[start:] {
		counts[o.Regime]++
	}

	var best knowledge.MarketRegime
	bestN := 0
	for r, n := range counts {
		if n > bestN {
			best = r
			bestN = n
		}
	}
	return best
}

func (t *agentTracker) totalOutcomes() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return len(t.outcomes)
}

// MetaObserverAgent memantau semua agen dan menghasilkan ExperienceReport
// ketika mendeteksi penurunan performa signifikan.
type MetaObserverAgent struct {
	trackers         map[string]*agentTracker
	rollingWindow    int
	dropThreshold    float64
	lossStreakTrigger int
	mu               sync.RWMutex
}

// NewMetaObserverAgent membuat instance baru dengan default config.
func NewMetaObserverAgent() *MetaObserverAgent {
	return &MetaObserverAgent{
		trackers:         make(map[string]*agentTracker),
		rollingWindow:    DefaultRollingWindow,
		dropThreshold:    DefaultDropThreshold,
		lossStreakTrigger: DefaultLossStreakTrigger,
	}
}

// NewMetaObserverAgentWithConfig membuat instance dengan konfigurasi custom.
func NewMetaObserverAgentWithConfig(cfg MetaObserverConfig) *MetaObserverAgent {
	agent := NewMetaObserverAgent()
	if cfg.RollingWindow > 0 {
		agent.rollingWindow = cfg.RollingWindow
	}
	if cfg.DropThreshold > 0 {
		agent.dropThreshold = cfg.DropThreshold
	}
	if cfg.LossStreakTrigger > 0 {
		agent.lossStreakTrigger = cfg.LossStreakTrigger
	}
	return agent
}

// Name mengembalikan identifier agent.
func (m *MetaObserverAgent) Name() string {
	return "MetaObserverAgent"
}

// RegisterAgent mendaftarkan agen ke dalam pengawasan MetaObserver.
func (m *MetaObserverAgent) RegisterAgent(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.trackers[name]; !exists {
		m.trackers[name] = &agentTracker{name: name}
		slog.Debug("MetaObserver: registered agent", "agent", name)
	}
}

// RecordOutcome dipanggil setelah setiap sinyal dievaluasi.
// Dipanggil dari evaluator goroutine setelah harga bergerak cukup.
func (m *MetaObserverAgent) RecordOutcome(outcome SignalOutcome) {
	m.mu.RLock()
	tracker, exists := m.trackers[outcome.AgentName]
	m.mu.RUnlock()

	if !exists {
		slog.Warn("MetaObserver: unknown agent, skipping", "agent", outcome.AgentName)
		return
	}

	tracker.record(outcome, m.rollingWindow)

	slog.Debug("MetaObserver: outcome recorded",
		"agent", outcome.AgentName,
		"pair", outcome.Pair,
		"correct", outcome.Correct,
		"loss_streak", tracker.lossStreak,
		"total_outcomes", tracker.totalOutcomes(),
	)
}

// Observe memindai semua tracker dan menghasilkan ExperienceReport
// jika ditemukan degradasi performa. Dipanggil setiap siklus pipeline.
func (m *MetaObserverAgent) Observe() []knowledge.ExperienceReport {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var reports []knowledge.ExperienceReport

	for _, tracker := range m.trackers {
		// Butuh minimal 1 window data untuk bisa evaluasi
		if tracker.totalOutcomes() < m.rollingWindow {
			continue
		}

		accNow := tracker.accuracy(m.rollingWindow)
		accPrev := tracker.accuracy(m.rollingWindow * 2)
		delta := accNow - accPrev
		regime := tracker.dominantRegime(m.rollingWindow)

		shouldReport := delta < -m.dropThreshold || tracker.lossStreak >= m.lossStreakTrigger

		if shouldReport {
			report := knowledge.ExperienceReport{
				AgentName:      tracker.name,
				AccuracyBefore: accPrev,
				AccuracyNow:    accNow,
				AccuracyDelta:  delta,
				LossStreak:     tracker.lossStreak,
				ActiveRegime:   regime,
				Timestamp:      time.Now(),
			}
			reports = append(reports, report)

			slog.Warn("🚨 MetaObserver: performance degradation detected",
				"agent", tracker.name,
				"accuracy_before", fmt.Sprintf("%.1f%%", accPrev*100),
				"accuracy_now", fmt.Sprintf("%.1f%%", accNow*100),
				"delta", fmt.Sprintf("%.1f%%", delta*100),
				"loss_streak", tracker.lossStreak,
				"regime", string(regime),
			)
		}
	}

	return reports
}

// GetMetrics mengembalikan AgentMetrics untuk semua agen yang dipantau.
// Berguna untuk logging dan dashboard.
func (m *MetaObserverAgent) GetMetrics() []knowledge.AgentMetrics {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var metrics []knowledge.AgentMetrics
	for _, tracker := range m.trackers {
		metric := knowledge.AgentMetrics{
			AgentName:    tracker.name,
			LossStreak:   tracker.lossStreak,
			Accuracy:     tracker.accuracy(m.rollingWindow),
			AccuracyPrev: tracker.accuracy(m.rollingWindow * 2),
			ActiveRegime: tracker.dominantRegime(m.rollingWindow),
			UpdatedAt:    time.Now(),
		}

		// Count wins/losses in current window
		tracker.mu.Lock()
		start := len(tracker.outcomes) - m.rollingWindow
		if start < 0 {
			start = 0
		}
		for _, o := range tracker.outcomes[start:] {
			if o.Correct {
				metric.WinCount++
			} else {
				metric.LossCount++
			}
		}
		tracker.mu.Unlock()

		metrics = append(metrics, metric)
	}
	return metrics
}
