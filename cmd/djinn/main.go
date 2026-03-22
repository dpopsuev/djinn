package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/dpopsuev/djinn/ari"
	djinnctx "github.com/dpopsuev/djinn/context"
	"github.com/dpopsuev/djinn/broker"
	"github.com/dpopsuev/djinn/cli/repl"
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

const (
	exitCodeError     = 1
	defaultSessionDir = ".config/djinn/sessions"
	defaultModel      = "claude-sonnet-4-6"
	pollInterval      = 50 * time.Millisecond
)

// Driver names.
const (
	driverClaude = "claude"
	driverOllama = "ollama"
)

func main() {
	if len(os.Args) < 2 {
		// No subcommand — default to repl
		runREPLCmd(os.Args[1:])
		return
	}

	switch os.Args[1] {
	case "repl":
		runREPLCmd(os.Args[2:])
	case "run":
		runHeadlessCmd(os.Args[2:])
	case "ls":
		runListCmd()
	case "attach":
		runAttachCmd(os.Args[2:])
	case "kill":
		runKillCmd(os.Args[2:])
	case "import":
		runImportCmd(os.Args[2:])
	case "doctor":
		runDoctorCmd()
	case "version":
		fmt.Println("djinn 0.1.0")
	case "--help", "-h", "help":
		printUsage()
	default:
		// Treat as prompt for repl
		runREPLCmd(os.Args[1:])
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `djinn — model-agnostic agent runtime

Usage:
  djinn [prompt]                      interactive REPL (default)
  djinn repl [flags] [prompt]         interactive REPL
  djinn run <prompt> [flags]          headless one-shot
  djinn import claude <file> -s <name> import Claude Code session
  djinn ls                            list sessions
  djinn attach <name>                 resume session
  djinn kill <name>                   delete session
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
  --no-persist                        don't save session to disk
`)
}

func sessionDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, defaultSessionDir)
}

