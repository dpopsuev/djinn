package djinnlog

import (
	"context"
	"log/slog"
)

// MultiHandler fans out log records to multiple handlers.
type MultiHandler struct {
	handlers []slog.Handler
}

// NewMultiHandler creates a handler that writes to all provided handlers.
func NewMultiHandler(handlers ...slog.Handler) *MultiHandler {
	return &MultiHandler{handlers: handlers}
}

func (h *MultiHandler) Enabled(_ context.Context, level slog.Level) bool {
	for _, handler := range h.handlers {
		if handler.Enabled(context.Background(), level) {
			return true
		}
	}
	return false
}

func (h *MultiHandler) Handle(ctx context.Context, r slog.Record) error { //nolint:gocritic // slog.Handler interface requires value receiver
	for _, handler := range h.handlers {
		if handler.Enabled(ctx, r.Level) {
			if err := handler.Handle(ctx, r.Clone()); err != nil {
				return err
			}
		}
	}
	return nil
}

func (h *MultiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newHandlers := make([]slog.Handler, len(h.handlers))
	for i, handler := range h.handlers {
		newHandlers[i] = handler.WithAttrs(attrs)
	}
	return NewMultiHandler(newHandlers...)
}

func (h *MultiHandler) WithGroup(name string) slog.Handler {
	newHandlers := make([]slog.Handler, len(h.handlers))
	for i, handler := range h.handlers {
		newHandlers[i] = handler.WithGroup(name)
	}
	return NewMultiHandler(newHandlers...)
}

// Ensure MultiHandler satisfies slog.Handler.
var _ slog.Handler = (*MultiHandler)(nil)
