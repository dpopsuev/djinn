package bugleport

import "github.com/dpopsuev/bugle/pool"

// Type aliases — definitions live in bugle/pool.
type (
	AgentPool    = pool.AgentPool
	Launcher     = pool.Launcher
	LaunchConfig = pool.LaunchConfig
	ExitStatus   = pool.ExitStatus
	ExitCode     = pool.ExitCode
	TreeNode     = pool.TreeNode
)

// Exit code constants.
const (
	ExitSuccess = pool.ExitSuccess
	ExitError   = pool.ExitError
	ExitBudget  = pool.ExitBudget
	ExitTimeout = pool.ExitTimeout
)

// Sentinel errors.
var (
	ErrNotFound = pool.ErrNotFound
	ErrNotOwner = pool.ErrNotOwner
)

// Constructor.
var NewAgentPool = pool.New
