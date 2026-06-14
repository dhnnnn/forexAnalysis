package chatbot

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// ════════════════════════════════════════════════════════════════════════
// GeminiChat — AI chat integration untuk WhatsApp bot
// ════════════════════════════════════════════════════════════════════════

// GeminiChat menangani percakapan AI dengan Gemini API + Groq fallback.
type GeminiChat struct {
	apiKey  string
	model   string
	client  *http.Client
	timeout time.Duration

	// Groq fallback
	groqAPIKey string
	groqModel  string
}

// NewGeminiChat membuat instance GeminiChat baru.
func NewGeminiChat(apiKey, model string, timeout time.Duration) *GeminiChat {
	return &GeminiChat{
		apiKey:  apiKey,
		model:   model,
		timeout: timeout,
		client: &http.Client{
			Timeout: timeout + 2*time.Second, // extra buffer
		},
	}
}

// SetGroqFallback mengatur Groq sebagai fallback jika Gemini kena rate limit.
func (g *GeminiChat) SetGroqFallback(apiKey, model string) {
	g.groqAPIKey = apiKey
	g.groqModel = model
}

// ChatContext berisi informasi user yang dikirim ke Gemini sebagai context.
type ChatContext struct {
	Balance     float64
	RiskPercent float64
	Pairs       []string
	LastSignal  string // "BUY EUR_USD 65%" atau "HOLD"
}

// Ask mengirim pertanyaan user ke AI dengan context trading.
// Coba Gemini dulu → kalau 429/error → fallback ke Groq.
func (g *GeminiChat) Ask(ctx context.Context, userMessage string, chatCtx ChatContext) string {
	prompt := g.buildChatPrompt(userMessage, chatCtx)

	// Try Gemini first
	if g.apiKey != "" {
		answer, err := g.callGemini(ctx, prompt)
		if err == nil {
			return answer
		}
		slog.Warn("GeminiChat error, trying Groq fallback", "error", err)
	}

	// Fallback to Groq
	if g.groqAPIKey != "" {
		answer, err := g.callGroq(ctx, prompt)
		if err == nil {
			return answer
		}
		slog.Warn("Groq fallback also failed", "error", err)
	}

	return "⚠️ AI sedang tidak tersedia (Gemini & Groq keduanya limit). Coba lagi nanti atau ketik /help."
}

// buildChatPrompt membuat prompt dengan system context + user message.
func (g *GeminiChat) buildChatPrompt(userMessage string, chatCtx ChatContext) string {
	var sb strings.Builder

	// System prompt — strict forex context
	sb.WriteString(`Kamu adalah asisten trading forex profesional bernama "ForexBot".

ATURAN KETAT:
1. Kamu HANYA boleh menjawab pertanyaan tentang forex trading, risk management, lot size, money management, analisis teknikal, dan topik terkait trading.
2. Jika user bertanya di luar topik forex/trading, tolak dengan sopan dan arahkan kembali ke topik trading.
3. Jawab dalam Bahasa Indonesia yang santai tapi tetap profesional.
4. Berikan angka konkret dan rekomendasi yang actionable.
5. Selalu ingatkan bahwa trading memiliki risiko dan ini bukan financial advice yang dijamin profit.
6. Jawab singkat dan padat (maksimal 200 kata). Gunakan emoji untuk clarity.
7. Jika user menyebut nominal modal, hitung lot size dan risk amount-nya.

FORMULA YANG KAMU GUNAKAN:
- Risk Amount = Balance × (Risk% / 100)
- Lot Size = Risk Amount / (SL_Pips × $10 per pip per lot)
- Risk:Reward ratio ideal = 1:2 minimum
- Max risk per trade = 1-2% untuk pemula, 2-5% untuk experienced

`)

	// User context
	sb.WriteString("DATA USER SAAT INI:\n")
	sb.WriteString(fmt.Sprintf("- Balance: $%.2f\n", chatCtx.Balance))
	sb.WriteString(fmt.Sprintf("- Risk per trade: %.1f%%\n", chatCtx.RiskPercent))
	sb.WriteString(fmt.Sprintf("- Risk amount: $%.2f\n", chatCtx.Balance*(chatCtx.RiskPercent/100)))
	sb.WriteString(fmt.Sprintf("- Pairs yang dimonitor: %s\n", strings.Join(chatCtx.Pairs, ", ")))
	if chatCtx.LastSignal != "" {
		sb.WriteString(fmt.Sprintf("- Signal terakhir: %s\n", chatCtx.LastSignal))
	}
	sb.WriteString("\n")

	// User message
	sb.WriteString(fmt.Sprintf("PERTANYAAN USER: %s\n", userMessage))

	return sb.String()
}

// geminiChatRequest and response structs
type geminiChatRequest struct {
	Contents []geminiChatContent `json:"contents"`
}

type geminiChatContent struct {
	Parts []geminiChatPart `json:"parts"`
}

type geminiChatPart struct {
	Text string `json:"text"`
}

type geminiChatResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

// callGemini performs the API call to Gemini.
func (g *GeminiChat) callGemini(ctx context.Context, prompt string) (string, error) {
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", g.model, g.apiKey)

	reqBody := geminiChatRequest{
		Contents: []geminiChatContent{
			{
				Parts: []geminiChatPart{
					{Text: prompt},
				},
			},
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal: %w", err)
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, g.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(timeoutCtx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("API call failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API error %d: %s", resp.StatusCode, string(body[:min(len(body), 200)]))
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	var gemResp geminiChatResponse
	if err := json.Unmarshal(respBody, &gemResp); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}

	if len(gemResp.Candidates) == 0 || len(gemResp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("empty response from Gemini")
	}

	answer := gemResp.Candidates[0].Content.Parts[0].Text
	answer = strings.TrimSpace(answer)

	// Truncate if too long for WhatsApp (max ~1000 chars)
	if len(answer) > 1000 {
		answer = answer[:997] + "..."
	}

	return answer, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ════════════════════════════════════════════════════════════════════════
// Groq Fallback — OpenAI-compatible API
// ════════════════════════════════════════════════════════════════════════

type groqRequest struct {
	Model    string        `json:"model"`
	Messages []groqMessage `json:"messages"`
}

type groqMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type groqResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// callGroq calls the Groq API (OpenAI-compatible).
func (g *GeminiChat) callGroq(ctx context.Context, prompt string) (string, error) {
	url := "https://api.groq.com/openai/v1/chat/completions"

	reqBody := groqRequest{
		Model: g.groqModel,
		Messages: []groqMessage{
			{Role: "user", Content: prompt},
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal: %w", err)
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, g.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(timeoutCtx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+g.groqAPIKey)

	resp, err := g.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("Groq API call failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Groq API error %d: %s", resp.StatusCode, string(body[:min(len(body), 200)]))
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	var groqResp groqResponse
	if err := json.Unmarshal(respBody, &groqResp); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}

	if len(groqResp.Choices) == 0 {
		return "", fmt.Errorf("empty response from Groq")
	}

	answer := strings.TrimSpace(groqResp.Choices[0].Message.Content)
	if len(answer) > 1000 {
		answer = answer[:997] + "..."
	}

	return answer, nil
}
