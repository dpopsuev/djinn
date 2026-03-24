package claude

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/dpopsuev/djinn/djinnlog"
	"github.com/dpopsuev/djinn/driver"
	"github.com/dpopsuev/djinn/tools/builtin"
)

func sseResponse(events ...string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		for _, e := range events {
			fmt.Fprint(w, e)
		}
	}
}

const textSSE = `event: message_start
data: {"type":"message_start","message":{"id":"msg-1","role":"assistant"}}

event: content_block_start
data: {"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Hello "}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"world"}}

event: content_block_stop
data: {"type":"content_block_stop","index":0}

event: message_delta
data: {"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":5}}

event: message_stop
data: {"type":"message_stop"}

`

func newTestAPIDriver(t *testing.T, handler http.HandlerFunc) *APIDriver {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	d := &APIDriver{
		config: driver.DriverConfig{Model: "claude-sonnet-4-6", MaxTokens: 1024},
		apiURL: srv.URL,
		apiKey: "test-key",
		log:    djinnlog.Nop(),
		client: srv.Client(),
	}
	d.Start(context.Background(), "")
	return d
}

func TestAPIDriver_InterfaceSatisfaction(t *testing.T) {
	var _ driver.Driver = (*APIDriver)(nil)
}

func TestAPIDriver_StreamText(t *testing.T) {
	d := newTestAPIDriver(t, sseResponse(textSSE))

	d.Send(context.Background(), driver.Message{Role: driver.RoleUser, Content: "hi"})

	events, err := d.Chat(context.Background())
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}

	var text string
	var gotDone bool
	for evt := range events {
		switch evt.Type {
		case driver.EventText:
			text += evt.Text
		case driver.EventDone:
			gotDone = true
		}
	}

	if text != "Hello world" {
		t.Fatalf("text = %q, want %q", text, "Hello world")
	}
	if !gotDone {
		t.Fatal("missing done event")
	}
}

const toolUseSSE = `event: message_start
data: {"type":"message_start","message":{"id":"msg-2","role":"assistant"}}

event: content_block_start
data: {"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Let me read that file."}}

event: content_block_stop
data: {"type":"content_block_stop","index":0}

event: content_block_start
data: {"type":"content_block_start","index":1,"content_block":{"type":"tool_use","id":"call-1","name":"Read","input":{}}}

event: content_block_delta
data: {"type":"content_block_delta","index":1,"delta":{"type":"input_json_delta","partial_json":"{\"path\": \"main.go\"}"}}

event: content_block_stop
data: {"type":"content_block_stop","index":1}

event: message_delta
data: {"type":"message_delta","delta":{"stop_reason":"tool_use"},"usage":{"output_tokens":20}}

event: message_stop
data: {"type":"message_stop"}

`

func TestAPIDriver_StreamToolUse(t *testing.T) {
	d := newTestAPIDriver(t, sseResponse(toolUseSSE))

	d.Send(context.Background(), driver.Message{Role: driver.RoleUser, Content: "read main.go"})

	events, err := d.Chat(context.Background())
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}

	var text string
	var toolCalls []driver.ToolCall
	for evt := range events {
		switch evt.Type {
		case driver.EventText:
			text += evt.Text
		case driver.EventToolUse:
			if evt.ToolCall != nil {
				toolCalls = append(toolCalls, *evt.ToolCall)
			}
		}
	}

	if text != "Let me read that file." {
		t.Fatalf("text = %q", text)
	}
	if len(toolCalls) != 1 {
		t.Fatalf("tool calls = %d, want 1", len(toolCalls))
	}
	if toolCalls[0].Name != "Read" {
		t.Fatalf("tool name = %q, want Read", toolCalls[0].Name)
	}
}

func TestAPIDriver_RecvBackwardCompat(t *testing.T) {
	d := newTestAPIDriver(t, sseResponse(textSSE))

	d.Send(context.Background(), driver.Message{Role: driver.RoleUser, Content: "hi"})

	ch := d.Recv(context.Background())
	msg := <-ch
	if msg.Role != driver.RoleAssistant {
		t.Fatalf("Role = %q", msg.Role)
	}
	if msg.Content != "Hello world" {
		t.Fatalf("Content = %q, want %q", msg.Content, "Hello world")
	}
}

