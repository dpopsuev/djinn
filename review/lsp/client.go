// client.go — Minimal JSON-RPC 2.0 stdio client for LSP servers.
//
// Adapted from Locus internal/survey/lsp_client.go.
// Spawns an LSP server subprocess, handles Content-Length framing,
// and provides Request/Notify methods for LSP protocol communication.
package lsp

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"sync"
)

// ErrMissingContentLength indicates a malformed LSP message.
var ErrMissingContentLength = errors.New("missing Content-Length header")

// Client communicates with an LSP server over stdin/stdout.
type Client struct {
	cmd    *exec.Cmd
	w      io.WriteCloser
	r      *bufio.Reader
	mu     sync.Mutex
	nextID int
}

// NewClient spawns an LSP server and returns a connected client.
func NewClient(command string, args ...string) (*Client, error) {
	cmd := exec.Command(command, args...) //nolint:gosec // LSP server command is from trusted config
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("lsp stdin: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("lsp stdout: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("lsp start %s: %w", command, err)
	}
	return &Client{
		cmd:    cmd,
		w:      stdin,
		r:      bufio.NewReader(stdout),
		nextID: 1,
	}, nil
}

// Request sends a JSON-RPC request and reads the response,
// skipping any interleaved notifications from the server.
func (c *Client) Request(method string, params any) (json.RawMessage, error) {
	c.mu.Lock()
	id := c.nextID
	c.nextID++
	c.mu.Unlock()

	req := jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}
	if err := c.writeMessage(req); err != nil {
		return nil, fmt.Errorf("lsp request %s: %w", method, err)
	}

	for {
		resp, err := c.readMessage()
		if err != nil {
			return nil, fmt.Errorf("lsp response %s: %w", method, err)
		}
		// Skip server-initiated notifications (no id) and server requests (has method).
		if resp.ID == nil || resp.Method != "" {
			continue
		}
		if *resp.ID == id {
			if resp.Error != nil {
				return nil, resp.Error
			}
			return resp.Result, nil
		}
	}
}

// Notify sends a JSON-RPC notification (no response expected).
func (c *Client) Notify(method string, params any) error {
	req := jsonRPCRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
	}
	return c.writeMessage(req)
}

// Close shuts down the LSP server gracefully.
func (c *Client) Close() error {
	// Send shutdown request, then exit notification.
	_, _ = c.Request("shutdown", nil)
	_ = c.Notify("exit", nil)
	_ = c.w.Close()
	return c.cmd.Wait()
}

type jsonRPCRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id,omitempty"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

type jsonRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *int            `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *jsonRPCError   `json:"error,omitempty"`
}

type jsonRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *jsonRPCError) Error() string {
	return fmt.Sprintf("LSP error %d: %s", e.Code, e.Message)
}

func (c *Client) writeMessage(msg any) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(body))
	if _, err := io.WriteString(c.w, header); err != nil {
		return err
	}
	_, err = c.w.Write(body)
	return err
}

func (c *Client) readMessage() (*jsonRPCResponse, error) {
	contentLen := -1
	for {
		line, err := c.r.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("reading header: %w", err)
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		if val, ok := strings.CutPrefix(line, "Content-Length:"); ok {
			contentLen, err = strconv.Atoi(strings.TrimSpace(val))
			if err != nil {
				return nil, fmt.Errorf("invalid Content-Length %q: %w", val, err)
			}
		}
	}
	if contentLen < 0 {
		return nil, ErrMissingContentLength
	}
	body := make([]byte, contentLen)
	if _, err := io.ReadFull(c.r, body); err != nil {
		return nil, fmt.Errorf("reading body: %w", err)
	}
	var resp jsonRPCResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	return &resp, nil
}
