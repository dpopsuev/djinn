// drivers.go — driver factory, isolates all driver package imports.
package app

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/dpopsuev/djinn/djinnlog"
	"github.com/dpopsuev/djinn/driver"
	acpdriver "github.com/dpopsuev/djinn/driver/acp"
	claudedriver "github.com/dpopsuev/djinn/driver/claude"
	codexdriver "github.com/dpopsuev/djinn/driver/codex"
	cursordriver "github.com/dpopsuev/djinn/driver/cursor"
	geminidriver "github.com/dpopsuev/djinn/driver/gemini"
	"github.com/dpopsuev/djinn/tools/builtin"
)

// CreateDriver creates a ChatDriver for the given backend name and model.
func CreateDriver(driverName, model, systemPrompt string, log ...*slog.Logger) (driver.ChatDriver, error) {
	var driverLog *slog.Logger
	if len(log) > 0 && log[0] != nil {
		driverLog = djinnlog.For(log[0], "driver")
	}
	switch driverName {
	case DriverClaude:
		opts := []claudedriver.APIDriverOption{
			claudedriver.WithTools(builtin.NewRegistry()),
		}
		if driverLog != nil {
			opts = append(opts, claudedriver.WithLogger(driverLog))
		}
		if systemPrompt != "" {
			opts = append(opts, claudedriver.WithAPISystemPrompt(systemPrompt))
		}
		return claudedriver.NewAPIDriver(driver.DriverConfig{Model: model}, opts...)
	case DriverCursor:
		opts := []cursordriver.Option{}
		if driverLog != nil {
			opts = append(opts, cursordriver.WithLogger(driverLog))
		}
		if systemPrompt != "" {
			opts = append(opts, cursordriver.WithSystemPrompt(systemPrompt))
		}
		return cursordriver.New(driver.DriverConfig{Model: model}, opts...), nil
	case "gemini":
		opts := []geminidriver.Option{}
		if driverLog != nil {
			opts = append(opts, geminidriver.WithLogger(driverLog))
		}
		if systemPrompt != "" {
			opts = append(opts, geminidriver.WithSystemPrompt(systemPrompt))
		}
		return geminidriver.New(driver.DriverConfig{Model: model}, opts...), nil
	case "codex":
		opts := []codexdriver.Option{}
		if driverLog != nil {
			opts = append(opts, codexdriver.WithLogger(driverLog))
		}
		if systemPrompt != "" {
			opts = append(opts, codexdriver.WithSystemPrompt(systemPrompt))
		}
		return codexdriver.New(driver.DriverConfig{Model: model}, opts...), nil
	case "acp":
		agentName := model
		if agentName == "" {
			agentName = "cursor"
		}
		if parts := strings.SplitN(agentName, "/", 2); len(parts) == 2 {
			agentName = parts[0]
			model = parts[1]
		}
		opts := []acpdriver.Option{}
		if driverLog != nil {
			opts = append(opts, acpdriver.WithLogger(driverLog))
		}
		if model != agentName {
			opts = append(opts, acpdriver.WithModel(model))
		}
		return acpdriver.New(agentName, opts...)
	case DriverOllama:
		return nil, fmt.Errorf("%w: %s", ErrDriverNotImpl, driverName)
	default:
		return nil, fmt.Errorf("%w: %q (supported: acp, cursor, claude, gemini, codex)", ErrUnknownDriver, driverName)
	}
}
