// instrumented.go — wraps a Cache with structured logging (ROGYB).
// ORANGE: warns on expected-warm misses.
// YELLOW: logs hits, promotions, warm starts, stats.
package cache

import (
	"context"
	"log/slog"

	"github.com/dpopsuev/djinn/telemetry"
)

var _ Cache = (*Instrumented)(nil)

// Instrumented wraps a Cache with slog instrumentation.
type Instrumented struct {
	inner Cache
	log   *slog.Logger
	name  string // "L1" or "L2" for log context
}

// NewInstrumented wraps a cache with structured logging.
func NewInstrumented(inner Cache, log *slog.Logger, name string) *Instrumented {
	return &Instrumented{inner: inner, log: log, name: name}
}

func (c *Instrumented) Put(scope, key string, data []byte) {
	c.inner.Put(scope, key, data)
	c.log.DebugContext(context.Background(), "cache put",
		slog.String(telemetry.KeyCache, c.name),
		slog.String(telemetry.KeyScope, scope),
		slog.String(telemetry.KeyKey, key),
		slog.Int(telemetry.KeyBytes, len(data)),
	)
}

func (c *Instrumented) Get(scope, key string) ([]byte, bool) {
	data, ok := c.inner.Get(scope, key)
	if ok {
		c.log.DebugContext(context.Background(), "cache hit",
			slog.String(telemetry.KeyCache, c.name),
			slog.String(telemetry.KeyScope, scope),
			slog.String(telemetry.KeyKey, key),
		)
	}
	return data, ok
}

func (c *Instrumented) Keys(scope string) []string {
	return c.inner.Keys(scope)
}

func (c *Instrumented) Evict(scope, key string) {
	c.inner.Evict(scope, key)
	c.log.DebugContext(context.Background(), "cache evict",
		slog.String(telemetry.KeyCache, c.name),
		slog.String(telemetry.KeyScope, scope),
		slog.String(telemetry.KeyKey, key),
	)
}

func (c *Instrumented) EvictScope(scope string) {
	c.inner.EvictScope(scope)
	c.log.InfoContext(context.Background(), "cache evict scope",
		slog.String(telemetry.KeyCache, c.name),
		slog.String(telemetry.KeyScope, scope),
	)
}

func (c *Instrumented) Scopes() []string { return c.inner.Scopes() }
func (c *Instrumented) Len() int         { return c.inner.Len() }

// WarnColdStart logs a warning when an agent starts without L2 entries.
// Call this when creating a new Vessel — if L2 has no entries for the scope,
// the agent starts cold (ORANGE signal).
func (c *Instrumented) WarnColdStart(scope string) {
	keys := c.inner.Keys(scope)
	if len(keys) == 0 {
		c.log.WarnContext(context.Background(), "cache cold start — no L2 entries for scope",
			slog.String(telemetry.KeyCache, c.name),
			slog.String(telemetry.KeyScope, scope),
		)
	} else {
		c.log.InfoContext(context.Background(), "cache warm start",
			slog.String(telemetry.KeyCache, c.name),
			slog.String(telemetry.KeyScope, scope),
			slog.Int(telemetry.KeyEntries, len(keys)),
		)
	}
}

// LogStats logs cache statistics (YELLOW).
func (c *Instrumented) LogStats() {
	c.log.InfoContext(context.Background(), "cache stats",
		slog.String(telemetry.KeyCache, c.name),
		slog.Int(telemetry.KeyScopes, len(c.inner.Scopes())),
		slog.Int(telemetry.KeyEntries, c.inner.Len()),
	)
}
