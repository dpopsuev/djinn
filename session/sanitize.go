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

	for i := range entries {
		for j := range entries[i].Blocks {
			block := &entries[i].Blocks[j]
			if block.Type == driver.BlockToolUse && block.ToolCall != nil {
				if block.ToolCall.Input == nil || string(block.ToolCall.Input) == "null" {
					block.ToolCall.Input = json.RawMessage(`{}`)
				}
			}
		}
	}

	// Repair orphaned tool_use blocks: inject synthetic tool_result.
	// Vertex requires every tool_use to have a matching tool_result
	// in the immediately following message (DJN-BUG-16).
	entries = repairOrphanedToolUse(entries)

	sess.History.SetEntries(entries)

	// Auto-compact oversized sessions
	if sess.History.Len() > DefaultMaxLoadEntries {
		Compact(sess, DefaultKeepRecent)
	}
}

// repairOrphanedToolUse finds tool_use blocks without a matching
// tool_result in the next message and injects synthetic results.
func repairOrphanedToolUse(entries []Entry) []Entry {
	var result []Entry

	for i, entry := range entries {
		result = append(result, entry)

		// Collect tool_use IDs from this entry
		var toolUseIDs []string
		for _, block := range entry.Blocks {
			if block.Type == driver.BlockToolUse && block.ToolCall != nil {
				toolUseIDs = append(toolUseIDs, block.ToolCall.ID)
			}
		}

		if len(toolUseIDs) == 0 {
			continue
		}

		// Check if the NEXT entry has matching tool_results
		nextHasResults := make(map[string]bool)
		if i+1 < len(entries) {
			for _, block := range entries[i+1].Blocks {
				if block.Type == driver.BlockToolResult && block.ToolResult != nil {
					nextHasResults[block.ToolResult.ToolCallID] = true
				}
			}
		}

		// Inject synthetic results for orphaned tool_use IDs
		var orphanBlocks []driver.ContentBlock
		for _, id := range toolUseIDs {
			if !nextHasResults[id] {
				orphanBlocks = append(orphanBlocks, driver.ContentBlock{
					Type: driver.BlockToolResult,
					ToolResult: &driver.ToolResult{
						ToolCallID: id,
						Output:     "(interrupted — session resumed)",
						IsError:    true,
					},
				})
			}
		}

		if len(orphanBlocks) > 0 {
			result = append(result, Entry{
				Role:   "user",
				Blocks: orphanBlocks,
			})
		}
	}

	return result
}
