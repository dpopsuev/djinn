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
  djinn debug session <name>    inspect session by name (from session dir)`)
		return nil
	}

	switch args[0] {
	case "session":
		if len(args) < 2 {
			return fmt.Errorf("usage: djinn debug session <file-or-name>")
		}
		return debugSession(args[1], stderr)
	default:
		return fmt.Errorf("unknown debug command: %s", args[0])
	}
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
