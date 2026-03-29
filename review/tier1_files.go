// tier1_files.go — Tier 1 heuristic: files touched + LOC delta (TSK-449).
//
// Pure git diff metrics. Zero parsing, zero external dependencies.
// Operates on pre-computed DiffSnapshot.
package review

import (
	"context"
	"fmt"
)

// FilesHeuristic checks files touched and LOC delta against thresholds.
type FilesHeuristic struct {
	MaxFiles int // 0 = disabled
	MaxLOC   int // 0 = disabled
}

// NewFilesHeuristic creates a heuristic with the given thresholds.
func NewFilesHeuristic(cfg Config) *FilesHeuristic {
	return &FilesHeuristic{
		MaxFiles: cfg.MaxFiles,
		MaxLOC:   cfg.MaxLOCDelta,
	}
}

func (h *FilesHeuristic) Name() string { return "tier1_files" }

func (h *FilesHeuristic) Evaluate(_ context.Context, diff *DiffSnapshot) ([]Signal, error) {
	var signals []Signal

	if h.MaxFiles > 0 {
		total := len(diff.ChangedFiles) + len(diff.AddedFiles) + len(diff.DeletedFiles)
		signals = append(signals, Signal{
			Metric:    "files_touched",
			Value:     float64(total),
			Threshold: float64(h.MaxFiles),
			Exceeded:  total >= h.MaxFiles,
			Detail:    fmt.Sprintf("%d files (changed:%d, added:%d, deleted:%d)", total, len(diff.ChangedFiles), len(diff.AddedFiles), len(diff.DeletedFiles)),
		})
	}

	if h.MaxLOC > 0 {
		loc := diff.LOCDelta
		if loc < 0 {
			loc = -loc
		}
		signals = append(signals, Signal{
			Metric:    "loc_delta",
			Value:     float64(loc),
			Threshold: float64(h.MaxLOC),
			Exceeded:  loc >= h.MaxLOC,
			Detail:    fmt.Sprintf("%d lines changed", diff.LOCDelta),
		})
	}

	return signals, nil
}
