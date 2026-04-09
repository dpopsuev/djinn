// drivers.go — driver factory, isolates all driver/provider imports.
//
// All LLM backends route through TroupeChatDriver wrapping anyllm.Provider.
// Provider resolution delegates to troupe/execution (credential checks, aliases).
package app

import (
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/dpopsuev/djinn/driver"
	troupedriver "github.com/dpopsuev/djinn/driver/troupe"
	"github.com/dpopsuev/djinn/telemetry"
	"github.com/dpopsuev/djinn/tools/builtin"
	"github.com/dpopsuev/troupe/execution"
	anyllm "github.com/mozilla-ai/any-llm-go/providers"
)

// CreateDriver creates a ChatDriver for the given backend name and model.
func CreateDriver(driverName, model, systemPrompt string, log ...*slog.Logger) (driver.ChatDriver, error) {
	var driverLog *slog.Logger
	if len(log) > 0 && log[0] != nil {
		driverLog = telemetry.For(log[0], "driver")
	}

	// Map driver names to provider names (Troupe handles aliases).
	providerName, err := resolveProviderName(driverName)
	if err != nil {
		return nil, err
	}

	provider, err := execution.NewProviderByName(providerName)
	if err != nil {
		return nil, fmt.Errorf("create provider %q: %w", providerName, err)
	}

	opts := []troupedriver.Option{
		troupedriver.WithTools(registryToAnyllmTools(builtin.NewRegistry())),
	}
	if driverLog != nil {
		opts = append(opts, troupedriver.WithLogger(driverLog))
	}
	if systemPrompt != "" {
		opts = append(opts, troupedriver.WithSystemPrompt(systemPrompt))
	}

	return troupedriver.New(provider, model, opts...), nil
}

// resolveProviderName maps djinn driver names to Troupe provider names.
func resolveProviderName(driverName string) (string, error) {
	switch driverName {
	case DriverClaude, DriverClaudeAPI:
		return "claude", nil // alias for "anthropic-api" in Troupe
	case DriverGemini:
		return "gemini", nil // alias for "gemini-api" in Troupe
	case DriverCursor, DriverCodex:
		return "", fmt.Errorf("%w: %s (CLI subprocess drivers removed — use API providers)", ErrDriverNotImpl, driverName)
	case DriverOllama:
		return "", fmt.Errorf("%w: %s", ErrDriverNotImpl, driverName)
	default:
		return "", fmt.Errorf("%w: %q (supported: claude, gemini)", ErrUnknownDriver, driverName)
	}
}

// registryToAnyllmTools converts a builtin.Registry to anyllm.Tool definitions.
func registryToAnyllmTools(reg *builtin.Registry) []anyllm.Tool {
	all := reg.All()
	tools := make([]anyllm.Tool, 0, len(all))
	for _, t := range all {
		var params map[string]any
		_ = json.Unmarshal(t.InputSchema(), &params)
		tools = append(tools, anyllm.Tool{
			Type: "function",
			Function: anyllm.Function{
				Name:        t.Name(),
				Description: t.Description(),
				Parameters:  params,
			},
		})
	}
	return tools
}
