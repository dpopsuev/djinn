// hub.go — GenSec relay: both shell and backend connect to it.
// Either side can disconnect and reconnect independently (hot-swap).
//
// The Hub is the General Secretary daemon — the persistent process
// that outlives both frontend and backend. It routes messages between
// them and queues messages when one side is temporarily disconnected.
package hotswap

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"github.com/dpopsuev/djinn/djinnlog"
)

// Sentinel errors for Hub operations.
var ErrUnknownRole = errors.New("unknown role")

// Hub is the GenSec relay — both shell and backend connect as clients.
type Hub struct {
	listener *SocketListener
	log      *slog.Logger

	mu      sync.RWMutex
	shell   *SocketTransport // current shell connection (nil if disconnected)
	backend *SocketTransport // current backend connection (nil if disconnected)

	shellQ   []BackendMsg // queued for shell while disconnected
	backendQ []ShellMsg   // queued for backend while disconnected

	// cancelFuncs for active relay goroutines — canceled on reconnect.
	shellRelay   context.CancelFunc
	backendRelay context.CancelFunc
}

// NewHub creates a hub listening on the given Unix socket path.
func NewHub(socketPath string, log *slog.Logger) (*Hub, error) {
	if log == nil {
		log = djinnlog.Nop()
	}
	ln, err := Listen(socketPath)
	if err != nil {
		return nil, err
	}
	return &Hub{listener: ln, log: log}, nil
}

// Run accepts connections and relays messages until ctx is canceled.
func (h *Hub) Run(ctx context.Context) error {
	go func() {
		<-ctx.Done()
		h.listener.Close()
	}()

	h.log.InfoContext(ctx, "hub started",
		slog.String(djinnlog.KeyComponent, "clutch"),
		slog.String(djinnlog.KeyPath, h.listener.Addr()),
	)

	for {
		conn, err := h.listener.listener.Accept()
		if err != nil {
			if ctx.Err() != nil {
				// Yellow: clean shutdown
				h.log.InfoContext(ctx, "hub shutdown",
					slog.String(djinnlog.KeyComponent, "clutch"),
					slog.String(djinnlog.KeyAction, "shutdown"),
				)
				return nil
			}
			// Orange: accept failure
			h.log.WarnContext(ctx, "hub accept error",
				slog.String(djinnlog.KeyComponent, "clutch"),
				slog.String(djinnlog.KeyError, err.Error()),
			)
			return fmt.Errorf("hub accept: %w", err)
		}

		transport := newSocketTransport(conn)

		// Read registration using the transport's decoder (same buffer).
		role, err := h.readRegistration(transport)
		if err != nil {
			// Orange: registration failure
			h.log.WarnContext(ctx, "registration rejected",
				slog.String(djinnlog.KeyComponent, "clutch"),
				slog.String(djinnlog.KeyError, err.Error()),
			)
			transport.Close()
			continue
		}

		// Yellow: successful registration
		h.log.InfoContext(ctx, "client registered",
			slog.String(djinnlog.KeyComponent, "clutch"),
			slog.String(djinnlog.KeyRole, role),
		)

		switch role {
		case "shell":
			h.registerShell(ctx, transport)
		case "backend":
			h.registerBackend(ctx, transport)
		default:
			transport.Close()
		}
	}
}

// Close stops the hub and cleans up.
func (h *Hub) Close() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.shellRelay != nil {
		h.shellRelay()
	}
	if h.backendRelay != nil {
		h.backendRelay()
	}
	if h.shell != nil {
		h.shell.Close()
	}
	if h.backend != nil {
		h.backend.Close()
	}
	return h.listener.Close()
}

// readRegistration reads the first JSON message to identify the client role.
// Uses the transport's decoder to avoid buffer conflicts.
func (h *Hub) readRegistration(transport *SocketTransport) (string, error) {
	var reg RegisterMsg
	if err := transport.dec.Decode(&reg); err != nil {
		return "", fmt.Errorf("read registration: %w", err)
	}
	if reg.Role != "shell" && reg.Role != "backend" {
		return "", fmt.Errorf("%w: %q", ErrUnknownRole, reg.Role)
	}
	return reg.Role, nil
}

func (h *Hub) registerShell(ctx context.Context, transport *SocketTransport) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Cancel previous shell relay if any.
	if h.shellRelay != nil {
		h.shellRelay()
	}
	if h.shell != nil {
		h.shell.Close()
	}

	h.shell = transport

	// Drain queued backend messages to the new shell.
	for i := range h.shellQ {
		transport.SendToShell(h.shellQ[i]) //nolint:errcheck // best-effort send, error logged by receiver
	}
	h.shellQ = nil

	// Start relay: shell → backend.
	relayCtx, cancel := context.WithCancel(ctx)
	h.shellRelay = cancel
	go h.relayShellToBackend(relayCtx, transport)
}

func (h *Hub) registerBackend(ctx context.Context, transport *SocketTransport) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Cancel previous backend relay if any.
	if h.backendRelay != nil {
		h.backendRelay()
	}
	if h.backend != nil {
		h.backend.Close()
	}

	h.backend = transport

	// Drain queued shell messages to the new backend.
	for _, msg := range h.backendQ {
		transport.SendToBackend(msg) //nolint:errcheck // best-effort send, error logged by receiver
	}
	h.backendQ = nil

	// Start relay: backend → shell.
	relayCtx, cancel := context.WithCancel(ctx)
	h.backendRelay = cancel
	go h.relayBackendToShell(relayCtx, transport)
}

//nolint:dupl // relay functions are directional mirrors, not extractable without generics
func (h *Hub) relayShellToBackend(ctx context.Context, transport *SocketTransport) {
	for {
		if ctx.Err() != nil {
			return
		}
		msg, err := transport.RecvFromShell()
		if err != nil {
			// Shell disconnected.
			h.mu.Lock()
			if h.shell == transport {
				h.shell = nil
			}
			h.mu.Unlock()
			return
		}

		h.mu.RLock()
		backend := h.backend
		h.mu.RUnlock()

		if backend != nil {
			if err := backend.SendToBackend(msg); err != nil {
				// Backend gone — queue the message.
				h.mu.Lock()
				h.backendQ = append(h.backendQ, msg)
				h.mu.Unlock()
			}
		} else {
			h.mu.Lock()
			h.backendQ = append(h.backendQ, msg)
			h.mu.Unlock()
		}
	}
}

// relayBackendToShell forwards backend messages to the shell.
// On disconnect, queues messages for the shell.
//
//nolint:dupl // relay functions are directional mirrors, not extractable without generics
func (h *Hub) relayBackendToShell(ctx context.Context, transport *SocketTransport) {
	for {
		if ctx.Err() != nil {
			return
		}
		msg, err := transport.RecvFromBackend()
		if err != nil {
			// Backend disconnected.
			h.mu.Lock()
			if h.backend == transport {
				h.backend = nil
			}
			h.mu.Unlock()
			return
		}

		h.mu.RLock()
		shell := h.shell
		h.mu.RUnlock()

		if shell != nil {
			if err := shell.SendToShell(msg); err != nil {
				// Shell gone — queue the message.
				h.mu.Lock()
				h.shellQ = append(h.shellQ, msg)
				h.mu.Unlock()
			}
		} else {
			h.mu.Lock()
			h.shellQ = append(h.shellQ, msg)
			h.mu.Unlock()
		}
	}
}
