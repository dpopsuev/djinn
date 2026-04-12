package supervisor

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestHeartbeat_AliveAfterPing(t *testing.T) {
	dir := t.TempDir()
	sock := filepath.Join(dir, "hb.sock")
	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))

	hb := NewHeartbeat(sock, 2*time.Second, log)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := hb.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer hb.Stop()

	// Not alive before any ping.
	if hb.Alive() {
		t.Fatal("should not be alive before first ping")
	}

	// Start sending heartbeats.
	go SendHeartbeat(ctx, sock, 100*time.Millisecond)

	// Wait for a ping.
	time.Sleep(300 * time.Millisecond)

	if !hb.Alive() {
		t.Fatal("should be alive after pings")
	}

	if hb.LastPing().IsZero() {
		t.Fatal("last ping should not be zero")
	}
}

func TestHeartbeat_DeadAfterTimeout(t *testing.T) {
	dir := t.TempDir()
	sock := filepath.Join(dir, "hb.sock")
	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))

	hb := NewHeartbeat(sock, 200*time.Millisecond, log)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := hb.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer hb.Stop()

	// Send one ping then stop.
	senderCtx, senderCancel := context.WithCancel(ctx)
	go SendHeartbeat(senderCtx, sock, 50*time.Millisecond)

	time.Sleep(150 * time.Millisecond)
	if !hb.Alive() {
		t.Fatal("should be alive during pings")
	}

	// Stop sender.
	senderCancel()
	time.Sleep(400 * time.Millisecond)

	// Should be dead now (timeout passed).
	if hb.Alive() {
		t.Fatal("should NOT be alive after timeout with no pings")
	}
}
