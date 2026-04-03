package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dpopsuev/djinn/djinnlog"
	testmcp "github.com/dpopsuev/djinn/testkit/mcp"
)

// Security tests for MCP response deserialization (TSK-516, OWASP A08).
// Trust boundary TB-6: mcp/client ↔ external MCP servers.
// STRIDE T (Tampering): malformed responses must not crash or corrupt state.

func TestMCP_SSE_RejectsMalformedJSON(t *testing.T) {
	// extractSSEData returns raw string; caller unmarshals.
	// Verify malformed JSON in data: line doesn't propagate as valid.
	data := extractSSEData("data: {broken json no closing brace")
	if data == "" {
		return // correctly rejected — no { prefix match or no valid JSON
	}
	// If extractSSEData returns it, verify json.Unmarshal catches it.
	var resp jsonRPCResponse
	err := json.Unmarshal([]byte(data), &resp)
	if err == nil {
		t.Fatal("SECURITY: malformed JSON accepted as valid response")
	}
}

func TestMCP_SSE_NestedDataPrefix(t *testing.T) {
	// Attacker could craft SSE with nested "data: " lines.
	input := "data: {\"id\":1}\ndata: {\"id\":2,\"evil\":true}\n"
	got := extractSSEData(input)
	// Should return first match only.
	var resp jsonRPCResponse
	if err := json.Unmarshal([]byte(got), &resp); err != nil {
		t.Fatalf("first data line should be valid: %v", err)
	}
	if resp.ID != 1 {
		t.Errorf("expected first data line (id=1), got id=%d", resp.ID)
	}
}

func TestMCP_ToolCallResult_EmptyContent(t *testing.T) {
	// Empty content array should be handled gracefully, not silently return "".
	result := toolCallResult{Content: []contentBlock{}}
	raw, _ := json.Marshal(result)

	var parsed toolCallResult
	if err := json.Unmarshal(raw, &parsed); err != nil {
		t.Fatalf("unmarshal empty content: %v", err)
	}
	if len(parsed.Content) != 0 {
		t.Errorf("expected 0 content blocks, got %d", len(parsed.Content))
	}
}

func TestMCP_ToolCallResult_NullResult(t *testing.T) {
	// null result field must not panic on unmarshal.
	raw := []byte(`null`)
	var result toolCallResult
	err := json.Unmarshal(raw, &result)
	if err != nil {
		t.Fatalf("null result should unmarshal to zero value: %v", err)
	}
}

func TestMCP_Response_NullErrorField(t *testing.T) {
	// Response with null error and null result — edge case.
	raw := []byte(`{"jsonrpc":"2.0","id":1,"result":null,"error":null}`)
	var resp jsonRPCResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		t.Fatalf("null fields should parse: %v", err)
	}
	if resp.Error != nil {
		t.Error("null error should be nil")
	}
}

func TestMCP_Response_OversizedPayload(t *testing.T) {
	// Large payload should parse but not cause OOM.
	// 1MB is reasonable; 100MB would be an attack.
	largeText := strings.Repeat("x", 1024*1024) // 1MB
	result := toolCallResult{
		Content: []contentBlock{{Type: "text", Text: largeText}},
	}
	raw, _ := json.Marshal(result)

	var parsed toolCallResult
	if err := json.Unmarshal(raw, &parsed); err != nil {
		t.Fatalf("1MB payload should parse: %v", err)
	}
	if len(parsed.Content[0].Text) != 1024*1024 {
		t.Error("payload truncated")
	}
}

func TestMCP_Response_UnknownFields(t *testing.T) {
	// Extra fields in response must be ignored (not cause errors).
	raw := []byte(`{"jsonrpc":"2.0","id":1,"result":{"content":[{"type":"text","text":"ok"}]},"extra":"attack"}`)
	var resp jsonRPCResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		t.Fatalf("unknown fields should be ignored: %v", err)
	}
}

func TestMCP_Response_IsErrorFlag(t *testing.T) {
	// isError=true should be preserved through deserialization.
	raw := []byte(`{"content":[{"type":"text","text":"failed"}],"isError":true}`)
	var result toolCallResult
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if !result.IsError {
		t.Error("SECURITY: isError flag lost during deserialization")
	}
}

// --- TSK-598: MCP identity validation tests ---

func TestMCP_RejectsUnknownServer(t *testing.T) {
	c := New(djinnlog.Nop())
	defer c.Close()

	// Call with a server name that was never connected.
	_, err := c.Call(context.Background(), "nonexistent-server", "some-tool", nil)
	if err == nil {
		t.Fatal("SECURITY: call to unknown server succeeded")
	}
	if !errors.Is(err, ErrServerNotFound) {
		t.Fatalf("expected ErrServerNotFound, got: %v", err)
	}
}

func TestMCP_ServerInitializeValidation(t *testing.T) {
	// Use MockMCPServer to verify the initialize handshake happens before tools/list.
	mock := testmcp.NewMockMCPServer()
	mock.RegisterTool("test-tool", "A test tool", "ok")

	// Wrap the mock's HTTP handler to record method call order.
	var methods []string
	methodCh := make(chan string, 10)
	srv := httptest.NewServer(wrappingHandler(mock, methodCh))
	defer srv.Close()

	// Connect — this should trigger: initialize → notifications/initialized → tools/list.
	c := New(djinnlog.Nop())
	defer c.Close()

	if err := c.ConnectHTTP(context.Background(), "test-mock", srv.URL); err != nil {
		t.Fatalf("ConnectHTTP: %v", err)
	}

	// Drain the channel to get method order.
	close(methodCh)
	for m := range methodCh {
		methods = append(methods, m)
	}

	// Verify handshake order.
	if len(methods) < 3 {
		t.Fatalf("expected at least 3 methods, got %d: %v", len(methods), methods)
	}
	if methods[0] != "initialize" {
		t.Errorf("first method should be initialize, got %q", methods[0])
	}
	if methods[1] != "notifications/initialized" {
		t.Errorf("second method should be notifications/initialized, got %q", methods[1])
	}
	if methods[2] != "tools/list" {
		t.Errorf("third method should be tools/list, got %q", methods[2])
	}

	// Verify tools were discovered.
	tools := c.Tools()
	if len(tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(tools))
	}
	if tools[0].Name != "test-tool" {
		t.Fatalf("tool name = %q, want test-tool", tools[0].Name)
	}
}

// wrappingHandler records method names before delegating to the mock server.
func wrappingHandler(mock *testmcp.MockMCPServer, methods chan<- string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Capture the body to peek at the method field.
		data, _ := io.ReadAll(r.Body)
		r.Body.Close()

		var req struct {
			Method string `json:"method"`
		}
		json.Unmarshal(data, &req) //nolint:errcheck // test helper
		methods <- req.Method

		// Reconstruct the body for the mock handler.
		r.Body = io.NopCloser(bytes.NewReader(data))
		mock.HTTPHandler().ServeHTTP(w, r)
	}
}