func runREPLCmd(args []string) {
	fs := flag.NewFlagSet("repl", flag.ExitOnError)
	driverName := fs.String("driver", driverClaude, "LLM backend: claude, ollama")
	model := fs.String("model", "", "model name (default depends on driver)")
	modelShort := fs.String("m", "", "model name (short)")
	sessionName := fs.String("session", "", "named session")
	sessionShort := fs.String("s", "", "named session (short)")
	cont := fs.Bool("continue", false, "resume most recent session")
	contShort := fs.Bool("c", false, "resume most recent (short)")
	maxTurns := fs.Int("max-turns", 20, "max agent turns per prompt")
	autoApprove := fs.Bool("auto-approve", false, "auto-approve all tool calls")
	systemPrompt := fs.String("system", "", "system prompt")
	systemFile := fs.String("system-file", "", "load system prompt from file")
	mode := fs.String("mode", "agent", "agent mode: ask, plan, agent, auto")
	verbose := fs.Bool("verbose", false, "verbose output")
	noPersist := fs.Bool("no-persist", false, "don't save session to disk")
	fs.Parse(args)

	// Load system prompt from file if specified
	if *systemFile != "" {
		fileContent := readSystemFile(*systemFile)
		if fileContent != "" {
			if *systemPrompt != "" {
				*systemPrompt = *systemPrompt + "\n\n" + fileContent
			} else {
				*systemPrompt = fileContent
			}
		}
	}

	_ = mode    // stub — wired in TSK-48
	_ = verbose // stub — wired when debug logging added

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

	// Resolve model default per driver
	if *model == "" {
		switch *driverName {
		case driverClaude:
			*model = defaultModel
		case driverOllama:
			*model = "qwen2.5-coder:14b"
		default:
			*model = defaultModel
		}
	}

	store, err := session.NewStore(sessionDir())
	if err != nil {
		fmt.Fprintf(os.Stderr, "djinn: %v\n", err)
		os.Exit(exitCodeError)
	}

	// Session: resume, continue, or new
	var sess *session.Session
	if *cont {
		sess, err = loadMostRecent(store)
		if err != nil {
			fmt.Fprintf(os.Stderr, "djinn: no session to continue: %v\n", err)
			os.Exit(exitCodeError)
		}
		fmt.Fprintf(os.Stderr, "djinn: resumed session %q (%d turns)\n", sess.Name, sess.History.Len())
	} else if *sessionName != "" {
		sess, err = store.Load(*sessionName)
		if err != nil {
			// New session with this name
			sess = session.New(*sessionName, *model, mustGetwd())
			sess.Name = *sessionName
			sess.Driver = *driverName
		} else {
			fmt.Fprintf(os.Stderr, "djinn: resumed session %q (%d turns)\n", sess.Name, sess.History.Len())
		}
	} else {
		id := fmt.Sprintf("djinn-%d", time.Now().Unix())
		sess = session.New(id, *model, mustGetwd())
		sess.Driver = *driverName
	}

	// Override model/driver if flags provided
	if *model != "" {
		sess.Model = *model
	}
	sess.Driver = *driverName

	// Auto-discover project context (CLAUDE.md, AGENTS.md, GEMINI.md, etc.)
	projectCtx := djinnctx.LoadProjectContext(mustGetwd())
	assembledPrompt := djinnctx.BuildSystemPrompt(projectCtx, *systemPrompt)

	// Create driver with assembled system prompt
	chatDriver, err := createDriver(*driverName, sess.Model, assembledPrompt)
	if err != nil {
		fmt.Fprintf(os.Stderr, "djinn: %v\n", err)
		os.Exit(exitCodeError)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	if err := chatDriver.Start(ctx, ""); err != nil {
		fmt.Fprintf(os.Stderr, "djinn: driver start: %v\n", err)
		os.Exit(exitCodeError)
	}
	defer chatDriver.Stop(ctx)

	// Replay history to driver if resuming
	for _, entry := range sess.Entries() {
		if entry.Role == driver.RoleUser {
			chatDriver.Send(ctx, driver.Message{Role: entry.Role, Content: entry.TextContent()})
		} else if entry.Role == driver.RoleAssistant {
			chatDriver.AppendAssistant(driver.RichMessage{
				Role:    entry.Role,
				Content: entry.TextContent(),
				Blocks:  entry.Blocks,
			})
		}
	}

	// Handle initial prompt from args
	prompt := strings.Join(fs.Args(), " ")

	replErr := repl.Run(ctx, repl.Config{
		Driver:       chatDriver,
		Tools:        builtin.NewRegistry(),
		Session:      sess,
		SystemPrompt: *systemPrompt,
		MaxTurns:     *maxTurns,
		AutoApprove:  *autoApprove,
	})

	_ = prompt // TODO: pass initial prompt to REPL

	// Save session on exit
	if !*noPersist {
		if saveErr := store.Save(sess); saveErr != nil {
			fmt.Fprintf(os.Stderr, "djinn: save session: %v\n", saveErr)
		}
	}

	if replErr != nil {
		fmt.Fprintf(os.Stderr, "djinn: %v\n", replErr)
		os.Exit(exitCodeError)
	}
}

func createDriver(driverName, model, systemPrompt string) (driver.ChatDriver, error) {
	switch driverName {
	case driverClaude:
		opts := []claudedriver.APIDriverOption{
			claudedriver.WithTools(builtin.NewRegistry()),
		}
		if systemPrompt != "" {
			opts = append(opts, claudedriver.WithAPISystemPrompt(systemPrompt))
		}
		return claudedriver.NewAPIDriver(
			driver.DriverConfig{Model: model},
			opts...,
		)
	case driverOllama:
		// TODO: implement Ollama driver (TSK-30)
		return nil, fmt.Errorf("ollama driver not yet implemented (use --driver claude)")
	default:
		return nil, fmt.Errorf("unknown driver: %q (supported: claude, ollama)", driverName)
	}
}

func runListCmd() {
	store, err := session.NewStore(sessionDir())
	if err != nil {
		fmt.Fprintf(os.Stderr, "djinn: %v\n", err)
		os.Exit(exitCodeError)
	}

	list, err := store.List()
	if err != nil {
		fmt.Fprintf(os.Stderr, "djinn: %v\n", err)
		os.Exit(exitCodeError)
	}

	if len(list) == 0 {
		fmt.Fprintln(os.Stderr, "no sessions")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tDRIVER\tMODEL\tTURNS\tUPDATED")
	for _, s := range list {
		name := s.Name
		if name == "" {
			name = s.ID
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\n",
			name, s.Driver, s.Model, s.Turns, s.UpdatedAt.Format(time.RFC3339))
	}
	w.Flush()
}

func runAttachCmd(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: djinn attach <name>")
		os.Exit(exitCodeError)
	}
	// Reuse repl with --session flag
	runREPLCmd(append([]string{"--session", args[0]}, args[1:]...))
}

func runKillCmd(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: djinn kill <name>")
		os.Exit(exitCodeError)
	}

	store, err := session.NewStore(sessionDir())
	if err != nil {
		fmt.Fprintf(os.Stderr, "djinn: %v\n", err)
		os.Exit(exitCodeError)
	}

	if err := store.Delete(args[0]); err != nil {
		fmt.Fprintf(os.Stderr, "djinn: %v\n", err)
		os.Exit(exitCodeError)
	}

	fmt.Fprintf(os.Stderr, "killed session: %s\n", args[0])
}

