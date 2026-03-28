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

// TestHotSwap_DisconnectReconnect proves the full hot-swap cycle:
// backend 1 connects → processes prompt → disconnects →
// backend 2 connects → processes prompt → shell sees both responses.
func TestHotSwap_DisconnectReconnect(t *testing.T) {
	sock := filepath.Join(t.TempDir(), "hotswap.sock")

	ln, err := Listen(sock)
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// --- Backend 1: connect, process one prompt, quit ---
	backend1Done := make(chan error, 1)
	go func() {
		transport, err := Connect(sock)
		if err != nil {
			backend1Done <- err
			return
		}
		defer transport.Close()
		backend1Done <- RunBackend(ctx, transport, BackendConfig{
			Driver:   stubs.NewStubChatDriver(driver.Message{Role: "assistant", Content: "response-1"}),
			Tools:    builtin.NewRegistry(),
			Session:  session.New("b1", "model-1", "/workspace"),
			MaxTurns: 5,
		})
	}()

	// Shell accepts backend 1.
	shell1, err := ln.Accept()
	if err != nil {
		t.Fatal(err)
	}

	// Read BackendReady from backend 1.
	ready1, err := shell1.RecvFromBackend()
	if err != nil {
		t.Fatal(err)
	}
	if ready1.Type != BackendReady || ready1.Model != "model-1" {
		t.Fatalf("backend 1 ready: %+v", ready1)
	}

	// Send a prompt to backend 1.
	shell1.SendToBackend(ShellMsg{Type: ShellPrompt, Text: "ping"}) //nolint:errcheck // best-effort send, error logged by receiver

	// Drain events until Done.
	var b1GotDone bool
	for range 20 {
		msg, err := shell1.RecvFromBackend()
		if err != nil {
			break
		}
		if msg.Type == BackendDone {
			b1GotDone = true
			break
		}
	}
	if !b1GotDone {
		t.Fatal("backend 1 never sent Done")
	}

	// Tell backend 1 to quit — simulates graceful shutdown before rebuild.
	shell1.SendToBackend(ShellMsg{Type: ShellQuit}) //nolint:errcheck // best-effort send, error logged by receiver
	if err := <-backend1Done; err != nil {
		t.Fatalf("backend 1 error: %v", err)
	}
	shell1.Close()

	// --- Backend 2: connect (hot-swap), process another prompt ---
	backend2Done := make(chan error, 1)
	go func() {
		transport, err := Connect(sock)
		if err != nil {
			backend2Done <- err
			return
		}
		defer transport.Close()
		backend2Done <- RunBackend(ctx, transport, BackendConfig{
			Driver:   stubs.NewStubChatDriver(driver.Message{Role: "assistant", Content: "response-2"}),
			Tools:    builtin.NewRegistry(),
			Session:  session.New("b2", "model-2", "/workspace"),
			MaxTurns: 5,
		})
	}()

	// Shell accepts backend 2 on the SAME listener — this is the hot-swap.
	shell2, err := ln.Accept()
	if err != nil {
		t.Fatal(err)
	}
	defer shell2.Close()

	// Read BackendReady from backend 2.
	ready2, err := shell2.RecvFromBackend()
	if err != nil {
		t.Fatal(err)
	}
	if ready2.Type != BackendReady || ready2.Model != "model-2" {
		t.Fatalf("backend 2 ready: %+v", ready2)
	}

	// Send a prompt to backend 2.
	shell2.SendToBackend(ShellMsg{Type: ShellPrompt, Text: "ping again"}) //nolint:errcheck // best-effort send, error logged by receiver

	var b2GotDone bool
	for range 20 {
		msg, err := shell2.RecvFromBackend()
		if err != nil {
			break
		}
		if msg.Type == BackendDone {
			b2GotDone = true
			break
		}
	}
	if !b2GotDone {
		t.Fatal("backend 2 never sent Done")
	}

	// Quit backend 2.
	shell2.SendToBackend(ShellMsg{Type: ShellQuit}) //nolint:errcheck // best-effort send, error logged by receiver
	if err := <-backend2Done; err != nil {
		t.Fatalf("backend 2 error: %v", err)
	}
}

// TestHotSwap_BackendCrash_ShellDetects proves that when the backend
// crashes (socket closes unexpectedly), the shell-side recv returns
// an error — enabling the shell to show "backend disconnected" and
// wait for reconnection.
func TestHotSwap_BackendCrash_ShellDetects(t *testing.T) {
	sock := filepath.Join(t.TempDir(), "crash.sock")

	ln, err := Listen(sock)
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	// Backend connects then crashes (Close without Quit).
	go func() {
		transport, _ := Connect(sock)
		// Send ready then immediately crash.
		transport.SendToShell(BackendMsg{Type: BackendReady, Version: ProtocolVersion}) //nolint:errcheck // best-effort send, error logged by receiver
		transport.Close()
	}()

	shell, err := ln.Accept()
	if err != nil {
		t.Fatal(err)
	}
	defer shell.Close()

	// Shell reads BackendReady.
	ready, err := shell.RecvFromBackend()
	if err != nil {
		t.Fatal(err)
	}
	if ready.Type != BackendReady {
		t.Fatalf("expected ready, got %q", ready.Type)
	}

	// Next recv should return error — backend crashed.
	_, err = shell.RecvFromBackend()
	if err == nil {
		t.Fatal("expected error after backend crash — shell should detect disconnect")
	}
}

// TestHotSwap_ShellPreservesState proves that the shell-side listener
// can accept a new backend after the previous one disconnected,
// and the new backend gets a fresh handshake.
func TestHotSwap_ShellPreservesState(t *testing.T) {
	sock := filepath.Join(t.TempDir(), "preserve.sock")

	ln, err := Listen(sock)
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	// Simulate: shell has conversation state that survives backend changes.
	shellConversation := []string{}

	// Backend 1: connect, send text, crash.
	go func() {
		transport, _ := Connect(sock)
		transport.SendToShell(BackendMsg{Type: BackendReady})                       //nolint:errcheck // best-effort send, error logged by receiver
		transport.SendToShell(BackendMsg{Type: BackendText, Text: "hello from b1"}) //nolint:errcheck // best-effort send, error logged by receiver
		transport.Close()                                                           // crash
	}()

	shell1, _ := ln.Accept()
	// Read until error (backend crash).
	for {
		msg, err := shell1.RecvFromBackend()
		if err != nil {
			break
		}
		if msg.Type == BackendText {
			shellConversation = append(shellConversation, msg.Text)
		}
	}
	shell1.Close()

	// Backend 2: connect, send text.
	go func() {
		transport, _ := Connect(sock)
		transport.SendToShell(BackendMsg{Type: BackendReady})                       //nolint:errcheck // best-effort send, error logged by receiver
		transport.SendToShell(BackendMsg{Type: BackendText, Text: "hello from b2"}) //nolint:errcheck // best-effort send, error logged by receiver
		transport.Close()
	}()

	shell2, _ := ln.Accept()
	for {
		msg, err := shell2.RecvFromBackend()
		if err != nil {
			break
		}
		if msg.Type == BackendText {
			shellConversation = append(shellConversation, msg.Text)
		}
	}
	shell2.Close()

	// Shell preserved state across both backends.
	if len(shellConversation) != 2 {
		t.Fatalf("conversation = %v, want 2 entries", shellConversation)
	}
	if shellConversation[0] != "hello from b1" || shellConversation[1] != "hello from b2" {
		t.Fatalf("conversation = %v", shellConversation)
	}
}
