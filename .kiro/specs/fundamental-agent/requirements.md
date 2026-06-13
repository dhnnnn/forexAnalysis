# Requirements Document

## Introduction

FundamentalAgent (Agent 3) is responsible for reading and analyzing economic news headlines to determine fundamental sentiment for a given currency pair. It fetches news from multiple sources (Alpha Vantage, Twelve Data, RSS feeds), uses Gemini API for NLP sentiment analysis, and caches results in Redis. The agent implements the existing `Agent` interface and produces a `FundamentalOutput` with sentiment, confidence, normalized score, and reasoning.

## Glossary

- **FundamentalAgent**: The Go struct implementing the Agent interface that orchestrates news fetching, sentiment analysis, and caching for fundamental analysis
- **NewsFetcher**: Component responsible for retrieving news headlines from external APIs and RSS feeds relevant to a currency pair
- **GeminiClient**: Component that sends headlines to the Gemini API and parses the sentiment response
- **SentimentCache**: Redis-based cache layer that stores sentiment results with a configurable TTL to avoid redundant API calls
- **Headline**: A single news article title string retrieved from a news source
- **SentimentResult**: The structured output from Gemini containing sentiment direction, confidence level, and reason
- **Score**: A normalized float64 value derived from sentiment and confidence, ranging from 0.0 to 1.0
- **Pair**: A currency pair string in the format "EUR_USD" or "GBP_USD"
- **Agent_Interface**: The contract (`Name() string` + `Run(ctx, AgentInput) AgentOutput`) that all agents implement

## Requirements

### Requirement 1: Agent Interface Compliance

**User Story:** As a pipeline orchestrator, I want FundamentalAgent to implement the Agent interface, so that it integrates seamlessly into the multi-agent pipeline.

#### Acceptance Criteria

1. THE FundamentalAgent SHALL implement the Agent interface by providing a `Name()` method that returns the string "FundamentalAgent"
2. THE FundamentalAgent SHALL implement the Agent interface by providing a `Run(ctx context.Context, input AgentInput) AgentOutput` method
3. WHEN the Run method completes successfully, THE FundamentalAgent SHALL return an AgentOutput with Success set to true, AgentName set to "FundamentalAgent", Timestamp set to the current time, and a non-nil Fundamental field
4. WHEN the context is cancelled before processing completes, THE FundamentalAgent SHALL return an AgentOutput with Success set to false and Error describing the cancellation

### Requirement 2: News Fetching

**User Story:** As a FundamentalAgent, I want to fetch relevant news headlines from multiple sources, so that I have comprehensive coverage of economic events affecting the currency pair.

#### Acceptance Criteria

1. WHEN the FundamentalAgent runs, THE NewsFetcher SHALL attempt to retrieve news headlines relevant to the specified currency pair
2. THE NewsFetcher SHALL support fetching from Alpha Vantage News Sentiment API using the configured API key
3. THE NewsFetcher SHALL support fetching from Twelve Data News API using the configured API key
4. THE NewsFetcher SHALL support fetching from RSS feeds (Reuters, Bloomberg Economic)
5. WHEN a news source returns an error, THE NewsFetcher SHALL continue fetching from remaining sources without failing the entire operation
6. WHEN all news sources return errors or no relevant headlines, THE NewsFetcher SHALL return an empty headline list

### Requirement 3: Gemini Sentiment Analysis

**User Story:** As a FundamentalAgent, I want to analyze news headlines using Gemini API, so that I can determine the sentiment impact on the currency pair.

#### Acceptance Criteria

1. WHEN headlines are available, THE GeminiClient SHALL send them to the Gemini API with a prompt requesting JSON output containing sentiment, confidence, and reason
2. THE GeminiClient SHALL enforce a 2-second timeout on the Gemini API request
3. WHEN Gemini responds successfully, THE GeminiClient SHALL parse the JSON response into a SentimentResult with sentiment as one of "bullish", "bearish", or "neutral", confidence as a float64 between 0.0 and 1.0, and reason as a string of 15 words or fewer
4. IF the Gemini API returns an error or times out, THEN THE GeminiClient SHALL return a fallback SentimentResult with sentiment "neutral", confidence 0.5, and reason "Gemini API unavailable"
5. IF the Gemini response contains invalid JSON or unexpected values, THEN THE GeminiClient SHALL return a fallback SentimentResult with sentiment "neutral", confidence 0.5, and reason "invalid Gemini response"

