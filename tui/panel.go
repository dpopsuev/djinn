// panel.go — re-exports from tui/core for backward compatibility.
// Consumers should import tui/core directly once callers are migrated.
package tui

import "github.com/dpopsuev/djinn/tui/core"

// Type aliases — transparent re-exports, no wrapper overhead.
type Panel = core.Panel
type SplitDir = core.SplitDir
type FocusManager = core.FocusManager

const (
	DirVertical   = core.DirVertical
	DirHorizontal = core.DirHorizontal
)

// NewFocusManager re-exports the core constructor.
var NewFocusManager = core.NewFocusManager
