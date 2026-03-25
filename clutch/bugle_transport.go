// bugle_transport.go — bridges Clutch Transport to Bugle LocalTransport.
// Registers the shell and backend as named agents in Bugle's transport
// so other agents can send messages to them via the A2A protocol.
package clutch

import (
	"context"

	"github.com/dpopsuev/djinn/bugleport"
)

// BugleTransportBridge wraps Bugle's LocalTransport as a Clutch-compatible layer.
// Registers "shell" and "backend" handlers in the Bugle transport.
type BugleTransportBridge struct {
	transport *bugleport.LocalTransport
}

// NewBugleTransportBridge creates a bridge.
func NewBugleTransportBridge(t *bugleport.LocalTransport) *BugleTransportBridge {
	return &BugleTransportBridge{transport: t}
}

// RegisterShell registers a handler for shell-bound messages.
func (b *BugleTransportBridge) RegisterShell(handler bugleport.Handler) {
	b.transport.Register("shell", handler)
}

// RegisterBackend registers a handler for backend-bound messages.
func (b *BugleTransportBridge) RegisterBackend(handler bugleport.Handler) {
	b.transport.Register("backend", handler)
}

// SendToBackend sends a message to the backend agent.
func (b *BugleTransportBridge) SendToBackend(ctx context.Context, content string) (*bugleport.Task, error) {
	return b.transport.SendMessage(ctx, "backend", bugleport.Message{
		From:    "shell",
		To:      "backend",
		Content: content,
	})
}

// SendToShell sends a message to the shell agent.
func (b *BugleTransportBridge) SendToShell(ctx context.Context, content string) (*bugleport.Task, error) {
	return b.transport.SendMessage(ctx, "shell", bugleport.Message{
		From:    "backend",
		To:      "shell",
		Content: content,
	})
}

// SendToAgent sends a message to any named agent.
func (b *BugleTransportBridge) SendToAgent(ctx context.Context, from, to, content string) (*bugleport.Task, error) {
	return b.transport.SendMessage(ctx, to, bugleport.Message{
		From:    from,
		To:      to,
		Content: content,
	})
}

// Transport returns the underlying Bugle transport for direct access.
func (b *BugleTransportBridge) Transport() *bugleport.LocalTransport {
	return b.transport
}
