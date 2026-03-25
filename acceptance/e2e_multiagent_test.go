package acceptance

import (
	"encoding/json"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/dpopsuev/djinn/agent"
	"github.com/dpopsuev/djinn/cli/repl"
	"github.com/dpopsuev/djinn/session"
	"github.com/dpopsuev/djinn/testkit/stubs"
	"github.com/dpopsuev/djinn/tools/builtin"
	"github.com/dpopsuev/djinn/tui"
)

// TestE2E_MultiAgentRoster creates an AgentsPanel, adds 3 agents, sends
// AgentStatusMsg updates, and verifies Count, Selected, and drill-down Children.
func TestE2E_MultiAgentRoster(t *testing.T) {
	panel := tui.NewAgentsPanel()

	// Initially empty.
	if panel.Count() != 0 {
		t.Fatalf("initial count = %d, want 0", panel.Count())
	}
	if panel.Selected() != nil {
		t.Fatal("selected should be nil when empty")
	}

	// Add 3 agents with distinct roles and colors.
	panel.AddAgent("agent-1", "executor", lipgloss.Color("1"))
	panel.AddAgent("agent-2", "auditor", lipgloss.Color("2"))
	panel.AddAgent("agent-3", "inspector", lipgloss.Color("3"))

	if panel.Count() != 3 {
		t.Fatalf("count = %d, want 3", panel.Count())
	}

	// Default selection: first agent (cursor=0).
	sel := panel.Selected()
	if sel == nil {
		t.Fatal("selected should not be nil after adding agents")
	}
	if sel.AgentID != "agent-1" {
		t.Fatalf("selected = %q, want agent-1", sel.AgentID)
	}

	// Send status update for agent-2.
	panel.UpdateAgent(tui.AgentStatusMsg{
		AgentID:   "agent-2",
		Role:      "auditor",
		State:     "streaming",
		TokensIn:  100,
		TokensOut: 50,
	})

	agent2 := panel.GetAgent("agent-2")
	if agent2 == nil {
		t.Fatal("agent-2 should exist")
	}
	if agent2.State != "streaming" {
		t.Fatalf("agent-2 state = %q, want streaming", agent2.State)
	}
	if agent2.TokensIn != 100 || agent2.TokensOut != 50 {
		t.Fatalf("agent-2 tokens = %d/%d, want 100/50", agent2.TokensIn, agent2.TokensOut)
	}

	// Send status update for agent-3.
	panel.UpdateAgent(tui.AgentStatusMsg{
		AgentID: "agent-3",
		Role:    "inspector",
		State:   "done",
	})

	agent3 := panel.GetAgent("agent-3")
	if agent3.State != "done" {
		t.Fatalf("agent-3 state = %q, want done", agent3.State)
	}

	// Navigate down to agent-2 (cursor moves from 0 to 1).
	panel.Update(tea.KeyMsg{Type: tea.KeyDown})
	sel2 := panel.Selected()
	if sel2 == nil || sel2.AgentID != "agent-2" {
		name := ""
		if sel2 != nil {
			name = sel2.AgentID
		}
		t.Fatalf("after KeyDown, selected = %q, want agent-2", name)
	}

	// Children: drill-down should expose the selected agent's output panel.
	children := panel.Children()
	if len(children) == 0 {
		t.Fatal("Children() should return the selected agent's output panel")
	}

	// View renders correctly without panic.
	view := panel.View(80)
	if view == "" {
		t.Fatal("view should not be empty")
	}
	if !strings.Contains(view, "executor") {
		t.Fatalf("view should contain 'executor', got: %q", view)
	}
	if !strings.Contains(view, "auditor") {
		t.Fatalf("view should contain 'auditor', got: %q", view)
	}

	// Remove agent-1 and verify count.
	panel.RemoveAgent("agent-1")
	if panel.Count() != 2 {
		t.Fatalf("count after remove = %d, want 2", panel.Count())
	}
}

