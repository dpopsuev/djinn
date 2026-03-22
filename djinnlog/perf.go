package djinnlog

import (
	"log/slog"
	"time"
)

// Performance attribute helpers. All return slog.Attr for use in slog.Group("perf", ...).

// TTFT returns a time-to-first-token attribute.
func TTFT(d time.Duration) slog.Attr {
	return slog.Duration("ttft", d)
}

// RTT returns a round-trip time attribute.
func RTT(d time.Duration) slog.Attr {
	return slog.Duration("rtt", d)
}

// ToolLatency returns a tool execution time attribute.
func ToolLatency(d time.Duration) slog.Attr {
	return slog.Duration("tool_ms", d)
}

// TokensIn returns an input token count attribute.
func TokensIn(n int) slog.Attr {
	return slog.Int("tokens_in", n)
}

// TokensOut returns an output token count attribute.
func TokensOut(n int) slog.Attr {
	return slog.Int("tokens_out", n)
}

// Throughput returns tokens-per-second attribute.
func Throughput(tokensOut int, elapsed time.Duration) slog.Attr {
	if elapsed <= 0 {
		return slog.Float64("throughput", 0)
	}
	tps := float64(tokensOut) / elapsed.Seconds()
	return slog.Float64("throughput", tps)
}

// Turns returns the ReAct loop turn count attribute.
func Turns(n int) slog.Attr {
	return slog.Int("turns", n)
}

// ContextPct returns context window usage as a fraction (0.0–1.0).
func ContextPct(used, total int) slog.Attr {
	if total <= 0 {
		return slog.Float64("context_pct", 0)
	}
	return slog.Float64("context_pct", float64(used)/float64(total))
}
