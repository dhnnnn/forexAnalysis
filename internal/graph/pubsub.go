package graph

import (
	"sync"

	"github.com/dhnnnn/forexAnalysis/internal/graph/model"
)

// ════════════════════════════════════════════════════════════════════════
// PubSub — in-memory pub/sub untuk GraphQL Subscriptions
// ════════════════════════════════════════════════════════════════════════

// PubSub manages subscription channels untuk real-time data ke frontend.
type PubSub struct {
	// Candle subscribers (per pair)
	candleSubs   map[string]map[string]chan *model.Candle // pair → {id → chan}
	candleMu     sync.RWMutex

	// Agent output subscribers
	agentSubs    map[string]chan *model.AgentDebateEntry // id → chan
	agentMu      sync.RWMutex

	// Signal subscribers
	signalSubs   map[string]chan *model.SignalEntry // id → chan
	signalMu     sync.RWMutex

	// Regime subscribers
	regimeSubs   map[string]chan *model.RegimeContext // id → chan
	regimeMu     sync.RWMutex

	// Rule subscribers
	ruleSubs     map[string]chan *model.KnowledgeRule // id → chan
	ruleMu       sync.RWMutex

	// Log subscribers
	logSubs      map[string]chan *model.SystemLog // id → chan
	logMu        sync.RWMutex

	// Pipeline event subscribers
	pipelineSubs map[string]chan *model.PipelineEvent // id → chan
	pipelineMu   sync.RWMutex
}

// NewPubSub creates a new PubSub instance.
func NewPubSub() *PubSub {
	return &PubSub{
		candleSubs:   make(map[string]map[string]chan *model.Candle),
		agentSubs:    make(map[string]chan *model.AgentDebateEntry),
		signalSubs:   make(map[string]chan *model.SignalEntry),
		regimeSubs:   make(map[string]chan *model.RegimeContext),
		ruleSubs:     make(map[string]chan *model.KnowledgeRule),
		logSubs:      make(map[string]chan *model.SystemLog),
		pipelineSubs: make(map[string]chan *model.PipelineEvent),
	}
}

// ── Candle Pub/Sub ────────────────────────────────────────────────────

func (ps *PubSub) SubscribeCandle(id, pair string) chan *model.Candle {
	ps.candleMu.Lock()
	defer ps.candleMu.Unlock()

	ch := make(chan *model.Candle, 10)
	if ps.candleSubs[pair] == nil {
		ps.candleSubs[pair] = make(map[string]chan *model.Candle)
	}
	ps.candleSubs[pair][id] = ch
	return ch
}

func (ps *PubSub) UnsubscribeCandle(id, pair string) {
	ps.candleMu.Lock()
	defer ps.candleMu.Unlock()

	if subs, ok := ps.candleSubs[pair]; ok {
		if ch, ok := subs[id]; ok {
			close(ch)
			delete(subs, id)
		}
	}
}

func (ps *PubSub) PublishCandle(pair string, candle *model.Candle) {
	ps.candleMu.RLock()
	defer ps.candleMu.RUnlock()

	if subs, ok := ps.candleSubs[pair]; ok {
		for _, ch := range subs {
			select {
			case ch <- candle:
			default: // drop if subscriber is slow
			}
		}
	}
}

// ── Agent Output Pub/Sub ──────────────────────────────────────────────

func (ps *PubSub) SubscribeAgent(id string) chan *model.AgentDebateEntry {
	ps.agentMu.Lock()
	defer ps.agentMu.Unlock()

	ch := make(chan *model.AgentDebateEntry, 20)
	ps.agentSubs[id] = ch
	return ch
}

func (ps *PubSub) UnsubscribeAgent(id string) {
	ps.agentMu.Lock()
	defer ps.agentMu.Unlock()

	if ch, ok := ps.agentSubs[id]; ok {
		close(ch)
		delete(ps.agentSubs, id)
	}
}

func (ps *PubSub) PublishAgentOutput(entry *model.AgentDebateEntry) {
	ps.agentMu.RLock()
	defer ps.agentMu.RUnlock()

	for _, ch := range ps.agentSubs {
		select {
		case ch <- entry:
		default:
		}
	}
}

// ── Signal Pub/Sub ────────────────────────────────────────────────────

