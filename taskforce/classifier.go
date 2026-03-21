package taskforce

import "github.com/dpopsuev/djinn/tier"

// Classifier determines the complexity band of a task.
// Takes primitive inputs (scopes, file count, repo count) — NOT ari.Intent.
// The broker owns the translation from Intent to these primitives (ISP).
type Classifier interface {
	Classify(scopes []tier.Scope, fileCount, repoCount int) ComplexityBand
}

// Thresholds for heuristic classification.
const (
	clearMaxFiles       = 1
	complicatedMaxFiles = 10
	multiRepoThreshold  = 2
)

// HeuristicClassifier classifies based on file count, repo count, and scope depth.
type HeuristicClassifier struct{}

// NewHeuristicClassifier creates a new heuristic-based classifier.
func NewHeuristicClassifier() *HeuristicClassifier {
	return &HeuristicClassifier{}
}

func (c *HeuristicClassifier) Classify(scopes []tier.Scope, fileCount, repoCount int) ComplexityBand {
	if repoCount >= multiRepoThreshold {
		return Complex
	}
	if fileCount <= clearMaxFiles && len(scopes) <= 1 {
		return Clear
	}
	if fileCount <= complicatedMaxFiles {
		return Complicated
	}
	return Complex
}
