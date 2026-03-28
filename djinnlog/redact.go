package djinnlog

import (
	"context"
	"log/slog"
	"strings"
)

const redacted = "[REDACTED]"

// Sensitive key patterns — any attribute key containing these substrings is redacted.
var sensitiveKeys = []string{
	"api_key", "apikey", "token", "secret", "password",
	"credential", "authorization", "bearer",
}

// RedactHandler is a slog.Handler middleware that strips sensitive
// attribute values before passing records to the wrapped handler.
// Belt & Suspenders: even if a caller accidentally logs a token,
// it never reaches the file or terminal.
type RedactHandler struct {
	next slog.Handler
}

// NewRedactHandler wraps a handler with sensitive data redaction.
func NewRedactHandler(next slog.Handler) *RedactHandler {
	return &RedactHandler{next: next}
}

func (h *RedactHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.next.Enabled(ctx, level)
}

func (h *RedactHandler) Handle(ctx context.Context, r slog.Record) error { //nolint:gocritic // slog.Handler interface requires value receiver
	// Clone and redact attributes
	clean := slog.NewRecord(r.Time, r.Level, r.Message, r.PC)
	r.Attrs(func(a slog.Attr) bool {
		clean.AddAttrs(redactAttr(a))
		return true
	})
	return h.next.Handle(ctx, clean)
}

func (h *RedactHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	cleaned := make([]slog.Attr, len(attrs))
	for i, a := range attrs {
		cleaned[i] = redactAttr(a)
	}
	return &RedactHandler{next: h.next.WithAttrs(cleaned)}
}

func (h *RedactHandler) WithGroup(name string) slog.Handler {
	return &RedactHandler{next: h.next.WithGroup(name)}
}

func redactAttr(a slog.Attr) slog.Attr {
	key := strings.ToLower(a.Key)
	for _, sensitive := range sensitiveKeys {
		if strings.Contains(key, sensitive) {
			return slog.String(a.Key, redacted)
		}
	}
	// Recurse into groups
	if a.Value.Kind() == slog.KindGroup {
		attrs := a.Value.Group()
		cleaned := make([]slog.Attr, len(attrs))
		for i, ga := range attrs {
			cleaned[i] = redactAttr(ga)
		}
		return slog.Group(a.Key, attrsToAny(cleaned)...)
	}
	return a
}

func attrsToAny(attrs []slog.Attr) []any {
	out := make([]any, len(attrs))
	for i, a := range attrs {
		out[i] = a
	}
	return out
}

var _ slog.Handler = (*RedactHandler)(nil)
