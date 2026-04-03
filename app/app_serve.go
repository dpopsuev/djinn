// app_serve.go — MCP server mode for dogfooding via Claude Code.
//
// `djinn serve` runs Djinn as an MCP server over stdio. Claude Code
// connects to it and gains Djinn's builtin tools with the Tool Operation
// Envelope baked in — SymbolGraph fires before Edit, WasteClassifier
// tracks calls, all transparent to the agent.
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
	"github.com/dpopsuev/djinn/tools/builtin"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// RunServe starts Djinn as an MCP server over stdio.
func RunServe(_ []string, stderr io.Writer) error {
	log := slog.New(slog.NewTextHandler(stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	log.InfoContext(context.Background(), "djinn serve starting",
		slog.String(djinnlog.KeyComponent, "serve"))

	// Middleware: SymbolGraph enricher for Edit calls.
	populator := agent.NewSymbolGraphPopulator(log, &agent.RegexProvider{})

	server := mcp.NewServer(&mcp.Implementation{
		Name:    "djinn",
		Version: Version,
	}, nil)

	// Register builtin tools with middleware baked in.
	registry := builtin.NewRegistry()
	for _, tool := range registry.All() {
		registerToolWithEnvelope(server, tool, registry, populator, log)
	}

	log.InfoContext(context.Background(), "djinn serve ready",
		slog.String(djinnlog.KeyComponent, "serve"),
		slog.Int(djinnlog.KeyCount, len(registry.Names())),
	)

	return server.Run(context.Background(), &mcp.StdioTransport{})
}

// registerToolWithEnvelope wraps a builtin tool with transparent middleware.
// Edit gets SymbolGraph enrichment. All tools get waste classification.
func registerToolWithEnvelope(
	server *mcp.Server,
	tool builtin.Tool,
	registry *builtin.Registry,
	populator *agent.SymbolGraphPopulator,
	log *slog.Logger,
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

		// --- ENRICH: SymbolGraph before Edit ---
		var symbolContext string
		if toolName == "Edit" {
			symbolContext = enrichWithSymbolGraph(ctx, inputBytes, populator, log)
		}

		// --- EXECUTE ---
		output, execErr := registry.Execute(ctx, toolName, inputBytes)
		if execErr != nil {
			return errorResult(fmt.Sprintf("tool error: %v", execErr)), nil, nil
		}

		// --- RECORD: append symbol context to response ---
		if symbolContext != "" {
			output = output + "\n\n" + symbolContext
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: output},
			},
		}, nil, nil
	})
}

// enrichWithSymbolGraph populates the symbol graph for the edited file
// and returns a caller impact summary. Transparent to the agent.
func enrichWithSymbolGraph(ctx context.Context, input json.RawMessage, populator *agent.SymbolGraphPopulator, log *slog.Logger) string {
	var params struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(input, &params); err != nil || params.Path == "" {
		return ""
	}

	sg, err := populator.Populate(ctx, params.Path)
	if err != nil {
		log.DebugContext(ctx, "symbol graph enrichment skipped",
			slog.String(djinnlog.KeyPath, params.Path),
			slog.String(djinnlog.KeyError, err.Error()),
		)
		return ""
	}

	formatted := sg.FormatContext()
	if formatted == "" {
		return ""
	}

	return formatted
}

func errorResult(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: msg},
		},
		IsError: true,
	}
}
