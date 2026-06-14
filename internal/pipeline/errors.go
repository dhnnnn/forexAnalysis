package pipeline

import "fmt"

// ════════════════════════════════════════════════════════════════════════
// Pipeline Errors — structured error types for pipeline operations
// ════════════════════════════════════════════════════════════════════════

// PipelineError wraps an error with context about where it occurred.
type PipelineError struct {
	Agent     string // agent yang mengalami error
	Pair      string // currency pair terkait
	Operation string // operasi yang gagal
	Err       error  // underlying error
}

func (e *PipelineError) Error() string {
	return fmt.Sprintf("[%s] %s %s: %v", e.Agent, e.Operation, e.Pair, e.Err)
}

func (e *PipelineError) Unwrap() error {
	return e.Err
}

// NewPipelineError creates a new PipelineError.
func NewPipelineError(agent, pair, operation string, err error) *PipelineError {
	return &PipelineError{
		Agent:     agent,
		Pair:      pair,
		Operation: operation,
		Err:       err,
	}
}

// PersistError represents a non-critical persistence failure.
type PersistError struct {
	Store     string // "postgres" or "redis"
	Operation string
	Err       error
}

func (e *PersistError) Error() string {
	return fmt.Sprintf("persist[%s] %s: %v", e.Store, e.Operation, e.Err)
}

func (e *PersistError) Unwrap() error {
	return e.Err
}

// ErrInsufficientData is returned when there's not enough candle data.
var ErrInsufficientData = fmt.Errorf("insufficient candle data")
