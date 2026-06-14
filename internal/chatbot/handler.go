package chatbot

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/dhnnnn/forex-agent/internal/agents"
)

// ════════════════════════════════════════════════════════════════════════
// ChatBot Handler — menerima pesan dari WhatsApp dan merespons
// ════════════════════════════════════════════════════════════════════════

// ChatRequest adalah payload yang diterima dari WhatsApp service.
type ChatRequest struct {
	Phone     string `json:"phone"`
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
}

// ChatResponse adalah balasan yang dikirim kembali ke WhatsApp service.
type ChatResponse struct {
	Reply string `json:"reply"`
}

// UserState menyimpan state per user (balance, risk, preferences).
type UserState struct {
	Balance     float64 `json:"balance"`
	RiskPercent float64 `json:"risk_percent"`
}

// Handler menangani incoming chat messages dan menghasilkan respons.
type Handler struct {
	userState   *UserState
	mu          sync.RWMutex
	getStatus   func() string // callback untuk mendapatkan pipeline status
	geminiChat  *GeminiChat   // AI chat integration
	pairs       []string      // pairs yang dimonitor
	lastSignal  string        // signal terakhir dari pipeline
}

// NewHandler membuat Handler baru dengan default state.
func NewHandler() *Handler {
	return &Handler{
		userState: &UserState{
			Balance:     1000.0,
			RiskPercent: 1.0,
		},
		pairs: []string{"EUR_USD", "GBP_USD"},
	}
}

// SetGeminiChat mengatur Gemini AI chat client.
func (h *Handler) SetGeminiChat(gc *GeminiChat) {
	h.geminiChat = gc
}

// SetPairs mengatur list pairs yang dimonitor.
func (h *Handler) SetPairs(pairs []string) {
	h.pairs = pairs
}

// SetLastSignal mengupdate signal terakhir dari pipeline.
func (h *Handler) SetLastSignal(signal string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.lastSignal = signal
}

// SetStatusFunc mengatur callback untuk mendapatkan pipeline status.
func (h *Handler) SetStatusFunc(fn func() string) {
	h.getStatus = fn
}

// GetUserState mengembalikan current user state (thread-safe).
func (h *Handler) GetUserState() UserState {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return *h.userState
}

// ServeHTTP handles POST /chat requests.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	slog.Info("📩 Chat received", "phone", req.Phone, "message", req.Message)

	// Process command and generate reply
	reply := h.processMessage(req.Message)

	resp := ChatResponse{Reply: reply}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// processMessage memproses pesan masuk dan menghasilkan balasan.
func (h *Handler) processMessage(msg string) string {
	msg = strings.TrimSpace(msg)
	lower := strings.ToLower(msg)

	// Command routing
	switch {
	case lower == "/help" || lower == "help" || lower == "menu":
		return h.cmdHelp()

	case lower == "/status" || lower == "status":
		return h.cmdStatus()

	case strings.HasPrefix(lower, "/set balance ") || strings.HasPrefix(lower, "set balance "):
		return h.cmdSetBalance(msg)

	case strings.HasPrefix(lower, "/set risk ") || strings.HasPrefix(lower, "set risk "):
		return h.cmdSetRisk(msg)

	case lower == "/analyze" || lower == "analyze" || lower == "scan":
		return h.cmdAnalyze()

	case lower == "/risk" || lower == "risk":
		return h.cmdRiskInfo()

	default:
		return h.cmdDefault(msg)
	}
}

// ════════════════════════════════════════════════════════════════════════
// Commands
// ════════════════════════════════════════════════════════════════════════

func (h *Handler) cmdHelp() string {
	return `🤖 *Forex Bot Commands*

📋 *Info*
• /status — Lihat setting & status bot
• /risk — Lihat kalkulasi risk management

⚙️ *Settings*
• /set balance <nominal> — Set balance trading
• /set risk <persen> — Set risk % per trade

📊 *Analysis*
• /analyze — Force scan semua pair sekarang

💡 Contoh:
• set balance 500
• set risk 2
• status`
}

func (h *Handler) cmdStatus() string {
	h.mu.RLock()
	state := *h.userState
	h.mu.RUnlock()

	status := "🟢 Running"
	if h.getStatus != nil {
		status = h.getStatus()
	}

	return fmt.Sprintf(`📊 *Bot Status*

💰 Balance: $%.2f
⚠️ Risk: %.1f%% per trade
📈 Risk Amount: $%.2f

🤖 Pipeline: %s
🕐 Pairs: EUR_USD, GBP_USD
⏰ Timeframe: 1h`,
		state.Balance,
		state.RiskPercent,
		state.Balance*(state.RiskPercent/100),
		status,
	)
}