func runDoctorCmd() {
	fmt.Fprintln(os.Stderr, "djinn doctor")
	fmt.Fprintln(os.Stderr, "  version: 0.1.0")

	// Check drivers
	fmt.Fprintln(os.Stderr, "\n  drivers:")

	// Claude
	if os.Getenv("ANTHROPIC_API_KEY") != "" {
		fmt.Fprintln(os.Stderr, "    claude: ANTHROPIC_API_KEY set ✓")
	} else if os.Getenv("ANTHROPIC_VERTEX_PROJECT_ID") != "" {
		project := os.Getenv("ANTHROPIC_VERTEX_PROJECT_ID")
		// Probe gcloud auth
		if out, err := exec.Command("gcloud", "auth", "print-access-token").Output(); err == nil && len(out) > 0 {
			fmt.Fprintf(os.Stderr, "    claude: Vertex AI (%s) — gcloud auth ✓\n", project)
		} else {
			fmt.Fprintf(os.Stderr, "    claude: Vertex AI (%s) — gcloud auth FAILED (run: gcloud auth login)\n", project)
		}
	} else {
		fmt.Fprintln(os.Stderr, "    claude: NOT CONFIGURED")
	}

	// Ollama
	if out, err := exec.Command("curl", "-s", "http://localhost:11434/api/tags").Output(); err == nil && len(out) > 0 {
		fmt.Fprintln(os.Stderr, "    ollama: running ✓")
	} else {
		fmt.Fprintln(os.Stderr, "    ollama: not running")
	}

	// Project context
	fmt.Fprintln(os.Stderr, "\n  context:")
	projectCtx := djinnctx.LoadProjectContext(mustGetwd())
	if projectCtx.ClaudeMD != "" {
		fmt.Fprintln(os.Stderr, "    CLAUDE.md: found ✓")
	}
	if projectCtx.AgentsMD != "" {
		fmt.Fprintln(os.Stderr, "    AGENTS.md: found ✓")
	}
	if projectCtx.GeminiMD != "" {
		fmt.Fprintln(os.Stderr, "    GEMINI.md: found ✓")
	}
	if projectCtx.CursorRules != "" {
		fmt.Fprintln(os.Stderr, "    .cursorrules: found ✓")
	}
	if projectCtx.ClaudeMD == "" && projectCtx.AgentsMD == "" && projectCtx.GeminiMD == "" {
		fmt.Fprintln(os.Stderr, "    no project instruction files found")
	}

	// Sessions
	fmt.Fprintln(os.Stderr, "\n  sessions:")
	dir := sessionDir()
	if _, err := os.Stat(dir); err == nil {
		store, _ := session.NewStore(dir)
		list, _ := store.List()
		fmt.Fprintf(os.Stderr, "    %d sessions in %s\n", len(list), dir)
	} else {
		fmt.Fprintf(os.Stderr, "    dir not found (%s)\n", dir)
	}

	// Tools
	fmt.Fprintln(os.Stderr, "\n  tools: "+strings.Join(builtin.NewRegistry().Names(), ", "))
}

