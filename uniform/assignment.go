// assignment.go — Assignment extends RoleAssignment with full agent configuration.
// It bridges the gear system (which spawns support roles) with the composition
// system (which instantiates Units in Formations).
package uniform

import (
	"time"

	"github.com/dpopsuev/djinn/uniform/execution"
	"github.com/dpopsuev/djinn/uniform/persona"
)

// Mode constants — string aliases matching agent.Mode names.
// We use strings here to avoid importing the agent package.
const (
	ModeAsk   = "ask"
	ModePlan  = "plan"
	ModeAgent = "agent"
	ModeAuto  = "auto"
)

// Assignment extends RoleAssignment with full agent configuration.
// It bridges the gear system (which spawns support roles) with the
// composition system (which instantiates Units in Formations).
type Assignment struct {
	Role         string           // staff role name (e.g. "gensec", "executor")
	Mode         string           // agent mode: "ask", "plan", "agent", "auto"
	Capabilities []string         // tool capability names from StaffConfig
	Budget       AssignmentBudget // resource limits
	Scope        AssignmentScope  // filesystem access
	Persona      string           // bugle persona name (from persona.RolePersona)
	Model        string           // preferred model (optional override)
}

// AssignmentBudget defines resource limits for an assignment.
type AssignmentBudget struct {
	MaxTokens   int
	MaxCost     float64
	MaxDuration time.Duration
}

// AssignmentScope defines filesystem access boundaries for an assignment.
type AssignmentScope struct {
	ReadPaths  []string
	WritePaths []string
}

// PrimordialGenSec returns the initial GenSec assignment before any split.
// It has all default capabilities, auto mode, and global scope.
func PrimordialGenSec() Assignment {
	cfg := DefaultConfig()
	roles := cfg.RoleMap()
	gensec := roles["gensec"]

	return Assignment{
		Role:         "gensec",
		Mode:         ModeAuto,
		Capabilities: gensec.ToolCapabilities,
		Persona:      persona.RolePersona["gensec"],
		Model:        gensec.Model,
		Scope: AssignmentScope{
			ReadPaths:  []string{"/"},
			WritePaths: []string{"/"},
		},
	}
}

// brokerCapabilities are the intake-only capabilities for the Broker half
// of a split GenSec: routing, interpretation, delegation — no execution tools.
var brokerCapabilities = []string{
	"WorkTracking",
	"IssueTracking",
	"SignalBroadcasting",
	"MemoryRecall",
}

// secretaryCapabilities are the execution capabilities for a scoped Secretary.
var secretaryCapabilities = []string{
	"WorkTracking",
	"FileEditing",
	"ShellExecution",
	"FileSearching",
	"QualityGating",
	"MemoryRecall",
}

// Broker returns the intake/routing half of a split GenSec.
// Mode=agent, intake-only capabilities (routing, interpretation, delegation).
func Broker() Assignment {
	return Assignment{
		Role:         "gensec",
		Mode:         ModeAgent,
		Capabilities: brokerCapabilities,
		Persona:      persona.RolePersona["gensec"],
		Scope: AssignmentScope{
			ReadPaths:  []string{"/"},
			WritePaths: nil,
		},
	}
}

// Secretary returns a scoped execution secretary.
// Mode=agent, scoped to the provided paths, execution capabilities.
func Secretary(scope AssignmentScope) Assignment {
	return Assignment{
		Role:         "executor",
		Mode:         ModeAgent,
		Capabilities: secretaryCapabilities,
		Persona:      persona.RolePersona["executor"],
		Scope:        scope,
	}
}

// AssignmentScheduler plans full Assignment objects for a gear.
// It wraps and extends execution.SupportScheduler with richer configuration.
type AssignmentScheduler interface {
	PlanAssignments(gear execution.Gear) []Assignment
}

// defaultAssignmentScheduler wraps the existing execution.SupportScheduler,
// enriching each RoleAssignment into a full Assignment using defaults.
type defaultAssignmentScheduler struct {
	inner execution.SupportScheduler
}

// DefaultAssignmentScheduler returns an AssignmentScheduler backed by the
// built-in execution.SupportScheduler. Each planned role is enriched with persona,
// mode, and capabilities from DefaultConfig.
func DefaultAssignmentScheduler() AssignmentScheduler {
	return &defaultAssignmentScheduler{inner: execution.DefaultSupportScheduler()}
}

// PlanAssignments calls the inner execution.SupportScheduler.Plan and enriches each
// RoleAssignment into a full Assignment using defaults from StaffConfig.
func (d *defaultAssignmentScheduler) PlanAssignments(gear execution.Gear) []Assignment {
	planned := d.inner.Plan(gear)
	if len(planned) == 0 {
		return nil
	}

	cfg := DefaultConfig()
	roles := cfg.RoleMap()

	assignments := make([]Assignment, len(planned))
	for i, ra := range planned {
		role, ok := roles[ra.Role]
		if !ok {
			// Unknown role — minimal assignment with just the role name.
			assignments[i] = Assignment{
				Role: ra.Role,
				Mode: ModePlan,
			}
			continue
		}

		assignments[i] = Assignment{
			Role:         ra.Role,
			Mode:         role.Mode,
			Capabilities: role.ToolCapabilities,
			Persona:      persona.RolePersona[ra.Role],
			Model:        role.Model,
		}
	}
	return assignments
}
