// base.go — re-exports from tui/core for backward compatibility.
package tui

import "github.com/dpopsuev/djinn/tui/core"

// Type alias — transparent re-export.
type BasePanel = core.BasePanel

// NewBasePanel re-exports the core constructor.
var NewBasePanel = core.NewBasePanel
