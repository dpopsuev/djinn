// discourse.go — forum-style conversation persistence for scoped agent sessions.
//
// DiscourseStore manages forums (one per scope), topics, threads, and messages.
// Thread-safe, file-backed with atomic writes (temp + rename), following the
// same pattern as TaskStore.
package tools

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"sync"
	"time"
)

// Sentinel errors for DiscourseStore operations.
var (
	ErrForumNotFound  = errors.New("forum not found")
	ErrTopicNotFound  = errors.New("topic not found")
	ErrThreadNotFound = errors.New("thread not found")
	ErrInvalidStatus  = errors.New("invalid topic status")
)

// ThreadMessage is a single message within a conversation thread.
type ThreadMessage struct {
	Role      string    `json:"role"` // "user" or "assistant"
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

// Thread is a linear sequence of messages within a topic.
type Thread struct {
	ID       string          `json:"id"`
	Messages []ThreadMessage `json:"messages"`
	Status   string          `json:"status"` // "active", "paused", "abandoned"
	Created  time.Time       `json:"created"`
	Updated  time.Time       `json:"updated"`
}

// Topic groups related threads under a title and kind.
type Topic struct {
	ID      string    `json:"id"`
	Title   string    `json:"title"`
	Kind    string    `json:"kind"`   // "feature", "bug", "refactor", "spike", "chore", "discussion"
	Status  string    `json:"status"` // "draft", "stash", "active", "complete", "archived"
	Threads []*Thread `json:"threads"`
	Created time.Time `json:"created"`
	Updated time.Time `json:"updated"`
}

// Forum is the top-level container for a single scope's conversation history.
type Forum struct {
	Scope  string   `json:"scope"` // e.g. "/aeon/djinn"
	Topics []*Topic `json:"topics"`
}

// Valid topic statuses.
const (
	TopicDraft    = "draft"
	TopicStash    = "stash"
	TopicActive   = "active"
	TopicComplete = "complete"
	TopicArchived = "archived"
)

// Valid thread statuses.
const (
	ThreadActive    = "active"
	ThreadPaused    = "paused"
	ThreadAbandoned = "abandoned"
)

// DiscourseStore is a thread-safe, file-backed collection of forums.
type DiscourseStore struct {
	mu      sync.RWMutex
	forums  map[string]*Forum // keyed by scope path
	path    string            // JSON file path
	nextIDs map[string]int    // per-scope auto-increment for topic/thread IDs
}

// NewDiscourseStore creates a DiscourseStore backed by the given file path.
func NewDiscourseStore(path string) *DiscourseStore {
	return &DiscourseStore{
		forums:  make(map[string]*Forum),
		path:    path,
		nextIDs: make(map[string]int),
	}
}

// GetForum returns the forum for a scope, creating one if it doesn't exist.
func (d *DiscourseStore) GetForum(scope string) *Forum {
	d.mu.Lock()
	defer d.mu.Unlock()

	f, ok := d.forums[scope]
	if !ok {
		f = &Forum{Scope: scope}
		d.forums[scope] = f
	}
	return f
}

// nextID generates the next sequential ID for a scope namespace.
func (d *DiscourseStore) nextID(namespace string) string {
	d.nextIDs[namespace]++
	return fmt.Sprintf("%s-%03d", namespace, d.nextIDs[namespace])
}

// CreateTopic adds a new topic to the forum for the given scope.
func (d *DiscourseStore) CreateTopic(scope, title, kind string) *Topic {
	d.mu.Lock()
	defer d.mu.Unlock()

	f, ok := d.forums[scope]
	if !ok {
		f = &Forum{Scope: scope}
		d.forums[scope] = f
	}

	now := time.Now()
	topic := &Topic{
		ID:      d.nextID("D"),
		Title:   title,
		Kind:    kind,
		Status:  TopicDraft,
		Created: now,
		Updated: now,
	}
	f.Topics = append(f.Topics, topic)
	return topic
}

// GetTopic returns a topic by ID within the given scope.
func (d *DiscourseStore) GetTopic(scope, topicID string) (*Topic, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	f, ok := d.forums[scope]
	if !ok {
		return nil, false
	}
	for _, t := range f.Topics {
		if t.ID == topicID {
			return t, true
		}
	}
	return nil, false
}

// UpdateTopicStatus changes a topic's status. Returns an error if the
// topic is not found or the status is invalid.
func (d *DiscourseStore) UpdateTopicStatus(scope, topicID, status string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	f, ok := d.forums[scope]
	if !ok {
		return fmt.Errorf("%w: scope %q", ErrForumNotFound, scope)
	}

	switch status {
	case TopicDraft, TopicStash, TopicActive, TopicComplete, TopicArchived:
	default:
		return fmt.Errorf("%w: %q", ErrInvalidStatus, status)
	}

	for _, t := range f.Topics {
		if t.ID == topicID {
			t.Status = status
			t.Updated = time.Now()
			return nil
		}
	}
	return fmt.Errorf("%w: %q in scope %q", ErrTopicNotFound, topicID, scope)
}

// CreateThread adds a new thread to a topic. Returns an error if the topic
// is not found.
func (d *DiscourseStore) CreateThread(scope, topicID string) (*Thread, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	f, ok := d.forums[scope]
	if !ok {
		return nil, fmt.Errorf("%w: scope %q", ErrForumNotFound, scope)
	}

	for _, topic := range f.Topics {
		if topic.ID != topicID {
			continue
		}
		now := time.Now()
		thread := &Thread{
			ID:      d.nextID("TH"),
			Status:  ThreadActive,
			Created: now,
			Updated: now,
		}
		topic.Threads = append(topic.Threads, thread)
		topic.Updated = now
		return thread, nil
	}
	return nil, fmt.Errorf("%w: %q in scope %q", ErrTopicNotFound, topicID, scope)
}

// AppendMessage adds a message to a thread. Returns an error if the
// topic or thread is not found.
func (d *DiscourseStore) AppendMessage(scope, topicID, threadID, role, content string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	f, ok := d.forums[scope]
	if !ok {
		return fmt.Errorf("%w: scope %q", ErrForumNotFound, scope)
	}

	for _, topic := range f.Topics {
		if topic.ID != topicID {
			continue
		}
		for _, thread := range topic.Threads {
			if thread.ID != threadID {
				continue
			}
			now := time.Now()
			thread.Messages = append(thread.Messages, ThreadMessage{
				Role:      role,
				Content:   content,
				Timestamp: now,
			})
			thread.Updated = now
			topic.Updated = now
			return nil
		}
		return fmt.Errorf("%w: %q in topic %q", ErrThreadNotFound, threadID, topicID)
	}
	return fmt.Errorf("%w: %q in scope %q", ErrTopicNotFound, topicID, scope)
}

// StaleTopics returns topics that haven't been updated within the given threshold.
// Only topics with open statuses (draft, stash, active) are considered.
func (d *DiscourseStore) StaleTopics(scope string, threshold time.Duration) []*Topic {
	d.mu.RLock()
	defer d.mu.RUnlock()

	f, ok := d.forums[scope]
	if !ok {
		return nil
	}

	cutoff := time.Now().Add(-threshold)
	var stale []*Topic
	for _, t := range f.Topics {
		switch t.Status {
		case TopicComplete, TopicArchived:
			continue
		}
		if t.Updated.Before(cutoff) {
			stale = append(stale, t)
		}
	}
	return stale
}

// OpenTopicCount returns the number of topics with open statuses
// (draft, stash, active) in the given scope.
func (d *DiscourseStore) OpenTopicCount(scope string) int {
	d.mu.RLock()
	defer d.mu.RUnlock()

	f, ok := d.forums[scope]
	if !ok {
		return 0
	}

	count := 0
	for _, t := range f.Topics {
		switch t.Status {
		case TopicDraft, TopicStash, TopicActive:
			count++
		}
	}
	return count
}

// discourseStoreState is the JSON-serialized form.
type discourseStoreState struct {
	Forums  []*Forum       `json:"forums"`
	NextIDs map[string]int `json:"next_ids"`
}

func (d *DiscourseStore) marshalState() discourseStoreState {
	forums := make([]*Forum, 0, len(d.forums))
	for _, f := range d.forums {
		forums = append(forums, f)
	}
	sort.Slice(forums, func(i, j int) bool { return forums[i].Scope < forums[j].Scope })
	return discourseStoreState{Forums: forums, NextIDs: d.nextIDs}
}

// Save writes the discourse store to disk as JSON. The write is atomic
// (write to temp then rename).
func (d *DiscourseStore) Save() error {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return atomicSaveJSON(d.path, d.marshalState(), "discourse")
}

// Load reads discourse data from the file. Existing in-memory data is replaced.
func (d *DiscourseStore) Load() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	data, err := os.ReadFile(d.path)
	if err != nil {
		return fmt.Errorf("read discourse: %w", err)
	}

	var state discourseStoreState
	if err := json.Unmarshal(data, &state); err != nil {
		return fmt.Errorf("unmarshal discourse: %w", err)
	}

	d.forums = make(map[string]*Forum, len(state.Forums))
	for _, f := range state.Forums {
		d.forums[f.Scope] = f
	}
	d.nextIDs = state.NextIDs
	if d.nextIDs == nil {
		d.nextIDs = make(map[string]int)
	}
	return nil
}
