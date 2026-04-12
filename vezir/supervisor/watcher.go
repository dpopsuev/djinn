// watcher.go — watches for binary changes and triggers rebuild + restart.
// Polls binary mtime rather than using fsnotify (simpler, cross-platform,
// EventLog IS the notification — NED-48).
package supervisor

import (
	"context"
	"log/slog"
	"os"
	"os/exec"
	"time"

	"github.com/dpopsuev/djinn/telemetry"
)

// Watcher polls a binary file for mtime changes and triggers a callback.
type Watcher struct {
	binaryPath string
	buildCmd   string // e.g. "go build -o ./djinn ./cmd/djinn"
	interval   time.Duration
	log        *slog.Logger
	onRebuild  func() // callback after successful rebuild
}

// NewWatcher creates a file watcher that polls binaryPath for changes.
func NewWatcher(binaryPath, buildCmd string, interval time.Duration, log *slog.Logger, onRebuild func()) *Watcher {
	return &Watcher{
		binaryPath: binaryPath,
		buildCmd:   buildCmd,
		interval:   interval,
		log:        log,
		onRebuild:  onRebuild,
	}
}

// Watch polls for changes until context is canceled. Blocking.
func (w *Watcher) Watch(ctx context.Context) {
	lastMod := w.mtime()

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			current := w.mtime()
			if current.After(lastMod) {
				w.log.InfoContext(ctx, "binary changed — rebuilding",
					slog.String(telemetry.KeyPath, w.binaryPath),
				)

				if err := w.rebuild(ctx); err != nil {
					w.log.WarnContext(ctx, "rebuild failed",
						slog.String(telemetry.KeyError, err.Error()),
					)
					continue
				}

				lastMod = w.mtime()
				w.log.InfoContext(ctx, "rebuild succeeded — triggering restart")

				if w.onRebuild != nil {
					w.onRebuild()
				}
			}
		}
	}
}

func (w *Watcher) rebuild(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "go", "build", "-o", w.binaryPath, w.buildCmd) //nolint:gosec // buildCmd is operator-configured
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (w *Watcher) mtime() time.Time {
	info, err := os.Stat(w.binaryPath)
	if err != nil {
		return time.Time{}
	}
	return info.ModTime()
}
