# Implementation Plan: FundamentalAgent

## Overview

Implement FundamentalAgent (Agent 3) for the forex multi-agent pipeline. The agent fetches economic news headlines from multiple sources concurrently, analyzes sentiment via Gemini API, caches results in Redis, and returns a normalized score for downstream decision making. All components use dependency injection for testability.

## Tasks

- [x] 1. Create sentiment package structure and interfaces
  - [x] 1.1 Create `internal/sentiment/interfaces.go` with `HeadlineFetcher`, `SentimentAnalyzer`, and `CacheStore` interfaces
    - Define `HeadlineFetcher` interface with `FetchForPair(ctx context.Context, pair string) ([]string, error)`
    - Define `SentimentAnalyzer` interface with `AnalyzeSentiment(ctx context.Context, pair string, headlines []string) SentimentResult`
    - Define `CacheStore` interface with `Get(ctx context.Context, pair string) (*SentimentResult, error)` and `Set(ctx context.Context, pair string, result SentimentResult) error`
    - Define `SentimentResult` struct with Sentiment, Confidence, Reason, FromCache fields
    - _Requirements: 3.3, 4.1, 4.2, 4.3, 7.1, 7.2, 7.4, 7.5_

  - [x] 1.2 Create `internal/sentiment/score.go` with score normalization function
    - Implement `ComputeScore(sentiment string, confidence float64) float64`
    - Bullish: `0.5 + (confidence × 0.5)`, Bearish: `0.5 - (confidence × 0.5)`, Neutral: `0.5`
    - Clamp confidence to [0.0, 1.0] before computing
    - _Requirements: 5.1, 5.2, 5.3_

  - [ ]* 1.3 Write property test for score normalization (Property 2)
    - **Property 2: Score Normalization Correctness**
    - Generate random sentiment values from {"bullish", "bearish", "neutral"} and random confidence in [0.0, 1.0]
    - Assert score matches the formula and stays within expected range
    - **Validates: Requirements 5.1, 5.2, 5.3, 7.3**

- [x] 2. Implement GeminiClient for sentiment analysis
  - [x] 2.1 Create `internal/sentiment/gemini.go` with GeminiClient struct and constructor
    - Implement `NewGeminiClient(apiKey, model string, timeout time.Duration) *GeminiClient`
    - Implement `AnalyzeSentiment(ctx context.Context, pair string, headlines []string) SentimentResult`
    - Build prompt using the defined template with pair and headlines
    - Enforce 2-second timeout on the HTTP request to Gemini API
    - Parse JSON response into SentimentResult; return neutral fallback on any error
    - Clamp confidence to [0.0, 1.0] and validate sentiment value
    - _Requirements: 3.1, 3.2, 3.3, 3.4, 3.5, 8.1, 8.2, 8.3, 8.4_

  - [ ]* 2.2 Write property test for prompt construction (Property 3)
    - **Property 3: Prompt Construction Completeness**
    - Generate random pair strings and random non-empty headline lists
    - Assert the constructed prompt contains the pair and every headline
    - **Validates: Requirements 3.1, 8.1, 8.2**

  - [ ]* 2.3 Write property test for Gemini response parsing (Property 4)
    - **Property 4: Gemini Response Parsing Safety**
    - Generate random strings (valid JSON, invalid JSON, empty, malformed)
    - Assert `AnalyzeSentiment` always returns a valid SentimentResult with Sentiment in {"bullish", "bearish", "neutral"} and Confidence in [0.0, 1.0]
    - **Validates: Requirements 3.3, 3.5**

- [x] 3. Implement NewsFetcher for multi-source headline retrieval
  - [x] 3.1 Create `internal/sentiment/news_fetcher.go` with NewsFetcher struct and constructor
    - Implement `NewNewsFetcher(avKey, tdKey string, rssURLs []string) *NewsFetcher`
    - Implement `FetchForPair(ctx context.Context, pair string) ([]string, error)`
    - Fetch from Alpha Vantage, Twelve Data, and RSS feeds concurrently using errgroup
    - Tolerate partial failures: continue with remaining sources if some fail
    - Return empty headline list (not error) when all sources fail
    - _Requirements: 2.1, 2.2, 2.3, 2.4, 2.5, 2.6_

  - [ ]* 3.2 Write property test for partial source failure resilience (Property 7)
    - **Property 7: Partial Source Failure Resilience**
    - Generate random subsets of source failures using mock HTTP servers
    - Assert NewsFetcher still returns headlines from successful sources without error
    - **Validates: Requirements 2.5**

