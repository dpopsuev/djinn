// icons.go — Re-exports from tui/icons/ for backward compatibility.
package tui

import "github.com/dpopsuev/djinn/tui/icons"

// NerdFontsAvailable is true when Nerd Font glyphs can be rendered.
var NerdFontsAvailable = icons.NerdFontsAvailable

// Icon holds a Nerd Font glyph and its ASCII fallback.
type Icon = icons.Icon

// Semantic icon registry — re-exports from tui/icons/.
var (
	IconFile    = icons.File
	IconFolder  = icons.Folder
	IconGit     = icons.Git
	IconBranch  = icons.Branch
	IconTag     = icons.Tag
	IconCheck   = icons.Check
	IconCross   = icons.Cross
	IconWarning = icons.Warning
	IconInfo    = icons.Info
	IconError   = icons.Error
	IconSpinner = icons.Spinner
	IconAgent   = icons.Agent
	IconTool    = icons.Tool
	IconClock   = icons.Clock
	IconBudget  = icons.Budget
)
