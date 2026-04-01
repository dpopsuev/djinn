package jerichoport

import "github.com/dpopsuev/jericho/transport"

// Type aliases — definitions live in jericho/transport.
type (
	Transport      = transport.Transport
	LocalTransport = transport.LocalTransport
	Message        = transport.Message
	Task           = transport.Task
	Event          = transport.Event
	MsgHandler     = transport.MsgHandler
	AgentCard      = transport.AgentCard
	TaskState      = transport.TaskState
)

// Task state constants.
const (
	TaskSubmitted = transport.TaskSubmitted
	TaskWorking   = transport.TaskWorking
	TaskCompleted = transport.TaskCompleted
	TaskFailed    = transport.TaskFailed
)

// Constructor.
var NewLocalTransport = transport.NewLocalTransport
