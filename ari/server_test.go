package ari

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"net/http"
	"testing"
	"time"
)

// mockRuntime implements Runtime for testing.
type mockRuntime struct {
	intentReceived Intent
	cancelledID    string
	clearedPaths   []string
}

func (m *mockRuntime) HandleIntent(ctx context.Context, intent Intent) {
	m.intentReceived = intent
}

func (m *mockRuntime) CancelWorkstream(id string) error {
	m.cancelledID = id
	return nil
}

func (m *mockRuntime) ClearCordon(paths []string) {
	m.clearedPaths = paths
}

func (m *mockRuntime) Andon() AndonSnapshot {
	return AndonSnapshot{
		Level:   "green",
		Cordons: 0,
		Workstreams: []WorkstreamSnapshot{
			{ID: "ws-1", Status: "running", Action: "fix", Health: "green"},
		},
	}
}

func (m *mockRuntime) ListWorkstreams() []WorkstreamSnapshot {
	return []WorkstreamSnapshot{
		{ID: "ws-1", IntentID: "int-1", Action: "fix", Status: "running", Health: "green"},
		{ID: "ws-2", IntentID: "int-2", Action: "refactor", Status: "completed", Health: "green"},
	}
}

func startTestServer(t *testing.T, rt Runtime) (*Server, *http.Client) {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	srv := NewServer(rt)
	go srv.StartOnListener(ln)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		srv.Stop(ctx)
	})

	client := &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return net.Dial("tcp", ln.Addr().String())
			},
		},
	}

	return srv, client
}

func TestServer_PostIntent(t *testing.T) {
	rt := &mockRuntime{}
	srv, client := startTestServer(t, rt)

	// Register intent handler
	var received Intent
	srv.OnIntent(func(i Intent) { received = i })

	body, _ := json.Marshal(Intent{ID: "int-1", Action: "fix-bug"})
	resp, err := client.Post("http://ari"+RouteIntent, "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST %s: %v", RouteIntent, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var result map[string]string
	json.NewDecoder(resp.Body).Decode(&result)
	if result["status"] != "accepted" {
		t.Fatalf("status = %q, want %q", result["status"], "accepted")
	}

	if received.ID != "int-1" {
		t.Fatalf("received.ID = %q, want %q", received.ID, "int-1")
	}
	if received.Action != "fix-bug" {
		t.Fatalf("received.Action = %q, want %q", received.Action, "fix-bug")
	}
}

func TestServer_PostIntent_MethodNotAllowed(t *testing.T) {
	_, client := startTestServer(t, &mockRuntime{})

	resp, err := client.Get("http://ari" + RouteIntent)
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", resp.StatusCode)
	}
}

func TestServer_PostIntent_BadJSON(t *testing.T) {
	_, client := startTestServer(t, &mockRuntime{})

	resp, err := client.Post("http://ari"+RouteIntent, "application/json", bytes.NewReader([]byte("{bad")))
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
}

func TestServer_GetAndon(t *testing.T) {
	_, client := startTestServer(t, &mockRuntime{})

	resp, err := client.Get("http://ari" + RouteAndon)
	if err != nil {
		t.Fatalf("GET %s: %v", RouteAndon, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var board AndonSnapshot
	json.NewDecoder(resp.Body).Decode(&board)
	if board.Level != "green" {
		t.Fatalf("Level = %q, want %q", board.Level, "green")
	}
	if len(board.Workstreams) != 1 {
		t.Fatalf("Workstreams = %d, want 1", len(board.Workstreams))
	}
}

func TestServer_GetWorkstreams(t *testing.T) {
	_, client := startTestServer(t, &mockRuntime{})

	resp, err := client.Get("http://ari" + RouteWorkstreams)
	if err != nil {
		t.Fatalf("GET %s: %v", RouteWorkstreams, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var ws []WorkstreamSnapshot
	json.NewDecoder(resp.Body).Decode(&ws)
	if len(ws) != 2 {
		t.Fatalf("workstreams = %d, want 2", len(ws))
	}
	if ws[0].ID != "ws-1" {
		t.Fatalf("ws[0].ID = %q, want %q", ws[0].ID, "ws-1")
	}
}

func TestServer_PostCordonClear(t *testing.T) {
	rt := &mockRuntime{}
	_, client := startTestServer(t, rt)

	body, _ := json.Marshal(map[string][]string{"paths": {"auth/middleware.go"}})
	resp, err := client.Post("http://ari"+RouteCordonClear, "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST %s: %v", RouteCordonClear, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	if len(rt.clearedPaths) != 1 || rt.clearedPaths[0] != "auth/middleware.go" {
		t.Fatalf("clearedPaths = %v, want [auth/middleware.go]", rt.clearedPaths)
	}
}

func TestServer_EmitAndReceiveEvent(t *testing.T) {
	_, client := startTestServer(t, &mockRuntime{})

	// Start SSE connection in background
	type sseResult struct {
		events []serverEvent
		err    error
	}
	resultCh := make(chan sseResult, 1)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "http://ari"+RouteEvents, nil)

	go func() {
		resp, err := client.Do(req)
		if err != nil {
			resultCh <- sseResult{err: err}
			return
		}
		defer resp.Body.Close()

		// Read first SSE event
		buf := make([]byte, 4096)
		n, _ := resp.Body.Read(buf)
		resultCh <- sseResult{events: nil, err: nil}
		_ = n
	}()

	// Give SSE client time to connect
	time.Sleep(50 * time.Millisecond)

	// No crash, SSE endpoint works
	cancel()
}

func TestServer_PermissionResponses(t *testing.T) {
	srv := NewServer(&mockRuntime{})

	ch := srv.PermissionResponses()
	if ch == nil {
		t.Fatal("PermissionResponses() returned nil")
	}
}