func runHeadlessCmd(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: djinn run <prompt> [--format json]")
		os.Exit(exitCodeError)
	}

	fs := flag.NewFlagSet("run", flag.ExitOnError)
	djinnfilePath := fs.String("f", "Djinnfile", "path to Djinnfile")
	misbahSocket := fs.String("misbah-socket", "", "Misbah daemon socket")
	workspace := fs.String("workspace", ".", "workspace root")
	fs.Parse(args[1:])

	prompt := args[0]

	f, err := os.Open(*djinnfilePath)
	if err != nil {
		// No Djinnfile — run simple headless with stubs
		fmt.Fprintf(os.Stderr, "djinn: no Djinnfile, using stubs\n")
		return
	}
	defer f.Close()

	df, err := djinnfile.Parse(f)
	if err != nil {
		fmt.Fprintf(os.Stderr, "djinn: %v\n", err)
		os.Exit(exitCodeError)
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

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	b.Start(ctx)

	op.SendIntent(ari.Intent{ID: df.Name + "-1", Action: prompt})

	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(pollInterval):
			if results := op.Results(); len(results) > 0 {
				for _, r := range results {
					if r.Success {
						fmt.Printf("OK: %s\n", r.Summary)
					} else {
						fmt.Printf("FAIL: %s\n", r.Summary)
					}
				}
				return
			}
		}
	}
}

func loadMostRecent(store *session.Store) (*session.Session, error) {
	list, err := store.List()
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, fmt.Errorf("no sessions found")
	}
	// List is sorted by most recent first
	return store.Load(list[0].Name)
}

func runImportCmd(args []string) {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: djinn import claude <session.jsonl> [-s name]")
		os.Exit(exitCodeError)
	}

	source := args[0]   // "claude"
	filePath := args[1] // path to JSONL

	fs := flag.NewFlagSet("import", flag.ExitOnError)
	name := fs.String("s", "", "session name")
	tokenBudget := fs.Int("token-budget", 0, "max tokens for imported history (0 = all)")
	fs.Parse(args[2:])

	switch source {
	case "claude":
		sess, err := session.ImportClaudeSession(filePath, *tokenBudget)
		if err != nil {
			fmt.Fprintf(os.Stderr, "djinn: import failed: %v\n", err)
			os.Exit(exitCodeError)
		}

		if *name != "" {
			sess.Name = *name
		}
		sess.Driver = driverClaude

		store, err := session.NewStore(sessionDir())
		if err != nil {
			fmt.Fprintf(os.Stderr, "djinn: %v\n", err)
			os.Exit(exitCodeError)
		}

		if err := store.Save(sess); err != nil {
			fmt.Fprintf(os.Stderr, "djinn: save: %v\n", err)
			os.Exit(exitCodeError)
		}

		displayName := sess.Name
		if displayName == "" {
			displayName = sess.ID
		}
		fmt.Fprintf(os.Stderr, "imported %d turns from Claude session → %s\n",
			sess.History.Len(), displayName)
		fmt.Fprintf(os.Stderr, "attach with: djinn attach %s\n", displayName)

	default:
		fmt.Fprintf(os.Stderr, "djinn: unsupported import source: %q (supported: claude)\n", source)
		os.Exit(exitCodeError)
	}
}

func readSystemFile(path string) string {
	if path == "" {
		return ""
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func mustGetwd() string {
	d, _ := os.Getwd()
	return d
}
