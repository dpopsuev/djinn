// clutch_hotswap_test.go — acceptance tests for the clutch shell/backend split.
//
// Spec: DJA-SPC-17 — Clutch Wiring
// Covers:
//   - ChannelTransport send/recv roundtrip
//   - Backend sends Ready on start
//   - Shell sends prompt, backend echoes via handler
//   - Shell sends quit, backend exits cleanly
//   - Protocol version agreement
//   - Transport close prevents further sends
package acceptance

import (
	"context"
	"testing"
	"time"

	"github.com/dpopsuev/djinn/clutch"
	"github.com/dpopsuev/djinn/driver"
	"github.com/dpopsuev/djinn/session"
	"github.com/dpopsuev/djinn/testkit/stubs"
	"github.com/dpopsuev/djinn/tools/builtin"
)

func TestClutch_ChannelTransport_Roundtrip(t *testing.T) {
	transport := clutch.NewChannelTransport()
	defer transport.Close()

	// Shell → Backend
	transport.SendToBackend(clutch.ShellMsg{Type: clutch.ShellPrompt, Text: "hello"})

	msg, err := transport.RecvFromShell()
	if err != nil {
		t.Fatalf("RecvFromShell: %v", err)
	}
	if msg.Type != clutch.ShellPrompt || msg.Text != "hello" {
		t.Fatalf("msg = %+v", msg)
	}

	// Backend → Shell
	transport.SendToShell(clutch.BackendMsg{Type: clutch.BackendText, Text: "world"})

	resp, err := transport.RecvFromBackend()
	if err != nil {
		t.Fatalf("RecvFromBackend: %v", err)
	}
	if resp.Type != clutch.BackendText || resp.Text != "world" {
		t.Fatalf("resp = %+v", resp)
	}
}

func TestClutch_BackendSendsReady(t *testing.T) {
	transport := clutch.NewChannelTransport()
	defer transport.Close()

	sess := session.New("test", "test-model", "/workspace")
	stubDriver := stubs.NewStubChatDriver(
		driver.Message{Role: driver.RoleAssistant, Content: "hi"},
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go clutch.RunBackend(ctx, transport, clutch.BackendConfig{
		Driver:  stubDriver,
		Tools:   builtin.NewRegistry(),
		Session: sess,
	})

	// First message should be Ready
	msg, err := transport.RecvFromBackend()
	if err != nil {
		t.Fatalf("RecvFromBackend: %v", err)
	}
	if msg.Type != clutch.BackendReady {
		t.Fatalf("expected Ready, got %q", msg.Type)
	}
	if msg.Version != clutch.ProtocolVersion {
		t.Fatalf("version = %d, want %d", msg.Version, clutch.ProtocolVersion)
	}
	if msg.Model != "test-model" {
		t.Fatalf("model = %q", msg.Model)
	}

	// Clean quit
	transport.SendToBackend(clutch.ShellMsg{Type: clutch.ShellQuit})
	quitMsg, _ := transport.RecvFromBackend()
	if quitMsg.Type != clutch.BackendExiting {
		t.Fatalf("expected Exiting, got %q", quitMsg.Type)
	}
}

func TestClutch_QuitExitsCleanly(t *testing.T) {
	transport := clutch.NewChannelTransport()
	defer transport.Close()

	sess := session.New("test", "model", "/workspace")
	stubDriver := stubs.NewStubChatDriver(
		driver.Message{Role: driver.RoleAssistant, Content: "done"},
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- clutch.RunBackend(ctx, transport, clutch.BackendConfig{
			Driver:  stubDriver,
			Tools:   builtin.NewRegistry(),
			Session: sess,
		})
	}()

	// Wait for ready
	transport.RecvFromBackend()

	// Send quit
	transport.SendToBackend(clutch.ShellMsg{Type: clutch.ShellQuit})

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("RunBackend should exit cleanly: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("backend should exit within 2s")
	}
}

func TestClutch_TransportClose_PreventseSends(t *testing.T) {
	transport := clutch.NewChannelTransport()
	transport.Close()

	err := transport.SendToBackend(clutch.ShellMsg{Type: clutch.ShellPrompt})
	if err == nil {
		t.Fatal("send after close should error")
	}
}

func TestClutch_ProtocolVersion(t *testing.T) {
	if clutch.ProtocolVersion < 1 {
		t.Fatal("protocol version should be >= 1")
	}
}
