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

// WrapText wraps lines longer than width at word boundaries.
func WrapText(text string, width int) string {
	if width <= 0 || len(text) <= width {
		return text
	}

	var result strings.Builder
	for _, line := range strings.Split(text, "\n") {
		if len(line) <= width {
			if result.Len() > 0 {
				result.WriteByte('\n')
			}
			result.WriteString(line)
			continue
		}
		// Wrap at word boundary
		for len(line) > width {
			breakAt := width
			// Find last space before width
			for i := width; i >= width/2; i-- {
				if line[i] == ' ' {
					breakAt = i
					break
				}
			}
			if result.Len() > 0 {
				result.WriteByte('\n')
			}
			result.WriteString(line[:breakAt])
			line = strings.TrimLeft(line[breakAt:], " ")
		}
		if len(line) > 0 {
			if result.Len() > 0 {
				result.WriteByte('\n')
			}
			result.WriteString(line)
		}
	}
	return result.String()
}
