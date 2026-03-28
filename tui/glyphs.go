// glyphs.go — configurable display symbols for the Djinn TUI.
// All rendering reads from ActiveGlyphs, never hardcoded strings.
// Loaded from djinn.yaml style.glyphs section.
package tui

// Glyphs holds all configurable display symbols.
type Glyphs struct {
	ToolCall    string `yaml:"tool_call"`
	ToolSuccess string `yaml:"tool_success"`
	ToolError   string `yaml:"tool_error"`
	UserPrompt  string `yaml:"user_prompt"`
	AssistLabel string `yaml:"assist_label"`
	ListCursor  string `yaml:"list_cursor"`
	Sandboxed   string `yaml:"sandboxed"`
	Unsandboxed string `yaml:"unsandboxed"`
	Shell       string `yaml:"shell"`
	Approved    string `yaml:"approved"`
	Denied      string `yaml:"denied"`
	Canceled    string `yaml:"canceled"`
	Queued      string `yaml:"queued"`
}

// DefaultGlyphs returns the default glyph set.
func DefaultGlyphs() Glyphs {
	return Glyphs{
		ToolCall:    "λ",
		ToolSuccess: "✓",
		ToolError:   "✗",
		UserPrompt:  "> ",
		AssistLabel: "",
		ListCursor:  "▸",
		Sandboxed:   "[>]",
		Unsandboxed: ">",
		Shell:       "$",
		Approved:    "approved",
		Denied:      "denied",
		Canceled:    "(canceled)",
		Queued:      "queued:",
	}
}

// ActiveGlyphs is the currently active glyph set.
var ActiveGlyphs = DefaultGlyphs()

// ApplyGlyphs sets the active glyph set and updates dependent variables.
func ApplyGlyphs(g Glyphs) {
	ActiveGlyphs = g
	// Update style variables that were previously const.
	LabelUser = g.UserPrompt
	LabelAssist = g.AssistLabel
	GlyphToolCall = g.ToolCall
	GlyphToolSuccess = g.ToolSuccess
	GlyphToolError = g.ToolError
	// Spinner frames stay in SpinnerFrames (theme-level, not glyph-level).
}
