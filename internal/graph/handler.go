package graph

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
)

// ════════════════════════════════════════════════════════════════════════
// GraphQL HTTP + WebSocket Handler
// ════════════════════════════════════════════════════════════════════════
//
// Simplified GraphQL handler tanpa gqlgen generated code.
// Mendukung:
// - POST /graphql — query & mutations
// - WS /graphql — subscriptions (graphql-ws protocol)
//
// Ini approach yang lebih straightforward untuk project ini.

// Handler handles GraphQL HTTP requests dan WebSocket subscriptions.
type Handler struct {
	resolver *Resolver
	upgrader websocket.Upgrader
}

// safeConn wraps a websocket.Conn with a mutex for concurrent writes.
// Multiple subscription goroutines write to the same connection.
type safeConn struct {
	conn *websocket.Conn
	mu   sync.Mutex
}

func (s *safeConn) WriteJSON(v interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.conn.WriteJSON(v)
}

// NewHandler creates a new GraphQL handler.
func NewHandler(resolver *Resolver) *Handler {
	return &Handler{
		resolver: resolver,
		upgrader: websocket.Upgrader{
			// Support graphql-transport-ws subprotocol used by Apollo Client / graphql-ws
			Subprotocols: []string{"graphql-transport-ws", "graphql-ws"},
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for development
			},
		},
	}
}

// ServeHTTP handles both HTTP queries and WebSocket subscription upgrades.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Check for WebSocket upgrade (subscriptions)
	if websocket.IsWebSocketUpgrade(r) {
		h.handleWebSocket(w, r)
		return
	}

	// Handle regular GraphQL query via HTTP POST
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req graphQLRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	result := h.executeQuery(r.Context(), req)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// graphQLRequest represents an incoming GraphQL request.
type graphQLRequest struct {
	Query         string                 `json:"query"`
	OperationName string                 `json:"operationName"`
	Variables     map[string]interface{} `json:"variables"`
}

// graphQLResponse represents a GraphQL response.
type graphQLResponse struct {
	Data   interface{} `json:"data,omitempty"`
	Errors []gqlError  `json:"errors,omitempty"`
}

type gqlError struct {
	Message string `json:"message"`
}

// executeQuery parses and executes a GraphQL query.
// This is a simplified query executor (not full spec) but handles our use case.
func (h *Handler) executeQuery(ctx context.Context, req graphQLRequest) graphQLResponse {
	query := strings.TrimSpace(req.Query)

	// Simple query routing based on field detection
	data := make(map[string]interface{})

	if strings.Contains(query, "candles") {
		pair := extractStringVar(req.Variables, "pair", "EUR_USD")
		timeframe := extractStringVar(req.Variables, "timeframe", "1h")
		limit := extractIntVar(req.Variables, "limit", 200)
		candles, err := h.resolver.Candles(ctx, pair, timeframe, &limit)
		if err != nil {
			return graphQLResponse{Errors: []gqlError{{Message: err.Error()}}}
		}
		data["candles"] = candles
	}

	if strings.Contains(query, "signals") {
		pair := extractStringVarPtr(req.Variables, "pair")
		limit := extractIntVar(req.Variables, "limit", 50)
		signals, err := h.resolver.Signals(ctx, pair, &limit, nil)
		if err != nil {
			return graphQLResponse{Errors: []gqlError{{Message: err.Error()}}}
		}
		data["signals"] = signals
	}

	if strings.Contains(query, "agentSummaries") {
		summaries, err := h.resolver.AgentSummaries(ctx)
		if err != nil {
			return graphQLResponse{Errors: []gqlError{{Message: err.Error()}}}
		}
		data["agentSummaries"] = summaries
	}

	if strings.Contains(query, "activeRules") {
		rules, err := h.resolver.ActiveRules(ctx)
		if err != nil {
			return graphQLResponse{Errors: []gqlError{{Message: err.Error()}}}
		}
		data["activeRules"] = rules
	}

	if strings.Contains(query, "performanceLogs") {
		agent := extractStringVarPtr(req.Variables, "agent")
		pair := extractStringVarPtr(req.Variables, "pair")
		limit := extractIntVar(req.Variables, "limit", 50)
		logs, _ := h.resolver.PerformanceLogs(ctx, agent, pair, &limit)
		data["performanceLogs"] = logs
	}

	if strings.Contains(query, "expiredRules") {
		limit := extractIntVar(req.Variables, "limit", 50)
		rules, _ := h.resolver.ExpiredRules(ctx, &limit)
		data["expiredRules"] = rules
	}

	if strings.Contains(query, "adaptiveWeights") {
		pair := extractStringVar(req.Variables, "pair", "EUR_USD")
		weights, _ := h.resolver.AdaptiveWeights(ctx, pair)
		data["adaptiveWeights"] = weights
	}

	if strings.Contains(query, "currentRegime") {
		pair := extractStringVar(req.Variables, "pair", "EUR_USD")
		regime, _ := h.resolver.CurrentRegime(ctx, pair)
		data["currentRegime"] = regime
	}

	if strings.Contains(query, "regimeHistory") {
		pair := extractStringVar(req.Variables, "pair", "EUR_USD")
		limit := extractIntVar(req.Variables, "limit", 50)
		history, _ := h.resolver.RegimeHistory(ctx, pair, &limit)
		data["regimeHistory"] = history
	}

	if strings.Contains(query, "regimeChanges") {
		pair := extractStringVar(req.Variables, "pair", "EUR_USD")
		limit := extractIntVar(req.Variables, "limit", 50)
		changes, _ := h.resolver.RegimeChanges(ctx, pair, &limit)
		data["regimeChanges"] = changes
	}

	if strings.Contains(query, "logs") && !strings.Contains(query, "logAdded") {
		level := extractStringVarPtr(req.Variables, "level")
		limit := extractIntVar(req.Variables, "limit", 100)
		logs, _ := h.resolver.Logs(ctx, level, &limit)
		data["logs"] = logs
	}

	if strings.Contains(query, "pairs") && !strings.Contains(query, "pair:") {
		pairs, _ := h.resolver.QueryPairs(ctx)
		data["pairs"] = pairs
	}

	if strings.Contains(query, "connectionStatus") {
		status, _ := h.resolver.ConnectionStatus(ctx)
		data["connectionStatus"] = status
	}

	return graphQLResponse{Data: data}
}

