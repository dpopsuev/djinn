// hub_hotswap_test.go — E2E acceptance tests for GenSec Hub hot-swap.
//
// Verifies the full dogfooding workflow:
// 1. Hub starts and accepts connections
// 2. Shell connects, backend connects, messages flow
// 3. Shell crashes → backend survives → new shell reconnects
// 4. Backend crashes → shell survives → new backend reconnects
// 5. Both crash → hub survives → both reconnect
package acceptance

import (
	"context"
	"encoding/json"
	"net"
	"path/filepath"
	"testing"
	"time"

	"github.com/dpopsuev/djinn/clutch"
)

// hubConnect dials the hub and registers with the given role.
func hubConnect(t *testing.T, socketPath, role string) *clutch.SocketTransport {
	t.Helper()
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		t.Fatalf("dial hub as %s: %v", role, err)
	}
	if err := json.NewEncoder(conn).Encode(clutch.RegisterMsg{Role: role}); err != nil {
		conn.Close()
		t.Fatalf("register as %s: %v", role, err)
	}
	return clutch.WrapConn(conn)
}

// TestHub_E2E_ShellHotSwap proves the dogfooding scenario:
// rebuild shell → restart → session preserved.
func TestHub_E2E_ShellHotSwap(t *testing.T) {
	sock := filepath.Join(t.TempDir(), "hub-e2e-shell.sock")
	hub, err := clutch.NewHub(sock)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	defer hub.Close()
	go hub.Run(ctx) //nolint:errcheck // test helper, error checked elsewhere
	time.Sleep(100 * time.Millisecond)

	// Backend connects and stays alive the whole time.
	backend := hubConnect(t, sock, "backend")
	defer backend.Close()
	backend.SendToShell(clutch.BackendMsg{Type: clutch.BackendReady, Model: "test"}) //nolint:errcheck // best-effort send, error logged by receiver

	// Shell 1 connects, receives BackendReady.
	shell1 := hubConnect(t, sock, "shell")
	ready, err := shell1.RecvFromBackend()
	if err != nil || ready.Type != clutch.BackendReady {
		t.Fatalf("shell1 ready: %+v, err: %v", ready, err)
	}

	// Shell 1 sends a prompt, backend receives it.
	shell1.SendToBackend(clutch.ShellMsg{Type: clutch.ShellPrompt, Text: "hello"}) //nolint:errcheck // best-effort send, error logged by receiver
	msg, err := backend.RecvFromShell()
	if err != nil || msg.Text != "hello" {
		t.Fatalf("backend recv: %+v, err: %v", msg, err)
	}

	// Backend responds.
	backend.SendToShell(clutch.BackendMsg{Type: clutch.BackendText, Text: "world"}) //nolint:errcheck // best-effort send, error logged by receiver
	resp, err := shell1.RecvFromBackend()
	if err != nil || resp.Text != "world" {
		t.Fatalf("shell1 recv: %+v, err: %v", resp, err)
	}

	// --- SHELL CRASHES (simulates rebuild + restart) ---
	shell1.Close()
	time.Sleep(100 * time.Millisecond)

	// Backend sends another message while shell is disconnected.
	backend.SendToShell(clutch.BackendMsg{Type: clutch.BackendText, Text: "queued-after-crash"}) //nolint:errcheck // best-effort send, error logged by receiver
	time.Sleep(100 * time.Millisecond)

	// Shell 2 connects (the rebuilt binary).
	shell2 := hubConnect(t, sock, "shell")
	defer shell2.Close()

	// Shell 2 receives the queued message from the backend.
	queued, err := shell2.RecvFromBackend()
	if err != nil {
		t.Fatalf("shell2 recv queued: %v", err)
	}
	if queued.Text != "queued-after-crash" {
		t.Fatalf("queued = %q, want 'queued-after-crash'", queued.Text)
	}

	// Shell 2 sends a new prompt, backend receives it.
	shell2.SendToBackend(clutch.ShellMsg{Type: clutch.ShellPrompt, Text: "from-rebuilt-shell"}) //nolint:errcheck // best-effort send, error logged by receiver
	msg2, err := backend.RecvFromShell()
	if err != nil || msg2.Text != "from-rebuilt-shell" {
		t.Fatalf("backend recv2: %+v, err: %v", msg2, err)
	}
}

