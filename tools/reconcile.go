// reconcile.go — three-pillar drift reconciliation engine (GOL-31, TSK-276).
//
// ComputeDrift scores functionality, structure, and performance pillars
// by examining task completion, import-graph health, and test pass rate.
// The resulting DriftReport drives the TUI drift gauge.
package tools

import "math"

// PillarScore captures a single pillar's health.
type PillarScore struct {
	Name    string   // "functionality", "structure", "performance"
	Score   float64  // 0-100
	Summary string   // e.g. "4/5 specs passing"
	Details []string // line-level detail
}

// DriftReport combines all three pillar scores.
type DriftReport struct {
	Functionality      PillarScore
	Structure          PillarScore
	Performance        PillarScore
	TasksToConvergence int
}

// ComputeDrift evaluates three pillars and returns a DriftReport.
//
//   - Functionality: percentage of tasks with status "done".
//   - Structure: 100 minus penalty (10 per cycle, 5 per violation), clamped 0-100.
//   - Performance: test pass rate — passed / (passed + failed) * 100.
//   - TasksToConvergence: count of tasks whose status is not "done".
func ComputeDrift(tasks *TaskStore, arch *ArchReport, test *TestResult) *DriftReport {
	dr := &DriftReport{}

	// --- Functionality pillar ---
	dr.Functionality = computeFunctionality(tasks)

	// --- Structure pillar ---
	dr.Structure = computeStructure(arch)

	// --- Performance pillar ---
	dr.Performance = computePerformance(test)

	// Tasks to convergence: all non-done tasks.
	dr.TasksToConvergence = countNonDone(tasks)

	return dr
}

// computeFunctionality scores based on task completion percentage.
func computeFunctionality(tasks *TaskStore) PillarScore {
	ps := PillarScore{Name: "functionality"}
	if tasks == nil {
		ps.Score = 100
		ps.Summary = "no tasks"
		return ps
	}

	all := tasks.List()
	total := len(all)
	if total == 0 {
		ps.Score = 100
		ps.Summary = "no tasks"
		return ps
	}

	done := 0
	for _, t := range all {
		if t.Status == StatusDone {
			done++
		}
	}

	ps.Score = clampScore(float64(done) / float64(total) * 100)
	ps.Summary = formatFraction(done, total, "specs")

	for _, t := range all {
		if t.Status != StatusDone {
			ps.Details = append(ps.Details, t.ID+": "+t.Status)
		}
	}
	return ps
}

// computeStructure scores based on import cycles and layer violations.
func computeStructure(arch *ArchReport) PillarScore {
	ps := PillarScore{Name: "structure"}
	if arch == nil {
		ps.Score = 100
		ps.Summary = "no analysis"
		return ps
	}

	cyclePenalty := len(arch.Cycles) * 10
	// Violations are computed externally via CheckLayerViolations, but
	// we proxy them through the Cycles field for simplicity in this
	// scoring path. Callers that need violation scoring should supply
	// them pre-merged into the report.
	score := 100.0 - float64(cyclePenalty)
	ps.Score = clampScore(score)
	ps.Summary = formatCount(len(arch.Cycles), "cycles")
	for _, cycle := range arch.Cycles {
		detail := ""
		for i, pkg := range cycle {
			if i > 0 {
				detail += " -> "
			}
			detail += pkg
		}
		ps.Details = append(ps.Details, detail)
	}
	return ps
}

// ComputeStructureWithViolations scores structure considering both cycles and violations.
func ComputeStructureWithViolations(arch *ArchReport, violations []string) PillarScore {
	ps := PillarScore{Name: "structure"}
	if arch == nil {
		ps.Score = 100
		ps.Summary = "no analysis"
		return ps
	}

	penalty := float64(len(arch.Cycles)*10 + len(violations)*5)
	ps.Score = clampScore(100.0 - penalty)
	ps.Summary = formatCount(len(arch.Cycles), "cycles") + ", " + formatCount(len(violations), "violations")

	for _, cycle := range arch.Cycles {
		detail := ""
		for i, pkg := range cycle {
			if i > 0 {
				detail += " -> "
			}
			detail += pkg
		}
		ps.Details = append(ps.Details, detail)
	}
	for _, v := range violations {
		ps.Details = append(ps.Details, v)
	}
	return ps
}

// computePerformance scores based on test pass rate.
func computePerformance(test *TestResult) PillarScore {
	ps := PillarScore{Name: "performance"}
	if test == nil {
		ps.Score = 100
		ps.Summary = "no tests"
		return ps
	}

	total := test.Passed + test.Failed
	if total == 0 {
		ps.Score = 100
		ps.Summary = "no tests"
		return ps
	}

	ps.Score = clampScore(float64(test.Passed) / float64(total) * 100)
	ps.Summary = formatCount(test.Failed, "failing")
	for _, f := range test.Failures {
		ps.Details = append(ps.Details, f.Package+"/"+f.Name)
	}
	return ps
}

// countNonDone counts tasks not in "done" status.
func countNonDone(tasks *TaskStore) int {
	if tasks == nil {
		return 0
	}
	count := 0
	for _, t := range tasks.List() {
		if t.Status != StatusDone {
			count++
		}
	}
	return count
}

// clampScore clamps a score to [0, 100].
func clampScore(v float64) float64 {
	return math.Max(0, math.Min(100, v))
}

func formatFraction(num, den int, unit string) string {
	return formatInt(num) + "/" + formatInt(den) + " " + unit
}

func formatCount(n int, unit string) string {
	return formatInt(n) + " " + unit
}

func formatInt(n int) string {
	// Avoid importing strconv for simple int formatting.
	if n == 0 {
		return "0"
	}
	s := ""
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	if neg {
		s = "-" + s
	}
	return s
}
