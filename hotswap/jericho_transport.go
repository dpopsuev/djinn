// bugle_transport.go — bridges Clutch Transport to Bugle LocalTransport.
// Registers the shell and backend as named agents in Bugle's transport
// so other agents can send messages to them via the A2A protocol.
package hotswap

import (
	"context"

	"github.com/dpopsuev/djinn/jerichoport"
)

// BugleTransportBridge wraps Bugle's LocalTransport as a Clutch-compatible layer.
// Registers "shell" and "backend" handlers in the Bugle transport.
type BugleTransportBridge struct {
	transport *jerichoport.LocalTransport
}

// NewBugleTransportBridge creates a bridge.
func NewBugleTransportBridge(t *jerichoport.LocalTransport) *BugleTransportBridge {
	return &BugleTransportBridge{transport: t}
}

// RegisterShell registers a handler for shell-bound messages.
func (b *BugleTransportBridge) RegisterShell(handler jerichoport.MsgHandler) {
	b.transport.Register("shell", handler)
}

// RegisterBackend registers a handler for backend-bound messages.
func (b *BugleTransportBridge) RegisterBackend(handler jerichoport.MsgHandler) {
	b.transport.Register("backend", handler)
}

// SendToBackend sends a message to the backend agent.
func (b *BugleTransportBridge) SendToBackend(ctx context.Context, content string) (*jerichoport.Task, error) {
	return b.transport.SendMessage(ctx, "backend", jerichoport.Message{
		From:    "shell",
		To:      "backend",
		Content: content,
	})
}

// SendToShell sends a message to the shell agent.
func (b *BugleTransportBridge) SendToShell(ctx context.Context, content string) (*jerichoport.Task, error) {
	return b.transport.SendMessage(ctx, "shell", jerichoport.Message{
		From:    "backend",
		To:      "shell",
		Content: content,
	})
}

// SendToAgent sends a message to any named agent.
func (b *BugleTransportBridge) SendToAgent(ctx context.Context, from, to jerichoport.AgentID, content string) (*jerichoport.Task, error) {
	return b.transport.SendMessage(ctx, to, jerichoport.Message{
		From:    from,
		To:      to,
		Content: content,
	})
}

// Transport returns the underlying Bugle transport for direct access.
func (b *BugleTransportBridge) Transport() *jerichoport.LocalTransport {
	return b.transport
}
