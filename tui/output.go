// output.go — OutputPanel wraps a viewport for scrollable conversation.
package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

// OutputPanel is the scrollable conversation area.
// State changes via messages (OutputAppendMsg, etc.) or legacy methods.
type OutputPanel struct {
	BasePanel
	vp          viewport.Model
	vpReady     bool
	lines       []string
	overlay     string // ephemeral content (spinner, streaming, approval)
	dirty       bool   // content changed since last View() — avoids flicker (BUG-25)
	lastContent string // previous frame's content for change detection
	streamBuf   strings.Builder // stream buffer (moved from Model)
}

const panelIDOutput = "output"

// NewOutputPanel creates the output panel.
func NewOutputPanel() *OutputPanel {
	return &OutputPanel{
		BasePanel: NewBasePanel(panelIDOutput, 0),
		dirty:     true,
	}
}

var _ Panel = (*OutputPanel)(nil)

func (p *OutputPanel) InitViewport(width, height int) {
	if height < 3 {
		height = 3
	}
	if !p.vpReady {
		p.vp = viewport.New(width, height)
		p.vpReady = true
		p.dirty = true
	} else {
		if p.vp.Width != width || p.vp.Height != height {
			p.vp.Width = width
			p.vp.Height = height
			p.dirty = true
		}
	}
}

// Append adds a line to the output.
func (p *OutputPanel) Append(line string) {
	p.lines = append(p.lines, line)
	p.dirty = true
}

// SetLine replaces a specific line by index.
func (p *OutputPanel) SetLine(idx int, line string) {
	if idx >= 0 && idx < len(p.lines) {
		p.lines[idx] = line
		p.dirty = true
	}
}

// LineCount returns the number of lines.
func (p *OutputPanel) LineCount() int {
	return len(p.lines)
}

// Lines returns all lines.
func (p *OutputPanel) Lines() []string {
	return p.lines
}

// SetOverlay sets ephemeral content rendered after the lines.
func (p *OutputPanel) SetOverlay(text string) {
	if p.overlay != text {
		p.overlay = text
		p.dirty = true
	}
}

// AppendToLast extends the last line (for stream flush).
func (p *OutputPanel) AppendToLast(text string) {
	if len(p.lines) > 0 {
		p.lines[len(p.lines)-1] += text
		p.dirty = true
	}
}

// StreamBufString returns the current stream buffer content (for overlay preview).
func (p *OutputPanel) StreamBufString() string {
	return p.streamBuf.String()
}

// Clear removes all lines.
func (p *OutputPanel) Clear() {
	p.lines = nil
	p.dirty = true
}

func (p *OutputPanel) Update(msg tea.Msg) (Panel, tea.Cmd) {
	switch msg := msg.(type) {
	case OutputAppendMsg:
		p.Append(msg.Line)
	case OutputSetLineMsg:
		p.SetLine(msg.Index, msg.Line)
	case OutputAppendLastMsg:
		p.AppendToLast(msg.Text)
	case OutputClearMsg:
		p.Clear()
	case OutputSetOverlayMsg:
		p.SetOverlay(msg.Text)
	case ResizeMsg:
		p.InitViewport(msg.Width, msg.Height)
	case TextMsg:
		p.streamBuf.WriteString(string(msg))
	case FlushStreamMsg:
		if p.streamBuf.Len() > 0 {
			p.AppendToLast(p.streamBuf.String())
			p.streamBuf.Reset()
		}
	default:
		if !p.focused || !p.vpReady {
			return p, nil
		}
		var cmd tea.Cmd
		p.vp, cmd = p.vp.Update(msg)
		return p, cmd
	}
	return p, nil
}

func (p *OutputPanel) View(width int) string {
	content := strings.Join(p.lines, "\n")
	if p.overlay != "" {
		content += "\n" + p.overlay
	}
	if p.vpReady {
		// Only update viewport content when dirty — avoids flicker (BUG-25).
		if p.dirty || content != p.lastContent {
			p.vp.SetContent(content)
			p.vp.GotoBottom()
			p.lastContent = content
			p.dirty = false
		}
		return p.vp.View()
	}
	return content
}
