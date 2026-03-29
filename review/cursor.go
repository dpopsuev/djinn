// cursor.go — Cursor tracking for view awareness (TSK-468).
//
// Tracks the operator's current focus position during review.
// Updated by TUI navigation events, read by ReviewContext.
package review

// Cursor tracks the operator's current focus in the review.
type Cursor struct {
	CircuitIndex int    // which circuit is focused (-1 = none)
	StopIndex    int    // which stop within the circuit (-1 = none)
	File         string // currently focused file
}

// NewCursor creates a cursor with no focus.
func NewCursor() *Cursor {
	return &Cursor{CircuitIndex: -1, StopIndex: -1}
}

// MoveTo updates the cursor position.
func (c *Cursor) MoveTo(circuit, stop int) {
	c.CircuitIndex = circuit
	c.StopIndex = stop
}

// FocusFile sets the currently focused file.
func (c *Cursor) FocusFile(file string) {
	c.File = file
}

// Reset clears the cursor.
func (c *Cursor) Reset() {
	c.CircuitIndex = -1
	c.StopIndex = -1
	c.File = ""
}
