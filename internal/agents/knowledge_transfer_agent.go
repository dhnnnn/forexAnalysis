package agents

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

	"github.com/dhnnnn/forexAnalysis/internal/knowledge"
	"github.com/google/uuid"
)

// ════════════════════════════════════════════════════════════════════════
// KnowledgeTransferAgent — mengubah ExperienceReport menjadi KnowledgeRule
// ════════════════════════════════════════════════════════════════════════
//
// Komponen novelty utama: LLM digunakan BUKAN untuk prediksi harga,
// melainkan untuk mengekstrak "mengapa sebuah agen gagal" dan menyusunnya
// menjadi aturan (KnowledgeRule) yang di-broadcast ke agen lain.

// KTAConfig menyimpan konfigurasi untuk KnowledgeTransferAgent.
type KTAConfig struct {
	GeminiAPIKey string
	GeminiModel  string
	GroqAPIKey   string
	GroqModel    string
	Timeout      time.Duration
	RuleTTL      time.Duration
	MinConfidence float64
}

// KnowledgeTransferAgent mengubah ExperienceReport menjadi KnowledgeRule
// menggunakan LLM (Gemini primary, Groq fallback) untuk reasoning.
type KnowledgeTransferAgent struct {
	geminiAPIKey string
	geminiModel  string
	groqAPIKey   string
	groqModel    string
	timeout      time.Duration
	client       *http.Client
	store        *knowledge.Store
	ruleTTL      time.Duration
	minConfidence float64
}

// NewKnowledgeTransferAgent membuat instance baru.
func NewKnowledgeTransferAgent(cfg KTAConfig, store *knowledge.Store) *KnowledgeTransferAgent {
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 10 * time.Second
	}
	ruleTTL := cfg.RuleTTL
	if ruleTTL == 0 {
		ruleTTL = 24 * time.Hour
	}
	minConf := cfg.MinConfidence
	if minConf == 0 {
		minConf = 0.60
	}

	return &KnowledgeTransferAgent{
		geminiAPIKey: cfg.GeminiAPIKey,
		geminiModel:  cfg.GeminiModel,
		groqAPIKey:   cfg.GroqAPIKey,
		groqModel:    cfg.GroqModel,
		timeout:      timeout,
		client:       &http.Client{Timeout: timeout + 2*time.Second},
		store:        store,
		ruleTTL:      ruleTTL,
		minConfidence: minConf,
	}
}

// Name mengembalikan identifier agent.
func (k *KnowledgeTransferAgent) Name() string {
	return "KnowledgeTransferAgent"
}

// Process menerima slice ExperienceReport dan menghasilkan KnowledgeRule.
// Dipanggil setelah MetaObserverAgent.Observe() menghasilkan report.
// Return: slice of rules yang berhasil dibuat dan disimpan ke KB.
func (k *KnowledgeTransferAgent) Process(ctx context.Context, reports []knowledge.ExperienceReport) []knowledge.KnowledgeRule {
	var rules []knowledge.KnowledgeRule

	for _, report := range reports {
		rule, err := k.processOne(ctx, report)
		if err != nil {
			slog.Warn("[KTA] failed to process report",
				"agent", report.AgentName,
				"error", err,
			)
			continue
		}

		if rule.Confidence < k.minConfidence {
			slog.Debug("[KTA] rule confidence too low, skipping",
				"agent", report.AgentName,
				"confidence", fmt.Sprintf("%.2f", rule.Confidence),
				"min_required", fmt.Sprintf("%.2f", k.minConfidence),
			)
			continue
		}

		// Simpan ke Redis KB
		if err := k.store.SaveRule(ctx, *rule); err != nil {
			slog.Warn("[KTA] failed to save rule", "error", err)
			continue
		}

		rules = append(rules, *rule)
		slog.Info("✨ KTA: new rule generated",
			"source_agent", rule.SourceAgent,
			"target_agent", rule.Action.TargetAgent,
			"regime", string(rule.Condition.Regime),
			"weight_delta", fmt.Sprintf("%.2f", rule.Action.WeightDelta),
			"confidence", fmt.Sprintf("%.2f", rule.Confidence),
			"reasoning", rule.Reasoning,
		)
	}

	return rules
}

