// Package keybind defines the keybinding interface for the REPL.
// Provides a registry of named commands mapped to key sequences.
package keybind

// Command is a named action that can be triggered by a keybinding.
type Command struct {
	Name        string // e.g., "cycle-mode", "quit", "clear"
	Description string
}

// Binding maps a key sequence to a command.
type Binding struct {
	Key     string // e.g., "alt+m", "ctrl+c", "ctrl+l"
	Command string // command name
}

// Table provides key-to-command lookup.
type Table interface {
	Lookup(key string) (Command, bool)
	Bindings() []Binding
}

// DefaultTable returns the built-in keybinding table.
func DefaultTable() Table {
	return &stubTable{}
}

type stubTable struct{}

var defaultBindings = []Binding{
	{Key: "ctrl+c", Command: "quit"},
	{Key: "alt+m", Command: "cycle-mode"},
	{Key: "up", Command: "history-prev"},
	{Key: "down", Command: "history-next"},
	{Key: "enter", Command: "submit"},
	{Key: "tab", Command: "complete"},
	{Key: "shift+tab", Command: "focus-next"},
	{Key: "pgup", Command: "scroll-up"},
	{Key: "pgdown", Command: "scroll-down"},
	{Key: "escape", Command: "back"},
	{Key: "alt+enter", Command: "newline"},
	{Key: "ctrl+r", Command: "review-edits"},
}

var defaultCommands = map[string]Command{
	"quit":         {Name: "quit", Description: "Exit the REPL"},
	"cycle-mode":   {Name: "cycle-mode", Description: "Cycle agent mode"},
	"history-prev": {Name: "history-prev", Description: "Previous input from history"},
	"history-next": {Name: "history-next", Description: "Next input from history"},
	"submit":       {Name: "submit", Description: "Submit current input"},
	"complete":     {Name: "complete", Description: "Tab-complete slash command or accept prediction"},
	"focus-next":   {Name: "focus-next", Description: "Focus next panel"},
	"focus-prev":   {Name: "focus-prev", Description: "Focus previous panel"},
	"focus-up":     {Name: "focus-up", Description: "Focus previous panel"},
	"focus-down":   {Name: "focus-down", Description: "Focus next panel"},
	"dive":         {Name: "dive", Description: "Enter child panel"},
	"climb":        {Name: "climb", Description: "Return to parent panel"},
	"back":         {Name: "back", Description: "Go back: cancel, climb, or dismiss"},
	"scroll-up":    {Name: "scroll-up", Description: "Scroll output up"},
	"scroll-down":  {Name: "scroll-down", Description: "Scroll output down"},
	"newline":      {Name: "newline", Description: "Insert newline in input"},
	"review-edits": {Name: "review-edits", Description: "Review pending file edits"},
}

// Mode represents a vim-style editing mode.
type Mode string

const (
	ModeInsert Mode = "insert"
	ModeNormal Mode = "normal"
)

// ModeTable provides mode-aware keybinding lookup.
type ModeTable struct {
	mode     Mode
	bindings map[Mode][]Binding
	commands map[string]Command
}

// NewModeTable creates a table with default bindings for both modes.
func NewModeTable() *ModeTable {
	t := &ModeTable{
		mode:     ModeInsert,
		commands: defaultCommands,
		bindings: map[Mode][]Binding{
			ModeInsert: defaultBindings,
			ModeNormal: {
				{Key: "j", Command: "focus-down"},
				{Key: "k", Command: "focus-up"},
				{Key: "i", Command: "enter-insert"},
				{Key: "enter", Command: "dive"},
				{Key: "escape", Command: "climb"},
				{Key: "q", Command: "quit"},
			},
		},
	}
	return t
}

// SetMode switches the active mode.
func (t *ModeTable) SetMode(m Mode) { t.mode = m }

// Mode returns the active mode.
func (t *ModeTable) CurrentMode() Mode { return t.mode }

// Lookup finds a command for the given key in the current mode.
func (t *ModeTable) Lookup(key string) (Command, bool) {
	for _, b := range t.bindings[t.mode] {
		if b.Key == key {
			cmd, ok := t.commands[b.Command]
			return cmd, ok
		}
	}
	return Command{}, false
}

// Bindings returns all bindings for the current mode.
func (t *ModeTable) Bindings() []Binding {
	out := make([]Binding, len(t.bindings[t.mode]))
	copy(out, t.bindings[t.mode])
	return out
}

func (t *stubTable) Lookup(key string) (Command, bool) {
	for _, b := range defaultBindings {
		if b.Key == key {
			cmd, ok := defaultCommands[b.Command]
			return cmd, ok
		}
	}
	return Command{}, false
}

func (t *stubTable) Bindings() []Binding {
	out := make([]Binding, len(defaultBindings))
	copy(out, defaultBindings)
	return out
}
