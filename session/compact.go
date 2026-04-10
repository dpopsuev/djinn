package session

import (
	"strings"
	"time"

	"github.com/dpopsuev/djinn/driver"
)

// DefaultKeepRecent is how many recent entries to preserve during compaction.
const DefaultKeepRecent = 4

// Compact summarizes old entries and keeps the most recent ones.
// Returns (before, after) entry counts. No-op if history is too short.
func Compact(sess *Session, keepRecent int) (before, after int) {
	before = sess.History.Len()
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

// SeedSession creates a new session pre-loaded with a compacted summary
// from an old session plus its most recent entries. Used by the relay
// to bootstrap a fresh context window.
func SeedSession(newID string, old *Session, summary string, keepRecent int) *Session {
	now := time.Now()
	s := &Session{
		ID:        newID,
		Name:      old.Name,
		Driver:    old.Driver,
		Model:     old.Model,
		Mode:      old.Mode,
		WorkDir:   old.WorkDir,
		WorkDirs:  old.WorkDirs,
		Workspace: old.Workspace,
		CreatedAt: now,
		UpdatedAt: now,
		History:   NewHistory(0),
	}

	// Inject summary as system context.
	if summary != "" {
		s.Append(Entry{
			Role:    driver.RoleUser,
			Content: "[Session context]\n" + summary,
		})
	}

	// Copy recent entries verbatim.
	entries := old.Entries()
	if keepRecent > 0 && len(entries) > keepRecent {
		entries = entries[len(entries)-keepRecent:]
	}
	for _, e := range entries {
		s.Append(e)
	}

	return s
}

// ExtractSummaryText builds a plain-text conversation excerpt from old entries
// suitable for feeding into an LLM summarizer. Falls back to truncated
// first-lines when the full text would exceed maxChars.
func ExtractSummaryText(entries []Entry, maxChars int) string {
	var sb strings.Builder
	for _, e := range entries {
		line := e.Role + ": " + e.TextContent() + "\n"
		if sb.Len()+len(line) > maxChars {
			break
		}
		sb.WriteString(line)
	}
	return sb.String()
}
