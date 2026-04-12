package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/dpopsuev/djinn/assignment"
)

// AssignmentTool exposes the Assignment manager as a builtin tool.
// Push-side work coordination: assign, unassign, list.
// Requires coordinate capability (REF-77).
type AssignmentTool struct {
	Manager assignment.Manager
}

func (t *AssignmentTool) Name() string { return "assignment" }
func (t *AssignmentTool) Description() string {
	return "Dispatch and track work assignments between agents: assign, unassign, list"
}

func (t *AssignmentTool) InputSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"action":   {"type": "string", "enum": ["assign", "unassign", "update", "list"]},
			"agent_id": {"type": "string", "description": "Target agent ID"},
			"task":     {"type": "string", "description": "Task description or ID"},
			"id":       {"type": "string", "description": "Assignment ID (for update/unassign)"},
			"status":   {"type": "string", "enum": ["assigned", "in_progress", "done", "failed"]},
			"result":   {"type": "string", "description": "Result text (for update)"},
			"comment":  {"type": "string", "description": "Required comment on state change"}
		},
		"required": ["action"]
	}`)
}

func (t *AssignmentTool) Execute(_ context.Context, input json.RawMessage) (string, error) {
	var req struct {
		Action  string `json:"action"`
		AgentID string `json:"agent_id"`
		Task    string `json:"task"`
		ID      string `json:"id"`
		Status  string `json:"status"`
		Result  string `json:"result"`
		Comment string `json:"comment"`
	}
	if err := json.Unmarshal(input, &req); err != nil {
		return "", fmt.Errorf("assignment: %w", err)
	}

	switch req.Action {
	case "assign":
		if req.AgentID == "" || req.Task == "" {
			return "", fmt.Errorf("assignment assign: agent_id and task required")
		}
		id, err := t.Manager.Assign(req.AgentID, req.Task)
		if err != nil {
			return "", fmt.Errorf("assignment assign: %w", err)
		}
		return fmt.Sprintf("Assigned %s to %s (id: %s)", req.Task, req.AgentID, id), nil

	case "update":
		if req.ID == "" || req.Status == "" {
			return "", fmt.Errorf("assignment update: id and status required")
		}
		if req.Comment == "" {
			return "", fmt.Errorf("assignment update: comment required on state change")
		}
		status := assignment.Status(req.Status)
		if err := t.Manager.Update(req.ID, status, req.Result); err != nil {
			return "", fmt.Errorf("assignment update: %w", err)
		}
		return fmt.Sprintf("Updated %s to %s", req.ID, req.Status), nil

	case "list":
		status := assignment.Assigned
		if req.Status != "" {
			status = assignment.Status(req.Status)
		}
		items := t.Manager.List(status)
		if len(items) == 0 {
			return "No assignments with status: " + string(status), nil
		}
		var sb strings.Builder
		for _, a := range items {
			fmt.Fprintf(&sb, "%s | %s | %s | %s\n", a.ID, a.AgentID, a.Task, a.Status)
		}
		return sb.String(), nil

	default:
		return "", fmt.Errorf("assignment: unknown action %q (expected: assign, update, list)", req.Action)
	}
}