### Requirement 4: Redis Caching

**User Story:** As a system operator, I want sentiment results cached in Redis, so that redundant API calls are avoided and latency is reduced.

#### Acceptance Criteria

1. WHEN a sentiment analysis is completed successfully, THE SentimentCache SHALL store the result in Redis with a TTL of 5 minutes
2. THE SentimentCache SHALL use a cache key derived from the currency pair to uniquely identify cached results
3. WHEN a cached result exists and has not expired, THE FundamentalAgent SHALL return the cached result with FromCache set to true without calling the Gemini API
4. WHEN no cached result exists or the cache has expired, THE FundamentalAgent SHALL proceed with news fetching and Gemini analysis
5. IF Redis is unavailable, THEN THE SentimentCache SHALL allow the FundamentalAgent to proceed without caching (cache miss behavior)

### Requirement 5: Score Normalization

**User Story:** As a DecisionAgent consumer, I want the fundamental score normalized to a 0.0-1.0 range, so that it can be combined with the technical score for decision making.

#### Acceptance Criteria

1. WHEN the sentiment is "bullish", THE FundamentalAgent SHALL compute the Score as 0.5 + (confidence × 0.5), producing a value in the range 0.5 to 1.0
2. WHEN the sentiment is "bearish", THE FundamentalAgent SHALL compute the Score as 0.5 - (confidence × 0.5), producing a value in the range 0.0 to 0.5
3. WHEN the sentiment is "neutral", THE FundamentalAgent SHALL set the Score to 0.5

### Requirement 6: Fallback Behavior

**User Story:** As a pipeline orchestrator, I want the FundamentalAgent to always produce a valid output, so that downstream agents can operate without null checks.

#### Acceptance Criteria

1. IF no news headlines are found for the pair, THEN THE FundamentalAgent SHALL return a FundamentalOutput with sentiment "neutral", confidence 0.5, Score 0.5, reason "no relevant news found", and FromCache false
2. IF the Gemini API fails and no cached result exists, THEN THE FundamentalAgent SHALL return a FundamentalOutput with sentiment "neutral", confidence 0.5, Score 0.5, reason "Gemini API unavailable", and FromCache false
3. THE FundamentalAgent SHALL return a successful AgentOutput (Success true) even when using fallback values, because a neutral sentiment is a valid analysis result

### Requirement 7: FundamentalOutput Structure

**User Story:** As a downstream consumer, I want the FundamentalOutput to contain all specified fields, so that I can use sentiment data for decision making.

#### Acceptance Criteria

1. THE FundamentalAgent SHALL populate the Sentiment field with one of "bullish", "bearish", or "neutral"
2. THE FundamentalAgent SHALL populate the Confidence field with a float64 between 0.0 and 1.0
3. THE FundamentalAgent SHALL populate the Score field with the normalized value computed from sentiment and confidence
4. THE FundamentalAgent SHALL populate the Reason field with a string of 15 words or fewer explaining the sentiment determination
5. THE FundamentalAgent SHALL populate the FromCache field with true when the result was served from Redis cache, and false otherwise

### Requirement 8: Gemini Prompt Construction

**User Story:** As a FundamentalAgent, I want to construct a well-defined prompt for Gemini, so that responses are consistent and parseable.

#### Acceptance Criteria

1. THE GeminiClient SHALL include the currency pair in the prompt to provide context for sentiment analysis
2. THE GeminiClient SHALL include all fetched headlines separated by newlines in the prompt
3. THE GeminiClient SHALL instruct Gemini to respond with a valid JSON object containing sentiment, confidence, and reason fields only
4. THE GeminiClient SHALL specify in the prompt that the reason field is limited to 15 words maximum
