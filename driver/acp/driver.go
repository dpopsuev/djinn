// Package acp provides a universal ChatDriver via the Agent Client Protocol.
// One driver for all ACP-compatible agents: Claude, Gemini, Codex, Cursor.
// Warm process, persistent context, standard JSON-RPC over stdio.
package acp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"

	"github.com/dpopsuev/djinn/driver"
)

// CommandFactory creates exec.Cmd — injectable for testing.
type CommandFactory func(ctx context.Context, name string, args ...string) *exec.Cmd

// Known ACP agent launch commands.
var AgentCommands = map[string][]string{
	"cursor": {"agent", "acp"},
	"claude": {"claude", "--experimental-acp"},
	"gemini": {"gemini", "--experimental-acp"},
	"codex":  {"codex-acp"},
}

// ACPDriver implements driver.ChatDriver via ACP JSON-RPC over stdio.
type ACPDriver struct {
	agentName string   // "cursor", "claude", "gemini", "codex"
	agentCmd  string   // binary name
	agentArgs []string // launch args
	model     string

	cmd       *exec.Cmd
	stdin     *json.Encoder
	scanner   *bufio.Scanner
	sessionID string
	messages  []driver.Message
	mu        sync.Mutex
	nextID    atomic.Int64
	log       *slog.Logger

	cmdFactory CommandFactory
}

type Option func(*ACPDriver)

func WithModel(m string) Option              { return func(d *ACPDriver) { d.model = m } }
func WithLogger(l *slog.Logger) Option       { return func(d *ACPDriver) { d.log = l } }
func WithCommandFactory(f CommandFactory) Option { return func(d *ACPDriver) { d.cmdFactory = f } }

// New creates an ACP driver for the named agent.
func New(agentName string, opts ...Option) (*ACPDriver, error) {
	args, ok := AgentCommands[agentName]
	if !ok {
		return nil, fmt.Errorf("unknown ACP agent %q (supported: cursor, claude, gemini, codex)", agentName)
	}

	d := &ACPDriver{
		agentName:  agentName,
		agentCmd:   args[0],
		agentArgs:  args[1:],
		log:        slog.Default(),
		cmdFactory: exec.CommandContext,
	}
	for _, o := range opts {
		o(d)
	}
	return d, nil
}

// Start launches the agent process and performs the ACP handshake.
func (d *ACPDriver) Start(ctx context.Context, _ driver.SandboxHandle) error {
	args := make([]string, len(d.agentArgs))
	copy(args, d.agentArgs)

	d.cmd = d.cmdFactory(ctx, d.agentCmd, args...)
	d.cmd.Stderr = os.Stderr

	stdinPipe, err := d.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("stdin pipe: %w", err)
	}

	stdoutPipe, err := d.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}

	if err := d.cmd.Start(); err != nil {
		return fmt.Errorf("start %s: %w", d.agentCmd, err)
	}

	d.stdin = json.NewEncoder(stdinPipe)
	d.scanner = bufio.NewScanner(stdoutPipe)
	d.scanner.Buffer(make([]byte, 4*1024*1024), 4*1024*1024) // 4MB buffer

	d.log.Info("ACP agent started", "agent", d.agentName, "pid", d.cmd.Process.Pid)

	// Initialize handshake.
	initResp, err := d.call("initialize", initializeParams{
		ProtocolVersion: ProtocolVersion,
		ClientInfo:      clientInfo{Name: "djinn", Version: "0.1.0"},
	})
	if err != nil {
		d.cmd.Process.Kill() //nolint:errcheck
		return fmt.Errorf("ACP initialize: %w", err)
	}

	var initResult initializeResult
	if initResp != nil {
		json.Unmarshal(*initResp, &initResult) //nolint:errcheck
	}
	d.log.Info("ACP initialized", "agent_name", initResult.AgentInfo.Name, "protocol", initResult.ProtocolVersion)

	// Create session.
	cwd, _ := os.Getwd()
	sessResp, err := d.call("session/new", newSessionParams{CWD: cwd})
	if err != nil {
		d.cmd.Process.Kill() //nolint:errcheck
		return fmt.Errorf("ACP session/new: %w", err)
	}

	var sessResult newSessionResult
	if sessResp != nil {
		json.Unmarshal(*sessResp, &sessResult) //nolint:errcheck
	}
	d.sessionID = sessResult.SessionID
	d.log.Info("ACP session created", "session_id", d.sessionID)

	return nil
}

// Stop cancels the session and kills the agent process.
func (d *ACPDriver) Stop(_ context.Context) error {
	if d.cmd == nil || d.cmd.Process == nil {
		return nil
	}
	// Send cancel notification (best-effort).
	d.notify("session/cancel", map[string]string{"sessionId": d.sessionID})
	d.cmd.Process.Kill() //nolint:errcheck
	return d.cmd.Wait()
}

