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
	"github.com/dpopsuev/djinn/djinnfile"
	"github.com/dpopsuev/djinn/driver"
	"github.com/dpopsuev/djinn/gate"
	"github.com/dpopsuev/djinn/orchestrator"
	sigsvc "github.com/dpopsuev/djinn/signal"
	"github.com/dpopsuev/djinn/testkit/stubs"
)

const (
	defaultDjinnfile = "Djinnfile"
	exitCodeError    = 1
	pollInterval     = 50 * time.Millisecond
)

func main() {
	djinnfilePath := flag.String("f", defaultDjinnfile, "path to Djinnfile (JSON)")
	intentStr := flag.String("intent", "", "intent action (e.g. 'fix:rate limiting bug')")
	flag.Parse()

	if *intentStr == "" {
		fmt.Fprintf(os.Stderr, "usage: djinn -intent <action[:description]> [-f Djinnfile]\n")
		os.Exit(exitCodeError)
	}

	if err := run(*djinnfilePath, *intentStr); err != nil {
		fmt.Fprintf(os.Stderr, "djinn: %v\n", err)
		os.Exit(exitCodeError)
	}
}

func run(djinnfilePath, intentStr string) error {
	f, err := os.Open(djinnfilePath)
	if err != nil {
		return fmt.Errorf("open djinnfile: %w", err)
	}
	defer f.Close()

	df, err := djinnfile.Parse(f)
	if err != nil {
		return err
	}

	// Wire hexagon with stubs (real Misbah/Claude deferred to TSK-7)
	bus := sigsvc.NewSignalBus()
	cordons := broker.NewCordonRegistry()
	op := stubs.NewStubOperatorPort()

	orch := orchestrator.NewSimpleOrchestrator(
		stubs.NewStubSandbox().Create,
		stubs.NewStubSandbox().Destroy,
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

	// Wait for result, printing progress
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
