package clutch

import (
	"context"
	"encoding/json"
	"net"
	"path/filepath"
	"testing"
	"time"
)

// connectToHub dials the hub and sends a registration message.
func connectToHub(t *testing.T, socketPath, role string) *SocketTransport {
	t.Helper()
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		t.Fatalf("dial hub: %v", err)
	}
	enc := json.NewEncoder(conn)
	if err := enc.Encode(RegisterMsg{Role: role}); err != nil {
		conn.Close()
		t.Fatalf("register %s: %v", role, err)
	}
	return newSocketTransport(conn)
}

func TestHub_FrontendHotSwap(t *testing.T) {
	sock := filepath.Join(t.TempDir(), "hub-fe.sock")
	hub, err := NewHub(sock)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	defer hub.Close()
	go hub.Run(ctx) //nolint:errcheck // test helper, error checked elsewhere

	// Give hub a moment to start accepting.
	time.Sleep(50 * time.Millisecond)

	// Backend registers and sends BackendReady.
	backend := connectToHub(t, sock, "backend")
	defer backend.Close()
	backend.SendToShell(BackendMsg{Type: BackendReady, Model: "test-model"}) //nolint:errcheck // best-effort send, error logged by receiver

	// Frontend 1 registers and receives BackendReady.
	fe1 := connectToHub(t, sock, "shell")
	msg1, err := fe1.RecvFromBackend()
	if err != nil {
		t.Fatalf("fe1 recv: %v", err)
	}
	if msg1.Type != BackendReady || msg1.Model != "test-model" {
		t.Fatalf("fe1 got %+v, want BackendReady", msg1)
	}

	// Frontend 1 sends a prompt, backend receives it.
	fe1.SendToBackend(ShellMsg{Type: ShellPrompt, Text: "hello from fe1"}) //nolint:errcheck // best-effort send, error logged by receiver
	shellMsg, err := backend.RecvFromShell()
	if err != nil {
		t.Fatalf("backend recv: %v", err)
	}
	if shellMsg.Text != "hello from fe1" {
		t.Fatalf("backend got %q, want 'hello from fe1'", shellMsg.Text)
	}

	// Frontend 1 crashes.
	fe1.Close()
	time.Sleep(50 * time.Millisecond)

	// Backend sends a message while shell is disconnected — should be queued.
	backend.SendToShell(BackendMsg{Type: BackendText, Text: "queued response"}) //nolint:errcheck // best-effort send, error logged by receiver
	time.Sleep(50 * time.Millisecond)

	// Frontend 2 registers and receives the queued message.
	fe2 := connectToHub(t, sock, "shell")
	defer fe2.Close()

	msg2, err := fe2.RecvFromBackend()
	if err != nil {
		t.Fatalf("fe2 recv: %v", err)
	}
	if msg2.Type != BackendText || msg2.Text != "queued response" {
		t.Fatalf("fe2 got %+v, want queued text", msg2)
	}

	// Frontend 2 sends a prompt, backend receives it.
	fe2.SendToBackend(ShellMsg{Type: ShellPrompt, Text: "hello from fe2"}) //nolint:errcheck // best-effort send, error logged by receiver
	shellMsg2, err := backend.RecvFromShell()
	if err != nil {
		t.Fatalf("backend recv 2: %v", err)
	}
	if shellMsg2.Text != "hello from fe2" {
		t.Fatalf("backend got %q, want 'hello from fe2'", shellMsg2.Text)
	}
}