func (d *ACPDriver) Send(_ context.Context, msg driver.Message) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.messages = append(d.messages, msg)
	return nil
}

func (d *ACPDriver) SendRich(_ context.Context, msg driver.RichMessage) error {
	return d.Send(context.Background(), msg.ToMessage())
}

func (d *ACPDriver) AppendAssistant(msg driver.RichMessage) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.messages = append(d.messages, msg.ToMessage())
}

func (d *ACPDriver) SetSystemPrompt(_ string) {
	// ACP agents manage their own system prompt.
}

// Chat sends a prompt and streams ACP session/update events.
func (d *ACPDriver) Chat(ctx context.Context) (<-chan driver.StreamEvent, error) {
	d.mu.Lock()
	if len(d.messages) == 0 {
		d.mu.Unlock()
		return nil, fmt.Errorf("no messages to send")
	}
	lastMsg := d.messages[len(d.messages)-1]
	d.mu.Unlock()

	// Send prompt.
	id := d.nextID.Add(1)
	err := d.stdin.Encode(jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      int(id),
		Method:  "session/prompt",
		Params: promptParams{
			SessionID: d.sessionID,
			Prompt:    []promptBlock{{Type: "text", Text: lastMsg.Content}},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("send prompt: %w", err)
	}

	ch := make(chan driver.StreamEvent, 64)

	go func() {
		defer close(ch)

		var fullText string

		for d.scanner.Scan() {
			line := d.scanner.Text()
			if line == "" {
				continue
			}

			var msg jsonRPCResponse
			if err := json.Unmarshal([]byte(line), &msg); err != nil {
				continue
			}

			// Response to our prompt request (stop reason).
			if msg.ID == int(id) && msg.Result != nil {
				var result promptResult
				json.Unmarshal(*msg.Result, &result) //nolint:errcheck
				ch <- driver.StreamEvent{Type: driver.EventDone}

				// Append assistant response to history.
				if fullText != "" {
					d.mu.Lock()
					d.messages = append(d.messages, driver.Message{
						Role:    driver.RoleAssistant,
						Content: fullText,
					})
					d.mu.Unlock()
				}
				return
			}

			// Notification (session/update).
			if msg.Method == "session/update" && msg.Params != nil {
				var notif sessionUpdateNotification
				if err := json.Unmarshal(*msg.Params, &notif); err != nil {
					continue
				}

				switch notif.Update.SessionUpdate {
				case "agent_message_chunk":
					if notif.Update.Content != nil && notif.Update.Content.Type == "text" {
						ch <- driver.StreamEvent{Type: driver.EventText, Text: notif.Update.Content.Text}
						fullText += notif.Update.Content.Text
					}
				case "tool_call":
					ch <- driver.StreamEvent{
						Type: driver.EventToolUse,
						ToolCall: &driver.ToolCall{
							ID:   notif.Update.ToolCallID,
							Name: notif.Update.Title,
						},
					}
				case "tool_call_update":
					// Tool progress — emit as text for now.
					if notif.Update.Content != nil && notif.Update.Content.Text != "" {
						ch <- driver.StreamEvent{Type: driver.EventText, Text: notif.Update.Content.Text}
					}
				}
			}

			// Error response.
			if msg.Error != nil {
				ch <- driver.StreamEvent{Type: driver.EventError, Error: msg.Error.Message}
				return
			}
		}

		// Scanner ended (process died).
		ch <- driver.StreamEvent{Type: driver.EventError, Error: "ACP agent process exited"}
	}()

	return ch, nil
}

// call sends a JSON-RPC request and reads the response.
func (d *ACPDriver) call(method string, params any) (*json.RawMessage, error) {
	id := int(d.nextID.Add(1))

	if err := d.stdin.Encode(jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}); err != nil {
		return nil, fmt.Errorf("send %s: %w", method, err)
	}

	// Read lines until we get a response with matching ID.
	for d.scanner.Scan() {
		line := d.scanner.Text()
		if line == "" {
			continue
		}

		var resp jsonRPCResponse
		if err := json.Unmarshal([]byte(line), &resp); err != nil {
			continue
		}

		if resp.ID == id {
			if resp.Error != nil {
				return nil, fmt.Errorf("%s error: %s", method, resp.Error.Message)
			}
			return resp.Result, nil
		}
	}

	return nil, fmt.Errorf("%s: no response (agent exited)", method)
}

// notify sends a JSON-RPC notification (no response expected).
func (d *ACPDriver) notify(method string, params any) {
	d.stdin.Encode(map[string]any{ //nolint:errcheck
		"jsonrpc": "2.0",
		"method":  method,
		"params":  params,
	})
}

var _ driver.ChatDriver = (*ACPDriver)(nil)