// ════════════════════════════════════════════════════════════════════════
// WebSocket Handler — graphql-ws protocol for subscriptions
// ════════════════════════════════════════════════════════════════════════

func (h *Handler) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	rawConn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("WebSocket upgrade failed", "error", err)
		return
	}
	defer rawConn.Close()

	// Wrap with mutex so subscription goroutines can write concurrently
	conn := &safeConn{conn: rawConn}

	slog.Info("GraphQL WebSocket client connected",
		"remote", r.RemoteAddr,
		"subprotocol", rawConn.Subprotocol())

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	// Handle graphql-ws protocol messages
	for {
		_, msg, err := rawConn.ReadMessage()
		if err != nil {
			slog.Debug("WebSocket client disconnected", "error", err)
			return
		}

		var wsMsg wsMessage
		if err := json.Unmarshal(msg, &wsMsg); err != nil {
			continue
		}

		switch wsMsg.Type {
		case "connection_init":
			// Acknowledge connection (graphql-transport-ws spec)
			conn.WriteJSON(map[string]string{"type": "connection_ack"})

		case "ping":
			// graphql-ws sends periodic pings — MUST respond with pong
			// or the client will close the connection with code 1006
			conn.WriteJSON(map[string]string{"type": "pong"})

		case "pong":
			// Client responded to our ping — no-op

		case "subscribe":
			// Start subscription in background goroutine
			go h.handleSubscription(ctx, conn, wsMsg)

		case "complete":
			// Client unsubscribed a specific operation
			slog.Debug("Client unsubscribed", "id", wsMsg.ID)
		}
	}
}

type wsMessage struct {
	ID      string          `json:"id"`
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type subscribePayload struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables"`
}

func (h *Handler) handleSubscription(ctx context.Context, conn *safeConn, msg wsMessage) {
	var payload subscribePayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		return
	}

	query := payload.Query

	// Route subscription based on query content
	switch {
	case strings.Contains(query, "candleUpdated"):
		pair := extractStringVar(payload.Variables, "pair", "EUR_USD")
		h.streamCandles(ctx, conn, msg.ID, pair)

	case strings.Contains(query, "agentOutput"):
		h.streamAgentOutput(ctx, conn, msg.ID)

	case strings.Contains(query, "signalGenerated"):
		h.streamSignals(ctx, conn, msg.ID)

	case strings.Contains(query, "regimeChanged"):
		h.streamRegime(ctx, conn, msg.ID)

	case strings.Contains(query, "ruleCreated"):
		h.streamRules(ctx, conn, msg.ID)

	case strings.Contains(query, "logAdded"):
		h.streamLogs(ctx, conn, msg.ID)

	case strings.Contains(query, "pipelineEvent"):
		h.streamPipelineEvents(ctx, conn, msg.ID)
	}
}

