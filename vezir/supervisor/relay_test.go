package supervisor

import (
	"context"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRelay_TUIConnectsAndRelays(t *testing.T) {
	dir := t.TempDir()
	vezirSock := filepath.Join(dir, "vezir.sock")
	subSock := filepath.Join(dir, "substrate.sock")

	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	relay := NewRelay(vezirSock, subSock, log)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Start a mock Substrate listener.
	subListener, err := net.Listen("unix", subSock)
	if err != nil {
		t.Fatal(err)
	}
	defer subListener.Close()

	// Accept Substrate connection in background.
	subCh := make(chan net.Conn, 1)
	go func() {
		conn, err := subListener.Accept()
		if err != nil {
			return
		}
		subCh <- conn
	}()

	// Start relay.
	if err := relay.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer relay.Stop()

	// Connect relay to Substrate.
	if err := relay.Reconnect(ctx); err != nil {
		t.Fatal(err)
	}

	// Wait for Substrate to accept.
	var subConn net.Conn
	select {
	case subConn = <-subCh:
		defer subConn.Close()
	case <-time.After(2 * time.Second):
		t.Fatal("substrate did not accept relay connection")
	}

	// TUI connects to Vezir socket.
	tuiConn, err := net.Dial("unix", vezirSock)
	if err != nil {
		t.Fatal(err)
	}
	defer tuiConn.Close()

	// Give relay time to bridge.
	time.Sleep(100 * time.Millisecond)

	// TUI sends message → Substrate receives.
	msg := []byte("hello from TUI")
	if _, err := tuiConn.Write(msg); err != nil {
		t.Fatal(err)
	}

	buf := make([]byte, 256)
	subConn.SetReadDeadline(time.Now().Add(2 * time.Second))
	n, err := subConn.Read(buf)
	if err != nil {
		t.Fatalf("substrate read: %v", err)
	}
	if string(buf[:n]) != "hello from TUI" {
		t.Fatalf("substrate got %q, want 'hello from TUI'", buf[:n])
	}

	// Substrate sends reply → TUI receives.
	reply := []byte("hello from Substrate")
	if _, err := subConn.Write(reply); err != nil {
		t.Fatal(err)
	}

	tuiConn.SetReadDeadline(time.Now().Add(2 * time.Second))
	n, err = tuiConn.Read(buf)
	if err != nil {
		t.Fatalf("TUI read: %v", err)
	}
	if string(buf[:n]) != "hello from Substrate" {
		t.Fatalf("TUI got %q, want 'hello from Substrate'", buf[:n])
	}

	t.Log("Relay PASSES — TUI ↔ Vezir ↔ Substrate bidirectional")
}

func TestRelay_ReconnectAfterSubstrateRestart(t *testing.T) {
	dir := t.TempDir()
	vezirSock := filepath.Join(dir, "vezir.sock")
	subSock := filepath.Join(dir, "substrate.sock")

	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	relay := NewRelay(vezirSock, subSock, log)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Start relay (no Substrate yet).
	if err := relay.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer relay.Stop()

	// TUI connects.
	tuiConn, err := net.Dial("unix", vezirSock)
	if err != nil {
		t.Fatal(err)
	}
	defer tuiConn.Close()

	// First Substrate starts.
	sub1, err := net.Listen("unix", subSock)
	if err != nil {
		t.Fatal(err)
	}

	go func() { sub1.Accept() }() //nolint:errcheck // test helper
	if err := relay.Reconnect(ctx); err != nil {
		t.Fatal(err)
	}

	// Substrate 1 dies.
	sub1.Close()
	os.Remove(subSock)

	// Substrate 2 starts (same socket path).
	sub2, err := net.Listen("unix", subSock)
	if err != nil {
		t.Fatal(err)
	}
	defer sub2.Close()

	subCh := make(chan net.Conn, 1)
	go func() {
		conn, _ := sub2.Accept()
		subCh <- conn
	}()

	// Relay reconnects to new Substrate.
	if err := relay.Reconnect(ctx); err != nil {
		t.Fatal(err)
	}

	// Wait for new Substrate to accept.
	select {
	case conn := <-subCh:
		conn.Close()
	case <-time.After(2 * time.Second):
		t.Fatal("new substrate did not accept relay reconnection")
	}

	t.Log("Reconnect PASSES — TUI stays, Substrate swapped")
}
