package jerichoport

import "github.com/dpopsuev/jericho/pool"

// Type aliases — definitions live in jericho/pool.
type (
	AgentPool       = pool.AgentPool
	AgentSupervisor = pool.AgentSupervisor
	AgentConfig     = pool.AgentConfig
	ExitStatus      = pool.ExitStatus
	ExitCode        = pool.ExitCode
	TreeNode        = pool.TreeNode
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
