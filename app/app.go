// Package app contains all Djinn CLI domain logic.
// The cmd/djinn binary is a thin shell that calls this package.
// Per hexagonal-architecture rule: all domain logic lives in library modules.
package app

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/dpopsuev/djinn/ari"
	"github.com/dpopsuev/djinn/broker"
	"github.com/dpopsuev/djinn/cli/repl"
	djinnconfig "github.com/dpopsuev/djinn/config"
	djinnctx "github.com/dpopsuev/djinn/context"
	"github.com/dpopsuev/djinn/djinnlog"
	mcpclient "github.com/dpopsuev/djinn/mcp/client"
	"github.com/dpopsuev/djinn/djinnfile"
	"github.com/dpopsuev/djinn/driver"
	claudedriver "github.com/dpopsuev/djinn/driver/claude"
	"github.com/dpopsuev/djinn/gate"
	"github.com/dpopsuev/djinn/orchestrator"
	"github.com/dpopsuev/djinn/session"
	msbsandbox "github.com/dpopsuev/djinn/sandbox/misbah"
	sigsvc "github.com/dpopsuev/djinn/signal"
	"github.com/dpopsuev/djinn/testkit/stubs"
	"github.com/dpopsuev/djinn/tier"
	"github.com/dpopsuev/djinn/tools/builtin"
)

// Constants.
const (
	Version           = "0.1.0"
	DefaultHomeDir    = ".djinn"
	DefaultSessionDir = ".djinn/sessions"
	DefaultModel      = "claude-sonnet-4-6"
	PollInterval      = 50 * time.Millisecond
)

// Driver names.
const (
	DriverClaude = "claude"
	DriverOllama = "ollama"
)

// HomeDir returns the Djinn home directory.
// Checks ~/.djinn first, falls back to ~/.config/djinn, defaults to ~/.djinn for new installs.
func HomeDir() string {
	home, _ := os.UserHomeDir()
	djinnDir := filepath.Join(home, ".djinn")
	if _, err := os.Stat(djinnDir); err == nil {
		return djinnDir
	}
	configDir := filepath.Join(home, ".config", "djinn")
	if _, err := os.Stat(configDir); err == nil {
		return configDir
	}
	return djinnDir // default for new installs
}

// SessionDir returns the default session directory.
func SessionDir() string {
	return filepath.Join(HomeDir(), "sessions")
}

// Run is the main entry point. Routes subcommands.
func Run(args []string, stderr io.Writer) error {
	if len(args) < 1 {
		return RunREPL(nil, stderr)
	}

	switch args[0] {
	case "repl":
		return RunREPL(args[1:], stderr)
	case "run":
		return RunHeadless(args[1:], stderr)
	case "ls":
		return RunList(stderr)
	case "attach":
		return RunAttach(args[1:], stderr)
	case "kill":
		return RunKill(args[1:], stderr)
	case "import":
		return RunImport(args[1:], stderr)
	case "config":
		return RunConfig(args[1:], stderr)
	case "log":
		return RunLog(stderr)
	case "doctor":
		return RunDoctor(stderr)
	case "version":
		fmt.Fprintln(stderr, "djinn "+Version)
		return nil
	case "--help", "-h", "help":
		PrintUsage(stderr)
		return nil
	default:
		// Treat as prompt for repl
		return RunREPL(args, stderr)
	}
}