- [x] 4. Implement SentimentCache for Redis caching
  - [x] 4.1 Create `internal/sentiment/cache.go` with SentimentCache struct and constructor
    - Implement `NewSentimentCache(client *redis.Client, ttl time.Duration) *SentimentCache`
    - Implement `Get(ctx context.Context, pair string) (*SentimentResult, error)` using key format `sentiment:{pair}`
    - Implement `Set(ctx context.Context, pair string, result SentimentResult) error` with 5-minute TTL
    - Handle Redis unavailability gracefully (return error on Get, log on Set failure)
    - _Requirements: 4.1, 4.2, 4.3, 4.4, 4.5_

  - [ ]* 4.2 Write property test for cache round-trip (Property 5)
    - **Property 5: Cache Round-Trip**
    - Generate random valid SentimentResult values
    - Store in miniredis, retrieve, and assert equivalence with FromCache=true
    - **Validates: Requirements 4.3, 7.5**

- [x] 5. Checkpoint - Ensure all component tests pass
  - Ensure all tests pass, ask the user if questions arise.

- [x] 6. Implement FundamentalAgent orchestration
  - [x] 6.1 Create `internal/agents/fundamental_agent.go` with FundamentalAgent struct and Run method
    - Implement `NewFundamentalAgent(gemini, news, cache)` constructor accepting interfaces
    - Implement `Name()` returning "FundamentalAgent"
    - Implement `Run(ctx context.Context, input AgentInput) AgentOutput` with full pipeline:
      1. Check context cancellation → return Success=false with error
      2. Check Redis cache → return cached result with FromCache=true if hit
      3. Fetch headlines → return neutral fallback if none found
      4. Call Gemini analysis → use result or fallback
      5. Compute normalized score
      6. Store in Redis cache (ignore errors)
      7. Build and return AgentOutput with Success=true
    - Validate Reason field does not exceed 15 words; truncate if needed
    - _Requirements: 1.1, 1.2, 1.3, 1.4, 4.3, 4.4, 6.1, 6.2, 6.3, 7.1, 7.2, 7.3, 7.4, 7.5_

  - [ ]* 6.2 Write property test for output invariants (Property 1)
    - **Property 1: Output Invariants**
    - Generate random failure combinations (news/Gemini/Redis states) using mock interfaces
    - Assert AgentOutput always has Success=true (except context cancellation), AgentName="FundamentalAgent", non-zero Timestamp, non-nil Fundamental, valid Sentiment, and Confidence in [0.0, 1.0]
    - **Validates: Requirements 1.3, 6.3, 7.1, 7.2**

  - [ ]* 6.3 Write property test for reason word limit (Property 6)
    - **Property 6: Reason Word Limit**
    - Generate random execution paths through the agent using mock interfaces
    - Assert the Reason field always contains 15 words or fewer
    - **Validates: Requirements 7.4**

  - [ ]* 6.4 Write unit tests for FundamentalAgent
    - Test Name() returns "FundamentalAgent"
    - Test context cancellation returns Success=false
    - Test cache hit returns cached result with FromCache=true
    - Test no headlines returns neutral fallback
    - Test Gemini failure without cache returns neutral fallback
    - Test Redis unavailable does not break pipeline
    - _Requirements: 1.1, 1.4, 4.3, 4.5, 6.1, 6.2_

- [ ] 7. Wire dependencies and integrate with pipeline
  - [x] 7.1 Update `cmd/main.go` or agent initialization code to construct FundamentalAgent
    - Add Redis client creation (reuse existing if available)
    - Create NewsFetcher with API keys from config
    - Create GeminiClient with API key, model, and 2s timeout from config
    - Create SentimentCache with Redis client and 5-minute TTL
    - Create FundamentalAgent with all dependencies injected
    - Register FundamentalAgent in the pipeline
    - _Requirements: 1.1, 1.2, 2.2, 2.3, 2.4, 3.2, 4.1_

  - [x] 7.2 Update `config/config.yaml` with new configuration entries
    - Add Gemini API key, model name fields
    - Add Alpha Vantage and Twelve Data API key fields
    - Add RSS feed URLs list
    - Add Redis connection settings (if not already present)
    - Add sentiment cache TTL setting
    - _Requirements: 2.2, 2.3, 2.4, 3.2, 4.1_

- [x] 8. Final checkpoint - Ensure all tests pass
  - Ensure all tests pass, ask the user if questions arise.

## Notes

- Tasks marked with `*` are optional and can be skipped for faster MVP
- Each task references specific requirements for traceability
- Checkpoints ensure incremental validation
- Property tests validate universal correctness properties using `pgregory.net/rapid`
- Unit tests validate specific examples and edge cases
- All external dependencies (HTTP, Redis) are injected via interfaces for testability
- Use `github.com/alicebob/miniredis/v2` for Redis tests and `httptest.Server` for HTTP mocks

## Task Dependency Graph

```json
{
  "waves": [
    { "id": 0, "tasks": ["1.1"] },
    { "id": 1, "tasks": ["1.2", "2.1", "3.1", "4.1"] },
    { "id": 2, "tasks": ["1.3", "2.2", "2.3", "3.2", "4.2"] },
    { "id": 3, "tasks": ["6.1"] },
    { "id": 4, "tasks": ["6.2", "6.3", "6.4"] },
    { "id": 5, "tasks": ["7.1", "7.2"] }
  ]
}
```