// TestHub_E2E_BackendHotSwap proves backend rebuild doesn't affect the shell.
func TestHub_E2E_BackendHotSwap(t *testing.T) {
	sock := filepath.Join(t.TempDir(), "hub-e2e-backend.sock")
	hub, err := clutch.NewHub(sock)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	defer hub.Close()
	go hub.Run(ctx) //nolint:errcheck // test helper, error checked elsewhere
	time.Sleep(100 * time.Millisecond)

	// Shell connects and stays alive.
	shell := hubConnect(t, sock, "shell")
	defer shell.Close()

	// Backend 1 connects.
	be1 := hubConnect(t, sock, "backend")
	be1.SendToShell(clutch.BackendMsg{Type: clutch.BackendReady, Model: "v1"}) //nolint:errcheck // best-effort send, error logged by receiver

	ready, _ := shell.RecvFromBackend()
	if ready.Model != "v1" {
		t.Fatalf("model = %q, want v1", ready.Model)
	}

	// Exchange messages.
	shell.SendToBackend(clutch.ShellMsg{Type: clutch.ShellPrompt, Text: "p1"}) //nolint:errcheck // best-effort send, error logged by receiver
	m1, _ := be1.RecvFromShell()
	if m1.Text != "p1" {
		t.Fatalf("be1 got %q", m1.Text)
	}
	be1.SendToShell(clutch.BackendMsg{Type: clutch.BackendText, Text: "r1"}) //nolint:errcheck // best-effort send, error logged by receiver
	r1, _ := shell.RecvFromBackend()
	if r1.Text != "r1" {
		t.Fatalf("shell got %q", r1.Text)
	}

	// --- BACKEND CRASHES ---
	be1.Close()
	time.Sleep(100 * time.Millisecond)

	// Shell sends a prompt while backend is down — hub queues it.
	shell.SendToBackend(clutch.ShellMsg{Type: clutch.ShellPrompt, Text: "queued-prompt"}) //nolint:errcheck // best-effort send, error logged by receiver
	time.Sleep(100 * time.Millisecond)

	// Backend 2 connects (rebuilt binary).
	be2 := hubConnect(t, sock, "backend")
	defer be2.Close()

	// Backend 2 receives the queued prompt.
	queued, err := be2.RecvFromShell()
	if err != nil {
		t.Fatalf("be2 recv queued: %v", err)
	}
	if queued.Text != "queued-prompt" {
		t.Fatalf("queued = %q, want 'queued-prompt'", queued.Text)
	}

	// Backend 2 announces itself.
	be2.SendToShell(clutch.BackendMsg{Type: clutch.BackendReady, Model: "v2"}) //nolint:errcheck // best-effort send, error logged by receiver
	ready2, _ := shell.RecvFromBackend()
	if ready2.Model != "v2" {
		t.Fatalf("model = %q, want v2", ready2.Model)
	}
}

// TestHub_E2E_FullRebuild proves both shell and backend can restart
// through the hub without losing the hub itself.
func TestHub_E2E_FullRebuild(t *testing.T) {
	sock := filepath.Join(t.TempDir(), "hub-e2e-full.sock")
	hub, err := clutch.NewHub(sock)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	defer hub.Close()
	go hub.Run(ctx) //nolint:errcheck // test helper, error checked elsewhere
	time.Sleep(100 * time.Millisecond)

	// Round 1: both connect, exchange a message.
	shell1 := hubConnect(t, sock, "shell")
	be1 := hubConnect(t, sock, "backend")
	time.Sleep(50 * time.Millisecond)

	be1.SendToShell(clutch.BackendMsg{Type: clutch.BackendReady, HistoryLen: 10}) //nolint:errcheck // best-effort send, error logged by receiver
	r1, _ := shell1.RecvFromBackend()
	if r1.HistoryLen != 10 {
		t.Fatalf("history = %d, want 10", r1.HistoryLen)
	}

	// Both crash simultaneously.
	shell1.Close()
	be1.Close()
	time.Sleep(200 * time.Millisecond)

	// Round 2: both reconnect (rebuilt binaries).
	shell2 := hubConnect(t, sock, "shell")
	defer shell2.Close()
	be2 := hubConnect(t, sock, "backend")
	defer be2.Close()
	time.Sleep(50 * time.Millisecond)

	// Backend 2 announces with restored session.
	be2.SendToShell(clutch.BackendMsg{Type: clutch.BackendReady, HistoryLen: 15}) //nolint:errcheck // best-effort send, error logged by receiver
	r2, err := shell2.RecvFromBackend()
	if err != nil {
		t.Fatalf("shell2 recv: %v", err)
	}
	if r2.HistoryLen != 15 {
		t.Fatalf("history = %d, want 15 (session survived full rebuild)", r2.HistoryLen)
	}

	// Full message roundtrip works.
	shell2.SendToBackend(clutch.ShellMsg{Type: clutch.ShellPrompt, Text: "alive"}) //nolint:errcheck // best-effort send, error logged by receiver
	m, _ := be2.RecvFromShell()
	if m.Text != "alive" {
		t.Fatalf("be2 got %q, want 'alive'", m.Text)
	}
}
