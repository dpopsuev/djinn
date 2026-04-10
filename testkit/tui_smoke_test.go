package testkit

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/dpopsuev/djinn/agent"
	"github.com/dpopsuev/djinn/cortex"
	"github.com/dpopsuev/djinn/driver"
	"github.com/dpopsuev/djinn/policy"
	"github.com/dpopsuev/djinn/terminal"
	"github.com/dpopsuev/djinn/testkit/stubs"
	"github.com/dpopsuev/djinn/tools/builtin"
)

// TestTUISmoke_SubmitPrompt_SeeResponse proves the functional flow:
// operator submits prompt → Terminal → agent.Run → tool call → response.
// No real TUI, no real LLM — TestOperator + ScriptedChatDriver.
func TestTUISmoke_SubmitPrompt_SeeResponse(t *testing.T) {
	// 1. Create Terminal
	op := stubs.NewTestOperator()
	djinn := op.Djinn()

	// 2. Wire agent into Terminal's OnSubmit
	drv := stubs.NewScriptedChatDriver(
		stubs.ScriptedTurn{
			Text: "I'll write the file",
			ToolCalls: []driver.ToolCall{{
				ID: "c1", Name: "Write",
				Input: stubs.MustJSON(map[string]string{
					"path":    t.TempDir() + "/test.go",
					"content": "package main",
				}),
			}},
		},
		stubs.ScriptedTurn{Text: "done writing"},
	)

	djinn.OnSubmit = func(ctx context.Context, prompt string) error {
		sess := cortex.New("smoke", "test-model", t.TempDir())
		reg := builtin.NewRegistry()

		result, err := agent.Run(ctx, agent.Config{
			Driver:       drv,
			Tools:        reg,
			Session:      sess,
			MaxTurns:     5,
			ToolsEnabled: true,
			Approve:      agent.AutoApprove,
			Enforcer:     policy.NopToolPolicyEnforcer{},
			Handler: &terminalEventHandler{
				djinn: djinn,
			},
		}, prompt)
		if err != nil {
			return err
		}

		// Emit final response as view event
		djinn.Emit(terminal.ViewEvent{
			Kind: terminal.EventDone,
			Text: result,
		})
		return nil
	}

	// 3. Submit prompt
	ctx := context.Background()
	if err := op.Submit(ctx, "write test.go"); err != nil {
		t.Fatalf("Submit: %v", err)
	}

	// 4. Wait for response
	ev, ok := op.WaitForEvent(terminal.EventDone, 5*time.Second)
	if !ok {
		t.Fatal("timeout waiting for agent response")
	}
	if ev.Text != "done writing" {
		t.Fatalf("response = %q, want 'done writing'", ev.Text)
	}

	t.Log("TUI smoke PASSES — submit prompt → agent works → response received")
}

// terminalEventHandler bridges agent events to Terminal view events.
type terminalEventHandler struct {
	djinn *terminal.Djinn
}

func (h *terminalEventHandler) OnText(text string) {
	h.djinn.Emit(terminal.ViewEvent{Kind: terminal.EventOutput, Text: text})
}
func (h *terminalEventHandler) OnThinking(text string) {}
func (h *terminalEventHandler) OnToolCall(call driver.ToolCall) {
	h.djinn.Emit(terminal.ViewEvent{Kind: terminal.EventToolCall, Text: call.Name})
}
func (h *terminalEventHandler) OnToolResult(id, name, output string, isError bool) {
	h.djinn.Emit(terminal.ViewEvent{Kind: terminal.EventToolResult, Text: name + ": " + output})
}
func (h *terminalEventHandler) OnDone(u *driver.Usage) {}
func (h *terminalEventHandler) OnError(err error)      {}

// ensure json import is used
var _ = json.Marshal
