package app

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/dpopsuev/djinn/driver"
	"github.com/dpopsuev/djinn/session"
)

// RunDebug handles the djinn debug subcommand.
func RunDebug(args []string, stderr io.Writer) error {
	if len(args) < 1 {
		fmt.Fprintln(stderr, `djinn debug — session and frame inspection

Usage:
  djinn debug session <file>    inspect session file for corruption
  djinn debug session <name>    inspect session by name (from session dir)
  djinn debug frame [file]      show latest TUI frame (default: /tmp/djinn-frames.jsonl)
  djinn debug frame --raw       show frame with ANSI codes
  djinn debug transcript        show output panel transcript (conversation lines)
  djinn debug input             show input panel state
  djinn debug dashboard         show dashboard state
  djinn debug panels            show all panel states + focus`)
		return nil
	}

	switch args[0] {
	case "session":
		if len(args) < 2 {
			return fmt.Errorf("usage: djinn debug session <file-or-name>")
		}
		return debugSession(args[1], stderr)
	case "frame":
		raw := false
		file := "/tmp/djinn-frames.jsonl"
		for _, arg := range args[1:] {
			if arg == "--raw" {
				raw = true
			} else {
				file = arg
			}
		}
		return debugFrame(file, raw, stderr)
	case "transcript":
		return debugComponent("/tmp/djinn-frames.jsonl", "transcript", stderr)
	case "input":
		return debugComponent("/tmp/djinn-frames.jsonl", "input", stderr)
	case "dashboard":
		return debugComponent("/tmp/djinn-frames.jsonl", "dashboard", stderr)
	case "panels":
		return debugComponent("/tmp/djinn-frames.jsonl", "panels", stderr)
	default:
		return fmt.Errorf("unknown debug command: %s", args[0])
	}
}

func debugFrame(file string, raw bool, w io.Writer) error {
	data, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("read %s: %w", file, err)
	}

	// Find last non-empty line (last frame).
	lines := splitLines(data)
	var lastLine []byte
	for i := len(lines) - 1; i >= 0; i-- {
		if len(lines[i]) > 0 {
			lastLine = lines[i]
			break
		}
	}
	if lastLine == nil {
		return fmt.Errorf("no frames in %s", file)
	}

	var frame struct {
		Frame  string `json:"frame"`
		State  string `json:"state"`
		Role   string `json:"role"`
		Width  int    `json:"width"`
		Height int    `json:"height"`
	}
	if err := json.Unmarshal(lastLine, &frame); err != nil {
		return fmt.Errorf("parse frame: %w", err)
	}

	fmt.Fprintf(w, "=== Frame: %dx%d  state=%s  role=%s ===\n", frame.Width, frame.Height, frame.State, frame.Role)

	if raw {
		fmt.Fprintln(w, frame.Frame)
		return nil
	}

	// Strip ANSI and show with line numbers + widths.
	frameLines := splitString(frame.Frame, '\n')
	for i, line := range frameLines {
		clean := stripANSI(line)
		fmt.Fprintf(w, "%3d (%3dc): %s\n", i, len([]rune(clean)), clean)
	}

	fmt.Fprintf(w, "\n--- %d lines, frame %dx%d ---\n", len(frameLines), frame.Width, frame.Height)
	return nil
}

// stripANSI removes ANSI escape codes from a string.
func stripANSI(s string) string {
	var out []byte
	inEscape := false
	for i := 0; i < len(s); i++ {
		if s[i] == '\x1b' {
			inEscape = true
			continue
		}
		if inEscape {
			if (s[i] >= 'a' && s[i] <= 'z') || (s[i] >= 'A' && s[i] <= 'Z') {
				inEscape = false
			}
			continue
		}
		out = append(out, s[i])
	}
	return string(out)
}

func splitLines(data []byte) [][]byte {
	var lines [][]byte
	start := 0
	for i, b := range data {
		if b == '\n' {
			lines = append(lines, data[start:i])
			start = i + 1
		}
	}
	if start < len(data) {
		lines = append(lines, data[start:])
	}
	return lines
}

func splitString(s string, sep byte) []string {
	var parts []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == sep {
			parts = append(parts, s[start:i])
			start = i + 1
		}
	}
	parts = append(parts, s[start:])
	return parts
}

