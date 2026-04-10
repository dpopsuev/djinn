// http_referee.go — Referee strategy for HTTP service scenarios.
//
// EXTERNAL harness — sits outside Djinn, verifies what the agent built.
// Uses tool.Executor for operations (same building blocks as agents).
// Deterministic — no LLM, only programmatic checks.
package crucible

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// HTTPReferee verifies an HTTP service scenario:
// compile → run → curl → verify response.
type HTTPReferee struct {
	Port    int
	Path    string
	Timeout time.Duration
}

var _ Referee = (*HTTPReferee)(nil)

// NewHTTPReferee creates a referee with sensible defaults.
func NewHTTPReferee() *HTTPReferee {
	return &HTTPReferee{
		Port:    8080,
		Path:    "/health",
		Timeout: 10 * time.Second,
	}
}

func (r *HTTPReferee) Check(ctx context.Context, _, projectPath string) (CheckResult, error) {
	var errs []string

	// 1. Compile
	buildCmd := exec.CommandContext(ctx, "go", "build", "-o", "server", ".")
	buildCmd.Dir = projectPath
	buildOut, err := buildCmd.CombinedOutput()
	if err != nil {
		errs = append(errs, fmt.Sprintf("build: %s\n%s", err, string(buildOut)))
		return CheckResult{Pass: false, Score: 0, Errors: errs}, nil
	}

	// 2. Find a free port if default is taken
	port := r.Port
	if ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port)); err != nil {
		// Port taken — find a free one
		ln2, err2 := net.Listen("tcp", "127.0.0.1:0") //nolint:gosec // test harness, localhost only
		if err2 != nil {
			errs = append(errs, fmt.Sprintf("no free port: %s", err2))
			return CheckResult{Pass: false, Score: 0.15, Errors: errs}, nil
		}
		port = ln2.Addr().(*net.TCPAddr).Port
		ln2.Close()
	} else {
		ln.Close()
	}

	// 3. Run
	binary := filepath.Join(projectPath, "server")
	serverCmd := exec.CommandContext(ctx, binary)
	serverCmd.Dir = projectPath
	serverCmd.Env = append(os.Environ(), fmt.Sprintf("PORT=%d", port))
	serverCmd.Stdout = io.Discard
	serverCmd.Stderr = os.Stderr
	if err := serverCmd.Start(); err != nil {
		errs = append(errs, fmt.Sprintf("start: %s", err))
		return CheckResult{Pass: false, Score: 0.1, Errors: errs}, nil
	}
	defer func() {
		serverCmd.Process.Kill() //nolint:errcheck // cleanup
		serverCmd.Wait()         //nolint:errcheck // already killed
	}()

	// 4. Wait for port
	addr := fmt.Sprintf("localhost:%d", port)
	if !waitForPort(addr, r.Timeout) {
		errs = append(errs, fmt.Sprintf("server not listening on %s after %v", addr, r.Timeout))
		return CheckResult{Pass: false, Score: 0.2, Errors: errs}, nil
	}

	// 4. Curl
	url := fmt.Sprintf("http://%s%s", addr, r.Path)
	resp, err := http.Get(url) //nolint:gosec // harness, controlled URL
	if err != nil {
		errs = append(errs, fmt.Sprintf("GET %s: %s", url, err))
		return CheckResult{Pass: false, Score: 0.3, Errors: errs}, nil
	}
	defer resp.Body.Close()

	// 5. Status code
	if resp.StatusCode != http.StatusOK {
		errs = append(errs, fmt.Sprintf("status %d, want 200", resp.StatusCode))
		return CheckResult{Pass: false, Score: 0.4, Errors: errs}, nil
	}

	// 6. JSON body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		errs = append(errs, fmt.Sprintf("read body: %s", err))
		return CheckResult{Pass: false, Score: 0.5, Errors: errs}, nil
	}

	var parsed map[string]any
	if err := json.Unmarshal(body, &parsed); err != nil {
		errs = append(errs, fmt.Sprintf("not valid JSON: %s", err))
		return CheckResult{Pass: false, Score: 0.6, Errors: errs}, nil
	}

	if _, ok := parsed["status"]; !ok {
		errs = append(errs, fmt.Sprintf("missing 'status' key: %s", string(body)))
		return CheckResult{Pass: false, Score: 0.8, Errors: errs}, nil
	}

	return CheckResult{Pass: true, Score: 1.0}, nil
}

func waitForPort(addr string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 100*time.Millisecond)
		if err == nil {
			conn.Close()
			return true
		}
		time.Sleep(50 * time.Millisecond)
	}
	return false
}
