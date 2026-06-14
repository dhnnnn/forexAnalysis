package sentiment

import "context"

// SentimentResult represents the output from sentiment analysis.
// It contains the sentiment direction, confidence level, reason, and cache status.
type SentimentResult struct {
	Sentiment  string  `json:"sentiment"`  // "bullish" | "bearish" | "neutral"
	Confidence float64 `json:"confidence"` // 0.0–1.0
	Reason     string  `json:"reason"`     // max 15 words
	FromCache  bool    `json:"-"`          // true if served from cache
}

// HeadlineFetcher retrieves news headlines relevant to a currency pair.
type HeadlineFetcher interface {
	FetchForPair(ctx context.Context, pair string) ([]string, error)
}

// SentimentAnalyzer analyzes headlines and produces a sentiment result.
type SentimentAnalyzer interface {
	AnalyzeSentiment(ctx context.Context, pair string, headlines []string) SentimentResult
}

// CacheStore provides caching for sentiment results.
type CacheStore interface {
	Get(ctx context.Context, pair string) (*SentimentResult, error)
	Set(ctx context.Context, pair string, result SentimentResult) error
}
