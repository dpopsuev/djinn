// dispatcher.go — EventDispatcher: unified hook runtime.
//
// Implements battery/middleware.Gate (pre-tool hooks) and
// battery/middleware.Recorder (post-tool hooks).
// Subscribes to signal.EventLog.OnEmit for async event hooks.
//
// GOL-161, TSK-1068
package hook

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"os/exec"
	"sync"
	"time"

	"github.com/dpopsuev/battery/middleware"
	"github.com/dpopsuev/djinn/telemetry"
	"github.com/dpopsuev/troupe/signal"
)

// SpawnFunc is called when a hook's action is spawn_slot.
type SpawnFunc func(ctx context.Context, role string) error

// EventDispatcher is the unified hook runtime.
type EventDispatcher struct {
	mu         sync.RWMutex
	hooks      []Hook
	preHooks   []Hook // Phase == PhasePreToolUse
	postHooks  []Hook // Phase == PhasePostToolUse
	eventHooks []Hook // Phase == PhaseEvent

	eventLog signal.EventLog
	spawnFn  SpawnFunc
	log      *slog.Logger

	// Stats (YELLOW, TSK-1075).
	dispatched int64
	denied     int64
	errors     int64
	statsMu    sync.Mutex
}

// Option configures an EventDispatcher.
type Option func(*EventDispatcher)

// WithSpawnFunc sets the callback for spawn_slot actions.
func WithSpawnFunc(fn SpawnFunc) Option {
	return func(d *EventDispatcher) { d.spawnFn = fn }
}

// WithLogger sets the structured logger.
func WithLogger(log *slog.Logger) Option {
	return func(d *EventDispatcher) { d.log = log }
}

// New creates an EventDispatcher and subscribes to the EventLog.
func New(hooks []Hook, eventLog signal.EventLog, opts ...Option) *EventDispatcher {
	d := &EventDispatcher{
		hooks:    hooks,
		eventLog: eventLog,
	}
	for _, opt := range opts {
		opt(d)
	}
	d.index()

	// Subscribe to EventLog for async event hooks.
	if eventLog != nil && len(d.eventHooks) > 0 {
		eventLog.OnEmit(d.handleEvent)
	}

	return d
}

// index splits hooks by phase for O(1) lookup.
func (d *EventDispatcher) index() {
	d.preHooks = nil
	d.postHooks = nil
	d.eventHooks = nil
	for i := range d.hooks {
		switch d.hooks[i].On {
		case PhasePreToolUse:
			d.preHooks = append(d.preHooks, d.hooks[i])
		case PhasePostToolUse:
			d.postHooks = append(d.postHooks, d.hooks[i])
		case PhaseEvent:
			d.eventHooks = append(d.eventHooks, d.hooks[i])
		}
	}
}

// Reload replaces the hook set atomically.
func (d *EventDispatcher) Reload(hooks []Hook) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.hooks = hooks
	d.index()
}

// Hooks returns the current hook set.
func (d *EventDispatcher) Hooks() []Hook {
	d.mu.RLock()
	defer d.mu.RUnlock()
	out := make([]Hook, len(d.hooks))
	copy(out, d.hooks)
	return out
}

// --- middleware.Gate (pre-tool hooks) ---

// Check evaluates all pre_tool_use hooks. First deny stops the chain.
func (d *EventDispatcher) Check(ctx context.Context, tool string, input json.RawMessage) (middleware.Verdict, error) {
	d.mu.RLock()
	hooks := d.preHooks
	d.mu.RUnlock()

	for i := range hooks {
		h := &hooks[i]
		if !h.Match.MatchesTool(tool) {
			continue
		}

		d.incDispatched()

		if h.Action.Shell != "" {
			if allowed, reason := d.runShell(ctx, *h, tool, input); !allowed {
				d.incDenied()
				return middleware.Verdict{Allowed: false, Reason: reason}, nil
			}
		}

		if h.Action.Deny != "" {
			d.incDenied()
			return middleware.Verdict{Allowed: false, Reason: h.Action.Deny}, nil
		}
	}

	return middleware.Verdict{Allowed: true}, nil
}

// --- middleware.Recorder (post-tool hooks) ---

