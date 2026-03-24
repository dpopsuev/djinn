// Package cursor provides a ChatDriver that wraps the Cursor Agent CLI.
// Uses `agent -p --output-format stream-json` for streaming responses.
package cursor

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
	"sync"

	"github.com/dpopsuev/djinn/driver"
)

// CommandFactory creates exec.Cmd — injectable for testing.
type CommandFactory func(ctx context.Context, name string, args ...string) *exec.Cmd

// CLIDriver wraps the Cursor Agent CLI (`agent`) as a ChatDriver.
type CLIDriver struct {
	model        string
	systemPrompt string
	messages     []driver.Message
	mu           sync.Mutex
	log          *slog.Logger
	cmdFactory   CommandFactory // injectable for testing
}

// Option configures the CLI driver.
type Option func(*CLIDriver)

// WithModel sets the model name (e.g., "sonnet-4", "gpt-5").
func WithModel(m string) Option { return func(d *CLIDriver) { d.model = m } }

// WithSystemPrompt sets the system prompt.
func WithSystemPrompt(p string) Option { return func(d *CLIDriver) { d.systemPrompt = p } }

// WithLogger sets the logger.
func WithLogger(l *slog.Logger) Option { return func(d *CLIDriver) { d.log = l } }

// WithCommandFactory overrides the exec.Cmd factory (for testing).
func WithCommandFactory(f CommandFactory) Option { return func(d *CLIDriver) { d.cmdFactory = f } }

// New creates a new Cursor CLI driver.
func New(cfg driver.DriverConfig, opts ...Option) *CLIDriver {
	d := &CLIDriver{
		model:      cfg.Model,
		log:        slog.Default(),
		cmdFactory: exec.CommandContext,
	}
	for _, o := range opts {
		o(d)
	}
	if d.model == "" {
		d.model = "sonnet-4"
	}
	return d
}

func (d *CLIDriver) Start(_ context.Context, _ driver.SandboxHandle) error { return nil }
func (d *CLIDriver) Stop(_ context.Context) error                          { return nil }

func (d *CLIDriver) Send(_ context.Context, msg driver.Message) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.messages = append(d.messages, msg)
	return nil
}

func (d *CLIDriver) SendRich(_ context.Context, msg driver.RichMessage) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.messages = append(d.messages, msg.ToMessage())
	return nil
}

func (d *CLIDriver) AppendAssistant(msg driver.RichMessage) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.messages = append(d.messages, msg.ToMessage())
}

func (d *CLIDriver) SetSystemPrompt(prompt string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.systemPrompt = prompt
}

// Chat runs `agent -p --output-format stream-json` with the last user message
// and streams events back via the channel.
func (d *CLIDriver) Chat(ctx context.Context) (<-chan driver.StreamEvent, error) {
	d.mu.Lock()
	if len(d.messages) == 0 {
		d.mu.Unlock()
		return nil, fmt.Errorf("no messages to send")
	}
	lastMsg := d.messages[len(d.messages)-1]
	d.mu.Unlock()

	args := []string{"-p", "--output-format", "stream-json", "--stream-partial-output"}
	if d.model != "" {
		args = append(args, "--model", d.model)
	}
	args = append(args, lastMsg.Content)

	d.log.Debug("cursor agent", "args", args)

	cmd := d.cmdFactory(ctx, "agent", args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start agent: %w", err)
	}

	ch := make(chan driver.StreamEvent, 64)

	go func() {
		defer close(ch)
		defer cmd.Wait() //nolint:errcheck

		scanner := bufio.NewScanner(stdout)
		scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

		var fullText strings.Builder

		for scanner.Scan() {
			line := scanner.Text()
			if line == "" {
				continue
			}

			var evt streamJSONEvent
			if err := json.Unmarshal([]byte(line), &evt); err != nil {
				// Non-JSON line — treat as plain text.
				ch <- driver.StreamEvent{Type: driver.EventText, Text: line}
				fullText.WriteString(line)
				continue
			}

			switch evt.Type {
			case "text_delta", "content":
				ch <- driver.StreamEvent{Type: driver.EventText, Text: evt.Content}
				fullText.WriteString(evt.Content)
			case "thinking":
				ch <- driver.StreamEvent{Type: driver.EventThinking, Thinking: evt.Content}
			case "done", "result":
				ch <- driver.StreamEvent{
					Type: driver.EventDone,
					Usage: &driver.Usage{
						InputTokens:  evt.InputTokens,
						OutputTokens: evt.OutputTokens,
					},
				}
			case "error":
				ch <- driver.StreamEvent{Type: driver.EventError, Error: evt.Content}
			}
		}

		// Append assistant response to history.
		if fullText.Len() > 0 {
			d.mu.Lock()
			d.messages = append(d.messages, driver.Message{
				Role:    driver.RoleAssistant,
				Content: fullText.String(),
			})
			d.mu.Unlock()
		}

		// If no done event was sent, send one now.
		ch <- driver.StreamEvent{Type: driver.EventDone}
	}()

	return ch, nil
}

// streamJSONEvent is the Cursor Agent CLI stream-json format.
type streamJSONEvent struct {
	Type         string `json:"type"`
	Content      string `json:"content,omitempty"`
	InputTokens  int    `json:"input_tokens,omitempty"`
	OutputTokens int    `json:"output_tokens,omitempty"`
}
