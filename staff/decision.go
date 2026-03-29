// decision.go — Three-layer control: Decision type + Zone routing (NED-16, TSK-471).
//
// Every signal follows the chain: Algo (score) → GenSec (interpret) → Operator (override).
// Zone determines how far up the chain a signal travels:
//   - Green:  algo auto-continues, zero LLM cost
//   - Yellow: GenSec interprets, operator notified
//   - Orange: GenSec proposes, operator can override
//   - Red:    operator must approve before action is taken
package staff

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/dpopsuev/djinn/signal"
)

// Action represents a corrective action proposed by GenSec.
type Action string

const (
	ActionContinue Action = "continue" // no change needed
	ActionReScope  Action = "re_scope" // narrow the scope
	ActionRePlan   Action = "re_plan"  // revise the plan
	ActionCordon   Action = "cordon"   // pause and escalate
	ActionRelay    Action = "relay"    // context relay to fresh agent
	ActionAbort    Action = "abort"    // stop all work
	ActionThrottle Action = "throttle" // reduce parallelism
	ActionSkip     Action = "skip"     // skip current task
)

// Decision captures GenSec's interpretation of a signal.
type Decision struct {
	Action     Action  `json:"action"`
	Reason     string  `json:"reason"`
	Confidence float64 `json:"confidence"`
	Pillar     string  `json:"pillar"` // signal category that triggered this
}

// ParseDecision extracts a Decision from GenSec's JSON response.
// Handles raw JSON and markdown-fenced JSON (```json ... ```).
func ParseDecision(response, pillar string) (Decision, error) {
	cleaned := stripMarkdownFences(response)
	var d Decision
	if err := json.Unmarshal([]byte(cleaned), &d); err != nil {
		return Decision{
			Action:     ActionContinue,
			Reason:     "failed to parse GenSec response: " + response,
			Confidence: 0,
			Pillar:     pillar,
		}, fmt.Errorf("parse decision: %w", err)
	}
	d.Pillar = pillar
	return d, nil
}

// stripMarkdownFences removes ```json ... ``` wrappers from LLM responses.
func stripMarkdownFences(s string) string {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "```") {
		return s
	}
	lines := strings.Split(s, "\n")
	jsonLines := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			continue
		}
		jsonLines = append(jsonLines, line)
	}
	return strings.TrimSpace(strings.Join(jsonLines, "\n"))
}

// Zone determines how far up the three-layer chain a signal travels.
type Zone int

const (
	ZoneGreen  Zone = iota // auto-continue, zero LLM
	ZoneYellow             // GenSec decides, operator notified
	ZoneOrange             // GenSec proposes, operator can override
	ZoneRed                // operator must approve
)

// Zone name constants.
const (
	zoneNameGreen   = "green"
	zoneNameYellow  = "yellow"
	zoneNameOrange  = "orange"
	zoneNameRed     = "red"
	zoneNameUnknown = "unknown"
)

// String returns the zone name.
func (z Zone) String() string {
	switch z {
	case ZoneGreen:
		return zoneNameGreen
	case ZoneYellow:
		return zoneNameYellow
	case ZoneOrange:
		return zoneNameOrange
	case ZoneRed:
		return zoneNameRed
	default:
		return zoneNameUnknown
	}
}

// ZoneFromLevel maps a signal flag level to a control zone.
func ZoneFromLevel(level signal.FlagLevel) Zone {
	switch level {
	case signal.Green:
		return ZoneGreen
	case signal.Yellow:
		return ZoneYellow
	case signal.Red:
		return ZoneOrange
	case signal.Black:
		return ZoneRed
	default:
		return ZoneGreen
	}
}

// AuditEntry records the full three-layer decision chain for a single signal.
type AuditEntry struct {
	Timestamp        time.Time `json:"ts"`
	Signal           signal.Signal
	Zone             Zone
	Decision         Decision
	OperatorOverride *Action `json:"operator_override,omitempty"`
}

// AuditLog is a thread-safe append-only log of decisions.
type AuditLog struct {
	entries []AuditEntry
}

// Append adds an entry to the audit log.
func (al *AuditLog) Append(entry AuditEntry) {
	al.entries = append(al.entries, entry)
}

// Entries returns a copy of all audit entries.
func (al *AuditLog) Entries() []AuditEntry {
	out := make([]AuditEntry, len(al.entries))
	copy(out, al.entries)
	return out
}

// Since returns audit entries after the given time.
func (al *AuditLog) Since(t time.Time) []AuditEntry {
	var out []AuditEntry
	for i := range al.entries {
		if al.entries[i].Timestamp.After(t) {
			out = append(out, al.entries[i])
		}
	}
	return out
}
