package testkit

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/dpopsuev/djinn/agent"
	"github.com/dpopsuev/djinn/cortex"
	"github.com/dpopsuev/djinn/driver"
	mcpclient "github.com/dpopsuev/djinn/mcp/client"
	"github.com/dpopsuev/djinn/policy"
	"github.com/dpopsuev/djinn/testkit/mcp"
	"github.com/dpopsuev/djinn/testkit/stubs"
	"github.com/dpopsuev/djinn/tools/builtin"
)

// TestSkeleton_AgentCallsMCPTool proves the full pipeline:
// ScriptedChatDriver → agent.Run → CompositeExecutor → MCP client → toy server → result back.
func TestSkeleton_AgentCallsMCPTool(t *testing.T) {
	// 1. Create toy MCP server with a canned tool
	mock := mcp.NewMockMCPServer()
	mock.RegisterTool("artifact", "list Scribe artifacts", `[{"id":"TSK-1","title":"fix bug"}]`)

	srv := httptest.NewServer(mock.HTTPHandler())
	defer srv.Close()

	// 2. Connect MCP client to toy server
	mcpClient := mcpclient.New(nil)
	if err := mcpClient.ConnectHTTP(context.Background(), "scribe", srv.URL); err != nil {
		t.Fatalf("ConnectHTTP: %v", err)
	}
	defer mcpClient.Close()

	// 3. Create CompositeExecutor (MCP + builtin)
	registry := builtin.NewRegistry()
	composite := builtin.NewCompositeExecutor(registry, mcpClient, nil)

	// 4. ScriptedChatDriver scripts calling the MCP tool
	drv := stubs.NewScriptedChatDriver(
		stubs.ScriptedTurn{
			Text: "Let me list the tasks",
			ToolCalls: []driver.ToolCall{{
				ID:    "call-1",
				Name:  "mcp__scribe__artifact",
				Input: json.RawMessage(`{"action":"list"}`),
			}},
		},
		stubs.ScriptedTurn{Text: "found TSK-1"},
	)

	// 5. Run agent with CompositeExecutor
	sess := cortex.New("mcp-test", "test-model", t.TempDir())

	result, err := agent.Run(context.Background(), agent.Config{
		Driver:       drv,
		Tools:        composite,
		Session:      sess,
		MaxTurns:     5,
		ToolsEnabled: true,
		Approve:      agent.AutoApprove,
		Enforcer:     policy.NopToolPolicyEnforcer{},
	}, "list my tasks")

	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result != "found TSK-1" {
		t.Fatalf("result = %q, want 'found TSK-1'", result)
	}

	// 6. Verify the MCP server received the call
	calls := mock.Calls()
	if len(calls) != 1 {
		t.Fatalf("MCP calls = %d, want 1", len(calls))
	}
	if calls[0].Name != "artifact" {
		t.Fatalf("tool called = %q, want 'artifact'", calls[0].Name)
	}

	// 7. Verify tool result was sent back to driver
	results := drv.ToolResults()
	if len(results) != 1 {
		t.Fatalf("tool results = %d, want 1", len(results))
	}
	if results[0].IsError {
		t.Fatalf("tool result is error: %s", results[0].Output)
	}

	t.Log("MCP skeleton PASSES — agent called toy MCP server and got results")
}

// TestSkeleton_AgentCallsBuiltinAndMCP proves CompositeExecutor routes
// builtin tools AND MCP tools in the same session.
func TestSkeleton_AgentCallsBuiltinAndMCP(t *testing.T) {
	// Toy MCP server
	mock := mcp.NewMockMCPServer()
	mock.RegisterTool("scan", "scan architecture", `{"packages":37}`)

	srv := httptest.NewServer(mock.HTTPHandler())
	defer srv.Close()

	mcpClient := mcpclient.New(nil)
	mcpClient.ConnectHTTP(context.Background(), "locus", srv.URL) //nolint:errcheck // test
	defer mcpClient.Close()

	registry := builtin.NewRegistry()
	composite := builtin.NewCompositeExecutor(registry, mcpClient, nil)

	// Script: turn 1 calls builtin Write, turn 2 calls MCP scan, turn 3 done
	ws := t.TempDir()
	drv := stubs.NewScriptedChatDriver(
		stubs.ScriptedTurn{ToolCalls: []driver.ToolCall{{
			ID: "c1", Name: "Write",
			Input: stubs.MustJSON(map[string]string{"path": ws + "/test.go", "content": "package main"}),
		}}},
		stubs.ScriptedTurn{ToolCalls: []driver.ToolCall{{
			ID: "c2", Name: "mcp__locus__scan",
			Input: json.RawMessage(`{}`),
		}}},
		stubs.ScriptedTurn{Text: "all done"},
	)

	sess := cortex.New("mixed-test", "test-model", ws)

	result, err := agent.Run(context.Background(), agent.Config{
		Driver:       drv,
		Tools:        composite,
		Session:      sess,
		MaxTurns:     5,
		ToolsEnabled: true,
		Approve:      agent.AutoApprove,
		Enforcer:     policy.NopToolPolicyEnforcer{},
	}, "write code and scan")

	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result != "all done" {
		t.Fatalf("result = %q", result)
	}

	// Builtin Write should have written the file
	results := drv.ToolResults()
	if len(results) != 2 {
		t.Fatalf("tool results = %d, want 2", len(results))
	}
	if results[0].IsError {
		t.Fatalf("Write failed: %s", results[0].Output)
	}

	// MCP scan should have been called
	if len(mock.Calls()) != 1 {
		t.Fatalf("MCP calls = %d, want 1", len(mock.Calls()))
	}

	t.Log("Mixed skeleton PASSES — builtin + MCP tools in same session")
}
