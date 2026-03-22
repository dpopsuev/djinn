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
}

var defaultCommands = map[string]Command{
	"quit":         {Name: "quit", Description: "Exit the REPL"},
	"cycle-mode":   {Name: "cycle-mode", Description: "Cycle agent mode (ask → plan → agent → auto)"},
	"history-prev": {Name: "history-prev", Description: "Previous input from history"},
	"history-next": {Name: "history-next", Description: "Next input from history"},
	"submit":       {Name: "submit", Description: "Submit current input"},
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
