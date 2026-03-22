// app_subcommands.go — CLI subcommands: ls, attach, kill, import, doctor, config, workspace, log, run.
package app

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/dpopsuev/djinn/ari"
	"github.com/dpopsuev/djinn/broker"
	djinnconfig "github.com/dpopsuev/djinn/config"
	djinnctx "github.com/dpopsuev/djinn/context"
	"github.com/dpopsuev/djinn/djinnfile"
	"github.com/dpopsuev/djinn/driver"
	"github.com/dpopsuev/djinn/gate"
	"github.com/dpopsuev/djinn/orchestrator"
	msbsandbox "github.com/dpopsuev/djinn/sandbox/misbah"
	"github.com/dpopsuev/djinn/session"
	sigsvc "github.com/dpopsuev/djinn/signal"
	"github.com/dpopsuev/djinn/testkit/stubs"
	"github.com/dpopsuev/djinn/tier"
	"github.com/dpopsuev/djinn/tools/builtin"
	djinnws "github.com/dpopsuev/djinn/workspace"
)

// Workspace role constants.
const (
	RolePrimary    = "primary"
	RoleDependency = "dependency"
)

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

// RunAttach resumes a session. No args = telescope picker with fuzzy search.
func RunAttach(args []string, stderr io.Writer) error {
	if len(args) >= 1 {
		return RunREPL(append([]string{"--session", args[0]}, args[1:]...), stderr)
	}

	// Telescope: list sessions, filter by optional query
	store, err := session.NewStore(SessionDir())
	if err != nil {
		return err
	}
	list, err := store.List()
	if err != nil {
		return err
	}
	if len(list) == 0 {
		fmt.Fprintln(stderr, "no sessions to attach (start one with: djinn repl -s <name>)")
		return nil
	}

	// Show all sessions as a numbered list
	fmt.Fprintln(stderr, "sessions:")
	for i, s := range list {
		name := s.Name
		if name == "" {
			name = s.ID
		}
		fmt.Fprintf(stderr, "  %d. %s (%s, %d turns, %s)\n",
			i+1, name, s.Model, s.Turns, s.UpdatedAt.Format("2006-01-02 15:04"))
	}
	fmt.Fprintf(stderr, "\nattach with: djinn attach <name>\n")
	return nil
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

	if out, err := exec.Command("curl", "-s", "http://localhost:11434/api/tags").Output(); err == nil && len(out) > 0 {
		fmt.Fprintln(w, "    ollama: running ✓")
	} else {
		fmt.Fprintln(w, "    ollama: not running")
	}

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

	fmt.Fprintln(w, "\n  sessions:")
	dir := SessionDir()
	if _, err := os.Stat(dir); err == nil {
		store, _ := session.NewStore(dir)
		list, _ := store.List()
		fmt.Fprintf(w, "    %d sessions in %s\n", len(list), dir)
	} else {
		fmt.Fprintf(w, "    dir not found (%s)\n", dir)
	}

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

// RunWorkspace handles the workspace subcommand.
func RunWorkspace(args []string, w io.Writer) error {
	if len(args) == 0 {
		return RunWorkspaceList(w)
	}
	switch args[0] {
	case "list", "ls":
		return RunWorkspaceList(w)
	case "create":
		if len(args) < 2 {
			return fmt.Errorf("usage: djinn workspace create <name> --repo <path> [--repo <path>...]")
		}
		name := args[1]
		ws := &djinnws.Workspace{Name: name}
		for i := 2; i < len(args); i++ {
			if args[i] == "--repo" && i+1 < len(args) {
				i++
				role := RolePrimary
				if len(ws.Repos) > 0 {
					role = RoleDependency
				}
				ws.Repos = append(ws.Repos, djinnws.Repo{Path: args[i], Role: role})
			}
		}
		if len(ws.Repos) == 0 {
			ws.Repos = []djinnws.Repo{{Path: Getwd(), Role: RolePrimary}}
		}
		if err := djinnws.Save(ws); err != nil {
			return err
		}
		fmt.Fprintf(w, "workspace %q created (%d repos)\n", name, len(ws.Repos))
		return nil
	default:
		return fmt.Errorf("unknown workspace command: %q (try: list, create)", args[0])
	}
}

// RunWorkspaceList lists saved workspaces.
func RunWorkspaceList(w io.Writer) error {
	list, err := djinnws.List()
	if err != nil {
		return err
	}
	if len(list) == 0 {
		fmt.Fprintln(w, "no workspaces (create one with: djinn workspace create <name> --repo <path>)")
		return nil
	}
	fmt.Fprintln(w, "NAME\tREPOS\tPRIMARY\tMODIFIED")
	for _, s := range list {
		fmt.Fprintf(w, "%s\t%d\t%s\t%s\n", s.Name, s.Repos, s.Primary, s.Modified)
	}
	return nil
}

// RunLog displays the log file contents.
func RunLog(w io.Writer) error {
	logPath := filepath.Join(HomeDir(), "djinn.log")
	data, err := os.ReadFile(logPath)
	if err != nil {
		return fmt.Errorf("cannot read log file at %s: %w", logPath, err)
	}
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
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
