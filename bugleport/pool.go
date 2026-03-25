package bugleport

import "github.com/dpopsuev/bugle/pool"

// Type aliases — definitions live in bugle/pool.
type (
	AgentPool    = pool.AgentPool
	Launcher     = pool.Launcher
	LaunchConfig = pool.LaunchConfig
)

// Constructor.
var NewAgentPool = pool.New
