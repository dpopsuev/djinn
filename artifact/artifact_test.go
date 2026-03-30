package artifact

import (
	"encoding/json"
	"testing"
	"time"
)

func TestArtifact_JSONRoundTrip(t *testing.T) {
	a := Artifact{
		ID:      "seg-1",
		Kind:    "plan-segment",
		Title:   "implement auth",
		Status:  StatusDraft,
		Owner:   "executor-1",
		Content: "acceptance criteria here",
		Sections: map[string]string{
			"content":      "acceptance criteria here",
			"verification": "go test ./auth/",
		},
		Components: ComponentMap{
			Directories: []string{"auth/"},
			Files:       []string{"auth/handler.go"},
			Symbols:     []string{"LoginHandler"},
		},
		DependsOn:   []string{"seg-0"},
		Children:    []string{"seg-2"},
		Parent:      "goal-1",
		Labels:      []string{"critical"},
		Annotations: []Annotation{{Kind: "+", Comment: "looks good"}},
		Version:     1,
		Priority:    1,
		Created:     time.Date(2026, 3, 30, 0, 0, 0, 0, time.UTC),
		Updated:     time.Date(2026, 3, 30, 0, 0, 0, 0, time.UTC),
	}

	data, err := json.Marshal(a)
	if err != nil {
		t.Fatal(err)
	}

	var decoded Artifact
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}

	if decoded.ID != a.ID {
		t.Errorf("ID = %q, want %q", decoded.ID, a.ID)
	}
	if decoded.Kind != a.Kind {
		t.Errorf("Kind = %q, want %q", decoded.Kind, a.Kind)
	}
	if decoded.Status != a.Status {
		t.Errorf("Status = %q, want %q", decoded.Status, a.Status)
	}
	if decoded.Content != a.Content {
		t.Errorf("Content = %q, want %q", decoded.Content, a.Content)
	}
	if len(decoded.Components.Files) != 1 || decoded.Components.Files[0] != "auth/handler.go" {
		t.Errorf("Components.Files = %v, want [auth/handler.go]", decoded.Components.Files)
	}
	if len(decoded.Sections) != 2 { //nolint:mnd // expected 2 sections
		t.Errorf("Sections count = %d, want 2", len(decoded.Sections))
	}
	if len(decoded.Annotations) != 1 || decoded.Annotations[0].Kind != "+" {
		t.Errorf("Annotations = %v, want [{+ looks good}]", decoded.Annotations)
	}
}

func TestStatus_StringValues(t *testing.T) {
	tests := []struct {
		status Status
		want   string
	}{
		{StatusDraft, "draft"},
		{StatusReady, "ready"},
		{StatusClaimed, "claimed"},
		{StatusInProgress, "in_progress"},
		{StatusComplete, "complete"},
		{StatusInvalidated, "invalidated"},
		{StatusBlocked, "blocked"},
		{StatusPending, "pending"},
		{StatusActive, "active"},
		{StatusDone, "done"},
	}
	for _, tt := range tests {
		if string(tt.status) != tt.want {
			t.Errorf("Status(%q) string = %q, want %q", tt.status, string(tt.status), tt.want)
		}
	}
}
