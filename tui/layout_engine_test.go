package tui

import (
	"strings"
	"testing"
)

func TestLayoutEngine_VisibleSlots_AllVisible(t *testing.T) {
	fm := NewFocusManager()
	e := NewLayoutEngine(fm)
	e.Register(PanelSlot{Panel: NewOutputPanel(), Focusable: true})
	e.Register(PanelSlot{Panel: NewInputPanel(), Focusable: true})

	if len(e.VisibleSlots()) != 2 {
		t.Fatalf("visible = %d, want 2", len(e.VisibleSlots()))
	}
}

func TestLayoutEngine_VisibleSlots_Conditional(t *testing.T) {
	fm := NewFocusManager()
	e := NewLayoutEngine(fm)
	show := false
	e.Register(PanelSlot{Panel: NewOutputPanel(), Focusable: true})
	e.Register(PanelSlot{Panel: NewQueuePanel(), Visible: func() bool { return show }, Focusable: true})
	e.Register(PanelSlot{Panel: NewInputPanel(), Focusable: true})

	if len(e.VisibleSlots()) != 2 {
		t.Fatalf("visible = %d, want 2 (queue hidden)", len(e.VisibleSlots()))
	}

	show = true
	if len(e.VisibleSlots()) != 3 {
		t.Fatalf("visible = %d, want 3 (queue shown)", len(e.VisibleSlots()))
	}
}

func TestLayoutEngine_FocusablePanels(t *testing.T) {
	fm := NewFocusManager()
	e := NewLayoutEngine(fm)
	e.Register(PanelSlot{Panel: NewOutputPanel(), Focusable: true})
	e.Register(PanelSlot{Panel: NewDashboardPanel(), Focusable: false}) // not focusable
	e.Register(PanelSlot{Panel: NewInputPanel(), Focusable: true})

	panels := e.FocusablePanels()
	if len(panels) != 2 {
		t.Fatalf("focusable = %d, want 2", len(panels))
	}
}

func TestLayoutEngine_ComputeHeights_FixedOnly(t *testing.T) {
	fm := NewFocusManager()
	e := NewLayoutEngine(fm)
	e.Resize(80, 24)
	e.Register(PanelSlot{Panel: NewInputPanel(), Border: BorderFocusDepth})   // height=1
	e.Register(PanelSlot{Panel: NewDashboardPanel(), Border: BorderFocusDepth}) // height=1

	heights := e.ComputeHeights()
	if heights["input"] != 1 || heights["dashboard"] != 1 {
		t.Fatalf("heights = %v", heights)
	}
}

func TestLayoutEngine_ComputeHeights_FlexDistribution(t *testing.T) {
	fm := NewFocusManager()
	e := NewLayoutEngine(fm)
	e.Resize(80, 30)
	e.Register(PanelSlot{Panel: NewOutputPanel(), Weight: 1, MinHeight: 3, Border: BorderOnly})
	e.Register(PanelSlot{Panel: NewDashboardPanel(), Border: BorderFocusDepth}) // fixed, height=1

	heights := e.ComputeHeights()
	if heights["output"] < 3 {
		t.Fatalf("output height = %d, want >= 3", heights["output"])
	}
}

func TestLayoutEngine_ComputeHeights_MinHeight(t *testing.T) {
	fm := NewFocusManager()
	e := NewLayoutEngine(fm)
	e.Resize(80, 5) // very small terminal
	e.Register(PanelSlot{Panel: NewOutputPanel(), Weight: 1, MinHeight: 3, Border: BorderOnly})

	heights := e.ComputeHeights()
	if heights["output"] < 3 {
		t.Fatalf("output height = %d, should respect MinHeight 3", heights["output"])
	}
}

func TestLayoutEngine_Render_ProducesOutput(t *testing.T) {
	fm := NewFocusManager()
	e := NewLayoutEngine(fm)
	e.Resize(80, 24)

	op := NewOutputPanel()
	op.Append("hello")
	e.Register(PanelSlot{Panel: op, Weight: 1, MinHeight: 3, Border: BorderOnly, Focusable: true})
	e.Register(PanelSlot{Panel: NewDashboardPanel(), Border: BorderFocusDepth, Focusable: true})

	result := e.Render()
	if result == "" {
		t.Fatal("render should produce output")
	}
	if !strings.Contains(result, "hello") {
		t.Fatal("render should contain panel content")
	}
}

func TestLayoutEngine_Render_SkipsInvisible(t *testing.T) {
	fm := NewFocusManager()
	e := NewLayoutEngine(fm)
	e.Resize(80, 24)

	q := NewQueuePanel()
	q.Update(QueueAddMsg{Prompt: "queued"})
	e.Register(PanelSlot{Panel: NewOutputPanel(), Weight: 1, Border: BorderOnly, Focusable: true})
	e.Register(PanelSlot{Panel: q, Visible: func() bool { return false }, Border: BorderFocusDepth, Focusable: true})

	result := e.Render()
	if strings.Contains(result, "queued") {
		t.Fatal("invisible panel should not appear in render")
	}
}

