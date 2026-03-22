package session

import (
	"strings"

	"github.com/dpopsuev/djinn/driver"
)

// DefaultKeepRecent is how many recent entries to preserve during compaction.
const DefaultKeepRecent = 4

// Compact summarizes old entries and keeps the most recent ones.
// Returns (before, after) entry counts. No-op if history is too short.
func Compact(sess *Session, keepRecent int) (int, int) {
	before := sess.History.Len()
	if before <= keepRecent {
		return before, before
	}

	entries := sess.Entries()
	keepFrom := len(entries) - keepRecent
	old := entries[:keepFrom]
	recent := entries[keepFrom:]

	var summaryParts []string
	for _, e := range old {
		first := e.Content
		if idx := strings.IndexByte(first, '\n'); idx >= 0 {
			first = first[:idx]
		}
		const maxLine = 60
		if len(first) > maxLine {
			first = first[:maxLine] + "..."
		}
		if first != "" {
			summaryParts = append(summaryParts, e.Role+": "+first)
		}
	}

	sess.History.Clear()
	if len(summaryParts) > 0 {
		sess.Append(Entry{
			Role:    driver.RoleUser,
			Content: "[Compacted history]\n" + strings.Join(summaryParts, "\n"),
		})
	}
	for _, e := range recent {
		sess.Append(e)
	}

	return before, sess.History.Len()
}
