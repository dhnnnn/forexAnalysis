package sentiment

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Compile-time interface satisfaction check.
var _ SentimentAnalyzer = (*GeminiClient)(nil)

// GeminiClient communicates with the Gemini API to perform sentiment analysis
// on forex news headlines. It implements the SentimentAnalyzer interface.
type GeminiClient struct {
	apiKey  string
	model   string
	timeout time.Duration
	client  *http.Client
}

// geminiRequest represents the request body for the Gemini API.
type geminiRequest struct {
	Contents []geminiContent `json:"contents"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

// geminiResponse represents the response structure from the Gemini API.
type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

// sentimentJSON is the expected JSON structure from the Gemini response text.
type sentimentJSON struct {
	Sentiment  string  `json:"sentiment"`
	Confidence float64 `json:"confidence"`
	Reason     string  `json:"reason"`
}

// NewGeminiClient creates a new GeminiClient with the given API key, model name,
// and timeout duration for HTTP requests.
func NewGeminiClient(apiKey, model string, timeout time.Duration) *GeminiClient {
	return &GeminiClient{
		apiKey:  apiKey,
		model:   model,
		timeout: timeout,
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

// AnalyzeSentiment sends the headlines to the Gemini API and returns a SentimentResult.
// On any error (timeout, invalid JSON, missing fields, etc.), it returns a neutral fallback.
func (g *GeminiClient) AnalyzeSentiment(ctx context.Context, pair string, headlines []string) SentimentResult {
	prompt := BuildPrompt(pair, headlines)

	result, err := g.callGemini(ctx, prompt)
	if err != nil {
		return neutralFallback(err.Error())
	}

	return result
}

// BuildPrompt constructs the Gemini prompt from a currency pair and list of headlines.
func BuildPrompt(pair string, headlines []string) string {
	var sb strings.Builder

	sb.WriteString("You are a professional forex market analyst.\n")
	sb.WriteString(fmt.Sprintf("Analyze the sentiment impact of these news headlines on the %s currency pair.\n", pair))
	sb.WriteString("\nHeadlines:\n")

	for _, h := range headlines {
		sb.WriteString(h)
		sb.WriteString("\n")
	}

	sb.WriteString("\nRespond ONLY with a valid JSON object. No explanation, no markdown:\n")
	sb.WriteString("{\n")
	sb.WriteString("  \"sentiment\": \"bullish\" OR \"bearish\" OR \"neutral\",\n")
	sb.WriteString("  \"confidence\": 0.0 to 1.0,\n")
	sb.WriteString("  \"reason\": \"max 15 words explanation\"\n")
	sb.WriteString("}\n")

	return sb.String()
}

// callGemini performs the HTTP POST to the Gemini API and parses the response.
func (g *GeminiClient) callGemini(ctx context.Context, prompt string) (SentimentResult, error) {
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", g.model, g.apiKey)

	reqBody := geminiRequest{
		Contents: []geminiContent{
			{
				Parts: []geminiPart{
					{Text: prompt},
				},
			},
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return SentimentResult{}, fmt.Errorf("marshal request: %w", err)
	}

	// Create request with a 2-second timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, g.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(timeoutCtx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return SentimentResult{}, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.client.Do(req)
	if err != nil {
		return SentimentResult{}, fmt.Errorf("Gemini API unavailable")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return SentimentResult{}, fmt.Errorf("Gemini API unavailable")
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return SentimentResult{}, fmt.Errorf("read response: %w", err)
	}

	return parseGeminiResponse(respBody)
}

// parseGeminiResponse extracts and validates the sentiment JSON from the Gemini API response.
func parseGeminiResponse(body []byte) (SentimentResult, error) {
	var gemResp geminiResponse
	if err := json.Unmarshal(body, &gemResp); err != nil {
		return SentimentResult{}, fmt.Errorf("invalid Gemini response")
	}

	if len(gemResp.Candidates) == 0 ||
		len(gemResp.Candidates[0].Content.Parts) == 0 {
		return SentimentResult{}, fmt.Errorf("invalid Gemini response")
	}

	text := gemResp.Candidates[0].Content.Parts[0].Text
	text = cleanJSONText(text)

	var parsed sentimentJSON
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		return SentimentResult{}, fmt.Errorf("invalid Gemini response")
	}

	// Validate sentiment value
	sentiment := strings.ToLower(strings.TrimSpace(parsed.Sentiment))
	if sentiment != "bullish" && sentiment != "bearish" && sentiment != "neutral" {
		sentiment = "neutral"
	}

	// Clamp confidence to [0.0, 1.0]
	confidence := clampConfidence(parsed.Confidence)

	return SentimentResult{
		Sentiment:  sentiment,
		Confidence: confidence,
		Reason:     parsed.Reason,
	}, nil
}

// cleanJSONText removes common markdown formatting from Gemini's response text.
func cleanJSONText(text string) string {
	text = strings.TrimSpace(text)
	// Remove markdown code fences if present
	text = strings.TrimPrefix(text, "```json")
	text = strings.TrimPrefix(text, "```")
	text = strings.TrimSuffix(text, "```")
	text = strings.TrimSpace(text)
	return text
}

// clampConfidence restricts a confidence value to [0.0, 1.0].
func clampConfidence(c float64) float64 {
	if c < 0.0 {
		return 0.0
	}
	if c > 1.0 {
		return 1.0
	}
	return c
}

// neutralFallback returns a neutral SentimentResult with the given reason.
func neutralFallback(reason string) SentimentResult {
	return SentimentResult{
		Sentiment:  "neutral",
		Confidence: 0.5,
		Reason:     reason,
	}
}
