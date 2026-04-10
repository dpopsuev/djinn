// Package discourse provides the planning coordination layer.
// Discourse = natural language deliberation. Program.
// Forum → Topic → Thread. GenSec stewards the General Discourse.
package discourse

// Forum is a scope-bound collection of topics for agent deliberation.
type Forum interface {
	// Post creates a new thread in the given topic.
	Post(topic, message string) (ThreadID, error)

	// Reply adds a message to an existing thread.
	Reply(id ThreadID, message string) error

	// Threads returns all threads in a topic.
	Threads(topic string) []Thread
}

// ThreadID uniquely identifies a thread.
type ThreadID string

// Thread is a sequence of messages in a topic.
type Thread struct {
	ID       ThreadID `json:"id"`
	Topic    string   `json:"topic"`
	Messages []string `json:"messages"`
}