// PrintUsage writes usage information.
func PrintUsage(w io.Writer) {
	fmt.Fprintf(w, `djinn — model-agnostic agent runtime

Usage:
  djinn [prompt]                      interactive REPL (default)
  djinn repl [flags] [prompt]         interactive REPL
  djinn run <prompt> [flags]          headless one-shot
  djinn import claude <file> -s <name> import Claude Code session
  djinn ls                            list sessions
  djinn attach <name>                 resume session
  djinn kill <name>                   delete session
  djinn config dump                   dump runtime config as YAML
  djinn log                           show recent log entries
  djinn doctor                        health check
  djinn version                       version info

Flags (repl/run):
  --driver <claude|ollama>            LLM backend (default: claude)
  -m, --model <model>                 model name
  -s, --session <name>                named session
  -c, --continue                      resume most recent session
  --max-turns <n>                     max agent turns (default: 20)
  --auto-approve                      auto-approve all tool calls
  --system <prompt>                   system prompt
  --system-file <path>                load system prompt from file
  --mode <ask|plan|agent|auto>        agent mode (default: agent)
  --config <file>                     load config from YAML file
  --workspace <path>                   project workspace (default: cwd)
  --add-dir <path>                    additional workspace directory
  --verbose                           show log output on terminal
  --no-persist                        don't save session to disk
`)
}

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
	workspace := fs.String("workspace", "", "project workspace directory (default: cwd)")
	addDir := fs.String("add-dir", "", "additional workspace directory")
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

	// Load config files (defaults → discovered files → explicit → CLI flags override)
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

	// Apply config file values back to flags (file values as defaults, CLI flags win)
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

	// Initialize workspace dirs
	// Priority: session stored > --workspace flag > cwd
	if len(sess.WorkDirs) == 0 {
		if *workspace != "" {
			sess.WorkDirs = []string{*workspace}
		} else {
			sess.WorkDirs = []string{Getwd()}
		}
	}
	if *addDir != "" {
		sess.WorkDirs = append(sess.WorkDirs, *addDir)
	}

	// Setup logging
	logResult := djinnlog.Setup(djinnlog.Options{
		Verbose: *verbose,
		LogFile: filepath.Join(HomeDir(), "djinn.log"),
	})
	log := djinnlog.For(logResult.Logger, "app")
	log.Info("session starting", "driver", *driverName, "model", *model, "mode", *mode)

	// Auto-discover project context from workspace dirs (upward walk + MEMORY.md)
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

	// Replay history to driver if resuming
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

	// Connect to MCP servers
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

	// Build unified tool registry: built-in + MCP tools
	registry := builtin.NewRegistry()
	for _, tool := range mcpClient.MCPTools() {
		registry.Register(tool)
	}
	log.Info("tools registered", "builtin", 6, "mcp", len(mcpClient.MCPTools()), "total", len(registry.Names()))

	replErr := repl.Run(ctx, repl.Config{
		Driver:       chatDriver,
		Tools:        registry,
		Session:      sess,
		SystemPrompt: assembledPrompt,
		MaxTurns:     *maxTurns,
		AutoApprove:  *autoApprove,
		Mode:         *mode,
		Log:          logResult.Logger,
		Ring:         logResult.Ring,
	})

	// Save session on exit
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

// RunList lists all sessions.
func RunList(w io.Writer) error {
	store, err := session.NewStore(SessionDir())
	if err != nil {
		return err
	}

	list, err := store.List()
	if err != nil {
		return err
	}

	if len(list) == 0 {
		fmt.Fprintln(w, "no sessions")
		return nil
	}

	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "NAME\tDRIVER\tMODEL\tTURNS\tUPDATED")
	for _, s := range list {
		name := s.Name
		if name == "" {
			name = s.ID
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%d\t%s\n",
			name, s.Driver, s.Model, s.Turns, s.UpdatedAt.Format(time.RFC3339))
	}
	tw.Flush()
	return nil
}

// RunAttach resumes a session.
func RunAttach(args []string, stderr io.Writer) error {
	if len(args) < 1 {
		return fmt.Errorf("attach requires a session name (usage: djinn attach <name>)")
	}
	return RunREPL(append([]string{"--session", args[0]}, args[1:]...), stderr)
}

// RunKill deletes a session.
func RunKill(args []string, stderr io.Writer) error {
	if len(args) < 1 {
		return fmt.Errorf("kill requires a session name (usage: djinn kill <name>)")
	}

	store, err := session.NewStore(SessionDir())
	if err != nil {
		return err
	}

	if err := store.Delete(args[0]); err != nil {
		return err
	}

	fmt.Fprintf(stderr, "killed session: %s\n", args[0])
	return nil
}

