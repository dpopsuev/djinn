// Package widgets provides all concrete Panel implementations for the Djinn TUI.
//
// Each widget embeds core.BasePanel and implements core.Panel (Update/View).
// Widgets import tui/ for message types and style vars, and tui/core for Panel/BasePanel.
// The parent tui/ package re-exports widget types via type aliases for backward compat.
package widgets
