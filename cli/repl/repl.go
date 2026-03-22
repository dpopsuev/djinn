// Package repl implements the interactive terminal interface for Djinn.
package repl

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/dpopsuev/djinn/agent"
	"github.com/dpopsuev/djinn/driver"
	claudedriver "github.com/dpopsuev/djinn/driver/claude"
	"github.com/dpopsuev/djinn/session"
	"github.com/dpopsuev/djinn/tools/builtin"
)

// ANSI color codes.
const (
	colorReset   = "\033[0m"
	colorRed     = "\033[31m"
	colorGreen   = "\033[32m"
	colorYellow  = "\033[33m"
	colorBlue    = "\033[34m"
	colorMagenta = "\033[35m"
	colorCyan    = "\033[36m"
	colorDim     = "\033[2m"
	colorBold    = "\033[1m"
)

// Prompts and labels.
const (
	promptUser   = colorGreen + "> " + colorReset
	labelAssist  = colorBlue + "djinn" + colorReset
	labelTool    = colorYellow + "tool" + colorReset
	labelThink   = colorDim + "thinking" + colorReset
	labelError   = colorRed + "error" + colorReset
)

// Slash commands.
const (
	cmdExit   = "/exit"
	cmdQuit   = "/quit"
	cmdClear  = "/clear"
	cmdModel  = "/model"
	cmdHelp   = "/help"
)

// Config configures the REPL.
type Config struct {
	Driver       *claudedriver.APIDriver
	Tools        *builtin.Registry
	Session      *session.Session
	SystemPrompt string
	MaxTurns     int
	AutoApprove  bool
}

// Run starts the interactive REPL loop. Blocks until /exit or ctrl-D.
func Run(ctx context.Context, cfg Config) error {
	scanner := bufio.NewScanner(os.Stdin)

	printWelcome(cfg)

	for {
		fmt.Fprint(os.Stderr, promptUser)

		if !scanner.Scan() {
			// EOF (ctrl-D)
			fmt.Fprintln(os.Stderr)
			return nil
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		// Handle slash commands
		if strings.HasPrefix(input, "/") {
			if handleCommand(input, cfg) {
				return nil // /exit
			}
			continue
		}

		// Run agent loop
		handler := &terminalHandler{}

		approve := agent.AutoApprove
		if !cfg.AutoApprove {
			approve = interactiveApprove
		}

		_, err := agent.Run(ctx, agent.Config{
			Driver:       cfg.Driver,
			Tools:        cfg.Tools,
			Session:      cfg.Session,
			SystemPrompt: cfg.SystemPrompt,
			MaxTurns:     cfg.MaxTurns,
			Approve:      approve,
			Handler:      handler,
		}, input)

		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", labelError, err)
		}

		fmt.Fprintln(os.Stderr) // blank line after response
	}
}

func printWelcome(cfg Config) {
	fmt.Fprintf(os.Stderr, "%sDjinn REPL%s", colorBold, colorReset)
	fmt.Fprintf(os.Stderr, " — model: %s%s%s", colorCyan, cfg.Session.Model, colorReset)
	fmt.Fprintf(os.Stderr, " — tools: %d", len(cfg.Tools.Names()))
	fmt.Fprintf(os.Stderr, " — /help for commands\n\n")
}

func handleCommand(input string, cfg Config) bool {
	parts := strings.Fields(input)
	cmd := parts[0]

	switch cmd {
	case cmdExit, cmdQuit:
		fmt.Fprintln(os.Stderr, "goodbye.")
		return true

	case cmdClear:
		cfg.Session.History.Clear()
		fmt.Fprintln(os.Stderr, "history cleared.")

	case cmdModel:
		if len(parts) > 1 {
			cfg.Session.Model = parts[1]
			fmt.Fprintf(os.Stderr, "model set to %s%s%s\n", colorCyan, parts[1], colorReset)
		} else {
			fmt.Fprintf(os.Stderr, "current model: %s%s%s\n", colorCyan, cfg.Session.Model, colorReset)
		}

	case cmdHelp:
		fmt.Fprintln(os.Stderr, "commands:")
		fmt.Fprintln(os.Stderr, "  /model [name]  — show or switch model")
		fmt.Fprintln(os.Stderr, "  /clear         — clear conversation history")
		fmt.Fprintln(os.Stderr, "  /exit          — quit the REPL")
		fmt.Fprintln(os.Stderr, "  /help          — show this help")

	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s (try /help)\n", cmd)
	}

	return false
}

// terminalHandler renders events to the terminal.
type terminalHandler struct {
	inText bool
}

func (h *terminalHandler) OnText(text string) {
	if !h.inText {
		fmt.Fprintf(os.Stderr, "\n%s: ", labelAssist)
		h.inText = true
	}
	fmt.Fprint(os.Stderr, text)
}

func (h *terminalHandler) OnThinking(text string) {
	fmt.Fprintf(os.Stderr, "%s%s%s", colorDim, text, colorReset)
}

func (h *terminalHandler) OnToolCall(call driver.ToolCall) {
	if h.inText {
		fmt.Fprintln(os.Stderr)
		h.inText = false
	}
	fmt.Fprintf(os.Stderr, "  %s: %s%s%s", labelTool, colorMagenta, call.Name, colorReset)
	// Show brief input
	if len(call.Input) > 0 && len(call.Input) < 100 {
		fmt.Fprintf(os.Stderr, " %s", string(call.Input))
	}
	fmt.Fprintln(os.Stderr)
}

func (h *terminalHandler) OnToolResult(callID, name, output string, isError bool) {
	if isError {
		fmt.Fprintf(os.Stderr, "  %s: %s%s%s %s\n", labelTool, colorRed, name, colorReset, output)
	} else {
		lines := strings.Count(output, "\n")
		if lines > 3 {
			fmt.Fprintf(os.Stderr, "  %s: %s%s%s (%d lines)\n", labelTool, colorGreen, name, colorReset, lines)
		} else if output != "" {
			fmt.Fprintf(os.Stderr, "  %s: %s%s%s %s\n", labelTool, colorGreen, name, colorReset, output)
		}
	}
}

func (h *terminalHandler) OnDone(usage *driver.Usage) {
	if h.inText {
		fmt.Fprintln(os.Stderr)
		h.inText = false
	}
	if usage != nil {
		fmt.Fprintf(os.Stderr, "%s[tokens: %d in, %d out]%s\n",
			colorDim, usage.InputTokens, usage.OutputTokens, colorReset)
	}
}

func (h *terminalHandler) OnError(err error) {
	fmt.Fprintf(os.Stderr, "\n%s: %v\n", labelError, err)
}

func interactiveApprove(call driver.ToolCall) bool {
	fmt.Fprintf(os.Stderr, "  %sapprove %s%s%s? [y/n] ",
		colorYellow, colorMagenta, call.Name, colorReset)

	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		return false
	}
	answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
	return answer == "y" || answer == "yes"
}