func (h *Handler) cmdSetBalance(msg string) string {
	// Parse number from message
	parts := strings.Fields(msg)
	if len(parts) < 3 {
		return "⚠️ Format: /set balance <nominal>\nContoh: /set balance 500"
	}

	var amount float64
	_, err := fmt.Sscanf(parts[len(parts)-1], "%f", &amount)
	if err != nil || amount <= 0 {
		return "⚠️ Nominal tidak valid. Masukkan angka positif.\nContoh: /set balance 500"
	}

	h.mu.Lock()
	h.userState.Balance = amount
	h.mu.Unlock()

	riskAmount := amount * (h.userState.RiskPercent / 100)
	lotSize := riskAmount / (20.0 * 10.0) // SL 20 pips, $10/pip per lot

	return fmt.Sprintf(`✅ *Balance diupdate!*

💰 Balance: $%.2f
⚠️ Risk per trade: %.1f%% = $%.2f
📐 Lot size (SL 20 pips): %.2f

Bot akan menggunakan setting ini untuk kalkulasi berikutnya.`,
		amount, h.userState.RiskPercent, riskAmount, lotSize)
}

func (h *Handler) cmdSetRisk(msg string) string {
	parts := strings.Fields(msg)
	if len(parts) < 3 {
		return "⚠️ Format: /set risk <persen>\nContoh: /set risk 2"
	}

	var pct float64
	_, err := fmt.Sscanf(parts[len(parts)-1], "%f", &pct)
	if err != nil || pct <= 0 || pct > 10 {
		return "⚠️ Risk harus antara 0.1% - 10%.\nContoh: /set risk 2"
	}

	h.mu.Lock()
	h.userState.RiskPercent = pct
	h.mu.Unlock()

	riskAmount := h.userState.Balance * (pct / 100)
	lotSize := riskAmount / (20.0 * 10.0)

	return fmt.Sprintf(`✅ *Risk diupdate!*

⚠️ Risk: %.1f%% per trade
💰 Balance: $%.2f
💸 Risk amount: $%.2f
📐 Lot size (SL 20 pips): %.2f

⚡ Semakin tinggi risk %%, semakin besar lot tapi semakin bahaya!`,
		pct, h.userState.Balance, riskAmount, lotSize)
}

func (h *Handler) cmdRiskInfo() string {
	h.mu.RLock()
	state := *h.userState
	h.mu.RUnlock()

	riskAmount := state.Balance * (state.RiskPercent / 100)
	lotSize := riskAmount / (20.0 * 10.0)

	// Risk per SL distance
	sl15 := riskAmount / (15.0 * 10.0)
	sl20 := lotSize
	sl30 := riskAmount / (30.0 * 10.0)
	sl50 := riskAmount / (50.0 * 10.0)

	return fmt.Sprintf(`📊 *Risk Management Calculator*

💰 Balance: $%.2f | Risk: %.1f%%
💸 Risk Amount: $%.2f

📐 *Lot Size by SL Distance:*
• SL 15 pips → %.2f lot
• SL 20 pips → %.2f lot
• SL 30 pips → %.2f lot
• SL 50 pips → %.2f lot

💡 *Rekomendasi:*
• Balance < $500 → risk 1%%, SL 15-20 pips
• Balance $500-$2000 → risk 1-2%%, SL 20-30 pips
• Balance > $2000 → risk 1-3%%, SL 20-50 pips

⚡ Gunakan /set balance dan /set risk untuk update.`,
		state.Balance, state.RiskPercent, riskAmount,
		sl15, sl20, sl30, sl50)
}

func (h *Handler) cmdAnalyze() string {
	return `🔍 *Scanning pairs...*

Bot secara otomatis memantau EUR_USD dan GBP_USD setiap 10 detik. Jika ada signal BUY/SELL dengan confidence ≥ 60%, kamu akan langsung menerima notifikasi.

📊 Status saat ini: menunggu cukup data candle (min 26).
🕐 Candle mock mode: ~130 detik untuk terkumpul.

Untuk force analysis manual, tunggu pipeline ready.`
}

func (h *Handler) cmdDefault(msg string) string {
	// Forward to Gemini AI for natural conversation
	if h.geminiChat != nil {
		h.mu.RLock()
		chatCtx := ChatContext{
			Balance:     h.userState.Balance,
			RiskPercent: h.userState.RiskPercent,
			Pairs:       h.pairs,
			LastSignal:  h.lastSignal,
		}
		h.mu.RUnlock()

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		return h.geminiChat.Ask(ctx, msg, chatCtx)
	}

	return fmt.Sprintf(`🤖 Saya belum mengerti "%s".

Ketik /help untuk melihat daftar perintah yang tersedia.

💡 AI chat belum aktif — set GEMINI_API_KEY untuk mengaktifkan.`, msg)
}

// ════════════════════════════════════════════════════════════════════════
// Utility
// ════════════════════════════════════════════════════════════════════════

// UpdateFromConfig mengupdate user state dari config (dipanggil saat startup).
func (h *Handler) UpdateFromConfig(balance, riskPercent float64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if balance > 0 {
		h.userState.Balance = balance
	}
	if riskPercent > 0 {
		h.userState.RiskPercent = riskPercent
	}
}

// AgentInput returns an AgentInput populated with user's current settings.
func (h *Handler) AgentInput() agents.AgentInput {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return agents.AgentInput{
		AccountBalance: h.userState.Balance,
		RiskPercent:    h.userState.RiskPercent,
	}
}
