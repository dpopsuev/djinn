// heartbeat.go — health monitoring between Vezir and Substrate.
// Substrate sends periodic pings. Vezir detects missed heartbeats → unhealthy.
package supervisor

import (
	"context"
	"log/slog"
	"net"
	"os"
	"sync"
	"time"

	"github.com/dpopsuev/djinn/telemetry"
)

// Heartbeat monitors Substrate health via a unix socket.
// Substrate connects and sends periodic pings. If no ping received
// within timeout, Substrate is declared unhealthy.
type Heartbeat struct {
	path    string // unix socket path for heartbeat
	timeout time.Duration
	log     *slog.Logger

	mu       sync.Mutex
	listener net.Listener
	lastPing time.Time
	alive    bool
}

// NewHeartbeat creates a heartbeat monitor on the given socket path.
func NewHeartbeat(path string, timeout time.Duration, log *slog.Logger) *Heartbeat {
	return &Heartbeat{
		path:    path,
		timeout: timeout,
		log:     log,
	}
}

// Start begins listening for heartbeat pings. Non-blocking.
func (h *Heartbeat) Start(ctx context.Context) error {
	os.Remove(h.path) //nolint:errcheck // best-effort cleanup of stale socket

	listener, err := net.Listen("unix", h.path)
	if err != nil {
		return err
	}

	h.mu.Lock()
	h.listener = listener
	h.alive = false
	h.mu.Unlock()

	go h.acceptLoop(ctx)

	h.log.InfoContext(ctx, "heartbeat listening",
		slog.String(telemetry.KeyPath, h.path),
		slog.Duration(telemetry.KeyDuration, h.timeout),
	)
	return nil
}

// Alive returns true if a heartbeat was received within the timeout window.
func (h *Heartbeat) Alive() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.lastPing.IsZero() {
		return false
	}
	return time.Since(h.lastPing) < h.timeout
}

// LastPing returns the timestamp of the most recent heartbeat.
func (h *Heartbeat) LastPing() time.Time {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.lastPing
}

// Stop closes the listener and cleans up.
func (h *Heartbeat) Stop() {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.listener != nil {
		h.listener.Close()
	}
	os.Remove(h.path) //nolint:errcheck // best-effort cleanup
}

func (h *Heartbeat) acceptLoop(ctx context.Context) {
	for {
		conn, err := h.listener.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			continue
		}
		go h.handleConn(ctx, conn)
	}
}

func (h *Heartbeat) handleConn(ctx context.Context, conn net.Conn) {
	defer conn.Close()
	buf := make([]byte, 64)

	for {
		_ = conn.SetReadDeadline(time.Now().Add(h.timeout))
		n, err := conn.Read(buf)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			h.mu.Lock()
			h.alive = false
			h.mu.Unlock()
			h.log.WarnContext(ctx, "heartbeat lost",
				slog.String(telemetry.KeyError, err.Error()),
			)
			return
		}

		if n > 0 {
			h.mu.Lock()
			h.lastPing = time.Now()
			h.alive = true
			h.mu.Unlock()
		}
	}
}

// SendHeartbeat connects to a heartbeat socket and sends periodic pings.
// Call this from the Substrate side.
func SendHeartbeat(ctx context.Context, path string, interval time.Duration) error {
	conn, err := net.Dial("unix", path)
	if err != nil {
		return err
	}
	defer conn.Close()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if _, err := conn.Write([]byte("ping")); err != nil {
				return err
			}
		}
	}
}
