// symbol_graph.go — SymbolGraph populator middleware (TSK-649).
//
// Tries providers in order (first success wins), builds the impact table,
// and renders it as a structured prompt injection.
package agent

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/dpopsuev/djinn/djinnlog"
)

// SymbolGraphPopulator builds SymbolGraphs using a chain of providers.
type SymbolGraphPopulator struct {
	providers []SymbolProvider
	log       *slog.Logger
}

// NewSymbolGraphPopulator creates a populator with providers tried in order.
func NewSymbolGraphPopulator(log *slog.Logger, providers ...SymbolProvider) *SymbolGraphPopulator {
	return &SymbolGraphPopulator{
		providers: providers,
		log:       log,
	}
}

// Populate builds the SymbolGraph for a file.
// Tries each provider in order; first success wins for symbols.
// References are resolved per-symbol (falling back on error).
func (p *SymbolGraphPopulator) Populate(ctx context.Context, file string) (*SymbolGraph, error) {
	var symbols []Symbol
	var providerName string

	for i, prov := range p.providers {
		var err error
		symbols, err = prov.Symbols(ctx, file)
		if err == nil {
			providerName = fmt.Sprintf("provider[%d]", i)
			break
		}
		p.log.WarnContext(context.Background(), "symbol provider failed, trying next",
			slog.String(djinnlog.KeyPath, file),
			slog.Int(djinnlog.KeyProvider, i),
			slog.String(djinnlog.KeyError, err.Error()),
		)
	}

	if symbols == nil {
		return &SymbolGraph{File: file}, nil
	}

	p.log.DebugContext(context.Background(), "symbols populated",
		slog.String(djinnlog.KeyPath, file),
		slog.Int(djinnlog.KeyCount, len(symbols)),
		slog.String(djinnlog.KeyComponent, providerName),
	)

	pkg := filepath.Dir(file)
	entries := make([]SymbolEntry, 0, len(symbols))

	for _, sym := range symbols {
		var allRefs []Reference
		for _, prov := range p.providers {
			refs, err := prov.References(ctx, file, sym.Name)
			if err == nil {
				allRefs = refs
				break
			}
			// Silently try next provider for references.
		}

		p.log.DebugContext(context.Background(), "references resolved",
			slog.String(djinnlog.KeyTool, sym.Name),
			slog.Int(djinnlog.KeyCallers, len(allRefs)),
		)

		var internal, external []Reference
		for _, ref := range allRefs {
			if filepath.Dir(ref.File) == pkg {
				internal = append(internal, ref)
			} else {
				external = append(external, ref)
			}
		}

		entries = append(entries, SymbolEntry{
			Symbol:   sym,
			Internal: internal,
			External: external,
		})
	}

	return &SymbolGraph{
		File:    file,
		Symbols: entries,
	}, nil
}

// FormatContext renders the SymbolGraph as a structured prompt injection string.
func (sg *SymbolGraph) FormatContext() string {
	if sg == nil || len(sg.Symbols) == 0 {
		return ""
	}

	var b strings.Builder
	fmt.Fprintf(&b, "<symbol-graph file=%q>\n", sg.File)

	for _, entry := range sg.Symbols {
		total := len(entry.Internal) + len(entry.External)
		allRefs := make([]Reference, 0, total)
		allRefs = append(allRefs, entry.Internal...)
		allRefs = append(allRefs, entry.External...)

		if total == 0 {
			fmt.Fprintf(&b, "  %s (%s, 0 callers) | safe to modify\n",
				entry.Symbol.Name, entry.Symbol.Kind)
			continue
		}

		// Format caller list (up to 3 references shown).
		var callers []string
		limit := 3
		if total < limit {
			limit = total
		}
		for i := range limit {
			ref := allRefs[i]
			callers = append(callers, fmt.Sprintf("%s:%d",
				filepath.Base(ref.File), ref.Line))
		}
		callerStr := strings.Join(callers, ", ")
		if total > limit {
			callerStr += ", ..."
		}

		fmt.Fprintf(&b, "  %s (%s, %d callers) | %s\n",
			entry.Symbol.Name, entry.Symbol.Kind, total, callerStr)
	}

	b.WriteString("</symbol-graph>")
	return b.String()
}
