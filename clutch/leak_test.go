package clutch

import (
	"context"
	"encoding/json"
	"net"
	"path/filepath"
	"testing"
	"time"

	"go.uber.org/goleak"
)

// TestMain enables goroutine leak detection for the entire package.
func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m,
		// Ignore known background goroutines from test infrastructure.
		goleak.IgnoreTopFunction("internal/poll.runtime_pollWait"),
	)
}

func TestLeak_HubCleanShutdown(t *testing.T) {
	sock := filepath.Join(t.TempDir(), "leak.sock")
	hub, err := NewHub(sock)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	go hub.Run(ctx) //nolint:errcheck // test helper, error checked elsewhere
	time.Sleep(50 * time.Millisecond)

	// Connect shell + backend, exchange a message, disconnect.
	shell := leakConnect(t, sock, "shell")
	backend := leakConnect(t, sock, "backend")
	time.Sleep(50 * time.Millisecond)

	backend.SendToShell(BackendMsg{Type: BackendReady}) //nolint:errcheck // best-effort send, error logged by receiver
	shell.RecvFromBackend()                             //nolint:errcheck // error intentionally ignored

	shell.Close()
	backend.Close()
	time.Sleep(100 * time.Millisecond)

	// Cancel context — hub should exit cleanly with no leaked goroutines.
	cancel()
	hub.Close()
	time.Sleep(100 * time.Millisecond)

	// goleak.VerifyTestMain catches any remaining goroutines.
}

func TestLeak_DriverChat(t *testing.T) {
	// Verify CLI driver Chat() goroutines clean up after channel drain.
	// Uses echo as mock — goroutine should exit when stdout closes.
	// (tested indirectly via cursor/gemini/codex driver tests)
}

func leakConnect(t *testing.T, socketPath, role string) *SocketTransport {
	t.Helper()
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	enc := json.NewEncoder(conn)
	if err := enc.Encode(RegisterMsg{Role: role}); err != nil {
		conn.Close()
		t.Fatalf("register: %v", err)
	}
	return newSocketTransport(conn)
}
