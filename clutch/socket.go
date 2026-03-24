// socket.go — Unix domain socket transport for shell ↔ backend communication.
//
// Each message is a JSON object on a single line, terminated by '\n'.
// The shell calls Listen() to create a server-side listener.
// The backend calls Connect() to connect as a client.
//
// The SocketTransport implements clutch.Transport — the same interface
// as ChannelTransport, so the shell and backend code doesn't change.
package clutch

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"sync"
)

// SocketTransport connects shell ↔ backend over a Unix domain socket.
// Bidirectional: both sides can send and receive both message types.
// Thread-safe: concurrent sends are serialized via mutex.
type SocketTransport struct {
	conn   net.Conn
	enc    *json.Encoder
	dec    *json.Decoder
	mu     sync.Mutex // serializes writes
	closed bool
}

// newSocketTransport wraps an established connection.
func newSocketTransport(conn net.Conn) *SocketTransport {
	return &SocketTransport{
		conn: conn,
		enc:  json.NewEncoder(conn),
		dec:  json.NewDecoder(conn),
	}
}

// Connect dials a Unix socket (backend → shell).
func Connect(socketPath string) (*SocketTransport, error) {
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return nil, fmt.Errorf("connect to %s: %w", socketPath, err)
	}
	return newSocketTransport(conn), nil
}

// SocketListener listens for backend connections on a Unix socket.
// The shell creates a listener and calls Accept() to wait for the backend.
type SocketListener struct {
	listener net.Listener
	path     string
}

// Listen creates a Unix socket server (shell side).
// Removes any existing socket file before binding.
func Listen(socketPath string) (*SocketListener, error) {
	// Remove stale socket file from previous run.
	os.Remove(socketPath) //nolint:errcheck
	ln, err := net.Listen("unix", socketPath)
	if err != nil {
		return nil, fmt.Errorf("listen on %s: %w", socketPath, err)
	}
	return &SocketListener{listener: ln, path: socketPath}, nil
}

// Accept waits for a backend to connect and returns a transport.
// Call this again after a disconnect to accept a reconnecting backend (hot-swap).
func (sl *SocketListener) Accept() (*SocketTransport, error) {
	conn, err := sl.listener.Accept()
	if err != nil {
		return nil, fmt.Errorf("accept: %w", err)
	}
	return newSocketTransport(conn), nil
}

// Close stops listening and removes the socket file.
func (sl *SocketListener) Close() error {
	err := sl.listener.Close()
	os.Remove(sl.path) //nolint:errcheck
	return err
}

// Addr returns the listener address (for logging/tests).
func (sl *SocketListener) Addr() string {
	return sl.path
}

// --- Transport interface implementation ---

// SendToBackend sends a shell message to the backend.
func (t *SocketTransport) SendToBackend(msg ShellMsg) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.closed {
		return ErrClosed
	}
	return t.enc.Encode(msg)
}

// RecvFromBackend receives a backend message (blocks until one arrives).
func (t *SocketTransport) RecvFromBackend() (BackendMsg, error) {
	var msg BackendMsg
	if err := t.dec.Decode(&msg); err != nil {
		return msg, fmt.Errorf("recv backend: %w", err)
	}
	return msg, nil
}

// SendToShell sends a backend message to the shell.
func (t *SocketTransport) SendToShell(msg BackendMsg) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.closed {
		return ErrClosed
	}
	return t.enc.Encode(msg)
}

// RecvFromShell receives a shell message (blocks until one arrives).
func (t *SocketTransport) RecvFromShell() (ShellMsg, error) {
	var msg ShellMsg
	if err := t.dec.Decode(&msg); err != nil {
		return msg, fmt.Errorf("recv shell: %w", err)
	}
	return msg, nil
}

// Close closes the connection.
func (t *SocketTransport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.closed {
		return nil
	}
	t.closed = true
	return t.conn.Close()
}