func (h *Handler) streamCandles(ctx context.Context, conn *safeConn, id, pair string) {
	ch, _ := h.resolver.CandleUpdated(ctx, pair)
	for {
		select {
		case <-ctx.Done():
			return
		case candle, ok := <-ch:
			if !ok {
				return
			}
			h.sendSubscriptionData(conn, id, map[string]interface{}{"candleUpdated": candle})
		}
	}
}

func (h *Handler) streamAgentOutput(ctx context.Context, conn *safeConn, id string) {
	ch, _ := h.resolver.AgentOutput(ctx, nil)
	for {
		select {
		case <-ctx.Done():
			return
		case entry, ok := <-ch:
			if !ok {
				return
			}
			h.sendSubscriptionData(conn, id, map[string]interface{}{"agentOutput": entry})
		}
	}
}

func (h *Handler) streamSignals(ctx context.Context, conn *safeConn, id string) {
	ch, _ := h.resolver.SignalGenerated(ctx, nil)
	for {
		select {
		case <-ctx.Done():
			return
		case entry, ok := <-ch:
			if !ok {
				return
			}
			h.sendSubscriptionData(conn, id, map[string]interface{}{"signalGenerated": entry})
		}
	}
}

func (h *Handler) streamRegime(ctx context.Context, conn *safeConn, id string) {
	ch, _ := h.resolver.RegimeChanged(ctx, nil)
	for {
		select {
		case <-ctx.Done():
			return
		case entry, ok := <-ch:
			if !ok {
				return
			}
			h.sendSubscriptionData(conn, id, map[string]interface{}{"regimeChanged": entry})
		}
	}
}

func (h *Handler) streamRules(ctx context.Context, conn *safeConn, id string) {
	ch, _ := h.resolver.RuleCreated(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case entry, ok := <-ch:
			if !ok {
				return
			}
			h.sendSubscriptionData(conn, id, map[string]interface{}{"ruleCreated": entry})
		}
	}
}

func (h *Handler) streamLogs(ctx context.Context, conn *safeConn, id string) {
	ch, _ := h.resolver.LogAdded(ctx, nil)
	for {
		select {
		case <-ctx.Done():
			return
		case entry, ok := <-ch:
			if !ok {
				return
			}
			h.sendSubscriptionData(conn, id, map[string]interface{}{"logAdded": entry})
		}
	}
}

func (h *Handler) streamPipelineEvents(ctx context.Context, conn *safeConn, id string) {
	ch, _ := h.resolver.PipelineEventSub(ctx, nil)
	for {
		select {
		case <-ctx.Done():
			return
		case entry, ok := <-ch:
			if !ok {
				return
			}
			h.sendSubscriptionData(conn, id, map[string]interface{}{"pipelineEvent": entry})
		}
	}
}

func (h *Handler) sendSubscriptionData(conn *safeConn, id string, data interface{}) {
	resp := map[string]interface{}{
		"id":      id,
		"type":    "next",
		"payload": map[string]interface{}{"data": data},
	}
	if err := conn.WriteJSON(resp); err != nil {
		slog.Debug("WebSocket write error", "error", err)
	}
}

// ════════════════════════════════════════════════════════════════════════
// Helper functions
// ════════════════════════════════════════════════════════════════════════

func extractStringVar(vars map[string]interface{}, key, defaultVal string) string {
	if v, ok := vars[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return defaultVal
}

func extractStringVarPtr(vars map[string]interface{}, key string) *string {
	if v, ok := vars[key]; ok {
		if s, ok := v.(string); ok {
			return &s
		}
	}
	return nil
}

func extractIntVar(vars map[string]interface{}, key string, defaultVal int) int {
	if v, ok := vars[key]; ok {
		switch n := v.(type) {
		case float64:
			return int(n)
		case int:
			return n
		}
	}
	return defaultVal
}
