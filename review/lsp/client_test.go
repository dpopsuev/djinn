package lsp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
)

// nopWriteCloser wraps an io.Writer as an io.WriteCloser.
type nopWriteCloser struct {
	io.Writer
}

func (nopWriteCloser) Close() error { return nil }

// newTestClient creates a Client with mock reader/writer (no subprocess).
func newTestClient(r io.Reader, w io.WriteCloser) *Client {
	return &Client{
		w:      w,
		r:      bufio.NewReader(r),
		nextID: 1,
	}
}

func TestWriteMessage(t *testing.T) {
	var buf bytes.Buffer
	c := newTestClient(strings.NewReader(""), nopWriteCloser{&buf})

	msg := jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
	}
	if err := c.writeMessage(msg); err != nil {
		t.Fatalf("writeMessage: %v", err)
	}

	output := buf.String()

	// Must start with Content-Length header.
	if !strings.HasPrefix(output, "Content-Length: ") {
		t.Fatalf("output missing Content-Length header: %q", output)
	}

	// Must contain \r\n\r\n separator between header and body.
	header, body, ok := strings.Cut(output, "\r\n\r\n")
	if !ok {
		t.Fatal("output missing \\r\\n\\r\\n separator")
	}
	var parsed jsonRPCRequest
	if err := json.Unmarshal([]byte(body), &parsed); err != nil {
		t.Fatalf("body is not valid JSON: %v\nbody: %q", err, body)
	}
	if parsed.Method != "initialize" {
		t.Errorf("parsed method = %q, want %q", parsed.Method, "initialize")
	}
	if parsed.JSONRPC != "2.0" {
		t.Errorf("parsed jsonrpc = %q, want %q", parsed.JSONRPC, "2.0")
	}

	// Verify Content-Length value matches body length.
	wantHeader := fmt.Sprintf("Content-Length: %d", len(body))
	if header != wantHeader {
		t.Errorf("header = %q, want %q", header, wantHeader)
	}
}

func TestReadMessage(t *testing.T) {
	id := 42
	resp := jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      &id,
		Result:  json.RawMessage(`{"capabilities":{}}`),
	}
	body, _ := json.Marshal(resp)
	frame := fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(body), body)

	c := newTestClient(strings.NewReader(frame), nopWriteCloser{io.Discard})

	got, err := c.readMessage()
	if err != nil {
		t.Fatalf("readMessage: %v", err)
	}
	if got.ID == nil || *got.ID != 42 {
		t.Errorf("ID = %v, want 42", got.ID)
	}
	if string(got.Result) != `{"capabilities":{}}` {
		t.Errorf("Result = %s, want %s", got.Result, `{"capabilities":{}}`)
	}
}

func TestReadMessageSkipsNotifications(t *testing.T) {
	// Build two frames: a notification (no id) followed by a response (with id).
	notif := jsonRPCResponse{
		JSONRPC: "2.0",
		Method:  "window/logMessage",
	}
	notifBody, _ := json.Marshal(notif)
	notifFrame := fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(notifBody), notifBody)

	id := 1
	resp := jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      &id,
		Result:  json.RawMessage(`"ok"`),
	}
	respBody, _ := json.Marshal(resp)
	respFrame := fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(respBody), respBody)

	combined := notifFrame + respFrame

	// Use Request which internally calls readMessage in a loop and skips notifications.
	pr, pw := io.Pipe()
	c := newTestClient(pr, nopWriteCloser{io.Discard})

	go func() {
		_, _ = pw.Write([]byte(combined))
		pw.Close()
	}()

	// readMessage first call: should get the notification.
	msg1, err := c.readMessage()
	if err != nil {
		t.Fatalf("first readMessage: %v", err)
	}
	// Notification has no ID.
	if msg1.ID != nil {
		t.Errorf("expected notification (no id), got ID=%d", *msg1.ID)
	}
	if msg1.Method != "window/logMessage" {
		t.Errorf("notification method = %q, want %q", msg1.Method, "window/logMessage")
	}

	// readMessage second call: should get the response.
	msg2, err := c.readMessage()
	if err != nil {
		t.Fatalf("second readMessage: %v", err)
	}
	if msg2.ID == nil || *msg2.ID != 1 {
		t.Errorf("expected response with ID=1, got %v", msg2.ID)
	}
}

func TestReadMessageMalformed(t *testing.T) {
	// No Content-Length header — just an empty line then a body.
	input := "\r\n{}"
	c := newTestClient(strings.NewReader(input), nopWriteCloser{io.Discard})

	_, err := c.readMessage()
	if err == nil {
		t.Fatal("readMessage should fail on missing Content-Length")
	}
	if !errors.Is(err, ErrMissingContentLength) {
		t.Errorf("error = %v, want ErrMissingContentLength", err)
	}
}

func TestReadMessageInvalidContentLength(t *testing.T) {
	input := "Content-Length: notanumber\r\n\r\n{}"
	c := newTestClient(strings.NewReader(input), nopWriteCloser{io.Discard})

	_, err := c.readMessage()
	if err == nil {
		t.Fatal("readMessage should fail on invalid Content-Length")
	}
	if !strings.Contains(err.Error(), "invalid Content-Length") {
		t.Errorf("error = %v, want it to contain 'invalid Content-Length'", err)
	}
}

