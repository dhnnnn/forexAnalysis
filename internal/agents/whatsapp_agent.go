package agents

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// ════════════════════════════════════════════════════════════════════════
// WhatsAppAgent (Agent 6) — mengirim notifikasi sinyal trading via WhatsApp
// ════════════════════════════════════════════════════════════════════════

// Compile-time check that WhatsAppAgent implements Agent.
var _ Agent = (*WhatsAppAgent)(nil)

// WhatsAppConfig menyimpan konfigurasi untuk WhatsApp notification.
type WhatsAppConfig struct {
	ServiceURL           string  // URL service Node.js (e.g. "http://localhost:3001")
	TargetPhone          string  // Nomor telepon tujuan
	MinConfidenceToAlert float64 // Minimum confidence untuk kirim alert (default: 0.60)
	RateLimitSeconds     int     // Rate limit antar pesan per pair (default: 180)
}

// WhatsAppAgent (Agent 6) mengirim notifikasi sinyal trading ke WhatsApp
// via HTTP POST ke service Node.js eksternal.
type WhatsAppAgent struct {
	config     WhatsAppConfig
	httpClient *http.Client
	lastSent   map[string]time.Time // rate limiter per pair
	mu         sync.Mutex
}

// NewWhatsAppAgent membuat instance WhatsAppAgent baru.
func NewWhatsAppAgent(config WhatsAppConfig) *WhatsAppAgent {
	return &WhatsAppAgent{
		config: config,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		lastSent: make(map[string]time.Time),
	}
}

// Name mengembalikan identifier agent.
func (a *WhatsAppAgent) Name() string {
	return "WhatsAppAgent"
}

// whatsAppPayload adalah struktur JSON yang dikirim ke service Node.js.
type whatsAppPayload struct {
	Phone   string         `json:"phone"`
	Message string         `json:"message"`
	Signal  *DecisionOutput `json:"signal"`
}

// Run mengeksekusi logika notifikasi WhatsApp.
// Hanya mengirim pesan jika:
// 1. Context belum cancelled
// 2. Ada DecisionOutput yang valid
// 3. Signal bukan HOLD
// 4. Confidence >= MinConfidenceToAlert
// 5. Rate limit per pair belum terlampaui
func (a *WhatsAppAgent) Run(ctx context.Context, input AgentInput) AgentOutput {
	// 1. Check context cancellation
	if ctx.Err() != nil {
		return errorOutput(a.Name(), fmt.Errorf("context cancelled: %w", ctx.Err()))
	}

	// 2. Validate DecisionOutput exists
	if input.Decision == nil {
		return AgentOutput{
			AgentName: a.Name(),
			Success:   true,
			Timestamp: time.Now(),
			// No notification needed — no decision available
		}
	}

	decision := input.Decision

	// 3. Skip HOLD signals — no notification needed
	if decision.Signal == "HOLD" {
		return AgentOutput{
			AgentName: a.Name(),
			Success:   true,
			Timestamp: time.Now(),
		}
	}

	// 4. Check minimum confidence threshold
	if decision.Confidence < a.config.MinConfidenceToAlert {
		return AgentOutput{
			AgentName: a.Name(),
			Success:   true,
			Timestamp: time.Now(),
		}
	}

	// 5. Check rate limit per pair
	if a.isRateLimited(input.Pair) {
		return AgentOutput{
			AgentName: a.Name(),
			Success:   true,
			Timestamp: time.Now(),
		}
	}

	// 6. Build message
	message := a.buildMessage(input.Pair, decision)

	// 7. Send HTTP POST to WhatsApp service
	err := a.sendNotification(ctx, message, decision)
	if err != nil {
		return errorOutput(a.Name(), fmt.Errorf("failed to send notification: %w", err))
	}

	// 8. Update rate limiter
	a.markSent(input.Pair)

	return AgentOutput{
		AgentName: a.Name(),
		Success:   true,
		Timestamp: time.Now(),
	}
}

// isRateLimited memeriksa apakah notifikasi untuk pair ini masih dalam rate limit.
func (a *WhatsAppAgent) isRateLimited(pair string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()

	lastTime, exists := a.lastSent[pair]
	if !exists {
		return false
	}

	cooldown := time.Duration(a.config.RateLimitSeconds) * time.Second
	return time.Since(lastTime) < cooldown
}

// markSent mencatat waktu pengiriman terakhir untuk rate limiting.
func (a *WhatsAppAgent) markSent(pair string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.lastSent[pair] = time.Now()
}

// buildMessage memformat pesan trading signal untuk WhatsApp.
func (a *WhatsAppAgent) buildMessage(pair string, d *DecisionOutput) string {
	emoji := "🟢"
	if d.Signal == "SELL" {
		emoji = "🔴"
	}

	msg := fmt.Sprintf(`%s *%s %s*

📊 Confidence: %d%% | Risk: %s

💰 Entry: %.5f
🛑 SL: %.5f
🎯 TP: %.5f
📐 Lot: %.2f

📈 Tech: %s (%.0f%%)
📰 Fund: %s (%.0f%%)

⏰ %s`,
		emoji, d.Signal, pair,
		d.ConfPct, d.RiskLevel,
		d.Entry,
		d.StopLoss,
		d.TakeProfit,
		d.LotSize,
		d.TechSignal, d.TechConf*100,
		d.FundSentiment, d.FundConf*100,
		time.Now().Format("15:04:05 MST"),
	)

	return msg
}

// sendNotification mengirim HTTP POST ke service Node.js WhatsApp.
func (a *WhatsAppAgent) sendNotification(ctx context.Context, message string, decision *DecisionOutput) error {
	payload := whatsAppPayload{
		Phone:   a.config.TargetPhone,
		Message: message,
		Signal:  decision,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	url := a.config.ServiceURL + "/send"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("whatsapp service returned status %d", resp.StatusCode)
	}

	return nil
}