func TestLayoutEngine_Render_SyncsFocusManager(t *testing.T) {
	fm := NewFocusManager()
	e := NewLayoutEngine(fm)
	e.Resize(80, 24)

	e.Register(PanelSlot{Panel: NewOutputPanel(), Weight: 1, Border: BorderOnly, Focusable: true})
	e.Register(PanelSlot{Panel: NewInputPanel(), Border: BorderFocusDepth, Focusable: true})

	e.Render()
	if fm.Count() != 2 {
		t.Fatalf("focus manager should have 2 panels, got %d", fm.Count())
	}
}

func TestLayoutEngine_BorderModes(t *testing.T) {
	fm := NewFocusManager()
	e := NewLayoutEngine(fm)
	e.Resize(80, 24)

	e.Register(PanelSlot{Panel: NewOutputPanel(), Weight: 1, Border: BorderOnly, Focusable: true})
	e.Register(PanelSlot{Panel: NewDashboardPanel(), Border: BorderNone, Focusable: false})

	result := e.Render()
	if result == "" {
		t.Fatal("should render with mixed border modes")
	}
}

func TestLayoutEngine_HorizontalGroup(t *testing.T) {
	fm := NewFocusManager()
	e := NewLayoutEngine(fm)
	e.Resize(80, 24)

	// Two panels in the same horizontal group.
	p1 := NewOutputPanel()
	p1.Append("LEFT_CONTENT")
	p2 := NewThinkingPanel()
	p2.Update(ThinkingMsg("RIGHT_CONTENT"))

	e.Register(PanelSlot{
		Panel:     p1,
		Weight:    1,
		Border:    BorderOnly,
		Focusable: true,
		Direction: DirHorizontal,
		Group:     "pair",
	})
	e.Register(PanelSlot{
		Panel:     p2,
		Weight:    1,
		Border:    BorderFocusDepth,
		Focusable: true,
		Direction: DirHorizontal,
		Group:     "pair",
	})

	result := e.Render()
	if result == "" {
		t.Fatal("horizontal group should produce output")
	}
	// Both panels should be present.
	if !strings.Contains(result, "LEFT_CONTENT") {
		t.Fatal("should contain left panel content")
	}
	if !strings.Contains(result, "RIGHT_CONTENT") {
		t.Fatal("should contain right panel content")
	}

	// They should be on the same line (side-by-side rendering).
	// Check that at least one line contains content from both panels.
	lines := strings.Split(result, "\n")
	found := false
	for _, line := range lines {
		if strings.Contains(line, "LEFT_CONTENT") || strings.Contains(line, "RIGHT_CONTENT") {
			// In side-by-side layout, there should be NO newline separating the two
			// bordered panels — they appear on the same row of lines.
			found = true
			break
		}
	}
	if !found {
		t.Fatal("horizontal group panels should appear in rendered output")
	}

	// Verify it is NOT a vertical layout: in vertical layout,
	// the border of the second panel would start on a new set of lines.
	// Count the number of top-border lines (╭).
	topBorders := 0
	for _, line := range lines {
		if strings.Contains(line, "╭") {
			topBorders++
		}
	}
	// Side-by-side: both top borders appear on the SAME line => 1 line with ╭.
	if topBorders != 1 {
		t.Fatalf("expected 1 line with top border (side-by-side), got %d", topBorders)
	}
}

func TestLayoutEngine_MixedGroups(t *testing.T) {
	fm := NewFocusManager()
	e := NewLayoutEngine(fm)
	e.Resize(80, 24)

	// Vertical panel above.
	topPanel := NewOutputPanel()
	topPanel.Append("TOP_PANEL")
	e.Register(PanelSlot{
		Panel:     topPanel,
		Weight:    1,
		MinHeight: 3,
		Border:    BorderOnly,
		Focusable: true,
	})

	// Horizontal group below.
	left := NewDashboardPanel()
	right := NewInputPanel()

	e.Register(PanelSlot{
		Panel:     left,
		Border:    BorderFocusDepth,
		Focusable: true,
		Direction: DirHorizontal,
		Group:     "bottom",
	})
	e.Register(PanelSlot{
		Panel:     right,
		Border:    BorderFocusDepth,
		Focusable: true,
		Direction: DirHorizontal,
		Group:     "bottom",
	})

	result := e.Render()
	if result == "" {
		t.Fatal("mixed layout should produce output")
	}
	if !strings.Contains(result, "TOP_PANEL") {
		t.Fatal("should contain top panel content")
	}

	// Top panel is vertical (own border lines), bottom group is horizontal.
	// Should have at least 2 lines containing ╭ (one for top, one for bottom group).
	lines := strings.Split(result, "\n")
	topBorderLines := 0
	for _, line := range lines {
		if strings.Contains(line, "╭") {
			topBorderLines++
		}
	}
	if topBorderLines < 2 {
		t.Fatalf("expected >= 2 lines with top border, got %d", topBorderLines)
	}
}
