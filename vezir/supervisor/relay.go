// relay.go — Unix socket relay between TUI and Substrate.
// TUI connects to Vezir's socket (permanent endpoint).
// Vezir relays bidirectionally to Substrate's socket.
// On Substrate restart: Vezir reconnects relay, TUI stays connected.
package supervisor

import (
	"context"
	"io"
	"log/slog"
	"net"
	"os"
	"sync"
	"time"

	"github.com/dpopsuev/djinn/telemetry"
)

// Relay manages TUI↔Substrate socket relay through Vezir.
type Relay struct {
	listenPath    string // Vezir's permanent socket (TUI connects here)
	substratePath string // Substrate's socket (Vezir connects here)
	log           *slog.Logger

	mu       sync.Mutex
	listener net.Listener
	tuiConn  net.Conn // current TUI connection
	subConn  net.Conn // current Substrate connection
	cancel   context.CancelFunc
}

// NewRelay creates a relay that listens on listenPath and connects to substratePath.
func NewRelay(listenPath, substratePath string, log *slog.Logger) *Relay {
	return &Relay{
		listenPath:    listenPath,
		substratePath: substratePath,
		log:           log,
	}
}

// Start begins listening for TUI connections. Non-blocking — runs in background.
func (r *Relay) Start(ctx context.Context) error {
	// Clean up stale socket.
	os.Remove(r.listenPath) //nolint:errcheck // best-effort cleanup of stale socket

	listener, err := net.Listen("unix", r.listenPath)
	if err != nil {
		return err
	}

	r.mu.Lock()
	r.listener = listener
	r.mu.Unlock()

	ctx, cancel := context.WithCancel(ctx)
	r.mu.Lock()
	r.cancel = cancel
	r.mu.Unlock()

	go r.acceptLoop(ctx)

	r.log.InfoContext(ctx, "relay listening",
		slog.String(telemetry.KeyPath, r.listenPath),
	)
	return nil
}

// Reconnect reconnects the relay to a new Substrate socket.
// Called after Substrate restart. TUI stays connected.
func (r *Relay) Reconnect(ctx context.Context) error {
	r.mu.Lock()
	// Close old Substrate connection.
	if r.subConn != nil {
		r.subConn.Close()
		r.subConn = nil
	}
	r.mu.Unlock()

	// Connect to new Substrate with retries.
	var conn net.Conn
	var err error
	for i := 0; i < 10; i++ {
		conn, err = net.Dial("unix", r.substratePath)
		if err == nil {
			break
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(200 * time.Millisecond):
		}
	}
	if err != nil {
		return err
	}

	r.mu.Lock()
	r.subConn = conn
	tuiConn := r.tuiConn
	r.mu.Unlock()

	r.log.InfoContext(ctx, "relay reconnected to substrate",
		slog.String(telemetry.KeyPath, r.substratePath),
	)

	// Resume bidirectional copy if TUI is connected.
	if tuiConn != nil {
		go r.bridge(ctx, tuiConn, conn)
	}

	return nil
}

// Stop closes all connections and the listener.
func (r *Relay) Stop() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.cancel != nil {
		r.cancel()
	}
	if r.tuiConn != nil {
		r.tuiConn.Close()
	}
	if r.subConn != nil {
		r.subConn.Close()
	}
	if r.listener != nil {
		r.listener.Close()
	}
	os.Remove(r.listenPath) //nolint:errcheck // best-effort cleanup
}

// Connected returns true if both TUI and Substrate are connected.
func (r *Relay) Connected() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.tuiConn != nil && r.subConn != nil
}

func (r *Relay) acceptLoop(ctx context.Context) {
	for {
		conn, err := r.listener.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return // shutting down
			}
			r.log.WarnContext(ctx, "relay accept error",
				slog.String(telemetry.KeyError, err.Error()),
			)
			continue
		}

		r.log.InfoContext(ctx, "TUI connected to relay")

		r.mu.Lock()
		// Close previous TUI connection if any.
		if r.tuiConn != nil {
			r.tuiConn.Close()
		}
		r.tuiConn = conn
		subConn := r.subConn
		r.mu.Unlock()

		// Start bridging if Substrate is already connected.
		if subConn != nil {
			go r.bridge(ctx, conn, subConn)
		}
	}
}

func (r *Relay) bridge(ctx context.Context, tui, sub net.Conn) {
	done := make(chan struct{}, 2) //nolint:mnd // two directions

	// TUI → Substrate
	go func() {
		io.Copy(sub, tui) //nolint:errcheck // connection closed is expected
		done <- struct{}{}
	}()

	// Substrate → TUI
	go func() {
		io.Copy(tui, sub) //nolint:errcheck // connection closed is expected
		done <- struct{}{}
	}()

	// Wait for either direction to close.
	select {
	case <-done:
	case <-ctx.Done():
	}

	r.log.InfoContext(ctx, "relay bridge closed")
}
