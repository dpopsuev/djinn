package mcp

import (
	"encoding/json"
	"testing"
)

func TestInitialize(t *testing.T) {
	srv := NewMockMCPServer()

	result, err := srv.Handle("initialize", 1, nil)
	if err != nil {
		t.Fatalf("initialize returned error: %v", err)
	}

	var resp map[string]any
	if err := json.Unmarshal(result, &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp["protocolVersion"] != "2024-11-05" {
		t.Errorf("expected protocolVersion 2024-11-05, got %v", resp["protocolVersion"])
	}
	info, ok := resp["serverInfo"].(map[string]any)
	if !ok {
		t.Fatalf("serverInfo not a map")
	}
	if info["name"] != "mock" {
		t.Errorf("expected server name mock, got %v", info["name"])
	}
}

func TestNotificationsInitialized(t *testing.T) {
	srv := NewMockMCPServer()

	result, err := srv.Handle("notifications/initialized", 0, nil)
	if err != nil {
		t.Fatalf("notifications/initialized returned error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil result for notification, got %s", string(result))
	}
}

func TestToolsList(t *testing.T) {
	srv := NewMockMCPServer()
	srv.RegisterTool("read_file", "Read a file", "file contents")
	srv.RegisterTool("write_file", "Write a file", "ok")

	result, err := srv.Handle("tools/list", 2, nil)
	if err != nil {
		t.Fatalf("tools/list returned error: %v", err)
	}

	var resp struct {
		Tools []ToolDef `json:"tools"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(resp.Tools) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(resp.Tools))
	}

	found := map[string]bool{}
	for _, tool := range resp.Tools {
		found[tool.Name] = true
	}
	if !found["read_file"] || !found["write_file"] {
		t.Errorf("expected read_file and write_file in tools list, got %v", found)
	}
}

func TestToolsCall(t *testing.T) {
	srv := NewMockMCPServer()
	srv.RegisterTool("greet", "Say hello", "hello world")

	params, _ := json.Marshal(map[string]any{
		"name":      "greet",
		"arguments": map[string]string{"who": "test"},
	})

	result, err := srv.Handle("tools/call", 3, params)
	if err != nil {
		t.Fatalf("tools/call returned error: %v", err)
	}

	var resp struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		IsError bool `json:"isError"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.IsError {
		t.Errorf("expected isError=false")
	}
	if len(resp.Content) != 1 || resp.Content[0].Text != "hello world" {
		t.Errorf("expected content text 'hello world', got %v", resp.Content)
	}
}

func TestToolsCallUnknown(t *testing.T) {
	srv := NewMockMCPServer()

	params, _ := json.Marshal(map[string]any{
		"name":      "nonexistent",
		"arguments": map[string]string{},
	})

	_, err := srv.Handle("tools/call", 4, params)
	if err == nil {
		t.Fatal("expected error for unknown tool")
	}
	if err.Error() != "unknown tool: nonexistent" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestCallsHistory(t *testing.T) {
	srv := NewMockMCPServer()
	srv.RegisterTool("tool_a", "Tool A", "result_a")
	srv.RegisterTool("tool_b", "Tool B", "result_b")

	paramsA, _ := json.Marshal(map[string]any{"name": "tool_a", "arguments": map[string]string{"x": "1"}})
	paramsB, _ := json.Marshal(map[string]any{"name": "tool_b", "arguments": map[string]string{"y": "2"}})

	srv.Handle("tools/call", 5, paramsA)
	srv.Handle("tools/call", 6, paramsB)

	calls := srv.Calls()
	if len(calls) != 2 {
		t.Fatalf("expected 2 calls, got %d", len(calls))
	}
	if calls[0].Name != "tool_a" {
		t.Errorf("expected first call to tool_a, got %s", calls[0].Name)
	}
	if calls[1].Name != "tool_b" {
		t.Errorf("expected second call to tool_b, got %s", calls[1].Name)
	}
}

func TestRegisterError(t *testing.T) {
	srv := NewMockMCPServer()
	srv.RegisterTool("flaky", "Flaky tool", "should not see this")
	srv.RegisterError("flaky", "something went wrong")

	params, _ := json.Marshal(map[string]any{
		"name":      "flaky",
		"arguments": map[string]string{},
	})

	result, err := srv.Handle("tools/call", 7, params)
	if err != nil {
		t.Fatalf("tools/call returned error: %v", err)
	}

	var resp struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		IsError bool `json:"isError"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if !resp.IsError {
		t.Errorf("expected isError=true")
	}
	if len(resp.Content) != 1 || resp.Content[0].Text != "something went wrong" {
		t.Errorf("expected error text 'something went wrong', got %v", resp.Content)
	}

	// Verify it was still recorded in call history
	calls := srv.Calls()
	if len(calls) != 1 || calls[0].Name != "flaky" {
		t.Errorf("expected flaky call recorded, got %v", calls)
	}
}

func TestUnknownMethod(t *testing.T) {
	srv := NewMockMCPServer()

	_, err := srv.Handle("resources/list", 8, nil)
	if err == nil {
		t.Fatal("expected error for unknown method")
	}
	if err.Error() != "unknown method: resources/list" {
		t.Errorf("unexpected error message: %v", err)
	}
}
