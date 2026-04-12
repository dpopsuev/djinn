// Vezir — Control Plane daemon for the Djinn runtime.
//
// Separate binary from day 0. Supervises the Substrate process.
// On crash: auto-restart with backoff. On code change: rebuild → restart.
// TUI connects here (permanent unix socket), Vezir relays to Substrate.
//
// Usage:
//
//	vezir                          # starts, spawns cmd/djinn
//	vezir --substrate ./cmd/djinn  # custom substrate binary path
//	vezir --socket /tmp/vezir.sock # custom TUI socket path
package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/dpopsuev/djinn/telemetry"
	"github.com/dpopsuev/djinn/vezir/supervisor"
)

func main() {
	substrateBin := flag.String("substrate", "./cmd/djinn", "path to Substrate binary or directory")
	socketPath := flag.String("socket", "/tmp/vezir.sock", "unix socket path for TUI relay")
	flag.Parse()

	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	log.InfoContext(ctx, "vezir starting",
		slog.String(telemetry.KeyPath, *substrateBin),
		slog.String(telemetry.KeySource, *socketPath),
	)

	sup := supervisor.New(*substrateBin, *socketPath, log)
	if err := sup.Run(ctx); err != nil {
		log.ErrorContext(ctx, "vezir fatal", slog.String(telemetry.KeyError, err.Error()))
	}
}
