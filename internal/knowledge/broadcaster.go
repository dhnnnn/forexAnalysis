package knowledge

import (
	"context"
	"log/slog"
	"sync"
)

// ════════════════════════════════════════════════════════════════════════
// Broadcaster — menyebarkan KnowledgeRule ke semua KnowledgeAware agents
// ════════════════════════════════════════════════════════════════════════

// KnowledgeAware adalah interface yang diimplementasi oleh agen
// yang bisa menerima dan bereaksi terhadap KnowledgeRule.
type KnowledgeAware interface {
	// ApplyKnowledge menerima rules aktif dan regime context saat ini.
	// Agen menggunakan informasi ini untuk menyesuaikan perilaku internalnya.
	ApplyKnowledge(rules []KnowledgeRule, regime RegimeContext)

	// AgentName mengembalikan nama agen (untuk logging).
	AgentName() string
}

// Broadcaster bertanggung jawab menyebarkan KnowledgeRule dari KB
// ke semua agen yang mengimplementasi KnowledgeAware.
type Broadcaster struct {
	store      *Store
	subscribers []KnowledgeAware
	mu          sync.RWMutex
}

// NewBroadcaster membuat instance Broadcaster baru.
func NewBroadcaster(store *Store) *Broadcaster {
	return &Broadcaster{
		store:      store,
		subscribers: make([]KnowledgeAware, 0),
	}
}

// Subscribe mendaftarkan agen ke dalam broadcast list.
func (b *Broadcaster) Subscribe(agent KnowledgeAware) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.subscribers = append(b.subscribers, agent)
	slog.Debug("Broadcaster: agent subscribed", "agent", agent.AgentName())
}

// Broadcast mengambil rules aktif dari KB untuk regime tertentu
// dan mengirimkannya ke semua subscriber.
// Dipanggil setiap siklus pipeline sebelum agen dijalankan.
func (b *Broadcaster) Broadcast(ctx context.Context, regime RegimeContext) int {
	rules, err := b.store.GetRulesForRegime(ctx, regime.Regime)
	if err != nil {
		slog.Debug("Broadcaster: failed to get rules", "error", err)
		return 0
	}

	if len(rules) == 0 {
		return 0
	}

	b.mu.RLock()
	defer b.mu.RUnlock()

	for _, agent := range b.subscribers {
		agent.ApplyKnowledge(rules, regime)
	}

	slog.Debug("🔊 Broadcaster: rules distributed",
		"regime", string(regime.Regime),
		"rule_count", len(rules),
		"subscriber_count", len(b.subscribers),
	)

	return len(rules)
}

// BroadcastAll mengambil SEMUA rules aktif (tanpa filter regime)
// dan mengirimkannya ke semua subscriber.
// Berguna untuk inisialisasi atau reset.
func (b *Broadcaster) BroadcastAll(ctx context.Context, regime RegimeContext) int {
	rules, err := b.store.GetActiveRules(ctx)
	if err != nil {
		slog.Debug("Broadcaster: failed to get all rules", "error", err)
		return 0
	}

	if len(rules) == 0 {
		return 0
	}

	b.mu.RLock()
	defer b.mu.RUnlock()

	for _, agent := range b.subscribers {
		agent.ApplyKnowledge(rules, regime)
	}

	return len(rules)
}

// SubscriberCount mengembalikan jumlah agen yang terdaftar.
func (b *Broadcaster) SubscriberCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.subscribers)
}
