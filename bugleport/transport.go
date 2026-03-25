package bugleport

import "github.com/dpopsuev/bugle/transport"

// Type aliases — definitions live in bugle/transport.
type (
	Transport      = transport.Transport
	LocalTransport = transport.LocalTransport
	Message        = transport.Message
	Task           = transport.Task
	Event          = transport.Event
	Handler        = transport.Handler
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
