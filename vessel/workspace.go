// workspace.go — WorkspaceVessel: real Vessel rooted in a workspace directory.
//
// Created by Substrate.Vessel(). Tools resolve relative paths against WorkDir.
// EventLog flows to the Substrate's unified log.
//
// DJN-TSK-1033
package vessel

import (
	"context"
	"log/slog"

	"github.com/dpopsuev/battery/tool"
	"github.com/dpopsuev/djinn/telemetry"
	"github.com/dpopsuev/troupe/signal"
)

var _ Vessel = (*WorkspaceVessel)(nil)

// WorkspaceVessel is a real Vessel rooted in a workspace directory.
// Tools resolve relative paths against workDir. EventLog flows to
// the Substrate's unified log.
type WorkspaceVessel struct {
	tools    tool.Executor
	eventLog signal.EventLog
	workDir  string
	log      *slog.Logger
}

// NewWorkspaceVessel creates a Vessel rooted at workDir.
// The caller provides a workspace-rooted tool executor and event log.
func NewWorkspaceVessel(workDir string, tools tool.Executor, eventLog signal.EventLog, opts ...VesselOption) *WorkspaceVessel {
	v := &WorkspaceVessel{
		tools:    tools,
		eventLog: eventLog,
		workDir:  workDir,
	}
	for _, opt := range opts {
		opt(v)
	}
	if v.log != nil {
		v.log.InfoContext(context.Background(), "vessel created",
			slog.String(telemetry.KeyWorkDir, workDir),
			slog.Int(telemetry.KeyCount, len(tools.Names())),
		)
	}
	return v
}

// VesselOption configures a WorkspaceVessel.
type VesselOption func(*WorkspaceVessel)

// WithLogger sets a structured logger on the Vessel.
func WithLogger(l *slog.Logger) VesselOption {
	return func(v *WorkspaceVessel) { v.log = l }
}

func (v *WorkspaceVessel) Tools() tool.Executor      { return v.tools }
func (v *WorkspaceVessel) EventLog() signal.EventLog { return v.eventLog }
func (v *WorkspaceVessel) WorkDir() string           { return v.workDir }

func (v *WorkspaceVessel) Close(_ context.Context) error {
	if v.log != nil {
		v.log.InfoContext(context.Background(), "vessel closed", slog.String(telemetry.KeyWorkDir, v.workDir))
	}
	return nil
}
