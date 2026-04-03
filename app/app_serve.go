// app_serve.go — MCP server mode for dogfooding via Claude Code.
//
// `djinn serve` runs Djinn as an MCP server over stdio. Claude Code
// connects to it and gains Djinn's builtin tools with the Tool Operation
// Envelope baked in — SecurityBundle + EnrichmentBundle + ObservabilityBundle,
// all transparent to the agent.
//
// No TUI. No Jericho. Single-agent headless mode.
package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"

	"github.com/dpopsuev/djinn/agent"
	"github.com/dpopsuev/djinn/djinnlog"
	"github.com/dpopsuev/djinn/policy"
	"github.com/dpopsuev/djinn/tools/builtin"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// RunServe starts Djinn as an MCP server over stdio.
func RunServe(_ []string, stderr io.Writer) error {
	log := slog.New(slog.NewTextHandler(stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	log.InfoContext(context.Background(), "djinn serve starting",
		slog.String(djinnlog.KeyComponent, "serve"))

	// Build tool registry.
	registry := builtin.NewRegistry()

	// PolicyEnforcer + token for serve mode — security by construction.
	enforcer := policy.NewDefaultToolPolicyEnforcer(djinnlog.For(log, "policy"))
	capToken := policy.CapabilityToken{
		WritablePaths: []string{Getwd()},
		DeniedPaths:   []string{"~/.ssh", "~/.gnupg", "~/.aws"},
	}

	// Middleware: SymbolGraph enricher + WasteClassifier recorder.
	symbolPopulator := agent.NewSymbolGraphPopulator(log, &agent.RegexProvider{})
	wasteClassifier := agent.NewWasteClassifier(djinnlog.For(log, "waste"))

	// Build envelope with all three layers.
	envelope, err := agent.NewEnvelopeBuilder(registry).
		WithGates(agent.SecurityBundle(enforcer, capToken)...).
		WithEnrichers(agent.EnrichmentBundle(symbolPopulator)...).
		WithRecorders(agent.ObservabilityBundle(wasteClassifier)...).
		Build()
	if err != nil {
		return fmt.Errorf("build envelope: %w", err)
	}

	server := mcp.NewServer(&mcp.Implementation{
		Name:    "djinn",
		Version: Version,
	}, nil)

	// Register builtin tools with Envelope-based execution.
	for _, tool := range registry.All() {
		registerToolWithEnvelope(server, tool, envelope)
	}

	log.InfoContext(context.Background(), "djinn serve ready",
		slog.String(djinnlog.KeyComponent, "serve"),
		slog.Int(djinnlog.KeyCount, len(registry.Names())),
	)

	return server.Run(context.Background(), &mcp.StdioTransport{})
}

// registerToolWithEnvelope wraps a builtin tool with the full Envelope pipeline.
// All calls go through Gate → Enrich → Execute → Record transparently.
func registerToolWithEnvelope(
	server *mcp.Server,
	tool builtin.Tool,
	envelope *agent.ToolEnvelope,
) {
	type rawInput struct {
		Input json.RawMessage `json:"input" jsonschema:"tool input as JSON object"`
	}
	toolName := tool.Name()

	mcp.AddTool(server, &mcp.Tool{
		Name:        "djinn_" + toolName,
		Description: tool.Description(),
		InputSchema: tool.InputSchema(),
	}, func(ctx context.Context, req *mcp.CallToolRequest, _ rawInput) (*mcp.CallToolResult, any, error) {
		inputBytes, err := json.Marshal(req.Params.Arguments)
		if err != nil {
			return errorResult(fmt.Sprintf("marshal input: %v", err)), nil, nil
		}

		// Envelope handles Gate → Enrich → Execute → Record.
		output, execErr := envelope.Execute(ctx, toolName, inputBytes)
		if execErr != nil {
			return errorResult(fmt.Sprintf("tool error: %v", execErr)), nil, nil
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: output},
			},
		}, nil, nil
	})
}

func errorResult(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: msg},
		},
		IsError: true,
	}
}