// processOne mengubah satu ExperienceReport menjadi KnowledgeRule via LLM.
func (k *KnowledgeTransferAgent) processOne(ctx context.Context, report knowledge.ExperienceReport) (*knowledge.KnowledgeRule, error) {
	prompt := k.buildPrompt(report)

	// Coba Gemini dulu, fallback ke Groq
	raw, err := k.callGemini(ctx, prompt)
	if err != nil {
		slog.Debug("[KTA] Gemini failed, trying Groq", "error", err)
		raw, err = k.callGroq(ctx, prompt)
		if err != nil {
			return nil, fmt.Errorf("both LLM clients failed: %w", err)
		}
	}

	return k.parseResponse(raw, report)
}

// buildPrompt membangun prompt yang dikirim ke LLM.
// Prompt dirancang untuk menghasilkan JSON terstruktur, bukan teks bebas.
func (k *KnowledgeTransferAgent) buildPrompt(r knowledge.ExperienceReport) string {
	return fmt.Sprintf(`You are an expert forex trading system analyst specializing in multi-agent systems.

An agent in our forex trading pipeline has experienced a significant performance drop:

Agent: %s
Accuracy before: %.1f%%
Accuracy now: %.1f%%
Accuracy drop: %.1f%%
Loss streak: %d consecutive incorrect signals
Market regime during failure: %s

Your task: Analyze WHY this agent is failing and generate exactly ONE knowledge rule that can be broadcast to other agents to prevent cascading failures.

The rule should reduce the weight of the failing agent in the specified market regime.

Respond ONLY with valid JSON in this exact format (no markdown, no explanation outside JSON):
{
  "condition": {
    "regime": "%s",
    "adx_below": null,
    "adx_above": null,
    "vol_below": null,
    "vol_above": null
  },
  "action": {
    "agent": "%s",
    "weight_delta": -0.2,
    "min_weight": 0.05
  },
  "confidence": 0.75,
  "reasoning": "one clear sentence explaining the failure cause"
}

RULES:
- weight_delta MUST be negative (between -0.5 and -0.1)
- confidence MUST be between 0.0 and 1.0
- Set adx_below/adx_above/vol_below/vol_above to numbers ONLY if they are relevant constraints, otherwise null
- reasoning must be ONE sentence, max 20 words`,
		r.AgentName,
		r.AccuracyBefore*100,
		r.AccuracyNow*100,
		r.AccuracyDelta*100,
		r.LossStreak,
		string(r.ActiveRegime),
		string(r.ActiveRegime),
		r.AgentName,
	)
}

// ════════════════════════════════════════════════════════════════════════
// LLM Response Parsing
// ════════════════════════════════════════════════════════════════════════

// ktaLLMResponse adalah struct untuk parsing JSON response dari LLM.
type ktaLLMResponse struct {
	Condition struct {
		Regime   string   `json:"regime"`
		ADXBelow *float64 `json:"adx_below"`
		ADXAbove *float64 `json:"adx_above"`
		VolBelow *float64 `json:"vol_below"`
		VolAbove *float64 `json:"vol_above"`
	} `json:"condition"`
	Action struct {
		Agent       string  `json:"agent"`
		WeightDelta float64 `json:"weight_delta"`
		MinWeight   float64 `json:"min_weight"`
	} `json:"action"`
	Confidence float64 `json:"confidence"`
	Reasoning  string  `json:"reasoning"`
}

