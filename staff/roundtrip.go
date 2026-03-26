// roundtrip.go — Human operator round-trip registry.
// Tracks messages sent to the operator and whether they have been
// acknowledged and responded to, enabling "pending message" tracking.
package staff

import (
	"fmt"
	"sync"
	"time"
)

// PendingMessage represents a message awaiting human response.
type PendingMessage struct {
	ID           string
	Content      string
	Received     time.Time
	Acknowledged bool
	Responded    bool
}

// RoundTripRegistry tracks messages sent to the human operator.
type RoundTripRegistry struct {
	mu       sync.Mutex
	messages map[string]*PendingMessage
	nextID   int
}

// NewRoundTripRegistry creates an empty registry.
func NewRoundTripRegistry() *RoundTripRegistry {
	return &RoundTripRegistry{
		messages: make(map[string]*PendingMessage),
	}
}

// Register records a new message and returns its ID.
func (r *RoundTripRegistry) Register(content string) string {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.nextID++
	id := fmt.Sprintf("msg-%d", r.nextID)
	r.messages[id] = &PendingMessage{
		ID:       id,
		Content:  content,
		Received: time.Now(),
	}
	return id
}

// Acknowledge marks a message as acknowledged (seen by the operator).
func (r *RoundTripRegistry) Acknowledge(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if msg, ok := r.messages[id]; ok {
		msg.Acknowledged = true
	}
}

// Respond marks a message as responded (operator sent a reply).
func (r *RoundTripRegistry) Respond(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if msg, ok := r.messages[id]; ok {
		msg.Responded = true
	}
}

// Pending returns all messages that have not been responded to.
// Returns a copy of each PendingMessage to avoid data races.
func (r *RoundTripRegistry) Pending() []*PendingMessage {
	r.mu.Lock()
	defer r.mu.Unlock()
	var result []*PendingMessage
	for _, msg := range r.messages {
		if !msg.Responded {
			cp := *msg
			result = append(result, &cp)
		}
	}
	return result
}

// PendingCount returns the number of unresponded messages.
func (r *RoundTripRegistry) PendingCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	count := 0
	for _, msg := range r.messages {
		if !msg.Responded {
			count++
		}
	}
	return count
}
