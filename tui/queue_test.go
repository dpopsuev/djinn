package tui

import (
	"strings"
	"testing"
)

func TestQueuePanel_AddAndView(t *testing.T) {
	q := NewQueuePanel()
	q.Update(QueueAddMsg{"fix the bug"})
	q.Update(QueueAddMsg{"run tests"})

	if q.Len() != 2 {
		t.Fatalf("len = %d, want 2", q.Len())
	}

	view := q.View(80)
	if !strings.Contains(view, "1.") || !strings.Contains(view, "fix the bug") {
		t.Fatalf("view missing item 1: %q", view)
	}
	if !strings.Contains(view, "2.") || !strings.Contains(view, "run tests") {
		t.Fatalf("view missing item 2: %q", view)
	}
}

func TestQueuePanel_EmptyView(t *testing.T) {
	q := NewQueuePanel()
	if q.View(80) != "" {
		t.Fatal("empty queue should return empty view")
	}
}

func TestQueuePanel_Drain(t *testing.T) {
	q := NewQueuePanel()
	q.Update(QueueAddMsg{"first"})
	q.Update(QueueAddMsg{"second"})
	q.Update(QueueDrainMsg{})

	if q.Len() != 1 {
		t.Fatalf("len = %d after drain, want 1", q.Len())
	}
	if q.Items()[0] != "second" {
		t.Fatalf("first item = %q, want second", q.Items()[0])
	}
}

func TestQueuePanel_Clear(t *testing.T) {
	q := NewQueuePanel()
	q.Update(QueueAddMsg{"one"})
	q.Update(QueueAddMsg{"two"})
	q.Update(QueueClearMsg{})

	if q.Len() != 0 {
		t.Fatalf("len = %d after clear", q.Len())
	}
}

func TestQueuePanel_Remove(t *testing.T) {
	q := NewQueuePanel()
	q.Update(QueueAddMsg{"a"})
	q.Update(QueueAddMsg{"b"})
	q.Update(QueueAddMsg{"c"})
	q.Update(QueueRemoveMsg{Index: 1}) // remove "b"

	if q.Len() != 2 {
		t.Fatalf("len = %d", q.Len())
	}
	if q.Items()[0] != "a" || q.Items()[1] != "c" {
		t.Fatalf("items = %v", q.Items())
	}
}

func TestQueuePanel_DrainEmpty(t *testing.T) {
	q := NewQueuePanel()
	q.Update(QueueDrainMsg{}) // no panic
	if q.Len() != 0 {
		t.Fatal("should still be empty")
	}
}
