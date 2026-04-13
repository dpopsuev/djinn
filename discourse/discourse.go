// Package discourse provides the communication layer for operator + agent interaction.
//
// Forum → Topic → Thread → Message. Every interaction flows through here.
// Operator is Entity 0 — posts like any agent. Trail persists across agent deaths.
//
// GOL-133, ADR-17
package discourse

import (
	"errors"
	"fmt"
	"time"
)

// ErrThreadNotFound is returned when replying to a nonexistent thread.
var ErrThreadNotFound = errors.New("discourse: thread not found")

// Forum is a scope-bound collection of topics for operator + agent communication.
type Forum interface {
	// Post creates a new thread in the given topic with an authored message.
	// Returns the thread ID for subsequent replies.
	Post(topic string, msg Message) (ThreadID, error)

	// Reply adds an authored message to an existing thread.
	Reply(id ThreadID, msg Message) error

	// Threads returns all threads in a topic.
	Threads(topic string) []Thread
}

// Message is an authored communication in a thread.
type Message struct {
	From    string    `json:"from"` // entity ID or role ("operator", "gensec", "coder-1")
	Content string    `json:"content"`
	Time    time.Time `json:"time"`
}

// ThreadID uniquely identifies a thread.
type ThreadID string

// Thread is a sequence of authored messages in a topic.
type Thread struct {
	ID       ThreadID  `json:"id"`
	Topic    string    `json:"topic"`
	Messages []Message `json:"messages"`
}

// StubForum is an in-memory Forum for testing. Records all posts and replies.
type StubForum struct {
	threads map[ThreadID]*Thread
	topics  map[string][]ThreadID
	nextID  int
}

// NewStubForum creates an empty in-memory Forum.
func NewStubForum() *StubForum {
	return &StubForum{
		threads: make(map[ThreadID]*Thread),
		topics:  make(map[string][]ThreadID),
	}
}

func (f *StubForum) Post(topic string, msg Message) (ThreadID, error) {
	f.nextID++
	id := ThreadID(fmt.Sprintf("thread-%d", f.nextID))
	if msg.Time.IsZero() {
		msg.Time = time.Now()
	}
	t := &Thread{
		ID:       id,
		Topic:    topic,
		Messages: []Message{msg},
	}
	f.threads[id] = t
	f.topics[topic] = append(f.topics[topic], id)
	return id, nil
}

func (f *StubForum) Reply(id ThreadID, msg Message) error {
	t, ok := f.threads[id]
	if !ok {
		return fmt.Errorf("%w: %q", ErrThreadNotFound, id)
	}
	if msg.Time.IsZero() {
		msg.Time = time.Now()
	}
	t.Messages = append(t.Messages, msg)
	return nil
}

func (f *StubForum) Threads(topic string) []Thread {
	ids := f.topics[topic]
	out := make([]Thread, 0, len(ids))
	for _, id := range ids {
		if t, ok := f.threads[id]; ok {
			out = append(out, *t)
		}
	}
	return out
}

// ThreadCount returns total thread count for testing.
func (f *StubForum) ThreadCount() int { return len(f.threads) }

// Interface compliance.
var _ Forum = (*StubForum)(nil)
