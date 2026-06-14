package knowledge

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// ════════════════════════════════════════════════════════════════════════
// Knowledge Store — Redis-based storage untuk KnowledgeRule dan AgentMetrics
// ════════════════════════════════════════════════════════════════════════

const (
	keyPrefixRule    = "kb:rule:"
	keyPrefixMetrics = "kb:metrics:"
	keyRuleIndex     = "kb:rule:index"
	defaultRuleTTL   = 24 * time.Hour
)

// Store menyediakan akses ke Redis untuk menyimpan knowledge base.
type Store struct {
	rdb *redis.Client
}

// NewStore membuat Store baru dengan Redis client.
func NewStore(rdb *redis.Client) *Store {
	return &Store{rdb: rdb}
}

// ════════════════════════════════════════════════════════════════════════
// Rule Operations
// ════════════════════════════════════════════════════════════════════════

// SaveRule menyimpan KnowledgeRule ke Redis dengan TTL.
func (s *Store) SaveRule(ctx context.Context, rule KnowledgeRule) error {
	data, err := json.Marshal(rule)
	if err != nil {
		return fmt.Errorf("marshal rule: %w", err)
	}

	key := keyPrefixRule + rule.ID
	ttl := time.Until(rule.ExpiresAt)
	if ttl <= 0 {
		ttl = defaultRuleTTL
	}

	pipe := s.rdb.Pipeline()
	pipe.Set(ctx, key, data, ttl)
	pipe.SAdd(ctx, keyRuleIndex, rule.ID)
	_, err = pipe.Exec(ctx)
	return err
}

// GetActiveRules mengambil semua rule yang masih berlaku.
func (s *Store) GetActiveRules(ctx context.Context) ([]KnowledgeRule, error) {
	ids, err := s.rdb.SMembers(ctx, keyRuleIndex).Result()
	if err != nil {
		return nil, fmt.Errorf("get rule index: %w", err)
	}

	var rules []KnowledgeRule
	for _, id := range ids {
		key := keyPrefixRule + id
		data, err := s.rdb.Get(ctx, key).Result()
		if err == redis.Nil {
			// Rule sudah expired, hapus dari index
			s.rdb.SRem(ctx, keyRuleIndex, id)
			continue
		}
		if err != nil {
			continue
		}

		var rule KnowledgeRule
		if err := json.Unmarshal([]byte(data), &rule); err != nil {
			continue
		}
		rules = append(rules, rule)
	}
	return rules, nil
}

// GetRulesForRegime mengambil rules yang berlaku untuk regime tertentu.
func (s *Store) GetRulesForRegime(ctx context.Context, regime MarketRegime) ([]KnowledgeRule, error) {
	all, err := s.GetActiveRules(ctx)
	if err != nil {
		return nil, err
	}

	var filtered []KnowledgeRule
	for _, r := range all {
		if r.Condition.Regime == regime {
			filtered = append(filtered, r)
		}
	}
	return filtered, nil
}

// ════════════════════════════════════════════════════════════════════════
// Metrics Operations
// ════════════════════════════════════════════════════════════════════════

// SaveMetrics menyimpan AgentMetrics ke Redis (untuk logging dan paper).
func (s *Store) SaveMetrics(ctx context.Context, m AgentMetrics) error {
	data, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("marshal metrics: %w", err)
	}

	key := keyPrefixMetrics + m.AgentName
	return s.rdb.Set(ctx, key, data, 0).Err()
}

// GetMetrics mengambil metrics sebuah agen dari Redis.
func (s *Store) GetMetrics(ctx context.Context, agentName string) (*AgentMetrics, error) {
	key := keyPrefixMetrics + agentName
	data, err := s.rdb.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get metrics: %w", err)
	}

	var m AgentMetrics
	if err := json.Unmarshal([]byte(data), &m); err != nil {
		return nil, fmt.Errorf("unmarshal metrics: %w", err)
	}
	return &m, nil
}

// SaveAllMetrics menyimpan batch metrics sekaligus.
func (s *Store) SaveAllMetrics(ctx context.Context, metrics []AgentMetrics) error {
	pipe := s.rdb.Pipeline()
	for _, m := range metrics {
		data, err := json.Marshal(m)
		if err != nil {
			continue
		}
		key := keyPrefixMetrics + m.AgentName
		pipe.Set(ctx, key, data, 0)
	}
	_, err := pipe.Exec(ctx)
	return err
}
