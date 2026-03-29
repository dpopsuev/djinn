// app_repl.go — interactive REPL startup, driver creation, workspace/MCP wiring.
package app

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dpopsuev/djinn/cli/repl"
	"github.com/dpopsuev/djinn/clutch"
	djinnconfig "github.com/dpopsuev/djinn/config"
	djinnctx "github.com/dpopsuev/djinn/context"
	"github.com/dpopsuev/djinn/djinnlog"
	mcpclient "github.com/dpopsuev/djinn/mcp/client"
	"github.com/dpopsuev/djinn/policy"
	"github.com/dpopsuev/djinn/sandbox"
	"github.com/dpopsuev/djinn/session"
	"github.com/dpopsuev/djinn/staff"
	"github.com/dpopsuev/djinn/tools/builtin"
	"github.com/dpopsuev/djinn/trace"
	"github.com/dpopsuev/djinn/tui"
	djinnws "github.com/dpopsuev/djinn/workspace"
)

// RunREPL starts the interactive REPL.
// Essential flags: -m (model), -s (session), -c (continue), -e (ecosystem),
// --config, --socket. Everything else lives in djinn.yaml.
func RunREPL(args []string, stderr io.Writer) error { //nolint:gocyclo,funlen // composition root, flag parsing + setup
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
	driverConf := &djinnconfig.DriverConfigurable{} // no default — djinn.yaml must specify
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

	// If no driver configured, auto-detect from installed CLIs.
	if driverConf.Name == "" { //nolint:nestif // auto-detect driver with fallback chain
		detected := ScanArsenal()
		if len(detected) > 0 {
			best := detected[0]
			driverConf.Name = best.ACPName()
			driverConf.Model = best.DefaultModel()
			fmt.Fprintf(stderr, "djinn: auto-detected %s", best.Name)
			if best.Binary != "" {
				fmt.Fprintf(stderr, " (%s)", best.Binary)
			}
			fmt.Fprintln(stderr)

			// Generate djinn.yaml for future runs.
			if err := GenerateConfig(Getwd(), best); err == nil {
				fmt.Fprintf(stderr, "djinn: created djinn.yaml — edit to customize\n")
			}
		} else {
			return fmt.Errorf("%w\n\n%s", ErrNoDriverDetected, FriendlyNoDriverError())
		}
	}

	store, err := session.NewStore(SessionDir())
	if err != nil {
		return fmt.Errorf("cannot open session store at %s: %w", SessionDir(), err)
	}

	// Session: resume, continue, or new
	var sess *session.Session
	switch {
	case *cont:
		sess, err = LoadMostRecent(store)
		if err != nil {
			return fmt.Errorf("no session to continue: %w", err)
		}
		fmt.Fprintf(stderr, "djinn: resumed session %q (%d turns)\n", sess.Name, sess.History.Len())
	case *sessionName != "":
		sess, err = store.Load(*sessionName)
		if err != nil {
			sess = session.New(*sessionName, driverConf.Model, Getwd())
			sess.Name = *sessionName
			sess.Driver = driverConf.Name
		} else {
			fmt.Fprintf(stderr, "djinn: resumed session %q (%d turns)\n", sess.Name, sess.History.Len())
		}
	default:
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

	// Trace ring — observable by default (Flywheel Tier 4).
	traceRing := trace.NewRing(1000) //nolint:mnd // 1000 events is a sensible default

	// Connect MCP servers
	mcpClient := mcpclient.New(djinnlog.For(logResult.Logger, "mcp"))
	mcpClient.Tracer = traceRing.For(trace.ComponentMCP)
	defer mcpClient.Close()

	// MCP config from djinn.yaml ONLY — Djinn owns MCP, agents get a mirror.
	var mcpFailures []tui.HealthReport
	mcpConfigs := mcpclient.LoadMCPConfig(Getwd())
	for name, cfg := range mcpConfigs {
		var connectErr error
		if cfg.IsHTTP() {
			connectErr = mcpClient.ConnectHTTP(ctx, name, cfg.URL)
		} else if cfg.Command != "" {
			connectErr = mcpClient.ConnectStdio(ctx, name, cfg.Command, cfg.Args, cfg.Env)
		}
		if connectErr != nil {
			// Distinguish: connection refused = offline, init error = degraded.
			status := tui.StatusOffline
			if strings.Contains(connectErr.Error(), "initialize") || strings.Contains(connectErr.Error(), "tools/list") {
				status = tui.StatusYellow // server running but init failed
			}
			log.Warn("MCP server offline", "server", name, "error", connectErr)
			mcpFailures = append(mcpFailures, tui.HealthReport{
				Component: name,
				Status:    status,
				Message:   fmt.Sprintf("start failed: %s", connectErr.Error()),
			})
		}
	}

	// Build health reports from MCP connection results
	serverNames := mcpClient.ServerNames()
	healthReports := make([]tui.HealthReport, 0, len(serverNames))
	for _, name := range serverNames {
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

	// ToolPolicyEnforcer — agent call mediation
	enforcer := policy.NewDefaultToolPolicyEnforcer()
	capToken := ws.ToCapabilityToken()
	log.Info("policy enforcer active", "writable", len(capToken.WritablePaths), "denied", len(capToken.DeniedPaths))

	// Build tool registry
	registry := builtin.NewRegistry()
	builtin.RegisterAeonShellTools(registry, ws.PrimaryPath(), HomeDir())
	builtin.RegisterDebugTrace(registry, traceRing)
	for _, tool := range mcpClient.MCPTools() {
		registry.Register(tool)
	}
	log.Info("tools registered", "builtin", len(registry.Names())-len(mcpClient.MCPTools()), "mcp", len(mcpClient.MCPTools()), "total", len(registry.Names()))

	// Load staff config: built-in defaults → user config → workspace config.
	staffCfg := staff.LoadConfigChain(
		filepath.Join(HomeDir(), "staff.yaml"),
		filepath.Join(ws.PrimaryPath(), "staff.yaml"),
	)
	if err := staffCfg.Validate(); err != nil {
		return fmt.Errorf("staff config: %w", err)
	}

	// Create tool clearance — filters tools by role's capabilities.
	slotRouter := staff.NewToolClearance(staffCfg, registry, "gensec")

	// Sandbox: if configured, create an isolated environment.
	// Sandbox: all agents run sandboxed by default. Only GenSec can jailbreak.
	var sandboxHandle string
	var sandboxExecFn func(ctx context.Context, cmd []string) (string, string, error)
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
		defer sb.Destroy(ctx, handle) //nolint:errcheck // best-effort cleanup on defer
		sandboxHandle = string(handle)
		sandboxExecFn = func(ctx context.Context, cmd []string) (string, string, error) {
			result, err := sb.Exec(ctx, handle, cmd, 120)
			return result.Stdout, result.Stderr, err
		}
		log.Info("sandbox active", "backend", sandboxConf.Backend, "level", sandboxConf.Level, "handle", handle)
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

	if hubSocket != "" { //nolint:nestif // hub connection setup with retry and error handling
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

	// TUIRecorder: configured via debug.tap_file and debug.live_debug in djinn.yaml.
	var tuiRecorder *tui.TUIRecorder
	if debugConf.TapFile != "" || debugConf.LiveDebug != "" { //nolint:nestif // debug tap setup with file and live server
		var tapErr error
		tuiRecorder, tapErr = tui.NewTUIRecorder(100, debugConf.TapFile)
		if tapErr != nil {
			fmt.Fprintf(stderr, "djinn: debug tap: %v\n", tapErr)
		} else {
			defer tuiRecorder.Close() //nolint:errcheck // best-effort cleanup
			if debugConf.TapFile != "" {
				fmt.Fprintf(stderr, "djinn: debug tap writing to %s\n", debugConf.TapFile)
			}
			if debugConf.LiveDebug != "" {
				ln, srvErr := tuiRecorder.ServeHTTP(debugConf.LiveDebug)
				if srvErr != nil {
					fmt.Fprintf(stderr, "djinn: debug server: %v\n", srvErr)
				} else {
					fmt.Fprintf(stderr, "djinn: debug server at http://%s/debug/\n", ln.Addr())
				}
			}
		}
	}

	replErr := repl.Run(ctx, repl.Config{
		Driver:         chatDriver,
		Tools:          registry,
		Session:        sess,
		SystemPrompt:   assembledPrompt,
		MaxTurns:       sessConf.MaxTurns,
		AutoApprove:    sessConf.AutoApprove,
		Mode:           modeConf.Mode,
		Log:            logResult.Logger,
		Ring:           logResult.Ring,
		Store:          store,
		InitialPrompt:  initialPrompt,
		WorkspaceBus:   wsBus,
		Transport:      transport,
		Enforcer:       enforcer,
		Token:          capToken,
		HealthReports:  healthReports,
		Router:         slotRouter,
		Version:        Version,
		TUIRecorder:    tuiRecorder,
		TraceRing:      traceRing,
		SandboxHandle:  sandboxHandle,
		SandboxExec:    sandboxExecFn,
		SandboxBackend: sandboxConf.Backend,
		SandboxLevel:   sandboxConf.Level,
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

// CreateDriver is in drivers.go — extracted to reduce fan-out.