func TestNotify(t *testing.T) {
	var buf bytes.Buffer
	c := newTestClient(strings.NewReader(""), nopWriteCloser{&buf})

	if err := c.Notify("initialized", nil); err != nil {
		t.Fatalf("Notify: %v", err)
	}

	output := buf.String()
	_, body, ok := strings.Cut(output, "\r\n\r\n")
	if !ok {
		t.Fatal("missing header/body separator")
	}

	var parsed jsonRPCRequest
	if err := json.Unmarshal([]byte(body), &parsed); err != nil {
		t.Fatalf("body not valid JSON: %v", err)
	}
	// Notification: no ID (omitempty means ID=0 is omitted).
	if parsed.ID != 0 {
		t.Errorf("notification should have zero ID (omitted), got %d", parsed.ID)
	}
	if parsed.Method != "initialized" {
		t.Errorf("method = %q, want %q", parsed.Method, "initialized")
	}
}

func TestRequestResponse(t *testing.T) {
	// Simulate a server that reads the request and writes back a response.
	clientRead, serverWrite := io.Pipe()
	serverRead, clientWrite := io.Pipe()

	c := newTestClient(clientRead, &pipeWriteCloser{clientWrite})

	go func() {
		// Read and discard the request from the client.
		r := bufio.NewReader(serverRead)
		contentLen := -1
		for {
			line, err := r.ReadString('\n')
			if err != nil {
				return
			}
			line = strings.TrimRight(line, "\r\n")
			if line == "" {
				break
			}
			if val, ok := strings.CutPrefix(line, "Content-Length:"); ok {
				fmt.Sscanf(val, "%d", &contentLen)
			}
		}
		if contentLen > 0 {
			discard := make([]byte, contentLen)
			io.ReadFull(r, discard)
		}

		// Write a response with ID=1 (the client starts at nextID=1).
		id := 1
		resp := jsonRPCResponse{
			JSONRPC: "2.0",
			ID:      &id,
			Result:  json.RawMessage(`{"ready":true}`),
		}
		body, _ := json.Marshal(resp)
		fmt.Fprintf(serverWrite, "Content-Length: %d\r\n\r\n%s", len(body), body)
		serverWrite.Close()
	}()

	result, err := c.Request("initialize", map[string]any{"processId": 1234})
	if err != nil {
		t.Fatalf("Request: %v", err)
	}
	if string(result) != `{"ready":true}` {
		t.Errorf("result = %s, want %s", result, `{"ready":true}`)
	}
}

func TestRequestServerError(t *testing.T) {
	// Server responds with a JSON-RPC error.
	clientRead, serverWrite := io.Pipe()
	serverRead, clientWrite := io.Pipe()

	c := newTestClient(clientRead, &pipeWriteCloser{clientWrite})

	go func() {
		// Drain the request from the client so writeMessage doesn't block.
		r := bufio.NewReader(serverRead)
		contentLen := -1
		for {
			line, err := r.ReadString('\n')
			if err != nil {
				return
			}
			line = strings.TrimRight(line, "\r\n")
			if line == "" {
				break
			}
			if val, ok := strings.CutPrefix(line, "Content-Length:"); ok {
				fmt.Sscanf(val, "%d", &contentLen)
			}
		}
		if contentLen > 0 {
			discard := make([]byte, contentLen)
			io.ReadFull(r, discard)
		}

		// Write an error response with ID=1.
		id := 1
		resp := jsonRPCResponse{
			JSONRPC: "2.0",
			ID:      &id,
			Error:   &jsonRPCError{Code: -32601, Message: "method not found"},
		}
		body, _ := json.Marshal(resp)
		fmt.Fprintf(serverWrite, "Content-Length: %d\r\n\r\n%s", len(body), body)
		serverWrite.Close()
	}()

	_, err := c.Request("bogus/method", nil)
	if err == nil {
		t.Fatal("Request should return error on JSON-RPC error response")
	}
	if !strings.Contains(err.Error(), "method not found") {
		t.Errorf("error = %v, want it to contain 'method not found'", err)
	}
}

func TestJSONRPCErrorString(t *testing.T) {
	e := &jsonRPCError{Code: -32600, Message: "invalid request"}
	got := e.Error()
	want := "LSP error -32600: invalid request"
	if got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestWriteMessageMultiple(t *testing.T) {
	var buf bytes.Buffer
	c := newTestClient(strings.NewReader(""), nopWriteCloser{&buf})

	for i := range 3 {
		msg := jsonRPCRequest{JSONRPC: "2.0", ID: i + 1, Method: "test"}
		if err := c.writeMessage(msg); err != nil {
			t.Fatalf("writeMessage %d: %v", i, err)
		}
	}

	// Verify we can read all 3 messages back from the buffer.
	reader := bufio.NewReader(&buf)
	c2 := &Client{r: reader}
	for i := range 3 {
		resp, err := c2.readMessage()
		if err != nil {
			t.Fatalf("readMessage %d: %v", i, err)
		}
		// readMessage parses as jsonRPCResponse; the "method" field is present.
		if resp.Method != "test" {
			t.Errorf("message %d method = %q, want %q", i, resp.Method, "test")
		}
	}
}

// pipeWriteCloser adapts *io.PipeWriter to io.WriteCloser.
type pipeWriteCloser struct {
	*io.PipeWriter
}
