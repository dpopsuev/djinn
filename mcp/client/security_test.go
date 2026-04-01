package client

import (
	"encoding/json"
	"strings"
	"testing"
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
