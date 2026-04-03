// scratch.go — ScratchPaper artifact kind for workstation state (TSK-652).
//
// Scratch paper is a child artifact of a task. It holds structured
// sections that agents write to communicate state across relay handoffs:
// understanding, plan, risks, notes. These are regular artifact sections.
package artifact

// KindScratchPaper is the artifact kind for workstation scratch paper.
const KindScratchPaper = "scratch-paper"

// ScratchPaperTemplate returns the template for scratch paper artifacts.
func ScratchPaperTemplate() Template {
	return Template{
		Kind:          KindScratchPaper,
		ValidStatuses: []Status{StatusDraft, StatusActive, StatusComplete},
		IDFormat:      "SP-%03d",
	}
}
