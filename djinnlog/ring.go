package djinnlog

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// Entry is a stored log record from the ring buffer.
type Entry struct {
	Time      time.Time
	Level     slog.Level
	Component string
	Message   string
	Attrs     map[string]any
}

// ringBuffer is the shared circular buffer. Multiple RingHandler
// instances (from WithAttrs/WithGroup) share the same buffer.
type ringBuffer struct {
	mu      sync.Mutex
	entries []Entry
	cap     int
	pos     int
	count   int
}

func (b *ringBuffer) write(e Entry) {
	b.mu.Lock()
	b.entries[b.pos] = e
	b.pos = (b.pos + 1) % b.cap
	if b.count < b.cap {
		b.count++
	}
	b.mu.Unlock()
}

func (b *ringBuffer) snapshot() []Entry {
	b.mu.Lock()
	defer b.mu.Unlock()

	result := make([]Entry, 0, b.count)
	if b.count < b.cap {
		result = append(result, b.entries[:b.count]...)
	} else {
		result = append(result, b.entries[b.pos:]...)
		result = append(result, b.entries[:b.pos]...)
	}
	return result
}

func (b *ringBuffer) len() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.count
}

// RingHandler stores the last N log records in a circular buffer.
// Thread-safe for concurrent writes from multiple goroutines.
type RingHandler struct {
	buf   *ringBuffer
	attrs []slog.Attr
}

// NewRingHandler creates a ring buffer handler with the given capacity.
func NewRingHandler(capacity int) *RingHandler {
	return &RingHandler{
		buf: &ringBuffer{
			entries: make([]Entry, capacity),
			cap:     capacity,
		},
	}
}

func (h *RingHandler) Enabled(_ context.Context, _ slog.Level) bool {
	return true
}

func (h *RingHandler) Handle(_ context.Context, r slog.Record) error {
	entry := Entry{
		Time:    r.Time,
		Level:   r.Level,
		Message: r.Message,
		Attrs:   make(map[string]any),
	}

	// Collect pre-set attributes (from WithAttrs)
	for _, a := range h.attrs {
		if a.Key == "component" {
			entry.Component = a.Value.String()
		}
		entry.Attrs[a.Key] = a.Value.Any()
	}

	// Collect record attributes
	r.Attrs(func(a slog.Attr) bool {
		if a.Key == "component" {
			entry.Component = a.Value.String()
		}
		entry.Attrs[a.Key] = a.Value.Any()
		return true
	})

	h.buf.write(entry)
	return nil
}

func (h *RingHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newAttrs := make([]slog.Attr, len(h.attrs)+len(attrs))
	copy(newAttrs, h.attrs)
	copy(newAttrs[len(h.attrs):], attrs)
	return &RingHandler{buf: h.buf, attrs: newAttrs}
}

func (h *RingHandler) WithGroup(name string) slog.Handler {
	// Groups are not tracked separately — attributes are flattened.
	return &RingHandler{buf: h.buf, attrs: h.attrs}
}

// Entries returns a snapshot of stored entries, oldest first.
func (h *RingHandler) Entries() []Entry {
	return h.buf.snapshot()
}

// Filter returns entries at or above the given level.
func (h *RingHandler) Filter(level slog.Level) []Entry {
	all := h.Entries()
	result := make([]Entry, 0, len(all))
	for _, e := range all {
		if e.Level >= level {
			result = append(result, e)
		}
	}
	return result
}

// Len returns the number of entries currently stored.
func (h *RingHandler) Len() int {
	return h.buf.len()
}

// Ensure RingHandler satisfies slog.Handler.
var _ slog.Handler = (*RingHandler)(nil)