// parseResponse mengubah teks JSON dari LLM menjadi KnowledgeRule struct.
func (k *KnowledgeTransferAgent) parseResponse(raw string, report knowledge.ExperienceReport) (*knowledge.KnowledgeRule, error) {
	// Bersihkan response jika ada markdown wrapper
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "```json")
	raw = strings.TrimPrefix(raw, "```")
	raw = strings.TrimSuffix(raw, "```")
	raw = strings.TrimSpace(raw)

	var parsed ktaLLMResponse
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return nil, fmt.Errorf("invalid JSON from LLM: %w (raw: %.200s)", err, raw)
	}

	// Validasi weight_delta harus negatif
	if parsed.Action.WeightDelta > 0 {
		parsed.Action.WeightDelta = -0.2 // force negatif
	}
	if parsed.Action.WeightDelta < -0.5 {
		parsed.Action.WeightDelta = -0.5 // cap at -0.5
	}

	// Validasi confidence
	if parsed.Confidence < 0 {
		parsed.Confidence = 0
	}
	if parsed.Confidence > 1.0 {
		parsed.Confidence = 1.0
	}

	// Validasi min_weight
	if parsed.Action.MinWeight <= 0 {
		parsed.Action.MinWeight = 0.05
	}

	rule := &knowledge.KnowledgeRule{
		ID: uuid.New().String(),
		Condition: knowledge.RuleCondition{
			Regime:   knowledge.MarketRegime(parsed.Condition.Regime),
			ADXBelow: parsed.Condition.ADXBelow,
			ADXAbove: parsed.Condition.ADXAbove,
			VolBelow: parsed.Condition.VolBelow,
			VolAbove: parsed.Condition.VolAbove,
		},
		Action: knowledge.RuleAction{
			TargetAgent: parsed.Action.Agent,
			WeightDelta: parsed.Action.WeightDelta,
			MinWeight:   parsed.Action.MinWeight,
		},
		SourceAgent: report.AgentName,
		Confidence:  parsed.Confidence,
		Reasoning:   parsed.Reasoning,
		CreatedAt:   time.Now(),
		ExpiresAt:   time.Now().Add(k.ruleTTL),
		ApplyCount:  0,
	}

	return rule, nil
}

// ════════════════════════════════════════════════════════════════════════
// Gemini API Call
// ════════════════════════════════════════════════════════════════════════

type ktaGeminiRequest struct {
	Contents []ktaGeminiContent `json:"contents"`
}

type ktaGeminiContent struct {
	Parts []ktaGeminiPart `json:"parts"`
}

type ktaGeminiPart struct {
	Text string `json:"text"`
}

type ktaGeminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

func (k *KnowledgeTransferAgent) callGemini(ctx context.Context, prompt string) (string, error) {
	if k.geminiAPIKey == "" {
		return "", fmt.Errorf("gemini API key not configured")
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s",
		k.geminiModel, k.geminiAPIKey)

	reqBody := ktaGeminiRequest{
		Contents: []ktaGeminiContent{
			{Parts: []ktaGeminiPart{{Text: prompt}}},
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, k.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(timeoutCtx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := k.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("gemini API call failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("gemini API error %d: %s", resp.StatusCode, string(body[:minInt(len(body), 200)]))
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	var gemResp ktaGeminiResponse
	if err := json.Unmarshal(respBody, &gemResp); err != nil {
		return "", fmt.Errorf("parse gemini response: %w", err)
	}

	if len(gemResp.Candidates) == 0 || len(gemResp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("empty response from gemini")
	}

	return gemResp.Candidates[0].Content.Parts[0].Text, nil
}

// ════════════════════════════════════════════════════════════════════════
// Groq API Call (OpenAI-compatible fallback)
// ════════════════════════════════════════════════════════════════════════

type ktaGroqRequest struct {
	Model    string           `json:"model"`
	Messages []ktaGroqMessage `json:"messages"`
}

type ktaGroqMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ktaGroqResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func (k *KnowledgeTransferAgent) callGroq(ctx context.Context, prompt string) (string, error) {
	if k.groqAPIKey == "" {
		return "", fmt.Errorf("groq API key not configured")
	}

	url := "https://api.groq.com/openai/v1/chat/completions"

	reqBody := ktaGroqRequest{
		Model: k.groqModel,
		Messages: []ktaGroqMessage{
			{Role: "user", Content: prompt},
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, k.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(timeoutCtx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+k.groqAPIKey)

	resp, err := k.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("groq API call failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("groq API error %d: %s", resp.StatusCode, string(body[:minInt(len(body), 200)]))
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	var groqResp ktaGroqResponse
	if err := json.Unmarshal(respBody, &groqResp); err != nil {
		return "", fmt.Errorf("parse groq response: %w", err)
	}

	if len(groqResp.Choices) == 0 {
		return "", fmt.Errorf("empty response from groq")
	}

	return groqResp.Choices[0].Message.Content, nil
}

// minInt returns the minimum of two ints.
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
