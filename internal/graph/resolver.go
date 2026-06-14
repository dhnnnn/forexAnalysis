package graph

import (
	"github.com/dhnnnn/forexAnalysis/internal/agents"
	"github.com/dhnnnn/forexAnalysis/internal/knowledge"
	"github.com/dhnnnn/forexAnalysis/internal/storage"
)

// Resolver holds dependencies yang dibutuhkan oleh GraphQL resolvers.
type Resolver struct {
	Store        *storage.Store
	KBStore      *knowledge.Store
	MarketAgent  *agents.MarketDataAgent
	MetaObserver *agents.MetaObserverAgent
	PubSub       *PubSub
	Pairs        []string
	Timeframes   []string
}
