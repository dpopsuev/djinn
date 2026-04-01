// layout_compat.go — temporary re-exports killed by TSK-533.
package tui

import "github.com/dpopsuev/djinn/tui/layout"

// Layout types re-exported for backward compatibility.
type (
	Breakpoint       = layout.Breakpoint
	LayoutEngine     = layout.LayoutEngine
	PanelSlot        = layout.PanelSlot
	Constraint       = layout.Constraint
	Fixed            = layout.Fixed
	MinHeight        = layout.MinHeight
	MaxHeight        = layout.MaxHeight
	Fill             = layout.Fill
	Percentage       = layout.Percentage
	PanelConstraints = layout.PanelConstraints
	ThirdsLayout     = layout.ThirdsLayout
	BorderMode       = layout.BorderMode
	LayoutConfig     = layout.LayoutConfig
	HeightBreakpoint = layout.HeightBreakpoint
	BorderRenderer   = layout.BorderRenderer
)

// Constructors and functions.
var (
	ComputeLayout       = layout.ComputeLayout
	ComputeThirdsLayout = layout.ComputeThirdsLayout
	Solve               = layout.Solve
	ClassifyHeight      = layout.ClassifyHeight
)

// NewLayoutEngine wraps layout.NewLayoutEngine with the default border renderer.
func NewLayoutEngine(fm *FocusManager) *LayoutEngine {
	return layout.NewLayoutEngine(fm, defaultBorderRenderer{})
}

// Breakpoint constants.
const (
	Small   = layout.Small
	Medium  = layout.Medium
	Large   = layout.Large
	Massive = layout.Massive
)

// BorderMode constants.
const (
	BorderFocusDepth = layout.BorderFocusDepth
	BorderOnly       = layout.BorderOnly
	BorderNone       = layout.BorderNone
)

// Dashboard style constants.
const (
	DashboardStyleCompact = layout.DashboardStyleCompact
	DashboardStyleFull    = layout.DashboardStyleFull
)

// Layout constraint constants.
const (
	MinOutputHeight    = layout.MinOutputHeight
	MinInputHeight     = layout.MinInputHeight
	MinDashboardHeight = layout.MinDashboardHeight
	InputAnchorRatio   = layout.InputAnchorRatio
	DashboardRatio     = layout.DashboardRatio
)

// Height breakpoint constants.
const (
	HeightTiny   = layout.HeightTiny
	HeightSmall  = layout.HeightSmall
	HeightMedium = layout.HeightMedium
	HeightLarge  = layout.HeightLarge
)

// defaultBorderRenderer implements layout.BorderRenderer using tui/focus.go functions.
type defaultBorderRenderer struct{}

func (defaultBorderRenderer) RenderWithDepth(content string, depth, width int) string {
	return RenderWithDepth(content, depth, width)
}

func (defaultBorderRenderer) RenderBorderOnly(content string, focused bool, width int) string {
	return RenderBorderOnly(content, focused, width)
}

func (defaultBorderRenderer) FocusDepths(count, focusedIdx int) []int {
	return FocusDepths(count, focusedIdx)
}
