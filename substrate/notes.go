// notes.go — sticky notes in Substrate. Expire on read.
// Not a Relic. Not in Reliquary. In-memory per node, file backup for restart.
// Agents access notes through Substrate. Cross-agent notes go through Substrate.
// "Latest" auto-note saved on agent death, read by successor on spawn.
package substrate

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

// ErrNoteNotFound is returned when a note key does not exist.
var ErrNoteNotFound = errors.New("note not found")

// Note is a sticky note between agents/operator. Read = expire.
type Note struct {
	From      string    `json:"from"`
	To        string    `json:"to"` // agent ID or "*" for broadcast
	Key       string    `json:"key"`
	Title     string    `json:"title"` // ≤100 tokens, shown in prompt enrichment
	Body      string    `json:"body"`  // ≤500 tokens, shown on read
	CreatedAt time.Time `json:"created_at"`
}

// NoteBoard holds sticky notes in Substrate. Thread-safe.
// Notes expire on read. "Latest" auto-note key is "_latest".
type NoteBoard struct {
	mu    sync.RWMutex
	notes map[string][]Note // keyed by "to" agent ID
}

// NewNoteBoard creates an empty note board.
func NewNoteBoard() *NoteBoard {
	return &NoteBoard{notes: make(map[string][]Note)}
}

// Leave adds a note for an agent. Use "*" for broadcast.
func (b *NoteBoard) Leave(n Note) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if n.CreatedAt.IsZero() {
		n.CreatedAt = time.Now().UTC()
	}
	b.notes[n.To] = append(b.notes[n.To], n)
}

// Pending returns unread note titles for an agent (for prompt enrichment nag).
// Includes broadcast notes ("*"). Does NOT expire them.
func (b *NoteBoard) Pending(agentID string) []Note {
	b.mu.RLock()
	defer b.mu.RUnlock()

	pending := make([]Note, 0, len(b.notes[agentID])+len(b.notes["*"]))
	pending = append(pending, b.notes[agentID]...)
	pending = append(pending, b.notes["*"]...)
	return pending
}

// Read returns a note's body and expires it (deletes).
// Returns error if not found.
func (b *NoteBoard) Read(agentID, key string) (Note, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Check agent-specific notes.
	if notes, ok := b.notes[agentID]; ok {
		for i, n := range notes {
			if n.Key == key {
				b.notes[agentID] = append(notes[:i], notes[i+1:]...)
				return n, nil
			}
		}
	}

	// Check broadcast notes.
	if notes, ok := b.notes["*"]; ok {
		for i, n := range notes {
			if n.Key == key {
				b.notes["*"] = append(notes[:i], notes[i+1:]...)
				return n, nil
			}
		}
	}

	return Note{}, fmt.Errorf("%w: %s/%s", ErrNoteNotFound, agentID, key)
}

// SaveLatest auto-saves a "latest" note for an agent (on death).
// Overwrites any previous "_latest" note for that agent.
func (b *NoteBoard) SaveLatest(agentID, title, body string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Remove existing _latest for this agent.
	if notes, ok := b.notes[agentID]; ok {
		for i, n := range notes {
			if n.Key == "_latest" {
				b.notes[agentID] = append(notes[:i], notes[i+1:]...)
				break
			}
		}
	}

	b.notes[agentID] = append(b.notes[agentID], Note{
		From:      "system",
		To:        agentID,
		Key:       "_latest",
		Title:     title,
		Body:      body,
		CreatedAt: time.Now().UTC(),
	})
}

// Count returns total unread notes across all agents.
func (b *NoteBoard) Count() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	total := 0
	for _, notes := range b.notes {
		total += len(notes)
	}
	return total
}
