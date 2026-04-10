// app_backend.go — djinn backend subcommand: headless agent runtime
// that connects to a shell process via Unix socket.
//
// Usage:
//
//	djinn backend --socket /tmp/djinn.sock [--driver claude] [--model ...]
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

	"github.com/dpopsuev/djinn/cortex"
	"github.com/dpopsuev/djinn/driver"
	troupedriver "github.com/dpopsuev/djinn/driver/troupe"
	"github.com/dpopsuev/djinn/hotswap"
	"github.com/dpopsuev/djinn/telemetry"
	"github.com/dpopsuev/djinn/tools/builtin"
	"github.com/dpopsuev/troupe/execution"
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
		return ErrSocketRequired
	}

	// Logging
	logResult := telemetry.Setup(telemetry.Options{Verbose: *verbose})
	log := telemetry.For(logResult.Logger, "backend")
	log.Info("starting backend", "socket", *socketPath)

	// Connect to hub (or legacy shell) via Unix socket.
	// Try hub registration first — if hub is listening, register as backend.
	// Falls back to direct connect for legacy shell mode.
	transport, err := ConnectToHubAsBackend(*socketPath)
	if err != nil {
		// Fallback: direct connect (legacy shell mode).
		transport, err = hotswap.Connect(*socketPath)
		if err != nil {
			return fmt.Errorf("connect to socket: %w", err)
		}
		log.Info("connected to shell (direct)")
	} else {
		log.Info("connected to hub as backend")
	}
	defer transport.Close()

	// Resolve model.
	modelName := *model
	if modelName == "" {
		modelName = DefaultModel
	}

	// Load cortex.
	sessDir := SessionDir()
	store, err := cortex.NewStore(sessDir)
	if err != nil {
		return fmt.Errorf("session store: %w", err)
	}

	var sess *cortex.Session
	if *sessionName != "" {
		sess, err = store.Load(*sessionName)
		if err != nil {
			sess = cortex.New(*sessionName, modelName, Getwd())
			sess.Name = *sessionName
		}
	} else {
		sess = cortex.New(fmt.Sprintf("backend-%d", os.Getpid()), modelName, Getwd())
	}
	sess.Driver = *driverName
	sess.Model = modelName

	// Workspace context for system prompt.
	workDir := Getwd()
	if *wsFlag != "" {
		workDir = *wsFlag
	}
	projectCtx := cortex.LoadProjectContext(workDir)

	prompt := *systemPrompt
	if *systemFile != "" {
		prompt = ReadSystemFile(*systemFile)
	}
	assembledPrompt := cortex.BuildSystemPrompt(projectCtx, prompt)

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
	defer chatDriver.Stop(ctx) //nolint:errcheck // best-effort shutdown

	// Replay session history into driver.
	if err := ReplayHistory(ctx, chatDriver, sess); err != nil {
		log.Warn("session replay failed, starting fresh", "error", err)
		sess.History.Clear()
	}

	// Build tool registry.
	registry := builtin.NewRegistry()
	builtin.RegisterBuiltinTools(registry, ".", HomeDir())

	log.Info("backend ready", "model", modelName, "driver", *driverName, "session", sess.Name)

	// Run the backend loop — blocks until shell sends Quit or context cancels.
	return hotswap.RunBackend(ctx, transport, hotswap.BackendConfig{
		Driver:       chatDriver,
		Tools:        registry,
		Session:      sess,
		SystemPrompt: assembledPrompt,
		MaxTurns:     *maxTurns,
	})
}

// createBackendDriver creates the LLM driver for the backend process.
func createBackendDriver(name, model, systemPrompt string, logger *slog.Logger) (driver.ChatDriver, error) {
	providerName, err := resolveProviderName(name)
	if err != nil {
		return nil, err
	}
	provider, err := execution.NewProviderByName(providerName)
	if err != nil {
		return nil, fmt.Errorf("create provider %q: %w", providerName, err)
	}

	opts := []troupedriver.Option{
		troupedriver.WithTools(registryToAnyllmTools(builtin.NewRegistry())),
	}
	if logger != nil {
		opts = append(opts, troupedriver.WithLogger(logger))
	}
	if systemPrompt != "" {
		opts = append(opts, troupedriver.WithSystemPrompt(systemPrompt))
	}
	return troupedriver.New(provider, model, opts...), nil
}
