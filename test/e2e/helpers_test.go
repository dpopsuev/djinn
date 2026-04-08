//go:build e2e

package e2e

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/dpopsuev/djinn/cli/repl"
)

func toModelPtr(m tea.Model) *repl.Model {
	switch v := m.(type) {
	case repl.Model:
		return &v
	case *repl.Model:
		return v
	default:
		panic("unexpected type")
	}
}

func multiUpdate(t *testing.T, m tea.Model, msgs ...tea.Msg) tea.Model {
	t.Helper()
	for i, msg := range msgs {
		m = safeUpdate(t, m, i, msg)
	}
	return m
}

func safeUpdate(t *testing.T, m tea.Model, step int, msg tea.Msg) (result tea.Model) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("PANIC at Update step %d (msg=%T): %v", step, msg, r)
		}
	}()
	result, _ = m.Update(msg)
	return result
}
