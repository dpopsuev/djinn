// locus.go — LocusAdapter implements StructuralAnalyzerPort via Locus MCP (GOL-53).
//
// Day 2 adapter: queries Locus for architecture analysis via MCP client.
package adapters

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/dpopsuev/djinn/hub"
	mcpclient "github.com/dpopsuev/djinn/mcp/client"
)

// LocusAdapter implements StructuralAnalyzerPort via the Locus MCP server.
type LocusAdapter struct {
	Client *mcpclient.Client
	Server string // MCP server name, e.g. "locus"
}

// NewLocusAdapter creates a Locus adapter for the given MCP client and server name.
func NewLocusAdapter(client *mcpclient.Client, server string) *LocusAdapter {
	return &LocusAdapter{Client: client, Server: server}
}

// Analyze runs architecture analysis via Locus MCP.
func (a *LocusAdapter) Analyze(ctx context.Context, paths []string) (hub.AnalysisResult, error) {
	input, err := json.Marshal(map[string]any{
		"action":    "violations",
		"format":    "json",
		"cache_key": "",
	})
	if err != nil {
		return hub.AnalysisResult{}, fmt.Errorf("locus analyze marshal: %w", err)
	}

	result, err := a.Client.Call(ctx, a.Server, "analysis", input)
	if err != nil {
		return hub.AnalysisResult{}, fmt.Errorf("locus analyze: %w", err)
	}

	var analysis struct {
		Components []string `json:"components"`
		Violations []string `json:"violations"`
	}
	if err := json.Unmarshal([]byte(result), &analysis); err != nil {
		return hub.AnalysisResult{}, fmt.Errorf("locus analyze unmarshal: %w", err)
	}

	return hub.AnalysisResult{
		Components: analysis.Components,
		Violations: analysis.Violations,
	}, nil
}

// Compile-time interface check.
var _ hub.StructuralAnalyzerPort = (*LocusAdapter)(nil)