func TestAPIDriver_WithTools(t *testing.T) {
	d := newTestAPIDriver(t, sseResponse(textSSE))
	d.tools = builtin.NewRegistry()

	d.Send(context.Background(), driver.Message{Role: driver.RoleUser, Content: "hi"})

	_, err := d.Chat(context.Background())
	if err != nil {
		t.Fatalf("Chat with tools: %v", err)
	}
}

func TestAPIDriver_APIError(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": "invalid api key", "request_id": "req_test_123"}`))
	}

	d := newTestAPIDriver(t, handler)
	d.Send(context.Background(), driver.Message{Role: driver.RoleUser, Content: "hi"})

	_, err := d.Chat(context.Background())
	if err == nil {
		t.Fatal("expected error for 401")
	}

	var de *driver.DriverError
	if !errors.As(err, &de) {
		t.Fatalf("expected DriverError, got %T: %v", err, err)
	}
	if de.StatusCode != 401 {
		t.Fatalf("StatusCode = %d, want 401", de.StatusCode)
	}
	if de.Retryable {
		t.Fatal("401 should not be retryable")
	}
	if de.Provider != "claude-direct" {
		t.Fatalf("Provider = %q, want claude-direct", de.Provider)
	}
	if de.RequestID != "req_test_123" {
		t.Fatalf("RequestID = %q, want req_test_123", de.RequestID)
	}
}

func TestAPIDriver_ToolUseInputAccumulated(t *testing.T) {
	// DJN-BUG-19: SSE parser must accumulate input_json_delta chunks
	// into the ToolCall.Input field. Previously Input was always nil.
	d := newTestAPIDriver(t, sseResponse(toolUseSSE))

	d.Send(context.Background(), driver.Message{Role: driver.RoleUser, Content: "read main.go"})

	events, err := d.Chat(context.Background())
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}

	var toolCalls []driver.ToolCall
	for evt := range events {
		if evt.Type == driver.EventToolUse && evt.ToolCall != nil {
			toolCalls = append(toolCalls, *evt.ToolCall)
		}
	}

	if len(toolCalls) != 1 {
		t.Fatalf("tool calls = %d, want 1", len(toolCalls))
	}

	tc := toolCalls[0]
	if tc.Input == nil {
		t.Fatal("BUG-19: ToolCall.Input is nil — input_json_delta not accumulated")
	}
	if string(tc.Input) == "null" || string(tc.Input) == "" {
		t.Fatalf("BUG-19: ToolCall.Input = %q — should contain accumulated JSON", string(tc.Input))
	}
}

func TestAPIDriver_NotStarted(t *testing.T) {
	d := &APIDriver{}
	err := d.Send(context.Background(), driver.Message{})
	if err == nil {
		t.Fatal("expected error when not started")
	}
}

func TestAPIDriver_NoAPIKey(t *testing.T) {
	// Clear env vars
	origKey := os.Getenv(envAPIKey)
	origProject := os.Getenv(envVertexProject)
	os.Unsetenv(envAPIKey)
	os.Unsetenv(envVertexProject)
	defer func() {
		if origKey != "" {
			os.Setenv(envAPIKey, origKey)
		}
		if origProject != "" {
			os.Setenv(envVertexProject, origProject)
		}
	}()

	_, err := NewAPIDriver(driver.DriverConfig{})
	if err == nil {
		t.Fatal("expected ErrNoAPIKey")
	}
}

func TestAPIDriver_SendRich(t *testing.T) {
	d := newTestAPIDriver(t, sseResponse(textSSE))

	// Send a tool result as rich message
	d.SendRich(context.Background(), driver.RichMessage{
		Role: driver.RoleUser,
		Blocks: []driver.ContentBlock{
			driver.NewToolResultBlock("call-1", "file contents", false),
		},
	})

	_, err := d.Chat(context.Background())
	if err != nil {
		t.Fatalf("Chat after SendRich: %v", err)
	}
}
