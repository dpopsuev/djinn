package ari

import (
	"time"

	ariTypes "github.com/dpopsuev/djinn/ari"
	"github.com/dpopsuev/djinn/orchestrator"
)

// MockARIClient provides a test-friendly interface for interacting with MockARIServer.
type MockARIClient struct {
	server *MockARIServer
}

// NewMockARIClient creates a client connected to the given mock server.
func NewMockARIClient(server *MockARIServer) *MockARIClient {
	return &MockARIClient{server: server}
}

// SendIntent sends an intent to the server.
func (c *MockARIClient) SendIntent(intent ariTypes.Intent) {
	c.server.InjectIntent(intent)
}

// RespondPermission sends a permission response to the server.
func (c *MockARIClient) RespondPermission(resp ariTypes.PermissionResponse) {
	c.server.permRespCh <- resp
}

// WaitForEvent polls until an event with the given kind appears, or times out.
func (c *MockARIClient) WaitForEvent(kind orchestrator.EventKind, timeout time.Duration) (orchestrator.Event, bool) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		events := c.server.RecordedEvents()
		for _, e := range events {
			if e.Kind == kind {
				return e, true
			}
		}
		time.Sleep(5 * time.Millisecond)
	}
	return orchestrator.Event{}, false
}

// WaitForResult polls until a result appears, or times out.
func (c *MockARIClient) WaitForResult(timeout time.Duration) (ariTypes.Result, bool) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		results := c.server.RecordedResults()
		if len(results) > 0 {
			return results[len(results)-1], true
		}
		time.Sleep(5 * time.Millisecond)
	}
	return ariTypes.Result{}, false
}

// Events returns all recorded events.
func (c *MockARIClient) Events() []orchestrator.Event {
	return c.server.RecordedEvents()
}

// Results returns all recorded results.
func (c *MockARIClient) Results() []ariTypes.Result {
	return c.server.RecordedResults()
}
