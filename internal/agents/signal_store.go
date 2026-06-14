package agents

import (
	"sync"
	"time"

	"github.com/dhnnnn/forexAnalysis/internal/knowledge"
)

// ════════════════════════════════════════════════════════════════════════
// PendingSignal — sinyal yang belum dievaluasi (menunggu harga bergerak)
// ════════════════════════════════════════════════════════════════════════

// PendingSignal menyimpan sinyal yang perlu dievaluasi setelah delay tertentu.
type PendingSignal struct {
	Pair      string
	Signal    string               // "BUY" | "SELL"
	Entry     float64              // harga entry saat sinyal dibuat
	Regime    knowledge.MarketRegime
	CreatedAt time.Time
	EvalAfter time.Time            // jangan evaluasi sebelum waktu ini
}

// SignalStore menyimpan pending signals untuk evaluasi oleh evaluator goroutine.
type SignalStore struct {
	pending []PendingSignal
	mu      sync.Mutex
}

// NewSignalStore membuat SignalStore baru.
func NewSignalStore() *SignalStore {
	return &SignalStore{
		pending: make([]PendingSignal, 0),
	}
}

// Add menambahkan sinyal baru ke pending evaluation.
func (s *SignalStore) Add(sig PendingSignal) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pending = append(s.pending, sig)
}

// GetReadyForEvaluation mengembalikan sinyal yang sudah melewati eval delay
// dan menghapusnya dari pending list.
func (s *SignalStore) GetReadyForEvaluation() []PendingSignal {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	var ready []PendingSignal
	var remaining []PendingSignal

	for _, sig := range s.pending {
		if now.After(sig.EvalAfter) {
			ready = append(ready, sig)
		} else {
			remaining = append(remaining, sig)
		}
	}

	s.pending = remaining
	return ready
}

// PendingCount mengembalikan jumlah sinyal yang belum dievaluasi.
func (s *SignalStore) PendingCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.pending)
}
