// app_print.go — headless print mode for CI/CD and scripting.
//
// djinn --print "fix the bug" sends one prompt, streams the response
// to stdout, and exits. No TUI, no interactive approval.
// --output-format json|text controls the output shape.
package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/dpopsuev/djinn/driver"
)

// PrintResult is the JSON output format for --print mode.
type PrintResult struct {
	Response string        `json:"response"`
	Usage    *driver.Usage `json:"usage,omitempty"`
	Duration string        `json:"duration"`
}

// RunPrint executes a single prompt in headless mode and writes the
// response to stdout. Returns the agent's response text.
func RunPrint(ctx context.Context, chatDriver driver.ChatDriver, prompt, outputFormat string, stderr io.Writer) error {
	start := time.Now()

	// Start driver.
	if err := chatDriver.Start(ctx, ""); err != nil {
		return fmt.Errorf("start driver: %w", err)
	}
	defer chatDriver.Stop(ctx) //nolint:errcheck // best-effort shutdown

	// Send prompt.
	if err := chatDriver.Send(ctx, driver.Message{
		Role:    driver.RoleUser,
		Content: prompt,
	}); err != nil {
		return fmt.Errorf("send: %w", err)
	}

	// Stream response.
	ch, err := chatDriver.Chat(ctx)
	if err != nil {
		return fmt.Errorf("chat: %w", err)
	}

	var response strings.Builder
	var usage *driver.Usage

	for ev := range ch {
		switch ev.Type {
		case driver.EventText:
			response.WriteString(ev.Text)
			if outputFormat == "text" {
				fmt.Fprint(os.Stdout, ev.Text)
			}
		case driver.EventDone:
			usage = ev.Usage
		case driver.EventError:
			fmt.Fprintf(stderr, "error: %s\n", ev.Error)
		}
	}

	if outputFormat == "text" {
		fmt.Fprintln(os.Stdout)
		return nil
	}

	// JSON output.
	result := PrintResult{
		Response: response.String(),
		Usage:    usage,
		Duration: time.Since(start).Round(time.Millisecond).String(),
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(result)
}
