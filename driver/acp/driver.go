// Package acp provides a ChatDriver adapter over bugle/acp.Client.
//
// The ACP protocol implementation lives in bugle/acp. This package wraps
// the Bugle client to satisfy Djinn's driver.ChatDriver interface.
package acp

import (
	"context"
	"log/slog"

	"github.com/dpopsuev/bugle/acp"
	"github.com/dpopsuev/djinn/driver"
)

// ACPDriver wraps bugle/acp.Client as a driver.ChatDriver.
type ACPDriver struct {
	client *acp.Client
}

// Option configures the ACPDriver (forwarded to bugle/acp.Client).
type Option func(*options)

type options struct {
	model      string
	logger     *slog.Logger
	cmdFactory acp.CommandFactory
}

func WithModel(m string) Option                      { return func(o *options) { o.model = m } }
func WithLogger(l *slog.Logger) Option               { return func(o *options) { o.logger = l } }
func WithCommandFactory(f acp.CommandFactory) Option { return func(o *options) { o.cmdFactory = f } }

// New creates an ACP driver for the named agent.
func New(agentName string, opts ...Option) (*ACPDriver, error) {
	o := &options{}
	for _, opt := range opts {
		opt(o)
	}

	clientOpts := []acp.Option{
		acp.WithClientInfo(acp.ClientInfo{Name: "djinn", Version: "0.1.0"}),
	}
	if o.model != "" {
		clientOpts = append(clientOpts, acp.WithModel(o.model))
	}
	if o.logger != nil {
		clientOpts = append(clientOpts, acp.WithLogger(o.logger))
	}
	if o.cmdFactory != nil {
		clientOpts = append(clientOpts, acp.WithCommandFactory(o.cmdFactory))
	}

	client, err := acp.NewClient(agentName, clientOpts...)
	if err != nil {
		return nil, err
	}

	return &ACPDriver{client: client}, nil
}

// Start launches the agent process and performs the ACP handshake.
func (d *ACPDriver) Start(ctx context.Context, _ driver.SandboxHandle) error {
	return d.client.Start(ctx)
}

// Stop cancels the session and kills the agent process.
func (d *ACPDriver) Stop(ctx context.Context) error {
	return d.client.Stop(ctx)
}

// Send appends a message to the conversation history.
func (d *ACPDriver) Send(_ context.Context, msg driver.Message) error {
	d.client.Send(acp.Message{Role: msg.Role, Content: msg.Content})
	return nil
}

// SendRich converts a rich message to plain and sends it.
func (d *ACPDriver) SendRich(ctx context.Context, msg driver.RichMessage) error {
	return d.Send(ctx, msg.ToMessage())
}

// AppendAssistant appends an assistant message to history.
func (d *ACPDriver) AppendAssistant(msg driver.RichMessage) {
	d.client.Send(acp.Message{Role: msg.Role, Content: msg.TextContent()})
}

// SetSystemPrompt is a no-op — ACP agents manage their own system prompt.
func (d *ACPDriver) SetSystemPrompt(_ string) {}

// ContextWindow returns the model's context window in tokens.
func (d *ACPDriver) ContextWindow() int { return 200_000 }

// Chat sends the last message as a prompt and streams events.
func (d *ACPDriver) Chat(ctx context.Context) (<-chan driver.StreamEvent, error) {
	acpCh, err := d.client.Chat(ctx)
	if err != nil {
		return nil, err
	}

	ch := make(chan driver.StreamEvent, 64)
	go func() {
		defer close(ch)
		for evt := range acpCh {
			ch <- convertEvent(evt)
		}
	}()

	return ch, nil
}

// convertEvent maps bugle/acp.StreamEvent to driver.StreamEvent.
func convertEvent(evt acp.StreamEvent) driver.StreamEvent {
	de := driver.StreamEvent{
		Type:     evt.Type,
		Text:     evt.Text,
		Thinking: evt.Thinking,
		Error:    evt.Error,
	}
	if evt.ToolCall != nil {
		de.ToolCall = &driver.ToolCall{
			ID:    evt.ToolCall.ID,
			Name:  evt.ToolCall.Name,
			Input: evt.ToolCall.Input,
		}
	}
	if evt.Usage != nil {
		de.Usage = &driver.Usage{
			InputTokens:  evt.Usage.InputTokens,
			OutputTokens: evt.Usage.OutputTokens,
		}
	}
	return de
}

// Client returns the underlying bugle/acp.Client for direct access.
func (d *ACPDriver) Client() *acp.Client { return d.client }

var _ driver.ChatDriver = (*ACPDriver)(nil)
