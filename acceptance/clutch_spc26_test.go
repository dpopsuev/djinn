// clutch_spc26_test.go — acceptance tests for SPC-26 Shell/Backend Split.
//
// These test the clutch protocol from the shell's perspective:
// handshake, disconnect detection, reconnect, and prompt queueing.
// All tests use real Unix sockets — no mocks.
package acceptance

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/dpopsuev/djinn/clutch"
	"github.com/dpopsuev/djinn/driver"
	"github.com/dpopsuev/djinn/session"
	"github.com/dpopsuev/djinn/testkit/stubs"
	"github.com/dpopsuev/djinn/tools/builtin"
)

func startTestBackend(t *testing.T, ctx context.Context, sock string, model string, responses ...driver.Message) chan error {
	t.Helper()
	done := make(chan error, 1)
	go func() {
		transport, err := clutch.Connect(sock)
		if err != nil {
			done <- err
			return
		}
		defer transport.Close()
		done <- clutch.RunBackend(ctx, transport, clutch.BackendConfig{
			Driver:   stubs.NewStubChatDriver(responses...),
			Tools:    builtin.NewRegistry(),
			Session:  session.New("test", model, "/workspace"),
			MaxTurns: 5,
		})
	}()
	return done
}

func TestSPC26_SocketHandshake(t *testing.T) {
	// SPC-26: Given the shell is running
	// When the backend connects via Unix socket
	// Then the backend sends BackendReady with session state
	sock := filepath.Join(t.TempDir(), "handshake.sock")

	ln, err := clutch.Listen(sock)
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	backendDone := startTestBackend(t, ctx, sock, "claude-opus-4-6")

	shell, err := ln.Accept()
	if err != nil {
		t.Fatal(err)
	}
	defer shell.Close()

	msg, err := shell.RecvFromBackend()
	if err != nil {
		t.Fatal(err)
	}
	if msg.Type != clutch.BackendReady {
		t.Fatalf("expected BackendReady, got %q", msg.Type)
	}
	if msg.Version != clutch.ProtocolVersion {
		t.Fatalf("version = %d", msg.Version)
	}
	if msg.Model != "claude-opus-4-6" {
		t.Fatalf("model = %q", msg.Model)
	}

	shell.SendToBackend(clutch.ShellMsg{Type: clutch.ShellQuit}) //nolint:errcheck
	<-backendDone
}

func TestSPC26_BackendDisconnect_ShellPreserves(t *testing.T) {
	// SPC-26: Given the backend crashes
	// When the shell detects disconnect
	// Then the shell preserves all received messages
	sock := filepath.Join(t.TempDir(), "disconnect.sock")

	ln, err := clutch.Listen(sock)
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	go func() {
		transport, _ := clutch.Connect(sock)
		transport.SendToShell(clutch.BackendMsg{Type: clutch.BackendReady}) //nolint:errcheck
		transport.SendToShell(clutch.BackendMsg{Type: clutch.BackendText, Text: "partial response"}) //nolint:errcheck
		transport.Close()
	}()

	shell, _ := ln.Accept()

	var messages []clutch.BackendMsg
	for {
		msg, err := shell.RecvFromBackend()
		if err != nil {
			break
		}
		messages = append(messages, msg)
	}
	shell.Close()

	if len(messages) < 2 {
		t.Fatalf("expected 2+ messages before crash, got %d", len(messages))
	}
	if messages[1].Text != "partial response" {
		t.Fatalf("text = %q", messages[1].Text)
	}
}

func TestSPC26_BackendReconnect_SessionPreserved(t *testing.T) {
	// SPC-26: Given the developer rebuilds the backend
	// When the new backend connects
	// Then it sends new session_state and the shell continues
	sock := filepath.Join(t.TempDir(), "reconnect.sock")

	ln, err := clutch.Listen(sock)
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Backend 1
	b1 := startTestBackend(t, ctx, sock, "model-v1",
		driver.Message{Role: "assistant", Content: "v1"})
	shell1, _ := ln.Accept()
	ready1, _ := shell1.RecvFromBackend()
	if ready1.Model != "model-v1" {
		t.Fatalf("b1 model = %q", ready1.Model)
	}
	shell1.SendToBackend(clutch.ShellMsg{Type: clutch.ShellQuit}) //nolint:errcheck
	<-b1
	shell1.Close()

	// Backend 2 — hot-swap
	b2 := startTestBackend(t, ctx, sock, "model-v2",
		driver.Message{Role: "assistant", Content: "v2"})
	shell2, err := ln.Accept()
	if err != nil {
		t.Fatal(err)
	}
	defer shell2.Close()

	ready2, _ := shell2.RecvFromBackend()
	if ready2.Model != "model-v2" {
		t.Fatalf("b2 model = %q — hot-swap should show new model", ready2.Model)
	}

	shell2.SendToBackend(clutch.ShellMsg{Type: clutch.ShellQuit}) //nolint:errcheck
	<-b2
}

func TestSPC26_ShellWithoutBackend_Queues(t *testing.T) {
	// SPC-26: Given the shell is running without a backend
	// When the backend connects later
	// Then the shell can send the queued prompt
	sock := filepath.Join(t.TempDir(), "queue.sock")

	ln, err := clutch.Listen(sock)
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Backend connects after delay.
	go func() {
		time.Sleep(100 * time.Millisecond)
		startTestBackend(t, ctx, sock, "model",
			driver.Message{Role: "assistant", Content: "got it"})
	}()

	// Shell blocks on Accept.
	shell, err := ln.Accept()
	if err != nil {
		t.Fatal(err)
	}
	defer shell.Close()

	ready, _ := shell.RecvFromBackend()
	if ready.Type != clutch.BackendReady {
		t.Fatal("expected ready")
	}

	// Send queued prompt.
	shell.SendToBackend(clutch.ShellMsg{Type: clutch.ShellPrompt, Text: "queued"}) //nolint:errcheck

	var gotText bool
	for range 20 {
		msg, err := shell.RecvFromBackend()
		if err != nil {
			break
		}
		if msg.Type == clutch.BackendText {
			gotText = true
		}
		if msg.Type == clutch.BackendDone {
			break
		}
	}
	if !gotText {
		t.Fatal("queued prompt should produce response after backend connects")
	}

	shell.SendToBackend(clutch.ShellMsg{Type: clutch.ShellQuit}) //nolint:errcheck
}
