// Package gemini provides a ChatDriver that wraps the Gemini CLI.
// Uses `gemini -p` for headless responses.
package gemini

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
	"sync"

	"github.com/dpopsuev/djinn/driver"
)

// Sentinel errors.
var ErrNoMessages = errors.New("no messages to send")

// CommandFactory creates exec.Cmd — injectable for testing.
type CommandFactory func(ctx context.Context, name string, args ...string) *exec.Cmd

// CLIDriver wraps the Gemini CLI as a ChatDriver.
type CLIDriver struct {
	model        string
	systemPrompt string
	messages     []driver.Message
	mu           sync.Mutex
	log          *slog.Logger
	cmdFactory   CommandFactory
}

type Option func(*CLIDriver)

func WithModel(m string) Option                  { return func(d *CLIDriver) { d.model = m } }
func WithSystemPrompt(p string) Option           { return func(d *CLIDriver) { d.systemPrompt = p } }
func WithLogger(l *slog.Logger) Option           { return func(d *CLIDriver) { d.log = l } }
func WithCommandFactory(f CommandFactory) Option { return func(d *CLIDriver) { d.cmdFactory = f } }

func New(cfg driver.DriverConfig, opts ...Option) *CLIDriver {
	d := &CLIDriver{model: cfg.Model, log: slog.Default(), cmdFactory: exec.CommandContext}
	for _, o := range opts {
		o(d)
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
	return d.Send(context.Background(), msg.ToMessage())
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

// ContextWindow returns the model's context window in tokens.
func (d *CLIDriver) ContextWindow() int { return 1_000_000 }

func (d *CLIDriver) Chat(ctx context.Context) (<-chan driver.StreamEvent, error) {
	d.mu.Lock()
	if len(d.messages) == 0 {
		d.mu.Unlock()
		return nil, ErrNoMessages
	}
	lastMsg := d.messages[len(d.messages)-1]
	d.mu.Unlock()

	args := []string{"-p", lastMsg.Content}
	if d.model != "" {
		args = append([]string{"--model", d.model}, args...)
	}

	d.log.Debug("gemini", "args", args)

	cmd := d.cmdFactory(ctx, "gemini", args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start gemini: %w", err)
	}

	ch := make(chan driver.StreamEvent, 64)
	go func() {
		defer close(ch)
		defer cmd.Wait() //nolint:errcheck // best-effort cleanup on defer

		scanner := bufio.NewScanner(stdout)
		scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
		var fullText strings.Builder

		for scanner.Scan() {
			line := scanner.Text()
			ch <- driver.StreamEvent{Type: driver.EventText, Text: line + "\n"}
			fullText.WriteString(line + "\n")
		}

		if fullText.Len() > 0 {
			d.mu.Lock()
			d.messages = append(d.messages, driver.Message{Role: driver.RoleAssistant, Content: fullText.String()})
			d.mu.Unlock()
		}
		ch <- driver.StreamEvent{Type: driver.EventDone}
	}()

	return ch, nil
}
