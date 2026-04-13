package terminal

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/dpopsuev/djinn/substrate"
	"github.com/dpopsuev/djinn/tui"
	"github.com/dpopsuev/djinn/uniform"
	"github.com/dpopsuev/djinn/uniform/execution"
	"github.com/dpopsuev/djinn/workspace"
)

// Sentinel errors.
var (
	ErrNoSubmitHandler  = errors.New("terminal: no submit handler configured")
	ErrNoShellHandler   = errors.New("terminal: no shell handler configured")
	ErrUnknownOperation = errors.New("terminal: unknown operation")
	ErrInvalidCapacity  = errors.New("terminal: invalid capacity")
	ErrUnknownCommand   = errors.New("terminal: unknown command")
	ErrInvalidFrequency = errors.New("terminal: invalid frequency")
	ErrUsage            = errors.New("terminal: usage error")
	ErrUnknownSubcmd    = errors.New("terminal: unknown subcommand")
)

// Ensure Djinn implements Terminal.
var _ Terminal = (*Djinn)(nil)

// Djinn is the concrete Terminal implementation.
// Holds domain state. TUI adapters delegate to this.
type Djinn struct {
	mu sync.RWMutex

	// Domain state
	operation   uniform.Operation
	capacity    *execution.AgentCapacity
	envelopeCfg substrate.EnvelopeConfig
	scopePath   string
	scopeType   string
	startedAt   time.Time

	// Subscriber management
	subMu       sync.RWMutex
	subscribers []chan<- ViewEvent

	// Workspace repo paths: name -> host path (for VFS mount creation).
	repos map[string]string

	// Pluggable handlers — set by the adapter (TUI, headless, etc.)
	// These allow Terminal to trigger actions without importing adapter packages.
	OnSubmit   func(ctx context.Context, prompt string) error
	OnShell    func(ctx context.Context, command string) (string, error)
	OnCommand  func(ctx context.Context, name string, args []string) (string, error)
	OnNavigate func(path string, scopeType string) error

	// VFS mount table — runtime path translation for agents.
	mounts *workspace.MountTable

	// Agent visibility control
	sightMgr *tui.SightManager

	// Observable state (written by adapter, read by Viewer)
	tokensIn    int
	tokensOut   int
	turns       int
	activeRole  string
	agentCount  int
	isStreaming bool
}

// NewDjinn creates a Terminal with default configuration.
func NewDjinn() *Djinn {
	return &Djinn{
		operation:   uniform.DefaultOperation(),
		capacity:    execution.NewAgentCapacity(1),
		envelopeCfg: substrate.DefaultEnvelopeConfig(),
		scopePath:   "/",
		mounts:      workspace.NewMountTable(slog.Default()),
		sightMgr:    tui.NewSightManager(nil), // TODO: inject real logger from app
	}
}

// --- Controller ---

func (d *Djinn) Submit(ctx context.Context, prompt string) error {
	if d.OnSubmit != nil {
		return d.OnSubmit(ctx, prompt)
	}
	return ErrNoSubmitHandler
}

func (d *Djinn) Shell(ctx context.Context, command string) (string, error) {
	if d.OnShell != nil {
		return d.OnShell(ctx, command)
	}
	return "", ErrNoShellHandler
}

func (d *Djinn) Command(ctx context.Context, name string, args []string) (string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	switch name {
	case "op":
		if len(args) == 0 {
			return fmt.Sprintf("operation: %s", d.operation), nil
		}
		op, ok := uniform.ParseOperation(args[0])
		if !ok {
			return "", fmt.Errorf("%w: %q (ask, plan, agent)", ErrUnknownOperation, args[0])
		}
		d.operation = op
		return fmt.Sprintf("operation → %s", d.operation), nil

	case "ac":
		if len(args) == 0 {
			return fmt.Sprintf("agents: %s", d.capacity), nil
		}
		var n int
		if _, err := fmt.Sscanf(args[0], "%d", &n); err != nil {
			return "", fmt.Errorf("%w: %s", ErrInvalidCapacity, args[0])
		}
		if err := d.capacity.SetCap(n); err != nil {
			return "", err
		}
		return fmt.Sprintf("capacity → %s", d.capacity), nil

	case "ac+":
		d.capacity.Inc()
		return fmt.Sprintf("capacity → %s", d.capacity), nil

	case "ac-":
		if err := d.capacity.Dec(); err != nil {
			return "", err
		}
		return fmt.Sprintf("capacity → %s", d.capacity), nil

	case "envelope":
		return d.commandEnvelope(args)

	case "sight":
		return d.commandSight(args)

	default:
		if d.OnCommand != nil {
			return d.OnCommand(ctx, name, args)
		}
		return "", fmt.Errorf("%w: %s", ErrUnknownCommand, name)
	}
}

