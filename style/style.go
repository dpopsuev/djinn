// Package style defines the style preset interface for the REPL.
// Presets control colors, labels, and visual appearance.
package style

// Preset defines a visual theme for the REPL.
type Preset struct {
	Name       string
	UserColor  string // hex color for user prompt
	AgentColor string // hex color for agent response
	ToolColor  string // hex color for tool calls
	ErrorColor string // hex color for errors
	DimColor   string // hex color for dim/secondary text
	LogoColor  string // hex color for logo/branding
	UserLabel  string // e.g., "> ", "you: "
	AgentLabel string // e.g., "djinn", "claude"
}

// Registry provides access to style presets.
type Registry interface {
	Get(name string) (Preset, bool)
	List() []Preset
	Default() Preset
}

// DefaultRegistry returns the built-in style presets.
func DefaultRegistry() Registry {
	return &builtinRegistry{}
}

type builtinRegistry struct{}

var builtinPresets = []Preset{
	{
		Name:       "djinn",
		UserColor:  "#4ade80",
		AgentColor: "#60a5fa",
		ToolColor:  "#facc15",
		ErrorColor: "#f87171",
		DimColor:   "#6b7280",
		LogoColor:  "#EE0000",
		UserLabel:  "> ",
		AgentLabel: "djinn",
	},
	{
		Name:       "minimal",
		UserColor:  "#a3a3a3",
		AgentColor: "#a3a3a3",
		ToolColor:  "#a3a3a3",
		ErrorColor: "#ef4444",
		DimColor:   "#525252",
		LogoColor:  "#a3a3a3",
		UserLabel:  "> ",
		AgentLabel: "ai",
	},
}

func (r *builtinRegistry) Get(name string) (Preset, bool) {
	for i := range builtinPresets {
		if builtinPresets[i].Name == name {
			return builtinPresets[i], true
		}
	}
	return Preset{}, false
}

func (r *builtinRegistry) List() []Preset {
	out := make([]Preset, len(builtinPresets))
	copy(out, builtinPresets)
	return out
}

func (r *builtinRegistry) Default() Preset {
	return builtinPresets[0]
}