func debugComponent(file, component string, w io.Writer) error {
	data, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("read %s: %w", file, err)
	}

	lines := splitLines(data)
	var lastLine []byte
	for i := len(lines) - 1; i >= 0; i-- {
		if len(lines[i]) > 0 {
			lastLine = lines[i]
			break
		}
	}
	if lastLine == nil {
		return fmt.Errorf("no frames in %s", file)
	}

	var frame struct {
		State      string `json:"state"`
		Role       string `json:"role"`
		Width      int    `json:"width"`
		Height     int    `json:"height"`
		Components *struct {
			Transcript []string `json:"transcript"`
			Overlay    string   `json:"overlay"`
			InputValue string   `json:"input_value"`
			InputFocus bool     `json:"input_focus"`
			FocusedIdx int      `json:"focused_idx"`
			Dashboard  string   `json:"dashboard"`
		} `json:"components"`
	}
	if err := json.Unmarshal(lastLine, &frame); err != nil {
		return fmt.Errorf("parse frame: %w", err)
	}

	if frame.Components == nil {
		return fmt.Errorf("no component data in frame (run with debug.tap_file enabled)")
	}

	c := frame.Components
	switch component {
	case "transcript":
		fmt.Fprintf(w, "=== Transcript (%d lines) ===\n", len(c.Transcript))
		for i, line := range c.Transcript {
			fmt.Fprintf(w, "%3d: %s\n", i, stripANSI(line))
		}
		if c.Overlay != "" {
			fmt.Fprintf(w, "\n--- Overlay ---\n%s\n", stripANSI(c.Overlay))
		}

	case "input":
		fmt.Fprintf(w, "=== Input Panel ===\n")
		fmt.Fprintf(w, "  focused: %v\n", c.InputFocus)
		fmt.Fprintf(w, "  value:   %q\n", c.InputValue)

	case "dashboard":
		fmt.Fprintf(w, "=== Dashboard ===\n")
		fmt.Fprintf(w, "  %s\n", stripANSI(c.Dashboard))

	case "panels":
		panelNames := []string{"output", "input", "dashboard"}
		fmt.Fprintf(w, "=== Panels (%dx%d, state=%s, role=%s) ===\n",
			frame.Width, frame.Height, frame.State, frame.Role)
		fmt.Fprintf(w, "  focused: %s (idx=%d)\n", panelNames[c.FocusedIdx], c.FocusedIdx)
		fmt.Fprintf(w, "\n  output:    %d lines", len(c.Transcript))
		if c.Overlay != "" {
			fmt.Fprintf(w, " + overlay")
		}
		fmt.Fprintf(w, "\n  input:     %q (focused=%v)\n", c.InputValue, c.InputFocus)
		fmt.Fprintf(w, "  dashboard: %s\n", stripANSI(c.Dashboard))
	}

	return nil
}

func debugSession(nameOrPath string, w io.Writer) error {
	// Try as file path first, then as session name
	var sess *session.Session
	var err error

	if _, statErr := os.Stat(nameOrPath); statErr == nil {
		// It's a file path
		data, readErr := os.ReadFile(nameOrPath)
		if readErr != nil {
			return readErr
		}
		sess = &session.Session{}
		if err = json.Unmarshal(data, sess); err != nil {
			return fmt.Errorf("parse session: %w", err)
		}
	} else {
		// Try as session name from default store
		store, storeErr := session.NewStore(SessionDir())
		if storeErr != nil {
			return storeErr
		}
		// Load WITHOUT sanitize to see raw state
		sess, err = store.LoadRaw(nameOrPath)
		if err != nil {
			return err
		}
	}

	entries := sess.History.Entries()
	fmt.Fprintf(w, "Session: %s (name=%s)\n", sess.ID, sess.Name)
	fmt.Fprintf(w, "Model: %s  Driver: %s  Mode: %s\n", sess.Model, sess.Driver, sess.Mode)
	fmt.Fprintf(w, "Entries: %d\n\n", len(entries))

	// Analyze
	var issues []string
	toolUseMap := map[string]int{}    // tool_use ID → entry index
	toolResultMap := map[string]int{} // tool_result call_id → entry index

	for i, entry := range entries {
		hasBlocks := len(entry.Blocks) > 0
		if !hasBlocks {
			continue
		}

		for _, block := range entry.Blocks {
			if block.Type == driver.BlockToolUse && block.ToolCall != nil {
				toolUseMap[block.ToolCall.ID] = i
				inputStatus := "valid"
				if block.ToolCall.Input == nil {
					inputStatus = "NIL"
					issues = append(issues, fmt.Sprintf("[%d] tool_use %q (%s): input is NIL", i, block.ToolCall.ID, block.ToolCall.Name))
				} else if string(block.ToolCall.Input) == "null" {
					inputStatus = "NULL"
					issues = append(issues, fmt.Sprintf("[%d] tool_use %q (%s): input is literal 'null'", i, block.ToolCall.ID, block.ToolCall.Name))
				}
				fmt.Fprintf(w, "  [%d] %s tool_use: id=%s name=%s input=%s\n",
					i, entry.Role, block.ToolCall.ID, block.ToolCall.Name, inputStatus)
			}
			if block.Type == driver.BlockToolResult && block.ToolResult != nil {
				toolResultMap[block.ToolResult.ToolCallID] = i
				fmt.Fprintf(w, "  [%d] %s tool_result: call_id=%s is_error=%v\n",
					i, entry.Role, block.ToolResult.ToolCallID, block.ToolResult.IsError)
			}
		}
	}

	// Check pairing
	fmt.Fprintf(w, "\nPairing:\n")
	for id, useIdx := range toolUseMap {
		if resultIdx, ok := toolResultMap[id]; ok {
			if resultIdx != useIdx+1 {
				issues = append(issues, fmt.Sprintf("  tool_use %q at [%d] has tool_result at [%d] (not immediately after)", id, useIdx, resultIdx))
				fmt.Fprintf(w, "  ⚠ %s: use=[%d] result=[%d] (gap=%d)\n", id, useIdx, resultIdx, resultIdx-useIdx)
			} else {
				fmt.Fprintf(w, "  ✓ %s: use=[%d] result=[%d]\n", id, useIdx, resultIdx)
			}
		} else {
			issues = append(issues, fmt.Sprintf("  tool_use %q at [%d] has NO matching tool_result (ORPHAN)", id, useIdx))
			fmt.Fprintf(w, "  ✗ %s: use=[%d] result=MISSING (ORPHAN)\n", id, useIdx)
		}
	}

	// Summary
	fmt.Fprintf(w, "\nIssues: %d\n", len(issues))
	for _, issue := range issues {
		fmt.Fprintf(w, "  • %s\n", issue)
	}

	if len(issues) == 0 {
		fmt.Fprintln(w, "  (none — session is clean)")
	}

	return nil
}