// commandEnvelope handles :envelope subcommands.
func (d *Djinn) commandEnvelope(args []string) (string, error) {
	if len(args) == 0 {
		status := "off"
		if d.envelopeCfg.Enabled {
			status = "on"
		}
		return fmt.Sprintf("envelope: %s (checkpoint every %d tasks, drift threshold %.0f%%)",
			status, d.envelopeCfg.CheckpointEvery, d.envelopeCfg.DriftThreshold*100), nil
	}
	switch args[0] {
	case "on":
		d.envelopeCfg.Enabled = true
		return "envelope → on", nil
	case "off":
		d.envelopeCfg.Enabled = false
		return "envelope → off", nil
	case "every":
		if len(args) < 2 {
			return "", fmt.Errorf("%w: :envelope every N", ErrUsage)
		}
		var n int
		if _, err := fmt.Sscanf(args[1], "%d", &n); err != nil || n < 1 {
			return "", fmt.Errorf("%w: %s", ErrInvalidFrequency, args[1])
		}
		d.envelopeCfg.CheckpointEvery = n
		return fmt.Sprintf("envelope checkpoint every %d tasks", n), nil
	default:
		return "", fmt.Errorf("%w: %q (on, off, every N)", ErrUnknownSubcmd, args[0])
	}
}

// commandSight handles :sight subcommands for agent visibility control.
//
//	:sight             — show current state
//	:sight on <panel>  — open panel gate (agent can see it)
//	:sight off <panel> — close panel gate (agent cannot see it)
//	:sight reveal <panel>.<field> — override sensitive, show to agent
//	:sight hide <panel>.<field>   — re-obscure a revealed field
func (d *Djinn) commandSight(args []string) (string, error) {
	if len(args) == 0 {
		return d.sightMgr.Status(), nil
	}

	switch args[0] {
	case "on":
		if len(args) < 2 { //nolint:mnd // subcommand + panel
			return "", fmt.Errorf("%w: :sight on <panel>", ErrUsage)
		}
		d.sightMgr.SetGate(args[1], true)
		return fmt.Sprintf("sight: %s → on", args[1]), nil

	case "off":
		if len(args) < 2 { //nolint:mnd // subcommand + panel
			return "", fmt.Errorf("%w: :sight off <panel>", ErrUsage)
		}
		d.sightMgr.SetGate(args[1], false)
		return fmt.Sprintf("sight: %s → off", args[1]), nil

	case "reveal":
		if len(args) < 2 { //nolint:mnd // subcommand + panel.field
			return "", fmt.Errorf("%w: :sight reveal <panel>.<field>", ErrUsage)
		}
		panel, field, ok := parsePanelField(args[1])
		if !ok {
			return "", fmt.Errorf("%w: expected panel.field, got %q", ErrUsage, args[1])
		}
		d.sightMgr.Reveal(panel, field)
		return fmt.Sprintf("sight: %s.%s → revealed", panel, field), nil

	case "hide":
		if len(args) < 2 { //nolint:mnd // subcommand + panel.field
			return "", fmt.Errorf("%w: :sight hide <panel>.<field>", ErrUsage)
		}
		panel, field, ok := parsePanelField(args[1])
		if !ok {
			return "", fmt.Errorf("%w: expected panel.field, got %q", ErrUsage, args[1])
		}
		d.sightMgr.Hide(panel, field)
		return fmt.Sprintf("sight: %s.%s → hidden", panel, field), nil

	default:
		return "", fmt.Errorf("%w: %q (on, off, reveal, hide)", ErrUnknownSubcmd, args[0])
	}
}

// parsePanelField splits "panel.field" into its components.
func parsePanelField(s string) (panel, field string, ok bool) {
	parts := strings.SplitN(s, ".", 2) //nolint:mnd // exactly 2 parts
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	return parts[0], parts[1], true
}

// SightManager returns the sight manager for external use (e.g., TUI model).
func (d *Djinn) SightManager() *tui.SightManager {
	return d.sightMgr
}

func (d *Djinn) SetOperation(op string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if parsed, ok := uniform.ParseOperation(op); ok {
		d.operation = parsed
	}
}

func (d *Djinn) SetCapacity(n int) error {
	return d.capacity.SetCap(n)
}

func (d *Djinn) NavigateScope(path, scopeType string) error {
	d.mu.Lock()
	d.scopePath = path
	d.scopeType = scopeType
	d.mu.Unlock()

	// Create VFS mount entries based on scope type and registered repos.
	st := workspace.ScopeType(scopeType)
	if st.Valid() {
		d.mountScopeRepos(path, st)
	}

	if d.OnNavigate != nil {
		return d.OnNavigate(path, scopeType)
	}
	return nil
}

// SetRepos registers host repo paths for VFS mount creation during scope
// navigation. Called during initialization to provide workspace repo info.
func (d *Djinn) SetRepos(repos map[string]string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.repos = repos
}