// RunImport imports a session from another CLI tool.
func RunImport(args []string, stderr io.Writer) error {
	if len(args) < 2 {
		return fmt.Errorf("import requires source and file (usage: djinn import claude <session.jsonl> [-s name])")
	}

	source := args[0]
	filePath := args[1]

	fs := flag.NewFlagSet("import", flag.ContinueOnError)
	fs.SetOutput(stderr)
	name := fs.String("s", "", "session name")
	tokenBudget := fs.Int("token-budget", 0, "max tokens for imported history")
	if err := fs.Parse(args[2:]); err != nil {
		return err
	}

	switch source {
	case "claude":
		sess, err := session.ImportClaudeSession(filePath, *tokenBudget)
		if err != nil {
			return fmt.Errorf("import: %w", err)
		}

		if *name != "" {
			sess.Name = *name
		}
		sess.Driver = DriverClaude

		store, err := session.NewStore(SessionDir())
		if err != nil {
			return err
		}

		if err := store.Save(sess); err != nil {
			return fmt.Errorf("save: %w", err)
		}

		displayName := sess.Name
		if displayName == "" {
			displayName = sess.ID
		}
		fmt.Fprintf(stderr, "imported %d turns from Claude session → %s\n",
			sess.History.Len(), displayName)
		fmt.Fprintf(stderr, "attach with: djinn attach %s\n", displayName)
		return nil

	default:
		return fmt.Errorf("%w: %q (supported: claude)", ErrUnknownImport, source)
	}
}

// RunDoctor checks system health.
func RunDoctor(w io.Writer) error {
	fmt.Fprintln(w, "djinn doctor")
	fmt.Fprintln(w, "  version: "+Version)

	fmt.Fprintln(w, "\n  drivers:")

	// Claude
	if os.Getenv("ANTHROPIC_API_KEY") != "" {
		fmt.Fprintln(w, "    claude: ANTHROPIC_API_KEY set ✓")
	} else if project := os.Getenv("ANTHROPIC_VERTEX_PROJECT_ID"); project != "" {
		if out, err := exec.Command("gcloud", "auth", "print-access-token").Output(); err == nil && len(out) > 0 {
			fmt.Fprintf(w, "    claude: Vertex AI (%s) — gcloud auth ✓\n", project)
		} else {
			fmt.Fprintf(w, "    claude: Vertex AI (%s) — gcloud auth FAILED (run: gcloud auth login)\n", project)
		}
	} else {
		fmt.Fprintln(w, "    claude: NOT CONFIGURED")
	}

	// Ollama
	if out, err := exec.Command("curl", "-s", "http://localhost:11434/api/tags").Output(); err == nil && len(out) > 0 {
		fmt.Fprintln(w, "    ollama: running ✓")
	} else {
		fmt.Fprintln(w, "    ollama: not running")
	}

	// Project context
	fmt.Fprintln(w, "\n  context:")
	projectCtx := djinnctx.LoadProjectContext(Getwd())
	found := false
	if projectCtx.ClaudeMD != "" {
		fmt.Fprintln(w, "    CLAUDE.md: found ✓")
		found = true
	}
	if projectCtx.AgentsMD != "" {
		fmt.Fprintln(w, "    AGENTS.md: found ✓")
		found = true
	}
	if projectCtx.GeminiMD != "" {
		fmt.Fprintln(w, "    GEMINI.md: found ✓")
		found = true
	}
	if projectCtx.CursorRules != "" {
		fmt.Fprintln(w, "    .cursorrules: found ✓")
		found = true
	}
	if !found {
		fmt.Fprintln(w, "    no project instruction files found")
	}

	// Sessions
	fmt.Fprintln(w, "\n  sessions:")
	dir := SessionDir()
	if _, err := os.Stat(dir); err == nil {
		store, _ := session.NewStore(dir)
		list, _ := store.List()
		fmt.Fprintf(w, "    %d sessions in %s\n", len(list), dir)
	} else {
		fmt.Fprintf(w, "    dir not found (%s)\n", dir)
	}

	// Tools
	fmt.Fprintln(w, "\n  tools: "+strings.Join(builtin.NewRegistry().Names(), ", "))
	return nil
}

// RunConfig handles the config subcommand.
func RunConfig(args []string, w io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: djinn config dump")
	}

	switch args[0] {
	case "dump":
		r := djinnconfig.NewRegistry()
		r.Register(&djinnconfig.ModeConfig{Mode: "agent"})
		r.Register(&djinnconfig.DriverConfigurable{Name: DriverClaude, Model: DefaultModel})
		r.Register(&djinnconfig.SessionConfigurable{MaxTurns: 20})
		r.Register(&djinnconfig.ToolsConfigurable{Enabled: builtin.NewRegistry().Names()})

		// Load discovered config files to show actual values
		if err := djinnconfig.LoadAll(r, Getwd(), ""); err != nil {
			fmt.Fprintf(w, "# warning: %v\n", err)
		}

		data, err := r.DumpYAML()
		if err != nil {
			return err
		}
		_, err = w.Write(data)
		return err

	default:
		return fmt.Errorf("unknown config command: %q (try: djinn config dump)", args[0])
	}
}

