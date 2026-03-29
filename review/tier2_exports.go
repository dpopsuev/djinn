// tier2_exports.go — Tier 2 heuristic: exported symbol changes (TSK-454).
//
// Detects public API surface mutations via LSP symbol diff.
// Language-agnostic export detection (Go uppercase, Python no underscore, etc.)
package review

import (
	"context"
	"fmt"
)

// ExportsHeuristic checks exported symbol changes against thresholds.
type ExportsHeuristic struct {
	MaxExportedChanges int
	differ             SymbolDiffer // injected for testability
}

// SymbolDiffer abstracts the LSP-based symbol diff for testing.
type SymbolDiffer interface {
	DiffFile(ctx context.Context, workDir, filePath string) (*SymbolDiff, error)
}

// NewExportsHeuristic creates a heuristic with the given config and differ.
func NewExportsHeuristic(cfg Config, differ SymbolDiffer) *ExportsHeuristic {
	return &ExportsHeuristic{
		MaxExportedChanges: cfg.MaxExportedChanges,
		differ:             differ,
	}
}

func (h *ExportsHeuristic) Name() string { return "tier2_exports" }

func (h *ExportsHeuristic) Evaluate(ctx context.Context, diff *DiffSnapshot) ([]Signal, error) {
	if h.MaxExportedChanges <= 0 || h.differ == nil {
		return nil, nil
	}

	exportedChanges := 0

	for _, file := range diff.ChangedFiles {
		sdiff, err := h.differ.DiffFile(ctx, diff.WorkDir, file)
		if err != nil || sdiff == nil {
			continue // graceful fallback for unsupported files
		}

		for _, s := range sdiff.Added {
			if s.Exported {
				exportedChanges++
			}
		}
		for _, s := range sdiff.Removed {
			if s.Exported {
				exportedChanges++
			}
		}
		for _, s := range sdiff.Modified {
			if s.Exported {
				exportedChanges++
			}
		}
	}

	return []Signal{{
		Metric:    "exported_symbols_changed",
		Value:     float64(exportedChanges),
		Threshold: float64(h.MaxExportedChanges),
		Exceeded:  exportedChanges >= h.MaxExportedChanges,
		Detail:    fmt.Sprintf("%d exported symbol changes", exportedChanges),
	}}, nil
}
