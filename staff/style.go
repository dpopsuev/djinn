// style.go — CommunicationStyle port for adaptive GenSec presentation.
//
// The General Secretary develops a communication style that matches
// the operator's preferences. This is NOT formatting (colors/borders).
// This is HOW the GenSec thinks and speaks: tone, density, structure.
//
// The operator never configures this explicitly. They just react
// naturally — "too noisy", "show me the full error" — and the GenSec
// adjusts by updating the style profile.
//
// Default: MapCommunicationStyle (map[string]string in memory).
// Sophia adapter (future): semantic preference learning across sessions.
package staff

// CommunicationStyle stores operator presentation preferences.
// Keys are style dimensions, values are the current preference.
//
// Dimensions:
//   - density: dense | sparse
//   - structure: narrative | bullet | table
//   - tone: formal | casual
//   - icons: emoji | ascii | none
//   - timestamps: always | failures | never
//   - names: heraldic | color | role
//   - failure_detail: full | summary
//   - success_detail: celebrate | silent
type CommunicationStyle interface {
	Get(key string) string
	Set(key, value string)
	All() map[string]string
}

// MapCommunicationStyle is the default in-memory implementation.
// Persists with session via RoleMemory (stored in GenSec's slice).
type MapCommunicationStyle struct {
	prefs map[string]string
}

// NewMapCommunicationStyle creates a style with sensible defaults.
func NewMapCommunicationStyle() *MapCommunicationStyle {
	return &MapCommunicationStyle{
		prefs: map[string]string{
			"density":        "sparse",
			"structure":      "bullet",
			"tone":           "casual",
			"icons":          "ascii",
			"timestamps":     "failures",
			"names":          "role",
			"failure_detail": "full",
			"success_detail": "silent",
		},
	}
}

func (s *MapCommunicationStyle) Get(key string) string {
	return s.prefs[key]
}

func (s *MapCommunicationStyle) Set(key, value string) {
	s.prefs[key] = value
}

func (s *MapCommunicationStyle) All() map[string]string {
	out := make(map[string]string, len(s.prefs))
	for k, v := range s.prefs {
		out[k] = v
	}
	return out
}

// Interface compliance.
var _ CommunicationStyle = (*MapCommunicationStyle)(nil)
