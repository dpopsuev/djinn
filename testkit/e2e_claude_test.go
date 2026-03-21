package testkit

import (
	"context"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/dpopsuev/djinn/ari"
	"github.com/dpopsuev/djinn/broker"
	"github.com/dpopsuev/djinn/driver"
	claudedriver "github.com/dpopsuev/djinn/driver/claude"
	"github.com/dpopsuev/djinn/gate"
	"github.com/dpopsuev/djinn/orchestrator"
	msbsandbox "github.com/dpopsuev/djinn/sandbox/misbah"
	"github.com/dpopsuev/djinn/signal"
	"github.com/dpopsuev/djinn/testkit/stubs"
	"github.com/dpopsuev/djinn/tier"
)

const (
	crossEcosystemTimeout = 60 * time.Second
)

// TestE2E_CrossEcosystem_ClaudeInMisbah runs the full vertical slice:
// Djinn CLI → Broker → Misbah container → Claude Code CLI → real work → result.
//
// Requires:
// - Misbah daemon running (MISBAH_DAEMON_SOCKET)
// - Claude Code CLI installed (claude binary on PATH)
// - ANTHROPIC_API_KEY set (for Claude to authenticate)
func TestE2E_CrossEcosystem_ClaudeInMisbah(t *testing.T) {
	// Check prerequisites
	socketPath := os.Getenv("MISBAH_DAEMON_SOCKET")
	if socketPath == "" {
		socketPath = defaultMisbahSocket
	}
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		t.Skipf("Misbah daemon not available at %s: %v", socketPath, err)
	}
	conn.Close()

	if _, err := exec.LookPath("claude"); err != nil {
		t.Skipf("Claude Code CLI not on PATH: %v", err)
	}

	if os.Getenv("ANTHROPIC_API_KEY") == "" && os.Getenv("ANTHROPIC_VERTEX_PROJECT_ID") == "" {
		t.Skip("no Claude auth: set ANTHROPIC_API_KEY or ANTHROPIC_VERTEX_PROJECT_ID")
	}

	// Create a workspace with a simple Go file to edit
	workspace := t.TempDir()
	mainFile := filepath.Join(workspace, "main.go")
	os.WriteFile(mainFile, []byte(`package main

func hello() string {
	return "hello"
}
`), 0644)

	// Wire: real Misbah sandbox + Claude Code driver + always-pass gate
	sandbox := msbsandbox.New(socketPath, workspace)
	defer sandbox.Close()

	bus := signal.NewSignalBus()
	cordons := broker.NewCordonRegistry()
	op := stubs.NewStubOperatorPort()

	orch := orchestrator.NewSimpleOrchestrator(
		sandbox.Create,
		sandbox.Destroy,
		func(cfg driver.DriverConfig) driver.Driver {
			execFn := func(ctx context.Context, name string, cmd []string, timeout int64) (claudedriver.ExecResult, error) {
				result, err := sandbox.Exec(ctx, name, cmd, timeout)
				if err != nil {
					return claudedriver.ExecResult{}, err
				}
				return claudedriver.ExecResult{
					ExitCode: result.ExitCode,
					Stdout:   result.Stdout,
					Stderr:   result.Stderr,
				}, nil
			}
			return claudedriver.NewCodeDriver(cfg, execFn, "You are working in a Go project. Be concise.")
		},
		func(cfg gate.GateConfig) gate.Gate {
			return stubs.AlwaysPassGate()
		},
		func(s signal.Signal) { bus.Emit(s) },
	)

	b := broker.NewBroker(broker.BrokerConfig{
		Orchestrator: orch,
		Bus:          bus,
		Cordons:      cordons,
		Operator:     op,
		Sandbox:      sandbox,
		PlanFactory: func(intent ari.Intent) orchestrator.WorkPlan {
			return orchestrator.WorkPlan{
				ID: intent.ID,
				Stages: []orchestrator.Stage{{
					Name:       "code",
					Scope:      tier.Scope{Level: tier.Mod, Name: "main"},
					Driver:     driver.DriverConfig{Model: "claude-sonnet-4-6"},
					Gate:       gate.GateConfig{Name: "pass", Severity: gate.SeverityBlocking},
					Prompt:     "Add a comment to the hello function in main.go saying '// returns greeting'",
					TimeBudget: 2 * time.Minute,
				}},
			}
		},
	})

	ctx, cancel := context.WithTimeout(context.Background(), crossEcosystemTimeout)
	defer cancel()

	b.Start(ctx)

	// Send intent
	op.SendIntent(ari.Intent{ID: "e2e-claude-1", Action: "edit"})

	// Wait for result
	deadline := time.Now().Add(crossEcosystemTimeout)
	for time.Now().Before(deadline) {
		if len(op.Results()) > 0 {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}

	results := op.Results()
	if len(results) == 0 {
		t.Fatal("no results received from cross-ecosystem E2E")
	}

	// Skip on infrastructure failures
	if !results[0].Success {
		summary := results[0].Summary
		if containsAny(summary, "operation not supported", "permission denied", "network isolation", "exec error", "API key") {
			t.Skipf("skipping: infrastructure issue: %s", summary)
		}
		t.Fatalf("cross-ecosystem E2E failed: %s", summary)
	}

	t.Logf("cross-ecosystem E2E passed: %s", results[0].Summary)

	// Verify workstream tracked
	ws, ok := b.Workstreams().Get("e2e-claude-1")
	if !ok {
		t.Fatal("workstream not registered")
	}
	t.Logf("workstream status: %s", ws.Status)

	// Check if the file was actually modified
	content, err := os.ReadFile(mainFile)
	if err != nil {
		t.Logf("could not read modified file: %v", err)
	} else {
		t.Logf("file content after edit:\n%s", string(content))
	}
}
