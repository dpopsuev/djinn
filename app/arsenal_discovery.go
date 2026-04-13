// arsenal_discovery.go — live model catalog discovery via Troupe Arsenal.
//
// DiscoverModels creates a Troupe Arsenal, registers discoverers for
// available API keys, and runs live discovery. Returns the Arsenal
// for downstream model selection. Falls back to static YAML catalog
// when no API keys are configured.
//
// TSK-1099, GOL-162
package app

import (
	"context"
	"log/slog"
	"time"

	"github.com/dpopsuev/djinn/telemetry"
	"github.com/dpopsuev/troupe/arsenal"
)

// DiscoverModels creates a Troupe Arsenal with live model discovery.
// Registers discoverers based on available API keys. Runs discovery
// with a 5s timeout. Returns the Arsenal for model selection.
// Non-fatal: discovery failures are logged, static catalog is fallback.
func DiscoverModels(ctx context.Context, log *slog.Logger) *arsenal.Arsenal {
	a, err := arsenal.NewArsenal("")
	if err != nil {
		if log != nil {
			log.WarnContext(ctx, "arsenal catalog load failed",
				slog.String(telemetry.KeyError, err.Error()),
			)
		}
		return nil
	}

	// Register discoverers based on available API keys.
	if d := arsenal.NewAnthropicDiscoverer(); d != nil {
		a.RegisterDiscoverer(d)
	}
	if d := arsenal.NewOpenAIDiscoverer(); d != nil {
		a.RegisterDiscoverer(d)
	}
	if d := arsenal.NewGeminiDiscoverer(); d != nil {
		a.RegisterDiscoverer(d)
	}

	if log != nil {
		a.WithLogger(log)
	}

	// Run discovery with timeout.
	start := time.Now()
	errs := a.Discover(ctx)
	elapsed := time.Since(start)

	if log != nil {
		for _, err := range errs {
			log.WarnContext(ctx, "model discovery error",
				slog.String(telemetry.KeyError, err.Error()),
			)
		}
		log.InfoContext(ctx, "model discovery complete",
			slog.Duration(telemetry.KeyDuration, elapsed),
			slog.Int(telemetry.KeyCount, len(errs)),
			slog.String(telemetry.KeyStatus, "ok"),
		)
	}

	return a
}