// Mounts returns the VFS mount table for external use (e.g., agent path translation).
func (d *Djinn) Mounts() *workspace.MountTable {
	return d.mounts
}

// mountScopeRepos creates VFS mount entries for repos matching the navigated scope.
// Mount semantics:
//   - Global: read-only mounts of all repos
//   - System: mount persists until session ends
//   - Operations: mount is ephemeral (caller unmounts when done)
func (d *Djinn) mountScopeRepos(scopePath string, st workspace.ScopeType) {
	d.mu.RLock()
	repos := d.repos
	d.mu.RUnlock()

	if repos == nil {
		return
	}

	readOnly := st == workspace.ScopeGlobal

	for name, hostPath := range repos {
		virtualPath := scopePath
		if !strings.HasSuffix(virtualPath, "/"+name) {
			virtualPath = scopePath + "/" + name
		}
		// Best-effort mount — skip conflicts (already mounted).
		_ = d.mounts.Mount(virtualPath, hostPath, readOnly, st)
	}
}

func (d *Djinn) SetEnvelopeEnabled(enabled bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.envelopeCfg.Enabled = enabled
}

// --- Viewer ---

func (d *Djinn) Subscribe(ch chan<- ViewEvent) {
	d.subMu.Lock()
	defer d.subMu.Unlock()
	d.subscribers = append(d.subscribers, ch)
}

func (d *Djinn) Unsubscribe(ch chan<- ViewEvent) {
	d.subMu.Lock()
	defer d.subMu.Unlock()
	for i, sub := range d.subscribers {
		if sub == ch {
			d.subscribers = append(d.subscribers[:i], d.subscribers[i+1:]...)
			return
		}
	}
}

func (d *Djinn) Status() RunState {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return RunState{
		Operation:   d.operation.String(),
		AgentCount:  d.agentCount,
		AgentCap:    d.capacity.Cap(),
		Turns:       d.turns,
		TokensIn:    d.tokensIn,
		TokensOut:   d.tokensOut,
		ActiveRole:  d.activeRole,
		ScopePath:   d.scopePath,
		ScopeType:   d.scopeType,
		EnvelopeOn:  d.envelopeCfg.Enabled,
		IsStreaming: d.isStreaming,
	}
}

func (d *Djinn) Introspect() IntrospectionReport {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return IntrospectionReport{
		RunState: RunState{
			Operation:   d.operation.String(),
			AgentCount:  d.agentCount,
			AgentCap:    d.capacity.Cap(),
			Turns:       d.turns,
			TokensIn:    d.tokensIn,
			TokensOut:   d.tokensOut,
			ActiveRole:  d.activeRole,
			ScopePath:   d.scopePath,
			ScopeType:   d.scopeType,
			EnvelopeOn:  d.envelopeCfg.Enabled,
			IsStreaming: d.isStreaming,
		},
		Uptime: time.Since(d.startedAt),
	}
}

// --- Lifecycle ---

func (d *Djinn) Start(ctx context.Context) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.startedAt = time.Now()
	return nil
}

func (d *Djinn) Stop() {
	// Notify subscribers that we're done.
	d.Emit(ViewEvent{Kind: EventDone, Timestamp: time.Now()})
}

// --- Event emission (used by adapters to push output to subscribers) ---

// Emit sends a ViewEvent to all subscribers. Non-blocking.
func (d *Djinn) Emit(ev ViewEvent) {
	if ev.Timestamp.IsZero() {
		ev.Timestamp = time.Now()
	}
	d.subMu.RLock()
	defer d.subMu.RUnlock()
	for _, ch := range d.subscribers {
		select {
		case ch <- ev:
		default:
			// Non-blocking: subscriber is slow, skip.
		}
	}
}

// --- State setters (called by adapters to update observable state) ---

// SetTokens updates the cumulative token counts.
func (d *Djinn) SetTokens(in, out int) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.tokensIn = in
	d.tokensOut = out
}

// SetTurns updates the conversation turn count.
func (d *Djinn) SetTurns(n int) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.turns = n
}

// SetActiveRole updates the currently active agent role.
func (d *Djinn) SetActiveRole(role string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.activeRole = role
}

// SetAgentCount updates the number of running agents.
func (d *Djinn) SetAgentCount(n int) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.agentCount = n
}

// SetStreaming updates the streaming state.
func (d *Djinn) SetStreaming(v bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.isStreaming = v
}

// Operation returns the current operation.
func (d *Djinn) Operation() uniform.Operation {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.operation
}

// Capacity returns the agent capacity tracker.
func (d *Djinn) Capacity() *execution.AgentCapacity {
	return d.capacity
}

// EnvelopeConfig returns the current envelope configuration.
func (d *Djinn) EnvelopeConfig() substrate.EnvelopeConfig {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.envelopeCfg
}
