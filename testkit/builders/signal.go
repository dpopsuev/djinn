package builders

import (
	"time"

	"github.com/dpopsuev/djinn/signal"
)

// SignalBuilder provides a fluent API for constructing Signals.
type SignalBuilder struct {
	s signal.Signal
}

// NewSignal starts building a signal.
func NewSignal(workstream string) *SignalBuilder {
	return &SignalBuilder{
		s: signal.Signal{
			Workstream: workstream,
			Level:      signal.Green,
			Timestamp:  time.Now(),
		},
	}
}

// WithLevel sets the flag level.
func (b *SignalBuilder) WithLevel(level signal.FlagLevel) *SignalBuilder {
	b.s.Level = level
	return b
}

// WithConfidence sets the confidence score.
func (b *SignalBuilder) WithConfidence(c float64) *SignalBuilder {
	b.s.Confidence = c
	return b
}

// WithSource sets the source agent/watchdog ID.
func (b *SignalBuilder) WithSource(src string) *SignalBuilder {
	b.s.Source = src
	return b
}

// WithScope sets the affected paths.
func (b *SignalBuilder) WithScope(paths ...string) *SignalBuilder {
	b.s.Scope = paths
	return b
}

// WithCategory sets the signal category.
func (b *SignalBuilder) WithCategory(cat string) *SignalBuilder {
	b.s.Category = cat
	return b
}

// WithMessage sets the message.
func (b *SignalBuilder) WithMessage(msg string) *SignalBuilder {
	b.s.Message = msg
	return b
}

// WithTimestamp sets the timestamp.
func (b *SignalBuilder) WithTimestamp(t time.Time) *SignalBuilder {
	b.s.Timestamp = t
	return b
}

// Build returns the constructed signal.
func (b *SignalBuilder) Build() signal.Signal {
	return b.s
}
