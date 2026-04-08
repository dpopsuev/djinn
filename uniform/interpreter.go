// interpreter.go — SignalInterpreter wires SignalBus to GenSec (NED-16, TSK-470).
//
// The interpreter subscribes to the signal bus and filters for yellow+ signals.
// Green zone costs zero LLM tokens. Yellow+ signals are formatted into structured
// prompts and sent to GenSec via the Bugle facade.Agent.Ask() interface.
// Responses are parsed into Decisions and routed through the zone system.
package uniform

import (
	"context"
	"sync"
	"time"

	"github.com/dpopsuev/djinn/telemetry"
)

// GenSecAgent is the interface the interpreter uses to communicate with GenSec.
// Satisfied by jericho agent handle and any mock.
type GenSecAgent interface {
	Ask(ctx context.Context, content string) (string, error)
}

// SignalInterpreter connects the signal bus to GenSec's stochastic interpretation layer.
type SignalInterpreter struct {
	bus        *telemetry.SignalBus
	gensec     GenSecAgent
	audit      AuditLog
	ctx        context.Context
	cancel     context.CancelFunc
	contextFn  func() SignalContext // provides current state for prompt enrichment
	onDecision func(AuditEntry)     // callback for TUI notification

	mu      sync.Mutex
	running bool
}

// NewSignalInterpreter creates an interpreter wired to a signal bus and GenSec agent.
func NewSignalInterpreter(bus *telemetry.SignalBus, gensec GenSecAgent) *SignalInterpreter {
	return &SignalInterpreter{
		bus:    bus,
		gensec: gensec,
	}
}

// SetContextProvider sets the function that provides current state for prompt enrichment.
func (si *SignalInterpreter) SetContextProvider(fn func() SignalContext) {
	si.contextFn = fn
}

// OnDecision registers a callback invoked after each decision is made.
func (si *SignalInterpreter) OnDecision(fn func(AuditEntry)) {
	si.onDecision = fn
}

// Start subscribes to the signal bus and begins interpreting yellow+ signals.
func (si *SignalInterpreter) Start(ctx context.Context) {
	si.mu.Lock()
	if si.running {
		si.mu.Unlock()
		return
	}
	si.ctx, si.cancel = context.WithCancel(ctx)
	si.running = true
	si.mu.Unlock()

	si.bus.OnSignal(func(s telemetry.Signal) {
		si.handleSignal(s)
	})
}

// Stop halts the interpreter.
func (si *SignalInterpreter) Stop() {
	si.mu.Lock()
	defer si.mu.Unlock()
	if si.cancel != nil {
		si.cancel()
	}
	si.running = false
}

// AuditEntries returns the full audit trail.
func (si *SignalInterpreter) AuditEntries() []AuditEntry {
	return si.audit.Entries()
}

const interpreterSource = "signal-interpreter"

func (si *SignalInterpreter) handleSignal(s telemetry.Signal) {
	// Ignore signals emitted by the interpreter itself to prevent loops.
	if s.Source == interpreterSource {
		return
	}

	zone := ZoneFromLevel(s.Level)

	// Green zone: auto-continue, zero LLM cost.
	if zone == ZoneGreen {
		return
	}

	si.mu.Lock()
	if !si.running {
		si.mu.Unlock()
		return
	}
	si.mu.Unlock()

	// Build context for prompt enrichment.
	var sctx SignalContext
	if si.contextFn != nil {
		sctx = si.contextFn()
	}

	// Format the signal into a structured prompt.
	prompt := FormatSignalPrompt(s, sctx)

	// Ask GenSec to interpret.
	response, err := si.gensec.Ask(si.ctx, prompt)
	if err != nil {
		si.bus.Emit(telemetry.Signal{
			Workstream: s.Workstream,
			Level:      telemetry.Yellow,
			Source:     interpreterSource,
			Category:   telemetry.CategoryLifecycle,
			Message:    "GenSec Ask failed: " + err.Error(),
		})
		return
	}

	// Parse GenSec's response into a Decision.
	decision := si.parseResponse(response, s.Category)

	// Record in audit log.
	entry := AuditEntry{
		Timestamp: time.Now(),
		Signal:    s,
		Zone:      zone,
		Decision:  decision,
	}
	si.audit.Append(entry)

	// Notify TUI / operator.
	if si.onDecision != nil {
		si.onDecision(entry)
	}

	// Route based on zone.
	si.routeDecision(zone, decision, s)
}

func (si *SignalInterpreter) parseResponse(response, pillar string) Decision {
	d, err := ParseDecision(response, pillar)
	if err != nil {
		// Unparseable response defaults to ActionContinue — safe fallback.
		_ = err
	}
	return d
}

func (si *SignalInterpreter) routeDecision(zone Zone, d Decision, s telemetry.Signal) {
	switch zone {
	case ZoneYellow:
		// Apply decision, operator is notified via onDecision callback.
		si.applyDecision(d, s)

	case ZoneOrange:
		// Apply decision, operator can override via :override command.
		si.applyDecision(d, s)

	case ZoneRed:
		// Emit cordon signal — operator must approve before action is taken.
		si.bus.Emit(telemetry.Signal{
			Workstream: s.Workstream,
			Level:      telemetry.Red,
			Source:     interpreterSource,
			Category:   telemetry.CategoryLifecycle,
			Message:    "cordon: " + d.Pillar + " — " + d.Reason,
		})
	}
}

func (si *SignalInterpreter) applyDecision(d Decision, s telemetry.Signal) {
	// Emit a lifecycle signal with the decision for downstream consumers.
	si.bus.Emit(telemetry.Signal{
		Workstream: s.Workstream,
		Level:      telemetry.Green,
		Source:     interpreterSource,
		Category:   telemetry.CategoryLifecycle,
		Message:    "decision: " + string(d.Action) + " — " + d.Reason,
	})
}
