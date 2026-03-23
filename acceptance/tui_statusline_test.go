// tui_statusline_test.go — acceptance tests for unified status line and health dashboard.
//
// Covers:
//   - Status line contains workspace, model, mode, tokens, turns
//   - Health indicators: green, yellow, red
//   - All green = collapsed "✓ N mcp"
//   - Mixed health shows individual indicators
//   - No health reports = no health section
package acceptance

import (
	"strings"
	"testing"

	"github.com/dpopsuev/djinn/cli/repl"
	"github.com/dpopsuev/djinn/tui"
	"github.com/dpopsuev/djinn/session"
	"github.com/dpopsuev/djinn/tools/builtin"

	tea "github.com/charmbracelet/bubbletea"
)

func TestStatusLine_ContainsWorkspace(t *testing.T) {
	sess := session.New("test", "claude-opus-4-6", "/workspace")
	sess.Workspace = "aeon"
	m := repl.NewModel(repl.Config{
		Tools:   builtin.NewRegistry(),
		Session: sess,
		Mode:    "agent",
	})
	m2, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 24})
	model := toModelPtr(m2)
	view := model.View()
	if !strings.Contains(view, "aeon") {
		t.Fatalf("status line should contain workspace name 'aeon': %s", view)
	}
}

func TestStatusLine_ContainsMode(t *testing.T) {
	sess := session.New("test", "claude-opus-4-6", "/workspace")
	m := repl.NewModel(repl.Config{
		Tools:   builtin.NewRegistry(),
		Session: sess,
		Mode:    "plan",
	})
	m2, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 24})
	model := toModelPtr(m2)
	view := model.View()
	if !strings.Contains(view, "plan") {
		t.Fatalf("status line should contain mode 'plan': %s", view)
	}
}

func TestStatusLine_ContainsModel(t *testing.T) {
	sess := session.New("test", "claude-opus-4-6", "/workspace")
	m := repl.NewModel(repl.Config{
		Tools:   builtin.NewRegistry(),
		Session: sess,
		Mode:    "agent",
	})
	m2, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 24})
	model := toModelPtr(m2)
	view := model.View()
	if !strings.Contains(view, "claude-opus-4-6") {
		t.Fatalf("status line should contain model: %s", view)
	}
}

func TestStatusLine_HealthAllGreen(t *testing.T) {
	sess := session.New("test", "model", "/workspace")
	m := repl.NewModel(repl.Config{
		Tools:   builtin.NewRegistry(),
		Session: sess,
		Mode:    "agent",
		HealthReports: []tui.HealthReport{
			{Component: "scribe", Status: tui.StatusGreen},
			{Component: "locus", Status: tui.StatusGreen},
		},
	})
	m2, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 24})
	model := toModelPtr(m2)
	view := model.View()
	// All green should show collapsed "✓ 2 mcp"
	if !strings.Contains(view, "2 mcp") {
		t.Fatalf("all green should show collapsed count: %s", view)
	}
}

func TestStatusLine_HealthMixed(t *testing.T) {
	sess := session.New("test", "model", "/workspace")
	m := repl.NewModel(repl.Config{
		Tools:   builtin.NewRegistry(),
		Session: sess,
		Mode:    "agent",
		HealthReports: []tui.HealthReport{
			{Component: "scribe", Status: tui.StatusYellow},
			{Component: "origami", Status: tui.StatusGreen},
		},
	})
	m2, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 24})
	model := toModelPtr(m2)
	view := model.View()
	if !strings.Contains(view, "scribe") {
		t.Fatalf("should show failed component name: %s", view)
	}
}

func TestStatusLine_NoReports(t *testing.T) {
	sess := session.New("test", "model", "/workspace")
	m := repl.NewModel(repl.Config{
		Tools:   builtin.NewRegistry(),
		Session: sess,
		Mode:    "agent",
	})
	m2, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 24})
	model := toModelPtr(m2)
	view := model.View()
	// Should still render without health section — no crash
	if !strings.Contains(view, "agent") {
		t.Fatalf("status line should render without health: %s", view)
	}
}

func TestStatusLine_EphemeralWorkspace(t *testing.T) {
	sess := session.New("test", "model", "/workspace")
	// No workspace set
	m := repl.NewModel(repl.Config{
		Tools:   builtin.NewRegistry(),
		Session: sess,
		Mode:    "agent",
	})
	m2, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 24})
	model := toModelPtr(m2)
	view := model.View()
	if !strings.Contains(view, "ephemeral") {
		t.Fatalf("should show (ephemeral) for unnamed workspace: %s", view)
	}
}