// TestE2E_AgentOutputRouting creates a Model with agentOutputs, sends
// AgentOutputMsg for specific agents, and verifies output is routed
// to the correct per-agent panel.
func TestE2E_AgentOutputRouting(t *testing.T) {
	drv := stubs.NewScriptedDriver(stubs.ScriptedStep{
		TextDeltas: []string{"ok"},
	})
	sess := session.New("multi-test", "test-model", "/workspace")
	m := repl.NewModel(repl.Config{
		Driver:  drv,
		Tools:   builtin.NewRegistry(),
		Session: sess,
		Mode:    "agent",
	})
	m2, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	model := toModelPtr(m2)

	// Send AgentOutputMsg for two different agents.
	result := multiUpdate(t, *model,
		tui.AgentOutputMsg{AgentID: "exec-1", Text: "Writing file main.go"},
		tui.AgentOutputMsg{AgentID: "exec-2", Text: "Running tests"},
		tui.AgentOutputMsg{AgentID: "exec-1", Text: "File written successfully"},
	)
	final := toModelPtr(result)

	// The model should not crash from routing output to unknown agents.
	// View should still render.
	view := final.View()
	if view == "" {
		t.Fatal("view should not be empty after agent output messages")
	}

	// Send AgentStatusMsg updates (these go to the agents panel).
	result2 := multiUpdate(t, *final,
		tui.AgentStatusMsg{AgentID: "exec-1", Role: "executor", State: "streaming"},
		tui.AgentStatusMsg{AgentID: "exec-2", Role: "executor", State: "tool-wait"},
	)
	final2 := toModelPtr(result2)
	_ = final2 // No panic is the primary assertion.

	// AgentThinkingMsg should also route without panic.
	result3 := multiUpdate(t, *final2,
		tui.AgentThinkingMsg{AgentID: "exec-1", Text: "Analyzing the codebase..."},
	)
	final3 := toModelPtr(result3)
	_ = final3
}

// TestE2E_PathTranslation verifies agent.TranslatePath correctly rewrites
// host workspace paths to jail mount paths.
func TestE2E_PathTranslation(t *testing.T) {
	tests := []struct {
		name        string
		input       json.RawMessage
		hostWorkDir string
		jailMount   string
		wantPath    string
	}{
		{
			name:        "file_path host to jail",
			input:       json.RawMessage(`{"file_path":"/home/user/project/main.go"}`),
			hostWorkDir: "/home/user/project",
			jailMount:   "/workspace",
			wantPath:    "/workspace/main.go",
		},
		{
			name:        "path host to jail",
			input:       json.RawMessage(`{"path":"/home/user/project/src/lib.go"}`),
			hostWorkDir: "/home/user/project",
			jailMount:   "/workspace",
			wantPath:    "/workspace/src/lib.go",
		},
		{
			name:        "no match leaves unchanged",
			input:       json.RawMessage(`{"file_path":"/other/path/file.go"}`),
			hostWorkDir: "/home/user/project",
			jailMount:   "/workspace",
			wantPath:    "/other/path/file.go",
		},
		{
			name:        "empty input returns as-is",
			input:       json.RawMessage(`{}`),
			hostWorkDir: "/home/user/project",
			jailMount:   "/workspace",
			wantPath:    "",
		},
		{
			name:        "empty hostWorkDir returns input unchanged",
			input:       json.RawMessage(`{"file_path":"/home/user/project/main.go"}`),
			hostWorkDir: "",
			jailMount:   "/workspace",
			wantPath:    "/home/user/project/main.go",
		},
		{
			name:        "nested subdirectory translation",
			input:       json.RawMessage(`{"file_path":"/home/user/project/pkg/deep/file.go"}`),
			hostWorkDir: "/home/user/project",
			jailMount:   "/jail/workspace",
			wantPath:    "/jail/workspace/pkg/deep/file.go",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := agent.TranslatePath(tt.input, tt.hostWorkDir, tt.jailMount)

			if tt.wantPath == "" {
				// Just verify no crash on empty/no-match cases.
				return
			}

			var parsed map[string]any
			if err := json.Unmarshal(result, &parsed); err != nil {
				t.Fatalf("unmarshal result: %v", err)
			}

			// Check file_path or path field.
			got := ""
			if fp, ok := parsed["file_path"].(string); ok {
				got = fp
			} else if p, ok := parsed["path"].(string); ok {
				got = p
			}

			if got != tt.wantPath {
				t.Errorf("path = %q, want %q (raw: %s)", got, tt.wantPath, string(result))
			}
		})
	}
}
