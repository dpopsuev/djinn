// notes.go — sticky notes in Substrate, backed by L2 cache.
// Typed wrapper over cache.Cache. Notes are L2 entries with "note:" key prefix.
// Expire on read. Cross-agent via scope. Broadcast via "*" scope.
package substrate

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	djinncache "github.com/dpopsuev/djinn/cache"
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

const notePrefix = "note:"

// NoteBoard is a typed wrapper over L2 cache for sticky notes.
// Notes are L2 entries with "note:" key prefix. Thread-safety via L2 cache.
type NoteBoard struct {
	l2 djinncache.Cache
}

// NewNoteBoard creates a note board backed by an L2 cache.
func NewNoteBoard(l2 djinncache.Cache) *NoteBoard {
	return &NoteBoard{l2: l2}
}

// Leave adds a note for an agent. Use "*" for broadcast.
func (b *NoteBoard) Leave(n Note) {
	if n.CreatedAt.IsZero() {
		n.CreatedAt = time.Now().UTC()
	}
	data, err := json.Marshal(n)
	if err != nil {
		return
	}
	b.l2.Put(n.To, notePrefix+n.Key, data)
}

// Pending returns unread notes for an agent (for prompt enrichment nag).
// Includes broadcast notes ("*"). Does NOT expire them.
func (b *NoteBoard) Pending(agentID string) []Note {
	var pending []Note
	// Agent-specific notes.
	for _, key := range b.l2.Keys(agentID) {
		if !strings.HasPrefix(key, notePrefix) {
			continue
		}
		if data, ok := b.l2.Get(agentID, key); ok {
			var n Note
			if err := json.Unmarshal(data, &n); err == nil {
				pending = append(pending, n)
			}
		}
	}
	// Broadcast notes.
	for _, key := range b.l2.Keys("*") {
		if !strings.HasPrefix(key, notePrefix) {
			continue
		}
		if data, ok := b.l2.Get("*", key); ok {
			var n Note
			if err := json.Unmarshal(data, &n); err == nil {
				pending = append(pending, n)
			}
		}
	}
	return pending
}

// Read returns a note's body and expires it (deletes from L2).
// Returns error if not found.
func (b *NoteBoard) Read(agentID, key string) (Note, error) {
	cacheKey := notePrefix + key

	// Check agent-specific.
	if data, ok := b.l2.Get(agentID, cacheKey); ok {
		b.l2.Evict(agentID, cacheKey)
		var n Note
		if err := json.Unmarshal(data, &n); err == nil {
			return n, nil
		}
	}

	// Check broadcast.
	if data, ok := b.l2.Get("*", cacheKey); ok {
		b.l2.Evict("*", cacheKey)
		var n Note
		if err := json.Unmarshal(data, &n); err == nil {
			return n, nil
		}
	}

	return Note{}, fmt.Errorf("%w: %s/%s", ErrNoteNotFound, agentID, key)
}

// SaveLatest auto-saves a "_latest" note for an agent (on death).
// Overwrites any previous "_latest" note.
func (b *NoteBoard) SaveLatest(agentID, title, body string) {
	b.Leave(Note{
		From:  "system",
		To:    agentID,
		Key:   "_latest",
		Title: title,
		Body:  body,
	})
}

// Count returns total unread notes across all scopes.
func (b *NoteBoard) Count() int {
	total := 0
	for _, scope := range b.l2.Scopes() {
		for _, key := range b.l2.Keys(scope) {
			if strings.HasPrefix(key, notePrefix) {
				total++
			}
		}
	}
	return total
}
