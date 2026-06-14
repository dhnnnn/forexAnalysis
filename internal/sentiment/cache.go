package sentiment

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

// Compile-time interface compliance check.
var _ CacheStore = (*SentimentCache)(nil)

// SentimentCache implements CacheStore using Redis for sentiment result caching.
type SentimentCache struct {
	client *redis.Client
	ttl    time.Duration
}

// NewSentimentCache creates a new SentimentCache with the given Redis client and TTL.
func NewSentimentCache(client *redis.Client, ttl time.Duration) *SentimentCache {
	return &SentimentCache{
		client: client,
		ttl:    ttl,
	}
}

// cacheKey returns the Redis key for a given currency pair.
func cacheKey(pair string) string {
	return fmt.Sprintf("sentiment:%s", pair)
}

// Get retrieves a cached SentimentResult for the given pair.
// Returns nil and an error if the key is not found or Redis is unavailable.
func (c *SentimentCache) Get(ctx context.Context, pair string) (*SentimentResult, error) {
	val, err := c.client.Get(ctx, cacheKey(pair)).Result()
	if err != nil {
		return nil, err
	}

	var result SentimentResult
	if err := json.Unmarshal([]byte(val), &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cached sentiment: %w", err)
	}

	result.FromCache = true
	return &result, nil
}

// Set stores a SentimentResult in Redis with the configured TTL.
// Logs a warning on failure but does not panic.
func (c *SentimentCache) Set(ctx context.Context, pair string, result SentimentResult) error {
	data, err := json.Marshal(result)
	if err != nil {
		log.Printf("[WARN] sentiment cache: failed to marshal result for %s: %v", pair, err)
		return fmt.Errorf("failed to marshal sentiment result: %w", err)
	}

	if err := c.client.Set(ctx, cacheKey(pair), data, c.ttl).Err(); err != nil {
		log.Printf("[WARN] sentiment cache: failed to store result for %s: %v", pair, err)
		return fmt.Errorf("failed to cache sentiment result: %w", err)
	}

	return nil
}
