// code_hub.go — CodeHub mediates code review operations (GOL-58).
//
// Wraps review.ReviewWindow and review.BudgetMonitor with five-step mediation.
// Day 1: internal budget monitoring. Day 2: CodeReviewerPort for external reviews.
package hub

import (
	"context"

	"github.com/dpopsuev/djinn/review"
	"github.com/dpopsuev/djinn/signal"
)

// CodeHub mediates between code review tools, budget monitoring, and display.
type CodeHub struct {
	HubCore
	Window *review.ReviewWindow
	Budget *review.BudgetMonitor
	// Reviewer port removed — Day 2 adapter will implement StructuralAnalyzerPort instead.
}

// NewCodeHub creates a code hub with the given review window.
func NewCodeHub(core HubCore, window *review.ReviewWindow) *CodeHub {
	ch := &CodeHub{HubCore: core, Window: window}
	if window != nil {
		ch.Budget = window.Budget
	}
	return ch
}

// Name returns the hub name.
func (h *CodeHub) Name() string { return codeHubName }

// Phase returns the DevOps phase.
func (h *CodeHub) Phase() string { return codePhase }

// RecordChange wraps ReviewWindow.RecordChange with mediation.
func (h *CodeHub) RecordChange(file string) {
	if h.Window == nil {
		return
	}

	h.Window.RecordChange(file)

	h.Trace("file-change", file)

	h.Render(DisplayMsg{
		Source:   codeHubName,
		Category: codeHubName,
		Content:  CodeEvent{Action: "change", File: file},
	})
}

// CheckBudget wraps ReviewWindow.CheckBudget with signal emission on breach.
func (h *CodeHub) CheckBudget(ctx context.Context) (bool, []review.Signal) {
	if h.Window == nil {
		return false, nil
	}

	exceeded, signals := h.Window.CheckBudget(ctx)

	if exceeded {
		h.Trace("budget-exceeded", "review backpressure triggered")

		h.Emit(signal.Signal{
			Category: codeHubName,
			Level:    signal.Yellow,
			Source:   codeHubName + "-hub",
			Message:  "budget thresholds exceeded — review recommended",
		})

		h.Render(DisplayMsg{
			Source:   codeHubName,
			Category: codeHubName,
			Content:  CodeEvent{Action: "budget-exceeded", Exceeded: true},
		})
	}

	return exceeded, signals
}

// CodeEvent is the display payload for code operations.
type CodeEvent struct {
	Action   string `json:"action"`
	File     string `json:"file,omitempty"`
	Exceeded bool   `json:"exceeded,omitempty"`
}

// Compile-time interface check.
var _ MediatorHub = (*CodeHub)(nil)
