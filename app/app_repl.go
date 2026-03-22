// app_repl.go — interactive REPL startup, driver creation, workspace/MCP wiring.
package app

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
	"strings"
	"time"

	"github.com/dpopsuev/djinn/cli/repl"
	djinnconfig "github.com/dpopsuev/djinn/config"
	djinnctx "github.com/dpopsuev/djinn/context"
	"github.com/dpopsuev/djinn/djinnlog"
	"github.com/dpopsuev/djinn/driver"
	claudedriver "github.com/dpopsuev/djinn/driver/claude"
	mcpclient "github.com/dpopsuev/djinn/mcp/client"
	"github.com/dpopsuev/djinn/session"
	"github.com/dpopsuev/djinn/tools/builtin"
	djinnws "github.com/dpopsuev/djinn/workspace"
)

// RunREPL starts the interactive REPL.
func RunREPL(args []string, stderr io.Writer) error {
	fs := flag.NewFlagSet("repl", flag.ContinueOnError)
	fs.SetOutput(stderr)
	driverName := fs.String("driver", DriverClaude, "LLM backend: claude, ollama")
	model := fs.String("model", "", "model name")
	modelShort := fs.String("m", "", "model name (short)")
	sessionName := fs.String("session", "", "named session")
	sessionShort := fs.String("s", "", "named session (short)")
	cont := fs.Bool("continue", false, "resume most recent session")
	contShort := fs.Bool("c", false, "resume most recent (short)")
	maxTurns := fs.Int("max-turns", 20, "max agent turns per prompt")
	autoApprove := fs.Bool("auto-approve", false, "auto-approve all tool calls")
	mode := fs.String("mode", "agent", "agent mode: ask, plan, agent, auto")
	configFile := fs.String("config", "", "load config from YAML file")
	systemPrompt := fs.String("system", "", "system prompt")
	systemFile := fs.String("system-file", "", "load system prompt from file")
	verbose := fs.Bool("verbose", false, "show log output on terminal")
	wsFlag := fs.String("w", "", "workspace name or manifest file")
	wsLong := fs.String("workspace", "", "workspace name or manifest file")
	noPersist := fs.Bool("no-persist", false, "don't save session to disk")
	if err := fs.Parse(args); err != nil {
		return err
	}

	// Resolve short flags
	if *modelShort != "" && *model == "" {
		*model = *modelShort
	}
	if *sessionShort != "" && *sessionName == "" {
		*sessionName = *sessionShort
	}
	if *contShort {
		*cont = true
	}

	// Load config files
	cfgRegistry := djinnconfig.NewRegistry()
	modeConf := &djinnconfig.ModeConfig{Mode: *mode}
	driverConf := &djinnconfig.DriverConfigurable{Name: *driverName, Model: *model}
	sessConf := &djinnconfig.SessionConfigurable{MaxTurns: *maxTurns, AutoApprove: *autoApprove}
	cfgRegistry.Register(modeConf)
	cfgRegistry.Register(driverConf)
	cfgRegistry.Register(sessConf)

	if err := djinnconfig.LoadAll(cfgRegistry, Getwd(), *configFile); err != nil {
		fmt.Fprintf(stderr, "djinn: config: %v\n", err)
	}

	// Apply config file values (CLI flags override)
	if *mode == "agent" && modeConf.Mode != "agent" {
		*mode = modeConf.Mode
	}
	if *model == "" && driverConf.Model != "" {
		*model = driverConf.Model
	}
	if *driverName == DriverClaude && driverConf.Name != DriverClaude {
		*driverName = driverConf.Name
	}
	if *maxTurns == 20 && sessConf.MaxTurns != 20 {
		*maxTurns = sessConf.MaxTurns
	}
	if !*autoApprove && sessConf.AutoApprove {
		*autoApprove = sessConf.AutoApprove
	}

	// Load system prompt from file
	if *systemFile != "" {
		if content := ReadSystemFile(*systemFile); content != "" {
			if *systemPrompt != "" {
				*systemPrompt = *systemPrompt + "\n\n" + content
			} else {
				*systemPrompt = content
			}
		}
	}

	// Resolve model default per driver
	if *model == "" {
		switch *driverName {
		case DriverClaude:
			*model = DefaultModel
		case DriverOllama:
			*model = "qwen2.5-coder:14b"
		default:
			*model = DefaultModel
		}
	}

	store, err := session.NewStore(SessionDir())
	if err != nil {
		return fmt.Errorf("cannot open session store at %s: %w", SessionDir(), err)
	}

	// Session: resume, continue, or new
	var sess *session.Session
	if *cont {
		sess, err = LoadMostRecent(store)
		if err != nil {
			return fmt.Errorf("no session to continue: %w", err)
		}
		fmt.Fprintf(stderr, "djinn: resumed session %q (%d turns)\n", sess.Name, sess.History.Len())
	} else if *sessionName != "" {
		sess, err = store.Load(*sessionName)
		if err != nil {
			sess = session.New(*sessionName, *model, Getwd())
			sess.Name = *sessionName
			sess.Driver = *driverName
		} else {
			fmt.Fprintf(stderr, "djinn: resumed session %q (%d turns)\n", sess.Name, sess.History.Len())
		}
	} else {
		id := fmt.Sprintf("djinn-%d", time.Now().Unix())
		sess = session.New(id, *model, Getwd())
		sess.Driver = *driverName
	}

	if *model != "" {
		sess.Model = *model
	}
	sess.Driver = *driverName

	// Load workspace
	wsName := *wsFlag
	if wsName == "" {
		wsName = *wsLong
	}

	var ws *djinnws.Workspace
	if wsName != "" {
		var wsErr error
		ws, wsErr = djinnws.Load(wsName)
		if wsErr != nil {
			return fmt.Errorf("workspace %q: %w", wsName, wsErr)
		}
		sess.Workspace = ws.Name
	} else if sess.Workspace != "" {
		ws, _ = djinnws.Load(sess.Workspace)
	}
	if ws == nil {
		// No workspace specified — empty workspace, no repos, no context.
		// CWD is NOT a workspace. Repos come from the manifest.
		ws = &djinnws.Workspace{}
	}

	// Workspace config overrides
	if ws.Driver != "" && *driverName == DriverClaude {
		*driverName = ws.Driver
	}
	if ws.Model != "" && *model == "" {
		*model = ws.Model
	}
	if ws.Mode != "" && *mode == "agent" {
		*mode = ws.Mode
	}
	sess.WorkDirs = ws.Paths()

	// Setup logging
	logResult := djinnlog.Setup(djinnlog.Options{
		Verbose: *verbose,
		LogFile: filepath.Join(HomeDir(), "djinn.log"),
	})
	log := djinnlog.For(logResult.Logger, "app")
	log.Info("session starting", "driver", *driverName, "model", *model, "mode", *mode)

	// Auto-discover project context
	projectCtx := djinnctx.LoadProjectContext(sess.WorkDirs...)
	assembledPrompt := djinnctx.BuildSystemPrompt(projectCtx, *systemPrompt)

	chatDriver, err := CreateDriver(*driverName, sess.Model, assembledPrompt, logResult.Logger)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := chatDriver.Start(ctx, ""); err != nil {
		return fmt.Errorf("cannot start %s driver: %w (try: djinn doctor)", *driverName, err)
	}
	defer chatDriver.Stop(ctx)

	// Replay history
	for _, entry := range sess.Entries() {
		switch entry.Role {
		case driver.RoleUser:
			chatDriver.Send(ctx, driver.Message{Role: entry.Role, Content: entry.TextContent()})
		case driver.RoleAssistant:
			chatDriver.AppendAssistant(driver.RichMessage{
				Role:    entry.Role,
				Content: entry.TextContent(),
				Blocks:  entry.Blocks,
			})
		}
	}

	// Connect MCP servers
	mcpClient := mcpclient.New(djinnlog.For(logResult.Logger, "mcp"))
	defer mcpClient.Close()

	mcpConfigs := mcpclient.LoadMCPConfig(Getwd(), filepath.Join(HomeDir()))
	for name, cfg := range mcpConfigs {
		var connectErr error
		if cfg.IsHTTP() {
			connectErr = mcpClient.ConnectHTTP(ctx, name, cfg.URL)
		} else if cfg.Command != "" {
			connectErr = mcpClient.ConnectStdio(ctx, name, cfg.Command, cfg.Args, cfg.Env)
		}
		if connectErr != nil {
			log.Warn("MCP server failed", "server", name, "error", connectErr)
		}
	}

	initialPrompt := strings.Join(fs.Args(), " ")

	// Workspace event bus
	wsBus := djinnws.NewBus(djinnlog.For(logResult.Logger, "workspace"))
	wsBus.On("driver-prompt", func(evt djinnws.Event) {
		if evt.New == nil {
			return
		}
		newCtx := djinnctx.LoadProjectContext(evt.New.Paths()...)
		newPrompt := djinnctx.BuildSystemPrompt(newCtx, *systemPrompt)
		chatDriver.SetSystemPrompt(newPrompt)
	})
	wsBus.On("session", func(evt djinnws.Event) {
		if evt.New == nil {
			return
		}
		sess.Workspace = evt.New.Name
		sess.WorkDirs = evt.New.Paths()
	})

	// Build tool registry
	registry := builtin.NewRegistry()
	for _, tool := range mcpClient.MCPTools() {
		registry.Register(tool)
	}
	log.Info("tools registered", "builtin", 6, "mcp", len(mcpClient.MCPTools()), "total", len(registry.Names()))

	replErr := repl.Run(ctx, repl.Config{
		Driver:        chatDriver,
		Tools:         registry,
		Session:       sess,
		SystemPrompt:  assembledPrompt,
		MaxTurns:      *maxTurns,
		AutoApprove:   *autoApprove,
		Mode:          *mode,
		Log:           logResult.Logger,
		Ring:          logResult.Ring,
		Store:         store,
		InitialPrompt: initialPrompt,
		WorkspaceBus:  wsBus,
	})

	if !*noPersist {
		if saveErr := store.Save(sess); saveErr != nil {
			fmt.Fprintf(stderr, "djinn: save session: %v\n", saveErr)
		}
	}

	return replErr
}

// CreateDriver creates a ChatDriver for the given driver name.
func CreateDriver(driverName, model, systemPrompt string, log ...*slog.Logger) (driver.ChatDriver, error) {
	var driverLog *slog.Logger
	if len(log) > 0 && log[0] != nil {
		driverLog = djinnlog.For(log[0], "driver")
	}
	switch driverName {
	case DriverClaude:
		opts := []claudedriver.APIDriverOption{
			claudedriver.WithTools(builtin.NewRegistry()),
		}
		if driverLog != nil {
			opts = append(opts, claudedriver.WithLogger(driverLog))
		}
		if systemPrompt != "" {
			opts = append(opts, claudedriver.WithAPISystemPrompt(systemPrompt))
		}
		return claudedriver.NewAPIDriver(driver.DriverConfig{Model: model}, opts...)
	case DriverOllama:
		return nil, fmt.Errorf("%w: %s (use --driver claude)", ErrDriverNotImpl, driverName)
	default:
		return nil, fmt.Errorf("%w: %q (supported: claude, ollama)", ErrUnknownDriver, driverName)
	}
}
