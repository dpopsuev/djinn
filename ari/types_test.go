package ari

import "testing"

func TestIntent_Construction(t *testing.T) {
	intent := Intent{
		ID:          "int-1",
		Action:      "fix-bug",
		Payload:     map[string]string{"file": "main.go"},
		Workstreams: []string{"ws-1"},
	}
	if intent.ID != "int-1" {
		t.Fatalf("ID = %q, want %q", intent.ID, "int-1")
	}
	if intent.Action != "fix-bug" {
		t.Fatalf("Action = %q, want %q", intent.Action, "fix-bug")
	}
	if intent.Payload["file"] != "main.go" {
		t.Fatalf("Payload[file] = %q, want %q", intent.Payload["file"], "main.go")
	}
}

func TestPermissionPayload_Construction(t *testing.T) {
	p := PermissionPayload{
		ExecID:      "exec-1",
		Stage:       "deploy",
		Description: "deploy to prod",
	}
	if p.ExecID != "exec-1" {
		t.Fatalf("ExecID = %q, want %q", p.ExecID, "exec-1")
	}
}

func TestPermissionResponse_Construction(t *testing.T) {
	r := PermissionResponse{ExecID: "exec-1", Approved: true}
	if !r.Approved {
		t.Fatal("Approved = false, want true")
	}
}

func TestResult_Construction(t *testing.T) {
	r := Result{
		IntentID: "int-1",
		Success:  true,
		Summary:  "done",
	}
	if !r.Success {
		t.Fatal("Success = false, want true")
	}
	if r.Summary != "done" {
		t.Fatalf("Summary = %q, want %q", r.Summary, "done")
	}
}

func TestResult_WithErrors(t *testing.T) {
	r := Result{
		IntentID: "int-2",
		Success:  false,
		Summary:  "failed",
		Errors:   []string{"lint failed", "test failed"},
	}
	if len(r.Errors) != 2 {
		t.Fatalf("len(Errors) = %d, want 2", len(r.Errors))
	}
}

func TestAlert_Construction(t *testing.T) {
	a := Alert{
		Source: "prometheus",
		Metric: "error_rate",
		Value:  7.2,
		Level:  "critical",
	}
	if a.Source != "prometheus" {
		t.Fatalf("Source = %q, want %q", a.Source, "prometheus")
	}
	if a.Value != 7.2 {
		t.Fatalf("Value = %f, want 7.2", a.Value)
	}
}
