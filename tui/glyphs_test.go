package tui

import "testing"

func TestDefaultGlyphs_AllFieldsSet(t *testing.T) {
	g := DefaultGlyphs()
	if g.ToolCall == "" {
		t.Fatal("ToolCall empty")
	}
	if g.ToolSuccess == "" {
		t.Fatal("ToolSuccess empty")
	}
	if g.ToolError == "" {
		t.Fatal("ToolError empty")
	}
	if g.UserPrompt == "" {
		t.Fatal("UserPrompt empty")
	}
	if g.ListCursor == "" {
		t.Fatal("ListCursor empty")
	}
	if g.Sandboxed == "" {
		t.Fatal("Sandboxed empty")
	}
}

func TestApplyGlyphs_UpdatesLabels(t *testing.T) {
	orig := ActiveGlyphs
	defer ApplyGlyphs(orig)

	custom := DefaultGlyphs()
	custom.ToolCall = "→"
	custom.UserPrompt = "$ "
	custom.AssistLabel = "ai"
	ApplyGlyphs(custom)

	if GlyphToolCall != "→" {
		t.Fatalf("GlyphToolCall = %q, want →", GlyphToolCall)
	}
	if LabelUser != "$ " {
		t.Fatalf("LabelUser = %q, want '$ '", LabelUser)
	}
	if LabelAssist != "ai" {
		t.Fatalf("LabelAssist = %q, want ai", LabelAssist)
	}
}

func TestApplyGlyphs_RestoresDefaults(t *testing.T) {
	orig := ActiveGlyphs
	custom := DefaultGlyphs()
	custom.ToolCall = "X"
	ApplyGlyphs(custom)
	ApplyGlyphs(orig)

	if GlyphToolCall != orig.ToolCall {
		t.Fatalf("not restored: %q", GlyphToolCall)
	}
}
