// app_repl.go — interactive REPL startup, driver creation, workspace/MCP wiring.
package app

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dpopsuev/djinn/cli/repl"
	"github.com/dpopsuev/djinn/clutch"
	djinnconfig "github.com/dpopsuev/djinn/config"
	djinnctx "github.com/dpopsuev/djinn/context"
	"github.com/dpopsuev/djinn/djinnlog"
	"github.com/dpopsuev/djinn/driver"
	claudedriver "github.com/dpopsuev/djinn/driver/claude"
	mcpclient "github.com/dpopsuev/djinn/mcp/client"
	"github.com/dpopsuev/djinn/policy"
	"github.com/dpopsuev/djinn/sandbox"
	"github.com/dpopsuev/djinn/session"
	"github.com/dpopsuev/djinn/staff"
	"github.com/dpopsuev/djinn/tools/builtin"
	"github.com/dpopsuev/djinn/tui"
	djinnws "github.com/dpopsuev/djinn/workspace"
)

// RunREPL starts the interactive REPL.
// Essential flags: -m (model), -s (session), -c (continue), -e (ecosystem),
// --config, --socket. Everything else lives in djinn.yaml.
func RunREPL(args []string, stderr io.Writer) error {
	fs := flag.NewFlagSet("repl", flag.ContinueOnError)
	fs.SetOutput(stderr)

	// Essential CLI flags — these are the only ones you need regularly.
	model := fs.String("m", "", "model name (overrides config)")
	sessionName := fs.String("s", "", "named session")
	cont := fs.Bool("c", false, "resume most recent session")
	ecoFlag := fs.String("e", "", "ecosystem or system scope (e.g. aeon, aeon/djinn)")
	configFile := fs.String("config", "", "config file path (default: djinn.yaml)")
	socketPath := fs.String("socket", "", "Unix socket for hot-swap")

	// Override flags — rarely needed, prefer djinn.yaml.
	driverName := fs.String("driver", "", "LLM backend (config: driver.name)")
	mode := fs.String("mode", "", "agent mode (config: mode)")
	systemPrompt := fs.String("system", "", "system prompt text")
	systemFile := fs.String("system-file", "", "system prompt from file")
	verbose := fs.Bool("verbose", false, "show log output (config: debug.verbose)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	// Load config files — djinn.yaml is the primary config source.
	cfgRegistry := djinnconfig.NewRegistry()
	modeConf := &djinnconfig.ModeConfig{Mode: "agent"}
	driverConf := &djinnconfig.DriverConfigurable{Name: DriverClaude}
	sessConf := &djinnconfig.SessionConfigurable{MaxTurns: 20}
	sandboxConf := &djinnconfig.SandboxConfigurable{Level: "namespace"}
	debugConf := &djinnconfig.DebugConfigurable{}
	cfgRegistry.Register(modeConf)
	cfgRegistry.Register(driverConf)
	cfgRegistry.Register(sessConf)
	cfgRegistry.Register(sandboxConf)
	cfgRegistry.Register(debugConf)

	if err := djinnconfig.LoadAll(cfgRegistry, Getwd(), *configFile); err != nil {
		fmt.Fprintf(stderr, "djinn: config: %v\n", err)
	}

	// CLI flags override config values.
	if *mode != "" {
		modeConf.Mode = *mode
	}
	if *model != "" {
		driverConf.Model = *model
	}
	if *driverName != "" {
		driverConf.Name = *driverName
	}
	if *verbose {
		debugConf.Verbose = true
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
	if driverConf.Model == "" {
		switch driverConf.Name {
		case DriverClaude:
			driverConf.Model = DefaultModel
		case DriverOllama:
			driverConf.Model = "qwen2.5-coder:14b"
		default:
			driverConf.Model = DefaultModel
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
			sess = session.New(*sessionName, driverConf.Model, Getwd())
			sess.Name = *sessionName
			sess.Driver = driverConf.Name
		} else {
			fmt.Fprintf(stderr, "djinn: resumed session %q (%d turns)\n", sess.Name, sess.History.Len())
		}
	} else {
		id := fmt.Sprintf("djinn-%d", time.Now().Unix())
		sess = session.New(id, driverConf.Model, Getwd())
		sess.Driver = driverConf.Name
	}

	sess.Model = driverConf.Model
	sess.Driver = driverConf.Name

	// Load workspace/ecosystem
	wsName := *ecoFlag
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
		ws = &djinnws.Workspace{}
	}

	// Workspace config overrides
	if ws.Driver != "" && driverConf.Name == DriverClaude {
		driverConf.Name = ws.Driver
	}
	if ws.Model != "" && driverConf.Model == DefaultModel {
		driverConf.Model = ws.Model
	}
	if ws.Mode != "" && modeConf.Mode == "agent" {
		modeConf.Mode = ws.Mode
	}
	sess.WorkDirs = ws.Paths()

	// Setup logging
	logResult := djinnlog.Setup(djinnlog.Options{
		Verbose: debugConf.Verbose,
		TUI:     true,
		LogFile: filepath.Join(HomeDir(), "djinn.log"),
	})
	log := djinnlog.For(logResult.Logger, "app")
	log.Info("session starting", "driver", driverConf.Name, "model", driverConf.Model, "mode", modeConf.Mode)

	// Auto-import Claude session for new empty sessions
	if sess.History.Len() == 0 && ws.PrimaryPath() != "" {
		home, _ := os.UserHomeDir()
		slug := strings.ReplaceAll(ws.PrimaryPath(), "/", "-")
		claudeDir := filepath.Join(home, ".claude", "projects", slug)
		if jsonl := findMostRecentJSONL(claudeDir); jsonl != "" {
			imported, importErr := session.ImportClaudeSession(jsonl, 0)
			if importErr == nil && imported.History.Len() > 0 {
				for _, entry := range imported.Entries() {
					sess.Append(entry)
				}
				log.Info("auto-imported Claude session", "turns", imported.History.Len())
			}
		}
	}

	// Auto-discover project context
	projectCtx := djinnctx.LoadProjectContext(sess.WorkDirs...)
	assembledPrompt := djinnctx.BuildSystemPrompt(projectCtx, *systemPrompt)

	chatDriver, err := CreateDriver(driverConf.Name, sess.Model, assembledPrompt, logResult.Logger)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := chatDriver.Start(ctx, ""); err != nil {
		return fmt.Errorf("cannot start %s driver: %w (try: djinn doctor)", *driverName, err)
	}
	defer chatDriver.Stop(ctx) //nolint:errcheck // best-effort cleanup on exit

	// Replay history into driver. If replay fails (corrupt session),
	// auto-recover by clearing history and starting fresh.
	if err := ReplayHistory(ctx, chatDriver, sess); err != nil {
		fmt.Fprintf(stderr, "djinn: session replay failed: %v\n", err)
		fmt.Fprintf(stderr, "djinn: clearing history and starting fresh\n")
		sess.History.Clear()
	}

	// Connect MCP servers
	mcpClient := mcpclient.New(djinnlog.For(logResult.Logger, "mcp"))
	defer mcpClient.Close()

	// Convert workspace MCP defs to client config
	var wsMCPConfig map[string]mcpclient.ServerConfig
	if len(ws.MCP) > 0 {
		wsMCPConfig = make(map[string]mcpclient.ServerConfig, len(ws.MCP))
		for name, def := range ws.MCP {
			wsMCPConfig[name] = mcpclient.ServerConfig{
				URL:     def.URL,
				Command: def.Command,
				Args:    def.Args,
				Env:     def.Env,
			}
		}
	}

	var mcpFailures []tui.HealthReport
	mcpConfigs := mcpclient.LoadMCPConfig(Getwd(), filepath.Join(HomeDir()), wsMCPConfig)
	for name, cfg := range mcpConfigs {
		var connectErr error
		if cfg.IsHTTP() {
			connectErr = mcpClient.ConnectHTTP(ctx, name, cfg.URL)
		} else if cfg.Command != "" {
			connectErr = mcpClient.ConnectStdio(ctx, name, cfg.Command, cfg.Args, cfg.Env)
		}
		if connectErr != nil {
			log.Warn("MCP server offline", "server", name, "error", connectErr)
			mcpFailures = append(mcpFailures, tui.HealthReport{
				Component: name,
				Status:    tui.StatusOffline,
				Message:   fmt.Sprintf("start failed: %s", connectErr.Error()),
			})
		}
	}

	// Build health reports from MCP connection results
	var healthReports []tui.HealthReport
	for _, name := range mcpClient.ServerNames() {
		tools, _ := mcpClient.ServerTools(name)
		healthReports = append(healthReports, tui.HealthReport{
			Component: name,
			Status:    tui.StatusGreen,
			Message:   fmt.Sprintf("%d tools", len(tools)),
		})
	}

	healthReports = append(healthReports, mcpFailures...)

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

	// PolicyEnforcer — agent call mediation
	enforcer := policy.NewDefaultEnforcer()
	capToken := ws.ToCapabilityToken()
	log.Info("policy enforcer active", "writable", len(capToken.WritablePaths), "denied", len(capToken.DeniedPaths))

	// Build tool registry
	registry := builtin.NewRegistry()
	for _, tool := range mcpClient.MCPTools() {
		registry.Register(tool)
	}
	log.Info("tools registered", "builtin", 6, "mcp", len(mcpClient.MCPTools()), "total", len(registry.Names()))

	// Create slot router — filters tools by role.
	// The agent sees only tools belonging to the current role's slots.
	staffCfg := staff.DefaultConfig()
	slotRouter := staff.NewSlotRouter(staffCfg, registry, "gensec")

	// Sandbox: if configured, create an isolated environment.
	if sandboxConf.Backend != "" {
		sb, sbErr := sandbox.Get(sandboxConf.Backend)
		if sbErr != nil {
			return fmt.Errorf("sandbox %q: %w (available: %v)", sandboxConf.Backend, sbErr, sandbox.Available())
		}
		repos := sess.AllWorkDirs()
		handle, createErr := sb.Create(ctx, sandboxConf.Level, repos)
		if createErr != nil {
			return fmt.Errorf("create sandbox: %w", createErr)
		}
		defer sb.Destroy(ctx, handle) //nolint:errcheck
		log.Info("sandbox created", "backend", sandboxConf.Backend, "level", sandboxConf.Level, "handle", handle)
	}

	// Start clutch shell/backend transport.
	// Priority: --socket flag → auto-detect hub → in-process channel.
	var transport clutch.Transport
	var useInProcess bool

	// backendCfg is shared across hub and in-process modes.
	backendCfg := clutch.BackendConfig{
		Driver:       chatDriver,
		Tools:        registry,
		Session:      sess,
		SystemPrompt: assembledPrompt,
		MaxTurns:     sessConf.MaxTurns,
	}

	hubSocket := *socketPath
	if hubSocket == "" {
		if detected, ok := HubSocketExists(); ok {
			hubSocket = detected
		}
	}

	if hubSocket != "" {
		// Hub mode: connect as shell, auto-spawn backend goroutine via hub.
		hubTransport, hubErr := ConnectToHub(hubSocket)
		if hubErr != nil {
			log.Warn("hub connect failed, falling back to in-process", "error", hubErr)
			useInProcess = true
		} else {
			defer hubTransport.Close()
			transport = hubTransport
			fmt.Fprintf(stderr, "djinn: connected to hub at %s\n", hubSocket)

			// Auto-spawn backend as goroutine connecting to the SAME hub.
			go func() {
				beTransport, beErr := ConnectToHubAsBackend(hubSocket)
				if beErr != nil {
					log.Error("backend connect to hub failed", "error", beErr)
					return
				}
				defer beTransport.Close()
				backendErr := clutch.RunBackend(ctx, beTransport, backendCfg)
				if backendErr != nil {
					log.Error("backend exited", "error", backendErr)
				}
			}()
		}
	} else {
		useInProcess = true
	}

	if useInProcess {
		channelTransport := clutch.NewChannelTransport()
		defer channelTransport.Close()
		transport = channelTransport

		go func() {
			backendErr := clutch.RunBackend(ctx, channelTransport, backendCfg)
			if backendErr != nil {
				log.Error("backend exited", "error", backendErr)
			}
		}()
	}

	// DebugTap: configured via debug.tap_file and debug.live_debug in djinn.yaml.
	var debugTap *tui.DebugTap
	if debugConf.TapFile != "" || debugConf.LiveDebug != "" {
		var tapErr error
		debugTap, tapErr = tui.NewDebugTap(100, debugConf.TapFile)
		if tapErr != nil {
			fmt.Fprintf(stderr, "djinn: debug tap: %v\n", tapErr)
		} else {
			defer debugTap.Close() //nolint:errcheck
			if debugConf.TapFile != "" {
				fmt.Fprintf(stderr, "djinn: debug tap writing to %s\n", debugConf.TapFile)
			}
			if debugConf.LiveDebug != "" {
				ln, srvErr := debugTap.ServeHTTP(debugConf.LiveDebug)
				if srvErr != nil {
					fmt.Fprintf(stderr, "djinn: debug server: %v\n", srvErr)
				} else {
					fmt.Fprintf(stderr, "djinn: debug server at http://%s/debug/\n", ln.Addr())
				}
			}
		}
	}

	replErr := repl.Run(ctx, repl.Config{
		Driver:        chatDriver,
		Tools:         registry,
		Session:       sess,
		SystemPrompt:  assembledPrompt,
		MaxTurns:      sessConf.MaxTurns,
		AutoApprove:   sessConf.AutoApprove,
		Mode:          modeConf.Mode,
		Log:           logResult.Logger,
		Ring:          logResult.Ring,
		Store:         store,
		InitialPrompt: initialPrompt,
		WorkspaceBus:  wsBus,
		Transport:     transport,
		Enforcer:      enforcer,
		Token:         capToken,
		HealthReports: healthReports,
		Router:        slotRouter,
		Version:       Version,
		DebugTap:      debugTap,
	})

	if !sessConf.NoPersist {
		if saveErr := store.Save(sess); saveErr != nil {
			fmt.Fprintf(stderr, "djinn: save session: %v\n", saveErr)
		}
	}

	return replErr
}

// findMostRecentJSONL scans a directory for .jsonl files and returns the most recent.
func findMostRecentJSONL(dir string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	var best string
	var bestTime int64
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".jsonl") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if info.ModTime().UnixNano() > bestTime {
			bestTime = info.ModTime().UnixNano()
			best = filepath.Join(dir, e.Name())
		}
	}
	return best
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
