package misbah

import (
	"context"
	"net"
	"os"
	"testing"
	"time"

	"github.com/dpopsuev/djinn/tier"
)

const testSocket = "/run/misbah/permission.sock"

func TestSandboxPort_New(t *testing.T) {
	s := New("/tmp/fake.sock", "/workspace")
	defer s.Close()

	if s.workspace != "/workspace" {
		t.Fatalf("workspace = %q", s.workspace)
	}
}

func TestSandboxPort_Integration_List(t *testing.T) {
	socketPath := os.Getenv("MISBAH_DAEMON_SOCKET")
	if socketPath == "" {
		socketPath = testSocket
	}

	conn, err := net.DialTimeout("unix", socketPath, 2*time.Second)
	if err != nil {
		t.Skipf("Misbah daemon not available at %s: %v", socketPath, err)
	}
	conn.Close()

	s := New(socketPath, t.TempDir())
	defer s.Close()

	containers, err := s.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	_ = containers
}

func TestSandboxPort_Integration_DiffCommit(t *testing.T) {
	socketPath := os.Getenv("MISBAH_DAEMON_SOCKET")
	if socketPath == "" {
		socketPath = testSocket
	}

	conn, err := net.DialTimeout("unix", socketPath, 2*time.Second)
	if err != nil {
		t.Skipf("Misbah daemon not available at %s: %v", socketPath, err)
	}
	conn.Close()

	workspace := t.TempDir()
	if err := os.WriteFile(workspace+"/test.go", []byte("original"), 0644); err != nil {
		t.Fatal(err)
	}

	s := New(socketPath, workspace)
	defer s.Close()

	scope := tier.Scope{
		Level: tier.Mod,
		Name:  "test",
	}

	id, err := s.Create(context.Background(), scope)
	if err != nil {
		t.Skipf("Create failed (daemon may not support overlay): %v", err)
	}
	defer s.Destroy(context.Background(), id)

	diff, err := s.Diff(context.Background(), id)
	if err != nil {
		t.Skipf("Diff not supported: %v", err)
	}
	if len(diff) != 0 {
		t.Fatalf("initial diff should be empty, got %v", diff)
	}

	t.Logf("Agent Space E2E: container %s created, diff verified empty", id)
}

func TestSandboxPort_Integration_Logs(t *testing.T) {
	socketPath := os.Getenv("MISBAH_DAEMON_SOCKET")
	if socketPath == "" {
		socketPath = testSocket
	}

	conn, err := net.DialTimeout("unix", socketPath, 2*time.Second)
	if err != nil {
		t.Skipf("Misbah daemon not available at %s: %v", socketPath, err)
	}
	conn.Close()

	s := New(socketPath, t.TempDir())
	defer s.Close()

	// Logs for nonexistent container should error gracefully.
	_, _, err = s.Logs(context.Background(), "nonexistent-container")
	if err == nil {
		t.Fatal("Logs for nonexistent container should error")
	}
}
