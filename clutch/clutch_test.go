package clutch

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/dpopsuev/djinn/driver"
	claudedriver "github.com/dpopsuev/djinn/driver/claude"
	"github.com/dpopsuev/djinn/session"
	"github.com/dpopsuev/djinn/tools/builtin"
)

func TestChannelTransport_RoundTrip(t *testing.T) {
	tr := NewChannelTransport()
	defer tr.Close()

	// Shell → Backend
	go func() {
		tr.SendToBackend(ShellMsg{Type: ShellPrompt, Text: "hello"})
	}()

	msg, err := tr.RecvFromShell()
	if err != nil {
		t.Fatalf("RecvFromShell: %v", err)
	}
	if msg.Type != ShellPrompt {
		t.Fatalf("Type = %q, want %q", msg.Type, ShellPrompt)
	}
	if msg.Text != "hello" {
		t.Fatalf("Text = %q, want %q", msg.Text, "hello")
	}

	// Backend → Shell
	go func() {
		tr.SendToShell(BackendMsg{Type: BackendText, Text: "world"})
	}()

	resp, err := tr.RecvFromBackend()
	if err != nil {
		t.Fatalf("RecvFromBackend: %v", err)
	}
	if resp.Type != BackendText {
		t.Fatalf("Type = %q", resp.Type)
	}
	if resp.Text != "world" {
		t.Fatalf("Text = %q", resp.Text)
	}
}

func TestChannelTransport_Close(t *testing.T) {
	tr := NewChannelTransport()
	tr.Close()

	err := tr.SendToBackend(ShellMsg{Type: ShellPrompt})
	if err != ErrClosed {
		t.Fatalf("SendToBackend after close: expected ErrClosed, got %v", err)
	}

	err = tr.SendToShell(BackendMsg{Type: BackendText})
	if err != ErrClosed {
		t.Fatalf("SendToShell after close: expected ErrClosed, got %v", err)
	}
}

func TestChannelTransport_DoubleClose(t *testing.T) {
	tr := NewChannelTransport()
	tr.Close()
	tr.Close() // should not panic
}

func TestBackendMsg_JSON(t *testing.T) {
	msg := BackendMsg{
		Type:  BackendToolCall,
		ToolCall: &driver.ToolCall{
			ID:   "call-1",
			Name: "Read",
			Input: json.RawMessage(`{"path":"main.go"}`),
		},
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded BackendMsg
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.ToolCall.Name != "Read" {
		t.Fatalf("ToolCall.Name = %q", decoded.ToolCall.Name)
	}
}

func TestRunBackend_QuitImmediately(t *testing.T) {
	tr := NewChannelTransport()

	// Send quit after backend announces ready
	go func() {
		// Wait for ready
		msg, _ := tr.RecvFromBackend()
		if msg.Type != BackendReady {
			t.Errorf("expected Ready, got %q", msg.Type)
		}
		if msg.Version != ProtocolVersion {
			t.Errorf("version = %d, want %d", msg.Version, ProtocolVersion)
		}
		tr.SendToBackend(ShellMsg{Type: ShellQuit})
	}()

	sess := session.New("test", "test-model", "/workspace")
	err := RunBackend(context.Background(), tr, BackendConfig{
		Session:  sess,
		Tools:    builtin.NewRegistry(),
		MaxTurns: 5,
	})
	if err != nil {
		t.Fatalf("RunBackend: %v", err)
	}
}

func TestRunBackend_SessionState(t *testing.T) {
	tr := NewChannelTransport()

	sess := session.New("sess-1", "claude-opus-4-6", "/workspace")
	sess.Append(session.Entry{Content: "old message"})
	sess.Append(session.Entry{Content: "another"})

	go func() {
		msg, _ := tr.RecvFromBackend()
		if msg.Model != "claude-opus-4-6" {
			t.Errorf("Model = %q", msg.Model)
		}
		if msg.HistoryLen != 2 {
			t.Errorf("HistoryLen = %d, want 2", msg.HistoryLen)
		}
		tr.SendToBackend(ShellMsg{Type: ShellQuit})
	}()

	RunBackend(context.Background(), tr, BackendConfig{
		Session: sess,
		Tools:   builtin.NewRegistry(),
	})
}

func TestRunBackend_TransportDisconnect(t *testing.T) {
	tr := NewChannelTransport()

	go func() {
		tr.RecvFromBackend() // consume ready
		time.Sleep(10 * time.Millisecond)
		tr.Close() // simulate disconnect
	}()

	sess := session.New("test", "model", "/workspace")
	err := RunBackend(context.Background(), tr, BackendConfig{
		Session: sess,
		Tools:   builtin.NewRegistry(),
	})

	// Should return with error (closed transport)
	if err == nil {
		t.Fatal("expected error on transport close")
	}
}

func TestBackendHandler_EventsReachShell(t *testing.T) {
	tr := NewChannelTransport()
	h := &backendHandler{transport: tr}

	go func() {
		h.OnText("hello")
		h.OnToolCall(driver.ToolCall{ID: "c1", Name: "Read"})
		h.OnToolResult("c1", "Read", "contents", false)
		h.OnDone(&driver.Usage{InputTokens: 100, OutputTokens: 50})
	}()

	msgs := make([]BackendMsg, 4)
	for i := range 4 {
		var err error
		msgs[i], err = tr.RecvFromBackend()
		if err != nil {
			t.Fatalf("recv %d: %v", i, err)
		}
	}

	if msgs[0].Type != BackendText || msgs[0].Text != "hello" {
		t.Fatalf("msg[0] = %v", msgs[0])
	}
	if msgs[1].Type != BackendToolCall || msgs[1].ToolCall.Name != "Read" {
		t.Fatalf("msg[1] = %v", msgs[1])
	}
	if msgs[2].Type != BackendToolResult || msgs[2].ToolName != "Read" {
		t.Fatalf("msg[2] = %v", msgs[2])
	}
	if msgs[3].Type != BackendDone || msgs[3].Usage.InputTokens != 100 {
		t.Fatalf("msg[3] = %v", msgs[3])
	}
}

// Verify unused import is satisfied
var _ = claudedriver.NewAPIDriver
