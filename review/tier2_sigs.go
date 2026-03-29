// tier2_sigs.go — Tier 2 heuristic: function signature changes (TSK-456).
//
// Detects parameter/return type changes on existing functions via LSP detail diff.
package review

import (
	"context"
	"fmt"

	"github.com/dpopsuev/djinn/review/lsp"
)

// SignatureHeuristic checks function signature changes.
type SignatureHeuristic struct {
	MaxSignatureChanges int
	differ              SymbolDiffer
}

// NewSignatureHeuristic creates a heuristic with the given config.
func NewSignatureHeuristic(cfg Config, differ SymbolDiffer) *SignatureHeuristic {
	return &SignatureHeuristic{
		MaxSignatureChanges: cfg.MaxSignatureChanges,
		differ:              differ,
	}
}

func (h *SignatureHeuristic) Name() string { return "tier2_signatures" }

func (h *SignatureHeuristic) Evaluate(ctx context.Context, diff *DiffSnapshot) ([]Signal, error) {
	if h.MaxSignatureChanges <= 0 || h.differ == nil {
		return nil, nil
	}

	sigChanges := 0

	for _, file := range diff.ChangedFiles {
		sdiff, err := h.differ.DiffFile(ctx, diff.WorkDir, file)
		if err != nil || sdiff == nil {
			continue
		}

		for _, s := range sdiff.Modified {
			if (s.Kind == lsp.SymbolFunction || s.Kind == lsp.SymbolMethod) && s.DetailBefore != s.DetailAfter {
				sigChanges++
			}
		}
	}

	return []Signal{{
		Metric:    "signature_changes",
		Value:     float64(sigChanges),
		Threshold: float64(h.MaxSignatureChanges),
		Exceeded:  sigChanges >= h.MaxSignatureChanges,
		Detail:    fmt.Sprintf("%d function signature changes", sigChanges),
	}}, nil
}
