package graph

import (
	"context"

	"github.com/dhnnnn/forexAnalysis/internal/graph/model"
	"github.com/google/uuid"
)

// ════════════════════════════════════════════════════════════════════════
// Subscription Resolvers — real-time data via WebSocket
// ════════════════════════════════════════════════════════════════════════

// CandleUpdated streams new candle data for a pair.
func (r *Resolver) CandleUpdated(ctx context.Context, pair string) (<-chan *model.Candle, error) {
	id := uuid.New().String()
	ch := r.PubSub.SubscribeCandle(id, pair)

	// Cleanup on context done
	go func() {
		<-ctx.Done()
		r.PubSub.UnsubscribeCandle(id, pair)
	}()

	return ch, nil
}

// AgentOutput streams agent debate entries.
func (r *Resolver) AgentOutput(ctx context.Context, pair *string) (<-chan *model.AgentDebateEntry, error) {
	id := uuid.New().String()
	ch := r.PubSub.SubscribeAgent(id)

	// Filter by pair if specified
	if pair != nil {
		filtered := make(chan *model.AgentDebateEntry, 20)
		go func() {
			defer close(filtered)
			for {
				select {
				case <-ctx.Done():
					r.PubSub.UnsubscribeAgent(id)
					return
				case entry, ok := <-ch:
					if !ok {
						return
					}
					if entry.Pair == *pair {
						select {
						case filtered <- entry:
						default:
						}
					}
				}
			}
		}()
		return filtered, nil
	}

	go func() {
		<-ctx.Done()
		r.PubSub.UnsubscribeAgent(id)
	}()

	return ch, nil
}

// SignalGenerated streams new trading signals.
func (r *Resolver) SignalGenerated(ctx context.Context, pair *string) (<-chan *model.SignalEntry, error) {
	id := uuid.New().String()
	ch := r.PubSub.SubscribeSignal(id)

	if pair != nil {
		filtered := make(chan *model.SignalEntry, 10)
		go func() {
			defer close(filtered)
			for {
				select {
				case <-ctx.Done():
					r.PubSub.UnsubscribeSignal(id)
					return
				case entry, ok := <-ch:
					if !ok {
						return
					}
					if entry.Pair == *pair {
						select {
						case filtered <- entry:
						default:
						}
					}
				}
			}
		}()
		return filtered, nil
	}

	go func() {
		<-ctx.Done()
		r.PubSub.UnsubscribeSignal(id)
	}()

	return ch, nil
}

// RegimeChanged streams regime changes.
func (r *Resolver) RegimeChanged(ctx context.Context, pair *string) (<-chan *model.RegimeContext, error) {
	id := uuid.New().String()
	ch := r.PubSub.SubscribeRegime(id)

	if pair != nil {
		filtered := make(chan *model.RegimeContext, 10)
		go func() {
			defer close(filtered)
			for {
				select {
				case <-ctx.Done():
					r.PubSub.UnsubscribeRegime(id)
					return
				case entry, ok := <-ch:
					if !ok {
						return
					}
					if entry.Pair == *pair {
						select {
						case filtered <- entry:
						default:
						}
					}
				}
			}
		}()
		return filtered, nil
	}

	go func() {
		<-ctx.Done()
		r.PubSub.UnsubscribeRegime(id)
	}()

	return ch, nil
}

// RuleCreated streams new knowledge rules.
func (r *Resolver) RuleCreated(ctx context.Context) (<-chan *model.KnowledgeRule, error) {
	id := uuid.New().String()
	ch := r.PubSub.SubscribeRule(id)

	go func() {
		<-ctx.Done()
		r.PubSub.UnsubscribeRule(id)
	}()

	return ch, nil
}

// LogAdded streams system logs.
func (r *Resolver) LogAdded(ctx context.Context, level *string) (<-chan *model.SystemLog, error) {
	id := uuid.New().String()
	ch := r.PubSub.SubscribeLog(id)

	if level != nil {
		filtered := make(chan *model.SystemLog, 50)
		go func() {
			defer close(filtered)
			for {
				select {
				case <-ctx.Done():
					r.PubSub.UnsubscribeLog(id)
					return
				case entry, ok := <-ch:
					if !ok {
						return
					}
					if entry.Level == *level {
						select {
						case filtered <- entry:
						default:
						}
					}
				}
			}
		}()
		return filtered, nil
	}

	go func() {
		<-ctx.Done()
		r.PubSub.UnsubscribeLog(id)
	}()

	return ch, nil
}

// PipelineEvent streams pipeline lifecycle events.
func (r *Resolver) PipelineEventSub(ctx context.Context, pair *string) (<-chan *model.PipelineEvent, error) {
	id := uuid.New().String()
	ch := r.PubSub.SubscribePipeline(id)

	if pair != nil {
		filtered := make(chan *model.PipelineEvent, 10)
		go func() {
			defer close(filtered)
			for {
				select {
				case <-ctx.Done():
					r.PubSub.UnsubscribePipeline(id)
					return
				case entry, ok := <-ch:
					if !ok {
						return
					}
					if entry.Pair == *pair {
						select {
						case filtered <- entry:
						default:
						}
					}
				}
			}
		}()
		return filtered, nil
	}

	go func() {
		<-ctx.Done()
		r.PubSub.UnsubscribePipeline(id)
	}()

	return ch, nil
}
