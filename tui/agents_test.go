package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestAgentsPanel_Empty(t *testing.T) {
	p := NewAgentsPanel()
	if p.Count() != 0 {
		t.Fatalf("count = %d, want 0", p.Count())
	}
	view := p.View(80)
	if !strings.Contains(view, "no agents") {
		t.Fatal("empty panel should show 'no agents'")
	}
}

func TestAgentsPanel_AddAndCount(t *testing.T) {
	p := NewAgentsPanel()
	p.AddAgent("a1", "executor", lipgloss.Color("2"))
	p.AddAgent("a2", "gensec", lipgloss.Color("4"))

	if p.Count() != 2 {
		t.Fatalf("count = %d, want 2", p.Count())
	}
}

func TestAgentsPanel_RemoveAgent(t *testing.T) {
	p := NewAgentsPanel()
	p.AddAgent("a1", "executor", lipgloss.Color("2"))
	p.AddAgent("a2", "gensec", lipgloss.Color("4"))

	p.RemoveAgent("a1")
	if p.Count() != 1 {
		t.Fatalf("count = %d after remove, want 1", p.Count())
	}
	if p.GetAgent("a1") != nil {
		t.Fatal("removed agent should not be found")
	}
}

func TestAgentsPanel_Selected(t *testing.T) {
	p := NewAgentsPanel()
	p.AddAgent("a1", "executor", lipgloss.Color("2"))
	p.AddAgent("a2", "gensec", lipgloss.Color("4"))

	sel := p.Selected()
	if sel == nil || sel.AgentID != "a1" {
		t.Fatal("initial selection should be first agent")
	}
}

func TestAgentsPanel_UpdateAgent(t *testing.T) {
	p := NewAgentsPanel()
	p.AddAgent("a1", "executor", lipgloss.Color("2"))

	p.UpdateAgent(AgentStatusMsg{
		AgentID:  "a1",
		State:    "streaming",
		TokensIn: 100,
	})

	agent := p.GetAgent("a1")
	if agent.State != "streaming" {
		t.Fatalf("state = %q, want streaming", agent.State)
	}
	if agent.TokensIn != 100 {
		t.Fatalf("tokensIn = %d, want 100", agent.TokensIn)
	}
}

func TestAgentsPanel_View_ShowsCursor(t *testing.T) {
	p := NewAgentsPanel()
	p.AddAgent("a1", "executor", lipgloss.Color("2"))
	p.AddAgent("a2", "gensec", lipgloss.Color("4"))

	view := p.View(80)
	if !strings.Contains(view, ">") {
		t.Fatal("view should show cursor indicator")
	}
}

func TestAgentsPanel_Children_ReturnsDiveTarget(t *testing.T) {
	p := NewAgentsPanel()
	p.AddAgent("a1", "executor", lipgloss.Color("2"))

	children := p.Children()
	if len(children) != 1 {
		t.Fatalf("children = %d, want 1 (agent's output panel)", len(children))
	}
}

func TestAgentStatusPanel_View(t *testing.T) {
	a := NewAgentStatusPanel("a1", "executor", lipgloss.Color("2"))
	a.State = "streaming"
	a.TokensIn = 150
	a.TokensOut = 45

	view := a.View(80)
	if !strings.Contains(view, "executor") {
		t.Fatal("should show role name")
	}
	if !strings.Contains(view, "streaming") {
		t.Fatal("should show state")
	}
	if !strings.Contains(view, "150/45") {
		t.Fatal("should show token counts")
	}
}

func TestAgentStatusPanel_OutputPanel(t *testing.T) {
	a := NewAgentStatusPanel("a1", "executor", lipgloss.Color("2"))
	if a.Output() == nil {
		t.Fatal("should have an output panel")
	}

	// Feed output via message.
	a.Update(AgentOutputMsg{AgentID: "a1", Text: "hello world"})
	if a.Output().LineCount() != 1 {
		t.Fatalf("output lines = %d, want 1", a.Output().LineCount())
	}
}

func TestAgentStatusPanel_IgnoresOtherAgentMessages(t *testing.T) {
	a := NewAgentStatusPanel("a1", "executor", lipgloss.Color("2"))
	a.Update(AgentStatusMsg{AgentID: "a2", State: "error"})
	if a.State != "idle" {
		t.Fatal("should ignore messages for other agents")
	}
}
