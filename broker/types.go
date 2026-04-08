// types.go — data types for broker's operator protocol.
//
// Formerly in ari/ — moved here because broker is the only producer.
package broker

// Intent represents a user's request to the agent runtime.
type Intent struct {
	ID          string
	Action      string
	Payload     map[string]string
	Workstreams []string
}

// PermissionPayload describes a permission request emitted to the operator.
type PermissionPayload struct {
	ExecID      string
	Stage       string
	Description string
}

// PermissionResponse is the operator's answer to a permission request.
type PermissionResponse struct {
	ExecID   string
	Approved bool
}

// Result represents the final outcome of an intent execution.
type Result struct {
	IntentID string
	Success  bool
	Summary  string
	Errors   []string
}

// Alert represents an event from an external monitoring system.
type Alert struct {
	Source string
	Metric string
	Value  float64
	Level  string
}

// WorkstreamSnapshot is a summary of a workstream for operator consumers.
type WorkstreamSnapshot struct {
	ID       string `json:"id"`
	IntentID string `json:"intent_id"`
	Action   string `json:"action"`
	Status   string `json:"status"`
	Health   string `json:"health"`
}

// AndonSnapshot is a summary of the factory Andon board for operator consumers.
type AndonSnapshot struct {
	Level       string               `json:"level"`
	Workstreams []WorkstreamSnapshot `json:"workstreams"`
	Cordons     int                  `json:"cordons"`
}

// SearchResult is defined in search.go (same package).
