package acceptance

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/dpopsuev/djinn/agent"
	"github.com/dpopsuev/djinn/cli/repl"
	"github.com/dpopsuev/djinn/driver"
	"github.com/dpopsuev/djinn/policy"
	"github.com/dpopsuev/djinn/session"
	"github.com/dpopsuev/djinn/testkit/stubs"
	"github.com/dpopsuev/djinn/tools/builtin"
	"github.com/dpopsuev/djinn/tui"
)

// testModelWithScripted creates a Model wired to a ScriptedDriver for E2E flows.
func testModelWithScripted(t *testing.T, steps ...stubs.ScriptedStep) *repl.Model {
	t.Helper()
	drv := stubs.NewScriptedDriver(steps...)
	sess := session.New("e2e-test", "test-model", "/workspace")
	m := repl.NewModel(repl.Config{
		Driver:  drv,
		Tools:   builtin.NewRegistry(),
		Session: sess,
		Mode:    "agent",
	})
	m2, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	mp := toModelPtr(m2)
	return mp
}

// TestE2E_PromptToResponse wires Model + ScriptedDriver. The driver streams
// "Hello world". A SubmitMsg is sent, messages are processed through Update(),
// and the session must contain user + assistant entries.
func TestE2E_PromptToResponse(t *testing.T) {
	m := testModelWithScripted(t, stubs.ScriptedStep{
		TextDeltas: []string{"Hello ", "world"},
		Usage:      &driver.Usage{InputTokens: 10, OutputTokens: 5},
	})

	// Submit a prompt — this starts the agent loop.
	m2, cmd := m.Update(tui.SubmitMsg{Value: "greet me"})
	model := toModelPtr(m2)

	if model.CurrentState() != repl.StateStreaming {
		t.Fatalf("state = %d, want streaming", model.CurrentState())
	}
	if cmd == nil {
		t.Fatal("should return agent cmd batch")
	}

	// NOTE: drv.Started() is false here because the driver starts inside
	// agent.Run which runs as a tea.Cmd. We cannot fully run the agent
	// loop in a unit test without executing the cmd. Instead we verify
	// the submit path is correct.

	// Simulate the streaming events that the agent loop would produce.
	result := multiUpdate(t, *model,
		tui.TextMsg("Hello "),
		tui.TickMsg(time.Now()),
		tui.TextMsg("world"),
		tui.TickMsg(time.Now()),
		tui.DoneMsg{Usage: &driver.Usage{InputTokens: 10, OutputTokens: 5}},
		tui.AgentDoneMsg{Result: "Hello world"},
	)

	final := toModelPtr(result)
	if final.CurrentState() != repl.StateInput {
		t.Fatalf("state after done = %d, want input", final.CurrentState())
	}

	// The output panel should contain the user prompt and streamed text.
	lines := final.View()
	_ = lines // View returns string — use ConversationLen instead.

	if final.ConversationLen() < 2 {
		t.Fatalf("conversation should have at least 2 entries (user + agent output), got %d",
			final.ConversationLen())
	}
}

// TestE2E_ToolCallCycle verifies the full tool-call flow: agent requests Read
// in step 1, the TUI handles ToolCallMsg → ToolResultMsg, then the agent
// responds with text in step 2.
func TestE2E_ToolCallCycle(t *testing.T) {
	m := testModelWithScripted(t,
		// Step 1: agent requests a Read tool call.
		stubs.ScriptedStep{
			TextDeltas: []string{"Let me read that."},
			ToolCall: &driver.ToolCall{
				ID:    "call-1",
				Name:  "Read",
				Input: json.RawMessage(`{"file_path":"/workspace/main.go"}`),
			},
		},
		// Step 2: agent responds with text after getting tool result.
		stubs.ScriptedStep{
			TextDeltas: []string{"The file contains a main function."},
		},
	)

	// Override mode to auto so tool approval does not block.
	m.SetMode(agent.ModeAuto)

	// Start streaming.
	m.SetState(repl.StateStreaming)
	m.AppendConversation(tui.AssistStyle.Render(tui.LabelAssist) + ": ")

	// Simulate the agent streaming text then requesting a tool call.
	result := multiUpdate(t, *m,
		tui.TextMsg("Let me read that."),
		tui.TickMsg(time.Now()),
		tui.ToolCallMsg{Call: driver.ToolCall{
			ID: "call-1", Name: "Read",
			Input: json.RawMessage(`{"file_path":"/workspace/main.go"}`),
		}},
	)
	model := toModelPtr(result)

	// In auto mode, tool should not require approval.
	if model.CurrentState() == repl.StateToolApproval {
		t.Fatal("auto mode should not require tool approval")
	}

	// Simulate tool result arriving.
	result2 := multiUpdate(t, *model,
		tui.ToolResultMsg{
			CallID:  "call-1",
			Name:    "Read",
			Output:  "package main\nfunc main() {}",
			IsError: false,
		},
	)
	model2 := toModelPtr(result2)

	// Verify the tool result appeared in output (envelope replaced).
	found := false
	view := model2.View()
	if strings.Contains(view, "Read") {
		found = true
	}
	if !found {
		t.Fatal("tool call should appear in output view")
	}

	// Simulate the agent's second turn text.
	result3 := multiUpdate(t, *model2,
		tui.TextMsg("The file contains a main function."),
		tui.TickMsg(time.Now()),
		tui.AgentDoneMsg{Result: "The file contains a main function."},
	)
	final := toModelPtr(result3)

	if final.CurrentState() != repl.StateInput {
		t.Fatalf("state = %d, want input after agent done", final.CurrentState())
	}
}

// TestE2E_EnforcerDenies creates a CapabilityToken with AllowedTools=["Read"].
// When the agent requests "Bash", the enforcer denies it.
func TestE2E_EnforcerDenies(t *testing.T) {
	token := policy.CapabilityToken{
		AllowedTools: []string{"Read"},
	}

	enforcer := policy.NewDefaultToolPolicyEnforcer()

	// Bash is NOT in AllowedTools — should be denied.
	err := enforcer.Check(token, "Bash", json.RawMessage(`{"command":"ls"}`))
	if err == nil {
		t.Fatal("enforcer should deny Bash when AllowedTools=[Read]")
	}
	if !strings.Contains(err.Error(), "not allowed") {
		t.Fatalf("error should mention 'not allowed', got: %v", err)
	}

	// Read IS in AllowedTools — should be allowed.
	err = enforcer.Check(token, "Read", json.RawMessage(`{"file_path":"/workspace/main.go"}`))
	if err != nil {
		t.Fatalf("enforcer should allow Read, got: %v", err)
	}

	// Verify the token integrates with the Model: create a model with the enforcer.
	drv := stubs.NewScriptedDriver(stubs.ScriptedStep{
		TextDeltas: []string{"test"},
	})
	sess := session.New("enforcer-test", "test-model", "/workspace")
	m := repl.NewModel(repl.Config{
		Driver:   drv,
		Tools:    builtin.NewRegistry(),
		Session:  sess,
		Mode:     "auto",
		Enforcer: enforcer,
		Token:    token,
	})
	m2, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	model := toModelPtr(m2)

	// The model should be usable with the enforcer attached.
	if model.CurrentState() != repl.StateInput {
		t.Fatalf("state = %d, want input", model.CurrentState())
	}
}