// Record evaluates all post_tool_use hooks.
func (d *EventDispatcher) Record(_ context.Context, tool string, _ json.RawMessage, _ string, _ error, _ time.Duration) {
	d.mu.RLock()
	hooks := d.postHooks
	d.mu.RUnlock()

	for i := range hooks {
		h := &hooks[i]
		if !h.Match.MatchesTool(tool) {
			continue
		}
		if h.Action.Emit != "" && d.eventLog != nil {
			d.eventLog.Emit(signal.Event{
				Source: "hook",
				Kind:   h.Action.Emit,
				Data:   map[string]string{"hook": h.Name, "tool": tool},
			})
		}
	}
}

// --- EventLog subscriber (async event hooks) ---

func (d *EventDispatcher) handleEvent(e signal.Event) {
	d.mu.RLock()
	hooks := d.eventHooks
	d.mu.RUnlock()

	// Collect matching hooks, then execute OUTSIDE the OnEmit callback
	// to avoid deadlock (MemLog holds mutex during OnEmit).
	var matched []Hook
	for i := range hooks {
		if hooks[i].Match.MatchesEvent(e) && hooks[i].MatchesScope(e.Source) {
			matched = append(matched, hooks[i])
		}
	}
	if len(matched) > 0 {
		go func() {
			for i := range matched {
				d.executeAction(matched[i], e)
			}
		}()
	}
}

func (d *EventDispatcher) executeAction(h Hook, e signal.Event) {
	if h.Action.Emit != "" && d.eventLog != nil {
		d.eventLog.Emit(signal.Event{
			Source:  "hook",
			Kind:    h.Action.Emit,
			TraceID: e.TraceID,
			Data:    map[string]string{"hook": h.Name, "trigger": e.Kind},
		})
	}
	if h.Action.SpawnSlot != "" && d.spawnFn != nil {
		go func() {
			if err := d.spawnFn(context.Background(), h.Action.SpawnSlot); err != nil && d.log != nil {
				d.log.WarnContext(context.Background(), "hook spawn_slot failed",
					slog.String(telemetry.KeyAction, h.Name),
					slog.String(telemetry.KeyRole, h.Action.SpawnSlot),
					slog.String(telemetry.KeyError, err.Error()),
				)
			}
		}()
	}
}

// --- Shell execution ---

const shellDenyExitCode = 2

func (d *EventDispatcher) runShell(ctx context.Context, h Hook, tool string, input json.RawMessage) (allowed bool, reason string) {
	payload, _ := json.Marshal(map[string]any{
		"hook_event": string(h.On),
		"tool_name":  tool,
		"tool_input": input,
	})

	cmd := exec.CommandContext(ctx, "sh", "-c", h.Action.Shell) //nolint:gosec // operator-defined hooks, same trust as agent/hook.go
	cmd.Stdin = bytes.NewReader(payload)

	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == shellDenyExitCode { //nolint:errorlint // need concrete type for ExitCode()
			r := stdout.String()
			if r == "" {
				r = "denied by hook " + h.Name
			}
			return false, r
		}
		return true, "" // non-exit errors: warn but allow
	}
	return true, ""
}

// DispatchStats returns hook dispatch metrics.
type DispatchStats struct {
	Dispatched int64 // total hook evaluations
	Denied     int64 // pre-tool denials
	Errors     int64 // action execution errors
}

// Stats returns current dispatch metrics.
func (d *EventDispatcher) Stats() DispatchStats {
	d.statsMu.Lock()
	defer d.statsMu.Unlock()
	return DispatchStats{
		Dispatched: d.dispatched,
		Denied:     d.denied,
		Errors:     d.errors,
	}
}

func (d *EventDispatcher) incDispatched() { d.statsMu.Lock(); d.dispatched++; d.statsMu.Unlock() }
func (d *EventDispatcher) incDenied()     { d.statsMu.Lock(); d.denied++; d.statsMu.Unlock() }
func (d *EventDispatcher) incErrors()     { d.statsMu.Lock(); d.errors++; d.statsMu.Unlock() } //nolint:unused // wired when shell hooks report errors

// Interface compliance.
var (
	_ middleware.Gate     = (*EventDispatcher)(nil)
	_ middleware.Recorder = (*EventDispatcher)(nil)
)
