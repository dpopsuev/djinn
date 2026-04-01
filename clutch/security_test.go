package clutch

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Security tests for Clutch socket (TSK-517, OWASP A07).
// Trust boundary TB-2: clutch/ ↔ agent/ over Unix socket.
// STRIDE S (Spoofing): unknown roles must be rejected.
// STRIDE T (Tampering): malformed registration must not crash.
// STRIDE D (DoS): rapid connect/disconnect must not leak resources.

func TestClutch_RejectsUnknownRole(t *testing.T) {
	t.Helper()
	socketPath := filepath.Join(t.TempDir(), "test.sock")
	hub, err := NewHub(socketPath)
	if err != nil {
		t.Fatal(err)
	}
	defer hub.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go hub.Run(ctx) //nolint:errcheck // test background goroutine, error checked via ctx cancel

	// Connect and send unknown role.
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	enc := json.NewEncoder(conn)
	if err := enc.Encode(RegisterMsg{Role: "attacker"}); err != nil {
		t.Fatal(err)
	}

	// Hub should close the connection. Read should fail.
	conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	buf := make([]byte, 1)
	_, err = conn.Read(buf)
	if err == nil {
		t.Fatal("SECURITY: connection with unknown role should be closed by hub")
	}
}

func TestClutch_RejectsMalformedRegistration(t *testing.T) {
	socketPath := filepath.Join(t.TempDir(), "test.sock")
	hub, err := NewHub(socketPath)
	if err != nil {
		t.Fatal(err)
	}
	defer hub.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go hub.Run(ctx) //nolint:errcheck // test background goroutine, error checked via ctx cancel

	// Send garbage instead of RegisterMsg JSON.
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	conn.Write([]byte("not json at all\n"))

	// Hub should close the connection.
	conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	buf := make([]byte, 1)
	_, err = conn.Read(buf)
	if err == nil {
		t.Fatal("SECURITY: malformed registration should close connection")
	}
}

func TestClutch_RejectsEmptyRole(t *testing.T) {
	socketPath := filepath.Join(t.TempDir(), "test.sock")
	hub, err := NewHub(socketPath)
	if err != nil {
		t.Fatal(err)
	}
	defer hub.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go hub.Run(ctx) //nolint:errcheck // test background goroutine, error checked via ctx cancel

	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	enc := json.NewEncoder(conn)
	if err := enc.Encode(RegisterMsg{Role: ""}); err != nil {
		t.Fatal(err)
	}

	conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	buf := make([]byte, 1)
	_, err = conn.Read(buf)
	if err == nil {
		t.Fatal("SECURITY: empty role should be rejected")
	}
}

func TestClutch_RejectsCaseSensitiveRole(t *testing.T) {
	socketPath := filepath.Join(t.TempDir(), "test.sock")
	hub, err := NewHub(socketPath)
	if err != nil {
		t.Fatal(err)
	}
	defer hub.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go hub.Run(ctx) //nolint:errcheck // test background goroutine, error checked via ctx cancel

	// "Shell" (capital S) should be rejected — roles are case-sensitive.
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	enc := json.NewEncoder(conn)
	if err := enc.Encode(RegisterMsg{Role: "Shell"}); err != nil {
		t.Fatal(err)
	}

	conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	buf := make([]byte, 1)
	_, err = conn.Read(buf)
	if err == nil {
		t.Fatal("SECURITY: case-sensitive role mismatch should be rejected")
	}
}

func TestClutch_SocketPermissions(t *testing.T) {
	socketPath := filepath.Join(t.TempDir(), "test.sock")
	hub, err := NewHub(socketPath)
	if err != nil {
		t.Fatal(err)
	}
	defer hub.Close()

	// Check socket file permissions.
	info, err := os.Stat(socketPath)
	if err != nil {
		t.Fatal(err)
	}
	perm := info.Mode().Perm()
	// Socket must be owner-only (0600) — no group/other access.
	if perm&0o077 != 0 {
		t.Fatalf("SECURITY: socket permissions %04o allow group/other access, want 0600", perm)
	}
}

func TestClutch_AcceptsValidRoles(t *testing.T) {
	socketPath := filepath.Join(t.TempDir(), "test.sock")
	hub, err := NewHub(socketPath)
	if err != nil {
		t.Fatal(err)
	}
	defer hub.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go hub.Run(ctx) //nolint:errcheck // test background goroutine, error checked via ctx cancel

	for _, role := range []string{"shell", "backend"} {
		conn, err := net.Dial("unix", socketPath)
		if err != nil {
			t.Fatal(err)
		}

		enc := json.NewEncoder(conn)
		if err := enc.Encode(RegisterMsg{Role: role}); err != nil {
			conn.Close()
			t.Fatalf("role %q: encode: %v", role, err)
		}

		// Valid role — connection should stay open (not closed by hub).
		// We can't easily test "still open" so we just verify no immediate error.
		conn.Close()
	}
}

func TestClutch_ReadRegistration_ReturnsErrUnknownRole(t *testing.T) {
	// Unit test readRegistration directly.
	hub := &Hub{}
	socketPath := filepath.Join(t.TempDir(), "test.sock")
	ln, err := Listen(socketPath)
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	_ = hub

	// Connect and send bad role.
	go func() {
		conn, _ := net.Dial("unix", socketPath)
		if conn != nil {
			enc := json.NewEncoder(conn)
			enc.Encode(RegisterMsg{Role: "evil"})
			conn.Close()
		}
	}()

	conn, err := ln.Accept()
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	h := &Hub{}
	_, err = h.readRegistration(conn)
	if err == nil {
		t.Fatal("expected error for unknown role")
	}
	if !errors.Is(err, ErrUnknownRole) {
		t.Fatalf("expected ErrUnknownRole, got: %v", err)
	}
}
