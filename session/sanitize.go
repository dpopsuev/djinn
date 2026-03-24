package session

import (
	"encoding/json"

	"github.com/dpopsuev/djinn/driver"
)

// DefaultMaxLoadEntries is the maximum entries before auto-compact on load.
const DefaultMaxLoadEntries = 200

// Sanitize repairs known corruption patterns in session history.
// Called automatically by Store.Load().
//
// Repairs:
//   - tool_use blocks with nil or "null" Input → defaults to {}
//
// Compacts:
//   - sessions with > DefaultMaxLoadEntries entries
func Sanitize(sess *Session) {
	if sess.History == nil {
		return
	}

	entries := sess.History.Entries()
	repaired := false

	for i := range entries {
		for j := range entries[i].Blocks {
			block := &entries[i].Blocks[j]
			if block.Type == driver.BlockToolUse && block.ToolCall != nil {
				if block.ToolCall.Input == nil || string(block.ToolCall.Input) == "null" {
					block.ToolCall.Input = json.RawMessage(`{}`)
					repaired = true
				}
			}
		}
	}

	if repaired {
		sess.History.SetEntries(entries)
	}

	// Auto-compact oversized sessions
	if sess.History.Len() > DefaultMaxLoadEntries {
		Compact(sess, DefaultKeepRecent)
	}
}
