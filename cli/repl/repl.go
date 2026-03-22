package repl

import (
	"context"
	"log/slog"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/dpopsuev/djinn/agent"
	"github.com/dpopsuev/djinn/driver"
	"github.com/dpopsuev/djinn/djinnlog"
	"github.com/dpopsuev/djinn/session"
	"github.com/dpopsuev/djinn/workspace"
	"github.com/dpopsuev/djinn/tools/builtin"
)

// Config configures the REPL.
type Config struct {
	Driver       driver.ChatDriver
	Tools        *builtin.Registry
	Session      *session.Session
	SystemPrompt string
	MaxTurns     int
	AutoApprove  bool
	Mode         string // "ask", "plan", "agent", "auto"
	Log          *slog.Logger
	Ring         *djinnlog.RingHandler
	Store         *session.Store      // for auto-save after each turn
	InitialPrompt string             // auto-submit on first render
	WorkspaceBus  *workspace.Bus     // workspace event bus for /workspace-switch
	Transport     interface{}        // clutch.Transport for hot-swap (nil = direct agent.Run)
}

// Run starts the interactive REPL. Blocks until /exit or ctrl-C.
func Run(ctx context.Context, cfg Config) error {
	m := NewModel(cfg)
	m.ctx = ctx

	p := tea.NewProgram(m)

	// Create handler that bridges agent events to Bubbletea messages
	handler := NewHandler(p)

	// Set the handler on the model (it needs access to send messages)
	// We do this by storing it in a package-level variable that runAgent reads.
	// This is necessary because Bubbletea's Model is copied by value.
	setGlobalHandler(handler)
	globalRing = cfg.Ring
	globalWorkspaceBus = cfg.WorkspaceBus

	_, err := p.Run()
	return err
}

// globalHandler is set before the program runs and read by runAgent.
// This bridges the gap between Bubbletea's value-copy Model and
// the agent.EventHandler which needs a stable program reference.
var globalHandler agent.EventHandler = agent.NilHandler{}

// globalRing is set before the program runs and read by /log command.
var globalRing *djinnlog.RingHandler

// globalWorkspaceBus is set before the program runs and read by /workspace-switch.
var globalWorkspaceBus *workspace.Bus

func setGlobalHandler(h agent.EventHandler) {
	globalHandler = h
}

// handler field on Model reads from the global. This is necessary
// because Bubbletea copies the Model by value in Update, so we
// can't store the handler on the model after program creation.
func init() {
	// Model.handler will be set in NewModel
}
