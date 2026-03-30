// templates.go — Built-in artifact templates for plan-segment and task (GOL-59).
//
// Each template defines required sections, valid statuses, and ID format.
// DefaultRegistry() returns a registry pre-loaded with both templates.
package artifact

// Built-in artifact kind constants.
const (
	KindPlanSegment = "plan-segment"
	KindTask        = "task"
)

// PlanSegmentTemplate returns the template for plan segments.
// Matches the lifecycle of the original plan.Segment.
func PlanSegmentTemplate() Template {
	return Template{
		Kind: KindPlanSegment,
		// Content is NOT required at creation — drafts start empty,
		// FillDraft adds content and transitions to ready.
		ValidStatuses: []Status{
			StatusDraft, StatusReady, StatusClaimed,
			StatusInProgress, StatusComplete, StatusInvalidated,
		},
		IDFormat: "seg-%d",
	}
}

// TaskTemplate returns the template for tasks.
// Matches the lifecycle of the original tools.Task.
func TaskTemplate() Template {
	return Template{
		Kind:          KindTask,
		ValidStatuses: []Status{StatusPending, StatusActive, StatusDone, StatusBlocked},
		IDFormat:      "T-%03d",
	}
}

// DefaultRegistry returns a registry pre-loaded with built-in templates.
func DefaultRegistry() *TemplateRegistry {
	r := NewTemplateRegistry()
	r.Register(PlanSegmentTemplate())
	r.Register(TaskTemplate())
	return r
}
