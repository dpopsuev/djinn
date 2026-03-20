package ari

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
	Source  string
	Metric string
	Value  float64
	Level  string
}
