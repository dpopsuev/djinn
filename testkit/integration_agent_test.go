package testkit

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/dpopsuev/djinn/driver"
	"github.com/dpopsuev/djinn/testkit/stubs"
)

// TestIntegration_AgentWritesFiles proves the full pipeline:
// ScriptedChatDriver → agent.Run() → real Write tool → files on disk.
// No LLM, no network — scripted tool calls with real filesystem.
func TestIntegration_AgentWritesFiles(t *testing.T) {
	ws := NewTestWorkspace(t)

	mainGoPath := filepath.Join(ws.Dir(), "main.go")
	goModPath := filepath.Join(ws.Dir(), "go.mod")

	mainGoContent := "package main\n\nfunc main() {}\n"
	goModContent := "module testproject\n\ngo 1.22\n"

	// Script: turn 1 writes main.go, turn 2 writes go.mod, turn 3 says "done"
	drv := stubs.NewScriptedChatDriver(
		stubs.ScriptedTurn{
			Text: "I'll create main.go",
			ToolCalls: []driver.ToolCall{{
				ID:   "call-1",
				Name: "Write",
				Input: stubs.MustJSON(map[string]string{
					"path":    mainGoPath,
					"content": mainGoContent,
				}),
			}},
		},
		stubs.ScriptedTurn{
			Text: "Now go.mod",
			ToolCalls: []driver.ToolCall{{
				ID:   "call-2",
				Name: "Write",
				Input: stubs.MustJSON(map[string]string{
					"path":    goModPath,
					"content": goModContent,
				}),
			}},
		},
		stubs.ScriptedTurn{Text: "done"},
	)

	f := NewAgentFixture(t,
		WithDriver(drv),
		WithMaxTurns(5),
	)

	result, err := f.Run(context.Background(), "create a Go project")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result != "done" {
		t.Fatalf("result = %q, want %q", result, "done")
	}

	// Verify files exist with correct content
	gotMain, err := os.ReadFile(mainGoPath)
	if err != nil {
		t.Fatalf("main.go not written: %v", err)
	}
	if string(gotMain) != mainGoContent {
		t.Fatalf("main.go content = %q, want %q", string(gotMain), mainGoContent)
	}

	gotMod, err := os.ReadFile(goModPath)
	if err != nil {
		t.Fatalf("go.mod not written: %v", err)
	}
	if string(gotMod) != goModContent {
		t.Fatalf("go.mod content = %q, want %q", string(gotMod), goModContent)
	}

	// Verify driver received tool results
	results := drv.ToolResults()
	if len(results) != 2 {
		t.Fatalf("tool results = %d, want 2", len(results))
	}
	if results[0].IsError {
		t.Fatalf("tool result 0 is error: %s", results[0].Output)
	}
	if results[1].IsError {
		t.Fatalf("tool result 1 is error: %s", results[1].Output)
	}

	// Verify 3 turns consumed (2 tool calls + 1 final text)
	if drv.TurnCount() != 3 {
		t.Fatalf("turns = %d, want 3", drv.TurnCount())
	}
}

// TestIntegration_AgentReadsThenWrites proves the read→write cycle:
// agent reads an existing file, then writes a modified version.
func TestIntegration_AgentReadsThenWrites(t *testing.T) {
	ws := NewTestWorkspace(t)

	// Seed a file in the workspace
	origPath := filepath.Join(ws.Dir(), "hello.txt")
	os.WriteFile(origPath, []byte("hello world"), 0o644)

	outPath := filepath.Join(ws.Dir(), "output.txt")

	drv := stubs.NewScriptedChatDriver(
		// Turn 1: read the file
		stubs.ScriptedTurn{
			ToolCalls: []driver.ToolCall{{
				ID:    "call-1",
				Name:  "Read",
				Input: stubs.MustJSON(map[string]string{"path": origPath}),
			}},
		},
		// Turn 2: write modified content
		stubs.ScriptedTurn{
			ToolCalls: []driver.ToolCall{{
				ID:   "call-2",
				Name: "Write",
				Input: stubs.MustJSON(map[string]string{
					"path":    outPath,
					"content": "MODIFIED: hello world",
				}),
			}},
		},
		// Turn 3: done
		stubs.ScriptedTurn{Text: "done"},
	)

	f := NewAgentFixture(t, WithDriver(drv), WithMaxTurns(5))

	result, err := f.Run(context.Background(), "read and modify")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result != "done" {
		t.Fatalf("result = %q", result)
	}

	// Read tool result should contain the file content
	results := drv.ToolResults()
	if len(results) != 2 {
		t.Fatalf("tool results = %d, want 2", len(results))
	}
	if results[0].IsError {
		t.Fatalf("Read failed: %s", results[0].Output)
	}

	// Verify output file was written
	got, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("output.txt not written: %v", err)
	}
	if string(got) != "MODIFIED: hello world" {
		t.Fatalf("output = %q", string(got))
	}
}

// TestIntegration_ToolError_AgentContinues proves the agent continues
// after a tool returns an error (e.g., reading a nonexistent file).
func TestIntegration_ToolError_AgentContinues(t *testing.T) {
	ws := NewTestWorkspace(t)

	drv := stubs.NewScriptedChatDriver(
		// Turn 1: read a file that doesn't exist
		stubs.ScriptedTurn{
			ToolCalls: []driver.ToolCall{{
				ID:    "call-1",
				Name:  "Read",
				Input: stubs.MustJSON(map[string]string{"path": filepath.Join(ws.Dir(), "nonexistent.go")}),
			}},
		},
		// Turn 2: agent recovers
		stubs.ScriptedTurn{Text: "file not found, skipping"},
	)

	f := NewAgentFixture(t, WithDriver(drv), WithMaxTurns(5))

	result, err := f.Run(context.Background(), "read a missing file")
	if err != nil {
		t.Fatalf("Run should not error on tool failure: %v", err)
	}
	if result != "file not found, skipping" {
		t.Fatalf("result = %q", result)
	}

	// Tool result should be an error
	results := drv.ToolResults()
	if len(results) != 1 {
		t.Fatalf("results = %d", len(results))
	}
	if !results[0].IsError {
		t.Fatal("expected error result for nonexistent file")
	}
}

// ensure json is used
var _ = json.Marshal
