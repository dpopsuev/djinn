// manager.go — real UndoManager backed by EventLog + Mirage Space.
package undo

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/dpopsuev/djinn/telemetry"
	"github.com/dpopsuev/mirage"
	"github.com/dpopsuev/troupe/signal"
)

// Sentinel errors.
var ErrInvalidIndex = errors.New("undo: invalid checkpoint index")

var _ Manager = (*SpaceManager)(nil)

// SpaceManager implements Manager using an EventLog for action tracking
// and a Mirage Space for filesystem rollback.
type SpaceManager struct {
	mu          sync.Mutex
	log         signal.EventLog
	space       mirage.Space
	checkpoints []Checkpoint
	current     int
	logger      *slog.Logger
}

// NewSpaceManager creates a real undo manager.
func NewSpaceManager(log signal.EventLog, space mirage.Space, logger ...*slog.Logger) *SpaceManager {
	l := slog.Default()
	if len(logger) > 0 && logger[0] != nil {
		l = logger[0]
	}
	return &SpaceManager{
		log:     log,
		space:   space,
		current: -1,
		logger:  l,
	}
}

func (m *SpaceManager) Checkpoint(name string) int {
	m.mu.Lock()
	defer m.mu.Unlock()

	idx := m.log.Len()
	cp := Checkpoint{
		Index:     idx,
		Name:      name,
		Timestamp: time.Now(),
	}
	m.checkpoints = append(m.checkpoints, cp)
	m.current = idx

	m.logger.InfoContext(context.Background(), "checkpoint created",
		slog.String(telemetry.KeyComponent, "undo"), slog.String(telemetry.KeyAction, name),
		slog.Int(telemetry.KeyCount, idx),
	)
	return idx
}

func (m *SpaceManager) Rollback(index int) ([]signal.Event, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if index < 0 || index > m.log.Len() {
		m.logger.WarnContext(context.Background(), "rollback to invalid index",
			slog.Int(telemetry.KeyCount, index),
			slog.Int(telemetry.KeyStatus, m.log.Len()),
		)
		return nil, fmt.Errorf("%w: %d (log has %d events)", ErrInvalidIndex, index, m.log.Len())
	}

	// Get events that will be undone
	undone := m.log.Since(index)
	if len(undone) == 0 {
		m.logger.WarnContext(context.Background(), "rollback with no changes to undo", slog.Int(telemetry.KeyCount, index))
		return nil, nil
	}

	// Reset the workspace filesystem
	if err := m.space.Reset(); err != nil {
		return nil, fmt.Errorf("undo: mirage reset: %w", err)
	}

	m.current = index

	m.logger.InfoContext(context.Background(), "rollback complete",
		slog.Int(telemetry.KeyCount, index),
		slog.Int(telemetry.KeyAction, len(undone)),
	)
	return undone, nil
}

func (m *SpaceManager) Current() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.current
}

func (m *SpaceManager) Checkpoints() []Checkpoint {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]Checkpoint, len(m.checkpoints))
	copy(out, m.checkpoints)
	return out
}
