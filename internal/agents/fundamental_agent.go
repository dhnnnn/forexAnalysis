package agents

import (
	"context"
	"strings"
	"time"

	"github.com/dhnnnn/forexAnalysis/internal/sentiment"
)

// Compile-time check that FundamentalAgent implements Agent.
var _ Agent = (*FundamentalAgent)(nil)

// FundamentalAgent analyzes economic news headlines to produce sentiment signals
// for currency pairs. It implements the Agent interface.
type FundamentalAgent struct {
	gemini sentiment.SentimentAnalyzer
	news   sentiment.HeadlineFetcher
	cache  sentiment.CacheStore
}

// NewFundamentalAgent creates a new FundamentalAgent with the given dependencies.
// All parameters are interfaces to enable dependency injection and testing.
func NewFundamentalAgent(gemini sentiment.SentimentAnalyzer, news sentiment.HeadlineFetcher, cache sentiment.CacheStore) *FundamentalAgent {
	return &FundamentalAgent{
		gemini: gemini,
		news:   news,
		cache:  cache,
	}
}

// Name returns the agent's identifier.
func (a *FundamentalAgent) Name() string {
	return "FundamentalAgent"
}

// Run executes the fundamental analysis pipeline:
// 1. Check context cancellation
// 2. Check Redis cache for cached result
// 3. Fetch news headlines
// 4. Analyze sentiment via Gemini
// 5. Compute normalized score
// 6. Store result in cache
// 7. Return AgentOutput
func (a *FundamentalAgent) Run(ctx context.Context, input AgentInput) AgentOutput {
	// 1. Check context cancellation
	if ctx.Err() != nil {
		return AgentOutput{
			AgentName: a.Name(),
			Success:   false,
			Error:     ctx.Err(),
			Timestamp: time.Now(),
		}
	}

	// 2. Check Redis cache
	cached, err := a.cache.Get(ctx, input.Pair)
	if err == nil && cached != nil {
		score := sentiment.ComputeScore(cached.Sentiment, cached.Confidence)
		return AgentOutput{
			AgentName: a.Name(),
			Success:   true,
			Timestamp: time.Now(),
			Fundamental: &FundamentalOutput{
				Sentiment:  cached.Sentiment,
				Confidence: cached.Confidence,
				Score:      score,
				Reason:     truncateReason(cached.Reason),
				FromCache:  true,
			},
		}
	}

	// 3. Fetch headlines
	headlines, _ := a.news.FetchForPair(ctx, input.Pair)
	if len(headlines) == 0 {
		return a.buildOutput(sentiment.SentimentResult{
			Sentiment:  "neutral",
			Confidence: 0.5,
			Reason:     "no relevant news found",
		})
	}

	// 4. Call Gemini analysis
	result := a.gemini.AnalyzeSentiment(ctx, input.Pair, headlines)

	// 5. Compute normalized score (done in buildOutput)

	// 6. Store in Redis cache (ignore errors)
	_ = a.cache.Set(ctx, input.Pair, result)

	// 7. Build and return AgentOutput
	return a.buildOutput(result)
}

// buildOutput constructs a successful AgentOutput from a SentimentResult.
func (a *FundamentalAgent) buildOutput(result sentiment.SentimentResult) AgentOutput {
	score := sentiment.ComputeScore(result.Sentiment, result.Confidence)
	return AgentOutput{
		AgentName: a.Name(),
		Success:   true,
		Timestamp: time.Now(),
		Fundamental: &FundamentalOutput{
			Sentiment:  result.Sentiment,
			Confidence: result.Confidence,
			Score:      score,
			Reason:     truncateReason(result.Reason),
			FromCache:  result.FromCache,
		},
	}
}

// truncateReason ensures the reason string does not exceed 15 words.
// If it does, it truncates to the first 15 words.
func truncateReason(reason string) string {
	words := strings.Fields(reason)
	if len(words) <= 15 {
		return reason
	}
	return strings.Join(words[:15], " ")
}
