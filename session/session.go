// Package session manages conversation history and context for
// multi-turn agent interactions.
package session

import (
	"time"

	"github.com/dpopsuev/djinn/driver"
	"github.com/dpopsuev/djinn/trace"
)

// Entry represents a single turn in a conversation.
type Entry struct {
	Role       string                `json:"role"`
	Content    string                `json:"content,omitempty"`
	Blocks     []driver.ContentBlock `json:"blocks,omitempty"`
	Timestamp  time.Time             `json:"timestamp"`
	TokenCount int                   `json:"token_count,omitempty"` // approximate
}

// TextContent returns the text from this entry.
func (e Entry) TextContent() string {
	if len(e.Blocks) == 0 {
		return e.Content
	}
	rm := driver.RichMessage{Content: e.Content, Blocks: e.Blocks}
	return rm.TextContent()
}

// Session holds the state of an interactive conversation.
type Session struct {
	ID        string    `json:"id"`
	Name      string    `json:"name,omitempty"`
	Driver    string    `json:"driver,omitempty"`
	Model     string    `json:"model"`
	Mode      string    `json:"mode,omitempty"`
	WorkDir   string    `json:"work_dir"`
	WorkDirs  []string  `json:"work_dirs,omitempty"` // deprecated: use Workspace
	Workspace string    `json:"workspace,omitempty"` // named workspace reference
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	History   *History  `json:"history"`

	// Trace snapshot — persisted across session save/load for self-heal validation.
	TraceSnapshot *trace.Archive `json:"trace_snapshot,omitempty"`
}

// New creates a new session.
func New(id, model, workDir string) *Session {
	now := time.Now()
	return &Session{
		ID:        id,
		Model:     model,
		WorkDir:   workDir,
		CreatedAt: now,
		UpdatedAt: now,
		History:   NewHistory(0), // unlimited by default
	}
}

// AllWorkDirs returns WorkDirs if set, otherwise [WorkDir].
func (s *Session) AllWorkDirs() []string {
	if len(s.WorkDirs) > 0 {
		return s.WorkDirs
	}
	if s.WorkDir != "" {
		return []string{s.WorkDir}
	}
	return nil
}

// Append adds a user or assistant entry to the session history.
func (s *Session) Append(entry Entry) {
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}
	s.History.Append(entry)
	s.UpdatedAt = time.Now()
}

// Entries returns the conversation history.
func (s *Session) Entries() []Entry {
	return s.History.Entries()
}

// TotalTokens returns the approximate total token count.
func (s *Session) TotalTokens() int {
	return s.History.TotalTokens()
}