// RunLog displays the log file contents.
func RunLog(w io.Writer) error {
	logPath := filepath.Join(HomeDir(), "djinn.log")
	data, err := os.ReadFile(logPath)
	if err != nil {
		return fmt.Errorf("cannot read log file at %s: %w", logPath, err)
	}
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	// Show last 50 lines
	if len(lines) > 50 {
		lines = lines[len(lines)-50:]
	}
	for _, line := range lines {
		fmt.Fprintln(w, line)
	}
	return nil
}

// RunHeadless runs a one-shot headless execution.
func RunHeadless(args []string, stderr io.Writer) error {
	if len(args) < 1 {
		return fmt.Errorf("run requires a prompt (usage: djinn run <prompt>)")
	}

	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	fs.SetOutput(stderr)
	djinnfilePath := fs.String("f", "Djinnfile", "path to Djinnfile")
	misbahSocket := fs.String("misbah-socket", "", "Misbah daemon socket")
	workspace := fs.String("workspace", ".", "workspace root")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}

	prompt := args[0]

	f, err := os.Open(*djinnfilePath)
	if err != nil {
		fmt.Fprintf(stderr, "djinn: no Djinnfile, using stubs\n")
		return nil
	}
	defer f.Close()

	df, err := djinnfile.Parse(f)
	if err != nil {
		return err
	}

	bus := sigsvc.NewSignalBus()
	cordons := broker.NewCordonRegistry()
	op := stubs.NewStubOperatorPort()

	var createSandbox func(ctx context.Context, scope tier.Scope) (string, error)
	var destroySandbox func(ctx context.Context, id string) error

	if *misbahSocket != "" {
		sandbox := msbsandbox.New(*misbahSocket, *workspace)
		defer sandbox.Close()
		createSandbox = sandbox.Create
		destroySandbox = sandbox.Destroy
	} else {
		stubSandbox := stubs.NewStubSandbox()
		createSandbox = stubSandbox.Create
		destroySandbox = stubSandbox.Destroy
	}

	orch := orchestrator.NewSimpleOrchestrator(
		createSandbox, destroySandbox,
		func(cfg driver.DriverConfig) driver.Driver {
			return stubs.NewStubDriver(driver.Message{Role: driver.RoleAssistant, Content: "completed"})
		},
		func(cfg gate.GateConfig) gate.Gate { return stubs.AlwaysPassGate() },
		func(s sigsvc.Signal) { bus.Emit(s) },
	)

	b := broker.NewBroker(broker.BrokerConfig{
		Orchestrator: orch, Bus: bus, Cordons: cordons, Operator: op,
		PlanFactory: func(intent ari.Intent) orchestrator.WorkPlan { return df.ToWorkPlan(intent.ID) },
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	b.Start(ctx)

	op.SendIntent(ari.Intent{ID: df.Name + "-1", Action: prompt})

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(PollInterval):
			if results := op.Results(); len(results) > 0 {
				for _, r := range results {
					if r.Success {
						fmt.Fprintf(stderr, "OK: %s\n", r.Summary)
					} else {
						fmt.Fprintf(stderr, "FAIL: %s\n", r.Summary)
					}
				}
				return nil
			}
		}
	}
}

// LoadMostRecent loads the most recently updated session.
func LoadMostRecent(store *session.Store) (*session.Session, error) {
	list, err := store.List()
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, ErrNoSessions
	}
	name := list[0].Name
	if name == "" {
		name = list[0].ID
	}
	return store.Load(name)
}

// ReadSystemFile reads a system prompt from a file. Returns empty on error.
func ReadSystemFile(path string) string {
	if path == "" {
		return ""
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// Getwd returns the current working directory.
func Getwd() string {
	d, _ := os.Getwd()
	return d
}
