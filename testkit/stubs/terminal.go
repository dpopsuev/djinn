package stubs

import (
	"context"
	"sync"
	"time"

	"github.com/dpopsuev/djinn/terminal"
)

// TestOperator wraps a terminal.Djinn for programmatic testing.
// Implements the operator side — submits prompts, observes events.
type TestOperator struct {
	djinn   *terminal.Djinn
	events  []terminal.ViewEvent
	eventCh chan terminal.ViewEvent
	mu      sync.Mutex
}

// NewTestOperator creates a TestOperator backed by a fresh Djinn instance.
func NewTestOperator() *TestOperator {
	d := terminal.NewDjinn()
	ch := make(chan terminal.ViewEvent, 100)
	d.Subscribe(ch)
	return &TestOperator{djinn: d, eventCh: ch}
}

// Djinn returns the underlying terminal instance.
func (t *TestOperator) Djinn() *terminal.Djinn { return t.djinn }

// Submit sends a prompt through the terminal.
func (t *TestOperator) Submit(ctx context.Context, prompt string) error {
	return t.djinn.Submit(ctx, prompt)
}

// Command executes a terminal command.
func (t *TestOperator) Command(ctx context.Context, name string, args []string) (string, error) {
	return t.djinn.Command(ctx, name, args)
}

// DrainEvents reads all pending events from the channel.
func (t *TestOperator) DrainEvents() []terminal.ViewEvent {
	t.mu.Lock()
	defer t.mu.Unlock()
	for {
		select {
		case ev := <-t.eventCh:
			t.events = append(t.events, ev)
		default:
			return t.events
		}
	}
}

// Events returns all collected events.
func (t *TestOperator) Events() []terminal.ViewEvent {
	t.DrainEvents()
	t.mu.Lock()
	defer t.mu.Unlock()
	return append([]terminal.ViewEvent{}, t.events...)
}

// WaitForEvent blocks until an event of the given kind arrives or timeout.
func (t *TestOperator) WaitForEvent(kind terminal.ViewEventKind, timeout time.Duration) (terminal.ViewEvent, bool) {
	deadline := time.After(timeout)
	for {
		select {
		case ev := <-t.eventCh:
			t.mu.Lock()
			t.events = append(t.events, ev)
			t.mu.Unlock()
			if ev.Kind == kind {
				return ev, true
			}
		case <-deadline:
			return terminal.ViewEvent{}, false
		}
	}
}
