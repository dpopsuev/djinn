// tier1_packages.go — Tier 1 heuristic: package spread + new/deleted files (TSK-450).
package review

import (
	"context"
	"fmt"
)

// PackagesHeuristic checks package spread and structural file changes.
type PackagesHeuristic struct {
	MaxPackages     int // 0 = disabled
	MaxNewFiles     int // 0 = disabled
	MaxDeletedFiles int // 0 = disabled
}

// NewPackagesHeuristic creates a heuristic with the given thresholds.
func NewPackagesHeuristic(cfg Config) *PackagesHeuristic {
	return &PackagesHeuristic{
		MaxPackages:     cfg.MaxPackages,
		MaxNewFiles:     cfg.MaxNewFiles,
		MaxDeletedFiles: cfg.MaxDeletedFiles,
	}
}

func (h *PackagesHeuristic) Name() string { return "tier1_packages" }

func (h *PackagesHeuristic) Evaluate(_ context.Context, diff *DiffSnapshot) ([]Signal, error) {
	var signals []Signal

	if h.MaxPackages > 0 {
		count := len(diff.PackagesHit)
		signals = append(signals, Signal{
			Metric:    "packages_touched",
			Value:     float64(count),
			Threshold: float64(h.MaxPackages),
			Exceeded:  count >= h.MaxPackages,
			Detail:    fmt.Sprintf("%d packages", count),
		})
	}

	if h.MaxNewFiles > 0 {
		count := len(diff.AddedFiles)
		signals = append(signals, Signal{
			Metric:    "new_files",
			Value:     float64(count),
			Threshold: float64(h.MaxNewFiles),
			Exceeded:  count >= h.MaxNewFiles,
			Detail:    fmt.Sprintf("%d new files", count),
		})
	}

	if h.MaxDeletedFiles > 0 {
		count := len(diff.DeletedFiles)
		signals = append(signals, Signal{
			Metric:    "deleted_files",
			Value:     float64(count),
			Threshold: float64(h.MaxDeletedFiles),
			Exceeded:  count >= h.MaxDeletedFiles,
			Detail:    fmt.Sprintf("%d deleted files", count),
		})
	}

	return signals, nil
}
