// scribe.go — ScribeAdapter implements ExecutionPlannerPort via Scribe MCP (GOL-57).
//
// Day 2 adapter: syncs plan segments bidirectionally with Scribe artifacts.
// Requires an MCP client connected to the "scribe" server.
package adapters

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/dpopsuev/djinn/hub"
	mcpclient "github.com/dpopsuev/djinn/mcp/client"
)

// ScribeAdapter implements ExecutionPlannerPort via the Scribe MCP server.
type ScribeAdapter struct {
	Client *mcpclient.Client
	Server string // MCP server name, e.g. "scribe"
}

// NewScribeAdapter creates a Scribe adapter for the given MCP client and server name.
func NewScribeAdapter(client *mcpclient.Client, server string) *ScribeAdapter {
	return &ScribeAdapter{Client: client, Server: server}
}

// SyncPlan pushes local plan segments to Scribe as task artifacts.
func (a *ScribeAdapter) SyncPlan(ctx context.Context, segments []hub.PlanSegmentDTO) error {
	for i := range segments {
		input, err := json.Marshal(map[string]any{
			"action": "update",
			"id":     segments[i].ID,
			"kind":   "task",
			"patch": map[string]string{
				"title":  segments[i].Title,
				"status": segments[i].Status,
			},
		})
		if err != nil {
			return fmt.Errorf("scribe sync marshal: %w", err)
		}

		if _, err := a.Client.Call(ctx, a.Server, "artifact", input); err != nil {
			return fmt.Errorf("scribe sync %s: %w", segments[i].ID, err)
		}
	}
	return nil
}

// FetchPlan retrieves plan segments from Scribe task artifacts.
func (a *ScribeAdapter) FetchPlan(ctx context.Context) ([]hub.PlanSegmentDTO, error) {
	input, err := json.Marshal(map[string]any{
		"action": "list",
		"kind":   "task",
	})
	if err != nil {
		return nil, fmt.Errorf("scribe fetch marshal: %w", err)
	}

	result, err := a.Client.Call(ctx, a.Server, "artifact", input)
	if err != nil {
		return nil, fmt.Errorf("scribe fetch: %w", err)
	}

	var artifacts []struct {
		ID     string `json:"id"`
		Title  string `json:"title"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal([]byte(result), &artifacts); err != nil {
		return nil, fmt.Errorf("scribe fetch unmarshal: %w", err)
	}

	dtos := make([]hub.PlanSegmentDTO, len(artifacts))
	for i := range artifacts {
		dtos[i] = hub.PlanSegmentDTO{
			ID:     artifacts[i].ID,
			Title:  artifacts[i].Title,
			Status: artifacts[i].Status,
		}
	}
	return dtos, nil
}

// Compile-time interface check.
var _ hub.ExecutionPlannerPort = (*ScribeAdapter)(nil)
