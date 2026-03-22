// Package djinnlog sets up structured logging for Djinn using log/slog.
// NOT a wrapper — components receive *slog.Logger directly.
package djinnlog

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
)

// Options configures the logging stack.
type Options struct {
	Verbose  bool   // enable terminal output at Info level
	LogFile  string // path to log file (empty = no file)
	RingSize int    // ring buffer capacity (default: 500)
}

// Result holds the logger and ring buffer for introspection.
type Result struct {
	Logger *slog.Logger
	Ring   *RingHandler
}

// Setup creates the multi-handler logger stack.
// File handler (JSON, Debug) is always active if LogFile is set.
// Terminal handler (text, Info) only if Verbose.
// Ring handler always active for /log introspection.
func Setup(opts Options) Result {
	if opts.RingSize == 0 {
		opts.RingSize = 500
	}

	ring := NewRingHandler(opts.RingSize)
	var handlers []slog.Handler

	// Always: ring buffer at Debug level
	handlers = append(handlers, ring)

	// File handler: JSON at Debug level
	if opts.LogFile != "" {
		if fh := newFileHandler(opts.LogFile); fh != nil {
			handlers = append(handlers, fh)
		}
	}

	// Terminal handler: text at Info level (only if verbose)
	if opts.Verbose {
		handlers = append(handlers, slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}))
	}

	var handler slog.Handler
	if len(handlers) == 1 {
		handler = handlers[0]
	} else {
		handler = NewMultiHandler(handlers...)
	}

	return Result{
		Logger: slog.New(handler),
		Ring:   ring,
	}
}

func newFileHandler(path string) slog.Handler {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return nil
	}
	// Note: file is never explicitly closed — it lives for the process lifetime.
	return slog.NewJSONHandler(f, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
}

// For creates a child logger with a component attribute.
func For(parent *slog.Logger, component string) *slog.Logger {
	if parent == nil {
		return Nop()
	}
	return parent.With("component", component)
}

// Nop returns a logger that discards all output. Useful for tests.
func Nop() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{
		Level: slog.Level(100), // higher than any real level
	}))
}
