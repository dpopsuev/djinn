package jerichoport

import "github.com/dpopsuev/jericho/orchestrate"

// Type aliases — definitions live in bugle/orchestrate.
type (
	WorkerManager = orchestrate.Manager
	WorkerConfig  = orchestrate.WorkerConfig
)

// Constructor.
var NewWorkerManager = orchestrate.NewManager
