package repl

import (
	"context"
	"log/slog"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/dpopsuev/djinn/agent"
	"github.com/dpopsuev/djinn/djinnlog"
	"github.com/dpopsuev/djinn/driver"
	"github.com/dpopsuev/djinn/policy"
	"github.com/dpopsuev/djinn/session"
	"github.com/dpopsuev/djinn/staff"
	"github.com/dpopsuev/djinn/tools/builtin"
	"github.com/dpopsuev/djinn/tui"
	"github.com/dpopsuev/djinn/workspace"
)

// Config configures the REPL.
type Config struct {
	Driver        driver.ChatDriver
	Tools         *builtin.Registry
	Session       *session.Session
	SystemPrompt  string
	MaxTurns      int
	AutoApprove   bool
	Mode          string // "ask", "plan", "agent", "auto"
	Log           *slog.Logger
	Ring          *djinnlog.RingHandler
	Store         *session.Store            // for auto-save after each turn
	InitialPrompt string                    // auto-submit on first render
	WorkspaceBus  *workspace.Bus            // workspace event bus for /workspace-switch
	Transport     interface{}               // clutch.Transport for hot-swap (nil = direct agent.Run)
	Router        *staff.ToolClearance      // capability-based tool routing (nil = use raw registry)
	Enforcer      policy.ToolPolicyEnforcer // ToolPolicyEnforcer for agent call mediation
	Token         policy.CapabilityToken
	HealthReports []tui.HealthReport // initial health from startup
	Version       string             // app version for MOTD (set via ldflags)
	TUIRecorder   *tui.TUIRecorder   // nil = disabled; captures rendered frames

	// Sandbox: when set, all agents except GenSec run inside the sandbox.
	SandboxHandle  string
	SandboxExec    func(ctx context.Context, cmd []string) (stdout, stderr string, err error)
	SandboxBackend string // for dashboard display
	SandboxLevel   string // for dashboard display
}

// Run starts the interactive REPL. Blocks until /exit or ctrl-C.
func Run(ctx context.Context, cfg Config) error { //nolint:gocritic // Config is a value type, changing would break all callers
	m := NewModel(cfg)
	m.ctx = ctx

	p := tea.NewProgram(m)

	// Create handler that bridges agent events to Bubbletea messages
	handler := tui.NewHandler(p)

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
