package clutch

import "errors"

const channelBuffer = 100

// Sentinel errors.
var (
	ErrClosed = errors.New("transport closed")
)

// ChannelTransport implements Transport using in-memory Go channels.
// This is the MVP transport — same binary, two goroutines.
type ChannelTransport struct {
	toBackend chan ShellMsg
	toShell   chan BackendMsg
	done      chan struct{}
}

// NewChannelTransport creates a new in-memory transport.
func NewChannelTransport() *ChannelTransport {
	return &ChannelTransport{
		toBackend: make(chan ShellMsg, channelBuffer),
		toShell:   make(chan BackendMsg, channelBuffer),
		done:      make(chan struct{}),
	}
}

func (t *ChannelTransport) SendToBackend(msg ShellMsg) error {
	select {
	case <-t.done:
		return ErrClosed
	default:
	}
	select {
	case t.toBackend <- msg:
		return nil
	case <-t.done:
		return ErrClosed
	}
}

func (t *ChannelTransport) RecvFromBackend() (BackendMsg, error) {
	// Check done first to avoid racing with buffered channel
	select {
	case <-t.done:
		return BackendMsg{}, ErrClosed
	default:
	}
	select {
	case msg, ok := <-t.toShell:
		if !ok {
			return BackendMsg{}, ErrClosed
		}
		return msg, nil
	case <-t.done:
		return BackendMsg{}, ErrClosed
	}
}

func (t *ChannelTransport) SendToShell(msg BackendMsg) error {
	select {
	case <-t.done:
		return ErrClosed
	default:
	}
	select {
	case t.toShell <- msg:
		return nil
	case <-t.done:
		return ErrClosed
	}
}

func (t *ChannelTransport) RecvFromShell() (ShellMsg, error) {
	select {
	case <-t.done:
		return ShellMsg{}, ErrClosed
	default:
	}
	select {
	case msg, ok := <-t.toBackend:
		if !ok {
			return ShellMsg{}, ErrClosed
		}
		return msg, nil
	case <-t.done:
		return ShellMsg{}, ErrClosed
	}
}

func (t *ChannelTransport) Close() error {
	select {
	case <-t.done:
		// already closed
	default:
		close(t.done)
	}
	return nil
}