func (ps *PubSub) SubscribeSignal(id string) chan *model.SignalEntry {
	ps.signalMu.Lock()
	defer ps.signalMu.Unlock()

	ch := make(chan *model.SignalEntry, 10)
	ps.signalSubs[id] = ch
	return ch
}

func (ps *PubSub) UnsubscribeSignal(id string) {
	ps.signalMu.Lock()
	defer ps.signalMu.Unlock()

	if ch, ok := ps.signalSubs[id]; ok {
		close(ch)
		delete(ps.signalSubs, id)
	}
}

func (ps *PubSub) PublishSignal(signal *model.SignalEntry) {
	ps.signalMu.RLock()
	defer ps.signalMu.RUnlock()

	for _, ch := range ps.signalSubs {
		select {
		case ch <- signal:
		default:
		}
	}
}

// ── Regime Pub/Sub ────────────────────────────────────────────────────

func (ps *PubSub) SubscribeRegime(id string) chan *model.RegimeContext {
	ps.regimeMu.Lock()
	defer ps.regimeMu.Unlock()

	ch := make(chan *model.RegimeContext, 10)
	ps.regimeSubs[id] = ch
	return ch
}

func (ps *PubSub) UnsubscribeRegime(id string) {
	ps.regimeMu.Lock()
	defer ps.regimeMu.Unlock()

	if ch, ok := ps.regimeSubs[id]; ok {
		close(ch)
		delete(ps.regimeSubs, id)
	}
}

func (ps *PubSub) PublishRegime(regime *model.RegimeContext) {
	ps.regimeMu.RLock()
	defer ps.regimeMu.RUnlock()

	for _, ch := range ps.regimeSubs {
		select {
		case ch <- regime:
		default:
		}
	}
}

// ── Rule Pub/Sub ──────────────────────────────────────────────────────

func (ps *PubSub) SubscribeRule(id string) chan *model.KnowledgeRule {
	ps.ruleMu.Lock()
	defer ps.ruleMu.Unlock()

	ch := make(chan *model.KnowledgeRule, 10)
	ps.ruleSubs[id] = ch
	return ch
}

func (ps *PubSub) UnsubscribeRule(id string) {
	ps.ruleMu.Lock()
	defer ps.ruleMu.Unlock()

	if ch, ok := ps.ruleSubs[id]; ok {
		close(ch)
		delete(ps.ruleSubs, id)
	}
}

func (ps *PubSub) PublishRule(rule *model.KnowledgeRule) {
	ps.ruleMu.RLock()
	defer ps.ruleMu.RUnlock()

	for _, ch := range ps.ruleSubs {
		select {
		case ch <- rule:
		default:
		}
	}
}

// ── Log Pub/Sub ───────────────────────────────────────────────────────

func (ps *PubSub) SubscribeLog(id string) chan *model.SystemLog {
	ps.logMu.Lock()
	defer ps.logMu.Unlock()

	ch := make(chan *model.SystemLog, 50)
	ps.logSubs[id] = ch
	return ch
}

func (ps *PubSub) UnsubscribeLog(id string) {
	ps.logMu.Lock()
	defer ps.logMu.Unlock()

	if ch, ok := ps.logSubs[id]; ok {
		close(ch)
		delete(ps.logSubs, id)
	}
}

func (ps *PubSub) PublishLog(log *model.SystemLog) {
	ps.logMu.RLock()
	defer ps.logMu.RUnlock()

	for _, ch := range ps.logSubs {
		select {
		case ch <- log:
		default:
		}
	}
}

// ── Pipeline Event Pub/Sub ────────────────────────────────────────────

func (ps *PubSub) SubscribePipeline(id string) chan *model.PipelineEvent {
	ps.pipelineMu.Lock()
	defer ps.pipelineMu.Unlock()

	ch := make(chan *model.PipelineEvent, 10)
	ps.pipelineSubs[id] = ch
	return ch
}

func (ps *PubSub) UnsubscribePipeline(id string) {
	ps.pipelineMu.Lock()
	defer ps.pipelineMu.Unlock()

	if ch, ok := ps.pipelineSubs[id]; ok {
		close(ch)
		delete(ps.pipelineSubs, id)
	}
}

func (ps *PubSub) PublishPipelineEvent(event *model.PipelineEvent) {
	ps.pipelineMu.RLock()
	defer ps.pipelineMu.RUnlock()

	for _, ch := range ps.pipelineSubs {
		select {
		case ch <- event:
		default:
		}
	}
}
