// render.go — markdown rendering for agent output.
// Uses glamour for styled terminal markdown with syntax highlighting.
package repl

import (
	"strings"
	"sync"

	"github.com/charmbracelet/glamour"
)

var (
	mdRenderer     *glamour.TermRenderer
	mdRendererOnce sync.Once
	mdWidth        int
)

// initRenderer creates the glamour markdown renderer.
// Called on first render or when terminal width changes.
func initRenderer(width int) {
	if width <= 0 {
		width = 80
	}
	mdWidth = width
	r, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(width-4), // leave margin
	)
	if err != nil {
		return
	}
	mdRenderer = r
}

// renderMarkdown renders markdown text to styled terminal output.
// Falls back to plain text if renderer is not initialized or fails.
func renderMarkdown(text string) string {
	if mdRenderer == nil {
		mdRendererOnce.Do(func() {
			initRenderer(80)
		})
	}
	if mdRenderer == nil || text == "" {
		return text
	}

	out, err := mdRenderer.Render(text)
	if err != nil {
		return text
	}

	// glamour adds trailing newlines — trim excess
	return strings.TrimRight(out, "\n")
}

// reinitRenderer recreates the renderer with a new width.
func reinitRenderer(width int) {
	if width != mdWidth && width > 0 {
		initRenderer(width)
	}
}
