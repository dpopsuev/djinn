package clutch

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/dpopsuev/djinn/driver"
	"github.com/dpopsuev/djinn/session"
	"github.com/dpopsuev/djinn/testkit/stubs"
	"github.com/dpopsuev/djinn/tools/builtin"
)

// TestSocketBackend_HandshakeOverSocket verifies that RunBackend
// sends BackendReady over a real Unix socket and the shell receives it.
func TestSocketBackend_HandshakeOverSocket(t *testing.T) {
	sock := filepath.Join(t.TempDir(), "handshake.sock")

	ln, err := Listen(sock)
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	// Start backend in a goroutine — it connects and sends BackendReady.
	sess := session.New("test", "test-model", "/workspace")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	backendDone := make(chan error, 1)
	go func() {
		transport, err := Connect(sock)
		if err != nil {
			backendDone <- err
			return
		}
		defer transport.Close()
		backendDone <- RunBackend(ctx, transport, BackendConfig{
			Driver:   &stubs.StubChatDriver{},
			Tools:    builtin.NewRegistry(),
			Session:  sess,
			MaxTurns: 5,
		})
	}()

	// Shell accepts and reads the handshake.
	shell, err := ln.Accept()
	if err != nil {
		t.Fatal(err)
	}
	defer shell.Close()

	msg, err := shell.RecvFromBackend()
	if err != nil {
		t.Fatal(err)
	}
	if msg.Type != BackendReady {
		t.Fatalf("expected BackendReady, got %q", msg.Type)
	}
	if msg.Version != ProtocolVersion {
		t.Fatalf("version = %d, want %d", msg.Version, ProtocolVersion)
	}
	if msg.Model != "test-model" {
		t.Fatalf("model = %q", msg.Model)
	}

	// Tell backend to quit.
	shell.SendToBackend(ShellMsg{Type: ShellQuit}) //nolint:errcheck // best-effort send, error logged by receiver
	if err := <-backendDone; err != nil {
		t.Fatalf("backend error: %v", err)
	}
}

// TestSocketBackend_PromptRoundTrip verifies that a prompt sent
// over the socket produces streaming text events back.
func TestSocketBackend_PromptRoundTrip(t *testing.T) {
	sock := filepath.Join(t.TempDir(), "prompt.sock")

	ln, err := Listen(sock)
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	// Backend with a stub driver that returns "pong".
	sess := session.New("test", "test-model", "/workspace")
	stubDriver := stubs.NewStubChatDriver(driver.Message{
		Role: driver.RoleAssistant, Content: "pong",
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	backendDone := make(chan error, 1)
	go func() {
		transport, err := Connect(sock)
		if err != nil {
			backendDone <- err
			return
		}
		defer transport.Close()
		backendDone <- RunBackend(ctx, transport, BackendConfig{
			Driver:   stubDriver,
			Tools:    builtin.NewRegistry(),
			Session:  sess,
			MaxTurns: 5,
		})
	}()

	shell, err := ln.Accept()
	if err != nil {
		t.Fatal(err)
	}
	defer shell.Close()

	// Read BackendReady.
	ready, _ := shell.RecvFromBackend()
	if ready.Type != BackendReady {
		t.Fatalf("expected ready, got %q", ready.Type)
	}

	// Send a prompt.
	shell.SendToBackend(ShellMsg{Type: ShellPrompt, Text: "ping"}) //nolint:errcheck // best-effort send, error logged by receiver

	// Collect backend events until Done.
	var gotText bool
	var gotDone bool
	for i := 0; i < 20; i++ {
		msg, err := shell.RecvFromBackend()
		if err != nil {
			t.Fatalf("recv: %v", err)
		}
		switch msg.Type {
		case BackendText:
			gotText = true
		case BackendDone:
			gotDone = true
		case BackendError:
			// Stub driver might produce errors — that's ok for this test.
		}
		if gotDone {
			break
		}
	}

	if !gotText {
		t.Fatal("expected at least one BackendText event")
	}
	if !gotDone {
		t.Fatal("expected BackendDone event")
	}

	// Quit.
	shell.SendToBackend(ShellMsg{Type: ShellQuit}) //nolint:errcheck // best-effort send, error logged by receiver
	<-backendDone
}
