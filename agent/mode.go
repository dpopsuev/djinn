package agent

import (
	"errors"
	"fmt"
)

// Mode represents the agent operating mode.
type Mode int

const (
	ModeAsk   Mode = iota // No tools, single turn, text only
	ModePlan              // No tools, thinking allowed
	ModeAgent             // Tools with operator approval
	ModeAuto              // Tools without approval
)

// ErrInvalidMode is returned when parsing an unknown mode string.
var ErrInvalidMode = errors.New("invalid agent mode")

var modeNames = [...]string{"ask", "plan", "agent", "auto"}

func (m Mode) String() string {
	if int(m) < len(modeNames) {
		return modeNames[m]
	}
	return fmt.Sprintf("Mode(%d)", m)
}

// ParseMode parses a mode name string into a Mode value.
func ParseMode(s string) (Mode, error) {
	for i, name := range modeNames {
		if name == s {
			return Mode(i), nil
		}
	}
	return 0, fmt.Errorf("%w: %q", ErrInvalidMode, s)
}

// ToolsEnabled returns whether this mode allows tool execution.
func (m Mode) ToolsEnabled() bool {
	return m == ModeAgent || m == ModeAuto
}

// DefaultApprove returns the approval function for this mode.
// Auto returns AutoApprove, Agent returns DenyAll (needs interactive override),
// Ask/Plan return nil (tools disabled).
func (m Mode) DefaultApprove() ApprovalFunc {
	switch m {
	case ModeAuto:
		return AutoApprove
	case ModeAgent:
		return DenyAll
	default:
		return nil
	}
}

// Next returns the next mode in the cycle: ask → plan → agent → auto → ask.
func (m Mode) Next() Mode {
	return Mode((int(m) + 1) % len(modeNames))
}