func TestHub_DaemonHotSwap(t *testing.T) {
	sock := filepath.Join(t.TempDir(), "hub-be.sock")
	hub, err := NewHub(sock)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	defer hub.Close()
	go hub.Run(ctx) //nolint:errcheck // test helper, error checked elsewhere
	time.Sleep(50 * time.Millisecond)

	// Shell registers first.
	shell := connectToHub(t, sock, "shell")
	defer shell.Close()

	// Shell sends a prompt — should be queued (no backend yet).
	shell.SendToBackend(ShellMsg{Type: ShellPrompt, Text: "prompt-1"}) //nolint:errcheck // best-effort send, error logged by receiver
	time.Sleep(50 * time.Millisecond)

	// Backend 1 registers and receives the queued prompt.
	be1 := connectToHub(t, sock, "backend")
	msg1, err := be1.RecvFromShell()
	if err != nil {
		t.Fatalf("be1 recv: %v", err)
	}
	if msg1.Text != "prompt-1" {
		t.Fatalf("be1 got %q, want 'prompt-1'", msg1.Text)
	}

	// Backend 1 responds.
	be1.SendToShell(BackendMsg{Type: BackendText, Text: "response-1"}) //nolint:errcheck // best-effort send, error logged by receiver
	resp1, err := shell.RecvFromBackend()
	if err != nil {
		t.Fatalf("shell recv: %v", err)
	}
	if resp1.Text != "response-1" {
		t.Fatalf("shell got %q, want 'response-1'", resp1.Text)
	}

	// Backend 1 crashes.
	be1.Close()
	time.Sleep(50 * time.Millisecond)

	// Shell sends another prompt — should be queued.
	shell.SendToBackend(ShellMsg{Type: ShellPrompt, Text: "prompt-2"}) //nolint:errcheck // best-effort send, error logged by receiver
	time.Sleep(50 * time.Millisecond)

	// Backend 2 registers and receives the queued prompt.
	be2 := connectToHub(t, sock, "backend")
	defer be2.Close()
	msg2, err := be2.RecvFromShell()
	if err != nil {
		t.Fatalf("be2 recv: %v", err)
	}
	if msg2.Text != "prompt-2" {
		t.Fatalf("be2 got %q, want 'prompt-2'", msg2.Text)
	}

	// Backend 2 responds.
	be2.SendToShell(BackendMsg{Type: BackendText, Text: "response-2"}) //nolint:errcheck // best-effort send, error logged by receiver
	resp2, err := shell.RecvFromBackend()
	if err != nil {
		t.Fatalf("shell recv 2: %v", err)
	}
	if resp2.Text != "response-2" {
		t.Fatalf("shell got %q, want 'response-2'", resp2.Text)
	}
}

func TestHub_BothHotSwap(t *testing.T) {
	sock := filepath.Join(t.TempDir(), "hub-both.sock")
	hub, err := NewHub(sock)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	defer hub.Close()
	go hub.Run(ctx) //nolint:errcheck // test helper, error checked elsewhere
	time.Sleep(50 * time.Millisecond)

	// Both connect and exchange a message.
	shell1 := connectToHub(t, sock, "shell")
	be1 := connectToHub(t, sock, "backend")
	time.Sleep(50 * time.Millisecond)

	be1.SendToShell(BackendMsg{Type: BackendReady, Model: "m1", HistoryLen: 3}) //nolint:errcheck // best-effort send, error logged by receiver
	ready, err := shell1.RecvFromBackend()
	if err != nil {
		t.Fatalf("shell1 recv: %v", err)
	}
	if ready.HistoryLen != 3 {
		t.Fatalf("history = %d, want 3", ready.HistoryLen)
	}

	// Both crash.
	shell1.Close()
	be1.Close()
	time.Sleep(100 * time.Millisecond)

	// Both reconnect. Backend sends BackendReady with restored session.
	shell2 := connectToHub(t, sock, "shell")
	defer shell2.Close()
	be2 := connectToHub(t, sock, "backend")
	defer be2.Close()
	time.Sleep(50 * time.Millisecond)

	be2.SendToShell(BackendMsg{Type: BackendReady, Model: "m2", HistoryLen: 5}) //nolint:errcheck // best-effort send, error logged by receiver
	ready2, err := shell2.RecvFromBackend()
	if err != nil {
		t.Fatalf("shell2 recv: %v", err)
	}
	if ready2.HistoryLen != 5 {
		t.Fatalf("history = %d, want 5 (session state survived restart)", ready2.HistoryLen)
	}
}

func TestHub_DaemonOnly_NoFrontend(t *testing.T) {
	sock := filepath.Join(t.TempDir(), "hub-noshell.sock")
	hub, err := NewHub(sock)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	defer hub.Close()
	go hub.Run(ctx) //nolint:errcheck // test helper, error checked elsewhere
	time.Sleep(50 * time.Millisecond)

	// Backend registers first — no shell yet.
	backend := connectToHub(t, sock, "backend")
	defer backend.Close()

	// Backend sends BackendReady — should be queued for shell.
	backend.SendToShell(BackendMsg{Type: BackendReady, Model: "early-model"}) //nolint:errcheck // best-effort send, error logged by receiver
	time.Sleep(50 * time.Millisecond)

	// Shell registers later and receives the queued BackendReady.
	shell := connectToHub(t, sock, "shell")
	defer shell.Close()

	msg, err := shell.RecvFromBackend()
	if err != nil {
		t.Fatalf("shell recv: %v", err)
	}
	if msg.Type != BackendReady || msg.Model != "early-model" {
		t.Fatalf("shell got %+v, want BackendReady with model 'early-model'", msg)
	}
}
