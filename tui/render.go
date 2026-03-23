// render.go — markdown rendering for agent output.
package tui

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

// InitRenderer creates the glamour markdown renderer.
func InitRenderer(width int) {
	if width <= 0 {
		width = 80
	}
	mdWidth = width
	r, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(width-4),
	)
	if err != nil {
		return
	}
	mdRenderer = r
}

// RenderMarkdown renders markdown text to styled terminal output.
func RenderMarkdown(text string) string {
	if mdRenderer == nil {
		mdRendererOnce.Do(func() {
			InitRenderer(80)
		})
	}
	if mdRenderer == nil || text == "" {
		return text
	}

	out, err := mdRenderer.Render(text)
	if err != nil {
		return text
	}

	return strings.TrimRight(out, "\n")
}

// ReinitRenderer recreates the renderer with a new width.
func ReinitRenderer(width int) {
	if width != mdWidth && width > 0 {
		InitRenderer(width)
	}
}
