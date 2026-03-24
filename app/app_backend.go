// app_backend.go — djinn backend subcommand: headless agent runtime
// that connects to a shell process via Unix socket.
//
// Usage:
//   djinn backend --socket /tmp/djinn.sock [--driver claude] [--model ...]
//
// The shell (TUI) runs separately and communicates via the clutch protocol.
// This enables hot-swap: rebuild the backend, restart it, and the shell
// picks up the new connection without losing conversation state.
package app

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/dpopsuev/djinn/clutch"
	djinnctx "github.com/dpopsuev/djinn/context"
	"github.com/dpopsuev/djinn/djinnlog"
	claudedriver "github.com/dpopsuev/djinn/driver/claude"
	"github.com/dpopsuev/djinn/driver"
	"github.com/dpopsuev/djinn/session"
	"github.com/dpopsuev/djinn/tools/builtin"
)

// RunBackendCmd starts the headless backend process.
// Connects to the shell via Unix socket and runs the agent loop.
func RunBackendCmd(args []string, stderr io.Writer) error {
	fs := flag.NewFlagSet("backend", flag.ContinueOnError)
	fs.SetOutput(stderr)
	socketPath := fs.String("socket", "", "Unix socket path to connect to shell")
	driverName := fs.String("driver", DriverClaude, "LLM backend: claude, ollama")
	model := fs.String("model", "", "model name")
	sessionName := fs.String("session", "", "named session to resume")
	maxTurns := fs.Int("max-turns", 20, "max agent turns per prompt")
	systemPrompt := fs.String("system", "", "system prompt")
	systemFile := fs.String("system-file", "", "load system prompt from file")
	wsFlag := fs.String("w", "", "workspace name or manifest file")
	verbose := fs.Bool("verbose", false, "show log output on terminal")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if *socketPath == "" {
		return fmt.Errorf("--socket is required for backend mode")
	}

	// Logging
	logResult := djinnlog.Setup(djinnlog.Options{Verbose: *verbose})
	log := djinnlog.For(logResult.Logger, "backend")
	log.Info("starting backend", "socket", *socketPath)

	// Connect to shell via Unix socket.
	transport, err := clutch.Connect(*socketPath)
	if err != nil {
		return fmt.Errorf("connect to shell: %w", err)
	}
	defer transport.Close()
	log.Info("connected to shell")

	// Resolve model.
	modelName := *model
	if modelName == "" {
		modelName = DefaultModel
	}

	// Load session.
	sessDir := SessionDir()
	store, err := session.NewStore(sessDir)
	if err != nil {
		return fmt.Errorf("session store: %w", err)
	}

	var sess *session.Session
	if *sessionName != "" {
		sess, err = store.Load(*sessionName)
		if err != nil {
			sess = session.New(*sessionName, modelName, Getwd())
			sess.Name = *sessionName
		}
	} else {
		sess = session.New(fmt.Sprintf("backend-%d", os.Getpid()), modelName, Getwd())
	}
	sess.Driver = *driverName
	sess.Model = modelName

	// Workspace context for system prompt.
	workDir := Getwd()
	if *wsFlag != "" {
		workDir = *wsFlag
	}
	projectCtx := djinnctx.LoadProjectContext(workDir)

	prompt := *systemPrompt
	if *systemFile != "" {
		prompt = ReadSystemFile(*systemFile)
	}
	assembledPrompt := djinnctx.BuildSystemPrompt(projectCtx, prompt)

	// Create driver.
	chatDriver, err := createBackendDriver(*driverName, modelName, assembledPrompt, log)
	if err != nil {
		return fmt.Errorf("create driver: %w", err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	if err := chatDriver.Start(ctx, ""); err != nil {
		return fmt.Errorf("start driver: %w", err)
	}
	defer chatDriver.Stop(ctx) //nolint:errcheck

	// Replay session history into driver.
	if err := ReplayHistory(ctx, chatDriver, sess); err != nil {
		log.Warn("session replay failed, starting fresh", "error", err)
		sess.History.Clear()
	}

	// Build tool registry.
	registry := builtin.NewRegistry()

	log.Info("backend ready", "model", modelName, "driver", *driverName, "session", sess.Name)

	// Run the backend loop — blocks until shell sends Quit or context cancels.
	return clutch.RunBackend(ctx, transport, clutch.BackendConfig{
		Driver:       chatDriver,
		Tools:        registry,
		Session:      sess,
		SystemPrompt: assembledPrompt,
		MaxTurns:     *maxTurns,
	})
}

// createBackendDriver creates the LLM driver for the backend process.
// Separate from app_repl.go's CreateDriver to avoid circular dependencies.
func createBackendDriver(name, model, systemPrompt string, logger *slog.Logger) (driver.ChatDriver, error) {
	switch name {
	case DriverClaude:
		return claudedriver.NewAPIDriver(driver.DriverConfig{
			Model: model,
		},
			claudedriver.WithAPISystemPrompt(systemPrompt),
			claudedriver.WithLogger(logger),
		)
	default:
		return nil, fmt.Errorf("unknown driver: %s (supported: claude)", name)
	}
}
