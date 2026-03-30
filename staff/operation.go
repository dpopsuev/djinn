// operation.go — Agent Operations decouple WHAT GenSec does with a prompt
// from input modes and capacity.
// Operations are API pointers — they hint at intent, not permissions.
// GenSec may silently upgrade Ask to Agent if the prompt requires writes.
package staff

// Operation represents what GenSec does with a prompt.
// Operations are API pointers — they hint at intent, not permissions.
// GenSec may silently upgrade Ask to Agent if the prompt requires writes.
type Operation string

const (
	// OpAsk is read-only introspection. Non-interrupting — does not
	// disturb running agents. GenSec answers directly or delegates to
	// the cheapest available agent.
	OpAsk Operation = "ask"

	// OpPlan discusses or modifies the current plan. Cascading —
	// changes may invalidate in-progress work, triggering interrupts.
	OpPlan Operation = "plan"

	// OpAgent is ad-hoc execution. Interrupting — GenSec decides
	// which agent to route to (or spawns a new one).
	OpAgent Operation = "agent"
)

// allOperations for iteration and validation.
var allOperations = []Operation{OpAsk, OpPlan, OpAgent}

// ParseOperation parses a string into an Operation.
// Returns false if the string is not a valid operation.
func ParseOperation(s string) (Operation, bool) {
	for _, op := range allOperations {
		if string(op) == s {
			return op, true
		}
	}
	return "", false
}

// String returns the operation name.
func (o Operation) String() string { return string(o) }

// IsInterrupting returns true if this operation may interrupt running agents.
func (o Operation) IsInterrupting() bool { return o == OpAgent }

// IsCascading returns true if this operation may cascade to dependent work.
func (o Operation) IsCascading() bool { return o == OpPlan }

// Next cycles to the next operation: ask → plan → agent → ask.
func (o Operation) Next() Operation {
	switch o {
	case OpAsk:
		return OpPlan
	case OpPlan:
		return OpAgent
	default:
		return OpAsk
	}
}

// DefaultOperation is the starting operation for new sessions.
func DefaultOperation() Operation { return OpAgent }
