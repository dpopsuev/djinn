package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/dpopsuev/djinn/ari"
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
	defaultDjinnfile = "Djinnfile"
	exitCodeError    = 1
	pollInterval     = 50 * time.Millisecond
)

func main() {
	// Subcommand: djinn repl
	if len(os.Args) > 1 && os.Args[1] == "repl" {
		replCmd := flag.NewFlagSet("repl", flag.ExitOnError)
		model := replCmd.String("model", "claude-sonnet-4-6", "model to use")
		systemPrompt := replCmd.String("system", "", "system prompt to append")
		maxTurns := replCmd.Int("max-turns", 20, "max agent turns per prompt")
		autoApprove := replCmd.Bool("auto-approve", false, "auto-approve all tool calls")
		replCmd.Parse(os.Args[2:])

		if err := runREPL(*model, *systemPrompt, *maxTurns, *autoApprove); err != nil {
			fmt.Fprintf(os.Stderr, "djinn: %v\n", err)
			os.Exit(exitCodeError)
		}
		return
	}

	// Legacy headless mode
	djinnfilePath := flag.String("f", defaultDjinnfile, "path to Djinnfile (JSON)")
	intentStr := flag.String("intent", "", "intent action (e.g. 'fix:rate limiting bug')")
	misbahSocket := flag.String("misbah-socket", "", "Misbah daemon socket path (empty = use stubs)")
	workspace := flag.String("workspace", ".", "workspace root to mount in containers")
	flag.Parse()

	if *intentStr == "" {
		fmt.Fprintf(os.Stderr, "usage:\n")
		fmt.Fprintf(os.Stderr, "  djinn repl [-model claude-sonnet-4-6]   interactive mode\n")
		fmt.Fprintf(os.Stderr, "  djinn -intent <action> [-f Djinnfile]   headless mode\n")
		os.Exit(exitCodeError)
	}

	if err := runHeadless(*djinnfilePath, *intentStr, *misbahSocket, *workspace); err != nil {
		fmt.Fprintf(os.Stderr, "djinn: %v\n", err)
		os.Exit(exitCodeError)
	}
}

func runREPL(model, systemPrompt string, maxTurns int, autoApprove bool) error {
	apiDriver, err := claudedriver.NewAPIDriver(
		driver.DriverConfig{Model: model},
		claudedriver.WithTools(builtin.NewRegistry()),
		claudedriver.WithAPISystemPrompt(systemPrompt),
	)
	if err != nil {
		return err
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	if err := apiDriver.Start(ctx, ""); err != nil {
		return err
	}
	defer apiDriver.Stop(ctx)

	sess := session.New("repl-1", model, ".")
	tools := builtin.NewRegistry()

	return repl.Run(ctx, repl.Config{
		Driver:       apiDriver,
		Tools:        tools,
		Session:      sess,
		SystemPrompt: systemPrompt,
		MaxTurns:     maxTurns,
		AutoApprove:  autoApprove,
	})
}

func runHeadless(djinnfilePath, intentStr, misbahSocket, workspace string) error {
	f, err := os.Open(djinnfilePath)
	if err != nil {
		return fmt.Errorf("open djinnfile: %w", err)
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

	if misbahSocket != "" {
		sandbox := msbsandbox.New(misbahSocket, workspace)
		defer sandbox.Close()
		createSandbox = sandbox.Create
		destroySandbox = sandbox.Destroy
		fmt.Fprintf(os.Stderr, "djinn: using Misbah daemon at %s\n", misbahSocket)
	} else {
		stubSandbox := stubs.NewStubSandbox()
		createSandbox = stubSandbox.Create
		destroySandbox = stubSandbox.Destroy
		fmt.Fprintf(os.Stderr, "djinn: using stub sandbox (no Misbah daemon)\n")
	}

	orch := orchestrator.NewSimpleOrchestrator(
		createSandbox,
		destroySandbox,
		func(cfg driver.DriverConfig) driver.Driver {
			return stubs.NewStubDriver(driver.Message{
				Role:    driver.RoleAssistant,
				Content: "completed",
			})
		},
		func(cfg gate.GateConfig) gate.Gate {
			return stubs.AlwaysPassGate()
		},
		func(s sigsvc.Signal) { bus.Emit(s) },
	)

	b := broker.NewBroker(broker.BrokerConfig{
		Orchestrator: orch,
		Bus:          bus,
		Cordons:      cordons,
		Operator:     op,
		PlanFactory: func(intent ari.Intent) orchestrator.WorkPlan {
			return df.ToWorkPlan(intent.ID)
		},
	})

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	b.Start(ctx)

	action, payload := parseIntent(intentStr)
	intentID := df.Name + "-1"

	fmt.Fprintf(os.Stderr, "djinn: project=%s stages=%d intent=%s\n", df.Name, len(df.Stages), action)

	op.SendIntent(ari.Intent{
		ID:      intentID,
		Action:  action,
		Payload: payload,
	})

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(pollInterval):
			results := op.Results()
			if len(results) > 0 {
				printResults(op, bus)
				return nil
			}
		}
	}
}

func printResults(op *stubs.StubOperatorPort, bus *sigsvc.SignalBus) {
	for _, e := range op.Events() {
		fmt.Fprintf(os.Stderr, "  [%s] %s %s\n", e.Kind, e.Stage, e.Message)
	}

	for _, r := range op.Results() {
		if r.Success {
			fmt.Printf("OK: %s\n", r.Summary)
		} else {
			fmt.Printf("FAIL: %s\n", r.Summary)
		}
	}

	health := bus.Health()
	worst := sigsvc.Green
	for _, h := range health {
		if h.Level > worst {
			worst = h.Level
		}
	}
	fmt.Fprintf(os.Stderr, "  andon: %s (%d workstreams)\n", worst, len(health))
}

func parseIntent(s string) (string, map[string]string) {
	parts := strings.SplitN(s, ":", 2)
	action := parts[0]
	payload := make(map[string]string)
	if len(parts) > 1 {
		payload["description"] = parts[1]
	}
	return action, payload
}
