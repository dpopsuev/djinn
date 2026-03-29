// tier2_iface.go — Tier 2 heuristic: interface + struct mutations (TSK-455).
//
// Detects new interface declarations, method count changes on existing
// interfaces, and struct/class field count changes via LSP symbol diff.
package review

import (
	"context"
	"fmt"

	"github.com/dpopsuev/djinn/review/lsp"
)

// InterfaceHeuristic checks interface/struct mutations.
type InterfaceHeuristic struct {
	MaxInterfacesIntroduced int
	OnInterfaceChange       bool
	MaxStructFieldChanges   int
	differ                  SymbolDiffer
}

// NewInterfaceHeuristic creates a heuristic with the given config.
func NewInterfaceHeuristic(cfg Config, differ SymbolDiffer) *InterfaceHeuristic {
	return &InterfaceHeuristic{
		MaxInterfacesIntroduced: cfg.MaxInterfacesIntroduced,
		OnInterfaceChange:       cfg.OnInterfaceChange,
		MaxStructFieldChanges:   cfg.MaxStructFieldChanges,
		differ:                  differ,
	}
}

func (h *InterfaceHeuristic) Name() string { return "tier2_interface" }

func (h *InterfaceHeuristic) Evaluate(ctx context.Context, diff *DiffSnapshot) ([]Signal, error) {
	if h.differ == nil {
		return nil, nil
	}

	var signals []Signal
	newInterfaces := 0
	interfaceMethodChanges := 0
	structFieldChanges := 0

	for _, file := range diff.ChangedFiles {
		sdiff, err := h.differ.DiffFile(ctx, diff.WorkDir, file)
		if err != nil || sdiff == nil {
			continue
		}

		for _, s := range sdiff.Added {
			if s.Kind == lsp.SymbolInterface {
				newInterfaces++
			}
		}

		for _, s := range sdiff.Modified {
			if s.Kind == lsp.SymbolInterface && s.ChildrenBefore != s.ChildrenAfter {
				interfaceMethodChanges++
			}
			if (s.Kind == lsp.SymbolStruct || s.Kind == lsp.SymbolClass) && s.ChildrenBefore != s.ChildrenAfter {
				diff := s.ChildrenAfter - s.ChildrenBefore
				if diff < 0 {
					diff = -diff
				}
				structFieldChanges += diff
			}
		}
	}

	if h.MaxInterfacesIntroduced > 0 {
		signals = append(signals, Signal{
			Metric:    "interfaces_introduced",
			Value:     float64(newInterfaces),
			Threshold: float64(h.MaxInterfacesIntroduced),
			Exceeded:  newInterfaces >= h.MaxInterfacesIntroduced,
			Detail:    fmt.Sprintf("%d new interfaces", newInterfaces),
		})
	}

	if h.OnInterfaceChange {
		signals = append(signals, Signal{
			Metric:    "interface_methods_changed",
			Value:     float64(interfaceMethodChanges),
			Threshold: 1,
			Exceeded:  interfaceMethodChanges > 0,
			Detail:    fmt.Sprintf("%d interfaces had method changes", interfaceMethodChanges),
		})
	}

	if h.MaxStructFieldChanges > 0 {
		signals = append(signals, Signal{
			Metric:    "struct_field_changes",
			Value:     float64(structFieldChanges),
			Threshold: float64(h.MaxStructFieldChanges),
			Exceeded:  structFieldChanges >= h.MaxStructFieldChanges,
			Detail:    fmt.Sprintf("%d struct/class field changes", structFieldChanges),
		})
	}

	return signals, nil
}
