// window.go — ReviewWindow state machine (TSK-436).
//
// Working → Reviewing → Approved | Rejected | Split.
// Each window tracks changed files, accumulated signals, and scope anchor.
package review

import (
	"context"
	"errors"
	"time"
)

// WindowState represents the review window lifecycle.
type WindowState int

const (
	WindowWorking   WindowState = iota // agent is producing changes
	WindowReviewing                    // operator is reviewing
	WindowApproved                     // operator approved all changes
	WindowRejected                     // operator rejected all changes
	WindowSplit                        // operator partially approved
)

// String returns the state name.
func (ws WindowState) String() string {
	switch ws {
	case WindowWorking:
		return "working"
	case WindowReviewing:
		return "reviewing"
	case WindowApproved:
		return "approved"
	case WindowRejected:
		return "rejected"
	case WindowSplit:
		return "split"
	default:
		return "unknown"
	}
}

// State machine errors.
var (
	ErrNotWorking   = errors.New("review: window is not in working state")
	ErrNotReviewing = errors.New("review: window is not in reviewing state")
)

// ReviewWindow manages the lifecycle of a single review cycle.
type ReviewWindow struct {
	State        WindowState
	Anchor       *ScopeAnchor
	Budget       *BudgetMonitor
	Signals      []Signal // accumulated from budget checks
	ChangedFiles []string
	StartedAt    time.Time
}

// NewReviewWindow creates a window in Working state.
func NewReviewWindow(request string, budget *BudgetMonitor) *ReviewWindow {
	return &ReviewWindow{
		State:     WindowWorking,
		Anchor:    NewScopeAnchor(request),
		Budget:    budget,
		StartedAt: time.Now(),
	}
}

// RecordChange tracks a file modification in the current window.
func (rw *ReviewWindow) RecordChange(file string) {
	for _, f := range rw.ChangedFiles {
		if f == file {
			return // already tracked
		}
	}
	rw.ChangedFiles = append(rw.ChangedFiles, file)
}

// CheckBudget evaluates all registered heuristics against current changes.
// Returns true if any signal exceeds its threshold.
func (rw *ReviewWindow) CheckBudget(ctx context.Context) (bool, []Signal) {
	if rw.Budget == nil {
		return false, nil
	}
	diff := &DiffSnapshot{
		ChangedFiles: rw.ChangedFiles,
		PackagesHit:  uniqueDirs(rw.ChangedFiles),
	}
	signals := rw.Budget.Check(ctx, diff)
	rw.Signals = append(rw.Signals, signals...)
	exceeded := Exceeded(signals)
	return len(exceeded) > 0, exceeded
}

// EnterReview transitions from Working to Reviewing.
func (rw *ReviewWindow) EnterReview() error {
	if rw.State != WindowWorking {
		return ErrNotWorking
	}
	rw.State = WindowReviewing
	return nil
}

// Approve transitions from Reviewing to Approved.
func (rw *ReviewWindow) Approve() error {
	if rw.State != WindowReviewing {
		return ErrNotReviewing
	}
	rw.State = WindowApproved
	return nil
}

// Reject transitions from Reviewing to Rejected.
func (rw *ReviewWindow) Reject() error {
	if rw.State != WindowReviewing {
		return ErrNotReviewing
	}
	rw.State = WindowRejected
	return nil
}

// Split transitions from Reviewing to Split with separate approve/reject sets.
func (rw *ReviewWindow) Split(_, _ []string) error {
	if rw.State != WindowReviewing {
		return ErrNotReviewing
	}
	rw.State = WindowSplit
	return nil
}

// uniqueDirs extracts unique parent directories from file paths.
func uniqueDirs(files []string) []string {
	seen := make(map[string]bool, len(files))
	var dirs []string
	for _, f := range files {
		dir := parentDir(f)
		if !seen[dir] {
			seen[dir] = true
			dirs = append(dirs, dir)
		}
	}
	return dirs
}

func parentDir(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' {
			return path[:i]
		}
	}
	return "."
}
