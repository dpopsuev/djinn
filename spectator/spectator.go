// Package spectator provides read-only observation of Djinn backend sessions.
//
// A spectator connects to a backend's clutch transport and receives all
// events (text, tool calls, done, errors) without being able to send
// prompts or approvals. This enables HITL (Human-in-the-Loop) monitoring
// where the Head of Staff watches an Executor work in real-time.
//
// Future: intervention capability (pause, override, redirect) via
// a secondary command channel.
//
// All implementations are stubs — real spectator mode requires
// multi-client socket support on the clutch listener.
package spectator

import (
	"errors"

	"github.com/dpopsuev/djinn/clutch"
)

// Sentinel errors.
var ErrNotImplemented = errors.New("spectator not implemented — requires multi-client socket support")

// Spectator observes a backend session in read-only mode.
// It receives all BackendMsg events but cannot send ShellMsg commands.
type Spectator interface {
	// Attach connects to a backend transport and starts receiving events.
	Attach(transport clutch.Transport) error
	// Detach disconnects from the backend.
	Detach() error
	// OnEvent registers a callback for each received backend event.
	OnEvent(handler func(clutch.BackendMsg))
}

// IsolationControl provides runtime sandbox manipulation.
// Used by the /mount, /unmount, /network, /resources slash commands
// to dynamically adjust the jail while the backend is running.
type IsolationControl interface {
	Mount(path string, readOnly bool) error
	Unmount(path string) error
	SetNetwork(enabled bool) error
	Resources() (ResourceInfo, error)
}

// ResourceInfo reports the current resource allocation of a sandbox.
type ResourceInfo struct {
	CPUs     int
	MemoryMB int
	DiskGB   int
	Network  bool
	Mounts   []MountInfo
}

// MountInfo describes a single filesystem mount inside the sandbox.
type MountInfo struct {
	HostPath  string
	GuestPath string
	ReadOnly  bool
}

// ReadOnlySpectator is the stub implementation.
// It wraps a transport and only receives — never sends.
type ReadOnlySpectator struct {
	transport clutch.Transport
	handler   func(clutch.BackendMsg)
	running   bool
}

func (s *ReadOnlySpectator) Attach(transport clutch.Transport) error {
	s.transport = transport
	s.running = true
	return ErrNotImplemented
}

func (s *ReadOnlySpectator) Detach() error {
	s.running = false
	return nil
}

func (s *ReadOnlySpectator) OnEvent(handler func(clutch.BackendMsg)) {
	s.handler = handler
}

// Interface compliance.
var _ Spectator = (*ReadOnlySpectator)(nil)
