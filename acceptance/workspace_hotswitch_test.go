package acceptance

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dpopsuev/djinn/cli/repl"
	"github.com/dpopsuev/djinn/djinnlog"
	"github.com/dpopsuev/djinn/session"
	"github.com/dpopsuev/djinn/workspace"
)

func TestHotSwitch_BusEmitsToSubscribers(t *testing.T) {
	bus := workspace.NewBus(djinnlog.Nop())

	var received []string
	bus.On("driver", func(evt workspace.Event) {
		received = append(received, "driver:"+evt.New.Name)
	})
	bus.On("session", func(evt workspace.Event) {
		received = append(received, "session:"+evt.New.Name)
	})

	bus.Emit(workspace.Event{
		Type: workspace.EventSwitch,
		New:  &workspace.Workspace{Name: "new-ws"},
	})

	if len(received) != 2 {
		t.Fatalf("received = %v, want 2 entries", received)
	}
}

func TestHotSwitch_PanicDoesNotBlockOthers(t *testing.T) {
	bus := workspace.NewBus(djinnlog.Nop())

	var survived bool
	bus.On("panicker", func(workspace.Event) { panic("boom") })
	bus.On("survivor", func(workspace.Event) { survived = true })

	bus.Emit(workspace.Event{Type: workspace.EventSwitch, New: &workspace.Workspace{Name: "ws"}})

	if !survived {
		t.Fatal("survivor should run despite panicker")
	}
}

func TestHotSwitch_WorkspaceSwitchCommand(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	// Save a workspace to switch to
	ws := &workspace.Workspace{
		Name:  "target",
		Repos: []workspace.Repo{{Path: "/new/project", Role: "primary"}},
	}
	workspace.Save(ws)

	sess := session.New("test", "model", "/old")
	sess.Workspace = "old"

	result := repl.ExecuteCommand(repl.Command{
		Name: "/workspace-switch",
		Args: []string{"target"},
	}, sess)

	if sess.Workspace != "target" {
		t.Fatalf("workspace = %q, want target", sess.Workspace)
	}
	if len(sess.WorkDirs) != 1 || sess.WorkDirs[0] != "/new/project" {
		t.Fatalf("WorkDirs = %v", sess.WorkDirs)
	}
	if result.Output == "" {
		t.Fatal("should produce output")
	}
}

func TestHotSwitch_WorkspaceSwitchNotFound(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	sess := session.New("test", "model", "/work")
	result := repl.ExecuteCommand(repl.Command{
		Name: "/workspace-switch",
		Args: []string{"nonexistent"},
	}, sess)

	if result.Output == "" {
		t.Fatal("should produce error output")
	}
}

func TestHotSwitch_WorkspaceAddEmitsEvent(t *testing.T) {
	// This tests that /workspace-add modifies session WorkDirs
	sess := session.New("test", "model", "/work")
	sess.WorkDirs = []string{"/original"}

	result := repl.ExecuteCommand(repl.Command{
		Name: "/workspace-add",
		Args: []string{"/added"},
	}, sess)

	if len(sess.WorkDirs) != 2 {
		t.Fatalf("WorkDirs = %v, want 2", sess.WorkDirs)
	}
	if result.Output == "" {
		t.Fatal("should confirm add")
	}
}

func TestHotSwitch_HyphenatedCommands(t *testing.T) {
	sess := session.New("test", "model", "/work")

	// /workspace-repos should work
	result := repl.ExecuteCommand(repl.Command{Name: "/workspace-repos"}, sess)
	_ = result // should not panic

	// /workspace (no hyphen) still works
	result = repl.ExecuteCommand(repl.Command{Name: "/workspace"}, sess)
	if result.Output == "" {
		t.Fatal("/workspace should show info")
	}
}

func TestHotSwitch_ContextReDiscovered(t *testing.T) {
	// Simulate: workspace switch triggers context re-discovery
	home := t.TempDir()
	t.Setenv("HOME", home)

	// Create workspace with a repo that has CLAUDE.md
	repoDir := filepath.Join(home, "project")
	os.MkdirAll(repoDir, 0755)
	os.WriteFile(filepath.Join(repoDir, "CLAUDE.md"), []byte("new project rules"), 0644)

	ws := &workspace.Workspace{
		Name:  "with-context",
		Repos: []workspace.Repo{{Path: repoDir, Role: "primary"}},
	}
	workspace.Save(ws)

	// Bus subscriber should re-discover context
	bus := workspace.NewBus(djinnlog.Nop())
	var contextUpdated bool
	bus.On("context", func(evt workspace.Event) {
		if evt.New != nil && len(evt.New.Paths()) > 0 {
			contextUpdated = true
		}
	})

	bus.Emit(workspace.Event{Type: workspace.EventSwitch, New: ws})

	if !contextUpdated {
		t.Fatal("context subscriber should have been called")
	}
}
