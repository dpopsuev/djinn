// Package assignment provides the execution coordination layer.
// Assignment = structured work unit downstream of Discourse decisions.
// Process — the running program. Tracks who does what, when, result.
package assignment

// Status tracks assignment lifecycle.
type Status string

const (
	Assigned   Status = "assigned"
	InProgress Status = "in_progress"
	Done       Status = "done"
	Failed     Status = "failed"
)

// Assignment is a unit of work assigned to an agent.
type Assignment struct {
	ID      string `json:"id"`
	AgentID string `json:"agent_id"`
	Task    string `json:"task"`
	Status  Status `json:"status"`
	Result  string `json:"result,omitempty"`
}

// Manager dispatches and tracks assignments.
type Manager interface {
	// Assign creates a new assignment for an agent.
	Assign(agentID, task string) (string, error)

	// Update changes the status of an assignment.
	Update(id string, status Status, result string) error

	// List returns assignments filtered by status.
	List(status Status) []Assignment
}
