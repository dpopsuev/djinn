package staff

import "testing"

func TestRoleMemory_AppendAndHistory(t *testing.T) {
	m := NewRoleMemory()
	id := m.Append("executor", Entry{Role: "assistant", Content: "wrote code"})

	if id != "MSG-0001" {
		t.Fatalf("id = %q, want MSG-0001", id)
	}

	h := m.History("executor")
	if len(h) != 1 {
		t.Fatalf("history len = %d", len(h))
	}
	if h[0].Content != "wrote code" {
		t.Fatalf("content = %q", h[0].Content)
	}
	if h[0].Speaker != "executor" {
		t.Fatalf("speaker = %q", h[0].Speaker)
	}
}

func TestRoleMemory_Isolation(t *testing.T) {
	m := NewRoleMemory()
	m.Append("executor", Entry{Role: "assistant", Content: "debug output line 1"})
	m.Append("executor", Entry{Role: "assistant", Content: "debug output line 2"})
	m.Append("auditor", Entry{Role: "assistant", Content: "reviewed spec"})

	if len(m.History("executor")) != 2 {
		t.Fatal("executor should have 2 entries")
	}
	if len(m.History("auditor")) != 1 {
		t.Fatal("auditor should have 1 entry")
	}
	if len(m.History("gensec")) != 0 {
		t.Fatal("gensec should have 0 entries")
	}
}

func TestRoleMemory_Briefing(t *testing.T) {
	m := NewRoleMemory()
	id := m.AppendBriefing(Entry{Content: "need captured: NED-47"})

	if id != "BRF-0001" {
		t.Fatalf("id = %q, want BRF-0001", id)
	}

	b := m.Briefing()
	if len(b) != 1 || b[0].Content != "need captured: NED-47" {
		t.Fatalf("briefing = %v", b)
	}
}

func TestRoleMemory_Context_IncludesBriefing(t *testing.T) {
	m := NewRoleMemory()
	m.AppendBriefing(Entry{Content: "shared context"})
	m.Append("executor", Entry{Content: "executor work"})
	m.Append("auditor", Entry{Content: "auditor review"})

	ctx := m.Context("executor")
	if len(ctx) != 2 {
		t.Fatalf("context len = %d, want 2 (1 briefing + 1 executor)", len(ctx))
	}
	if ctx[0].Content != "shared context" {
		t.Fatal("briefing should come first in context")
	}
	if ctx[1].Content != "executor work" {
		t.Fatal("role history should follow briefing")
	}

	// Auditor should NOT see executor's entries
	audCtx := m.Context("auditor")
	if len(audCtx) != 2 {
		t.Fatalf("auditor context len = %d, want 2 (1 briefing + 1 auditor)", len(audCtx))
	}
}

func TestRoleMemory_Get(t *testing.T) {
	m := NewRoleMemory()
	id := m.Append("executor", Entry{Content: "some work"})

	e, ok := m.Get(id)
	if !ok {
		t.Fatal("should find entry by ID")
	}
	if e.Content != "some work" {
		t.Fatalf("content = %q", e.Content)
	}

	_, ok = m.Get("MSG-9999")
	if ok {
		t.Fatal("should not find non-existent ID")
	}
}

func TestRoleMemory_Search(t *testing.T) {
	m := NewRoleMemory()
	m.Append("executor", Entry{Content: "fixed the Bobby bug"})
	m.Append("auditor", Entry{Content: "reviewed Bobby fix"})
	m.Append("executor", Entry{Content: "unrelated work"})

	results := m.Search("bobby")
	if len(results) != 2 {
		t.Fatalf("search results = %d, want 2", len(results))
	}
}

func TestRoleMemory_Search_CaseInsensitive(t *testing.T) {
	m := NewRoleMemory()
	m.Append("gensec", Entry{Content: "Bobby the Dog is important"})

	results := m.Search("BOBBY")
	if len(results) != 1 {
		t.Fatalf("case-insensitive search failed: %d results", len(results))
	}
}

func TestRoleMemory_Len(t *testing.T) {
	m := NewRoleMemory()
	m.Append("a", Entry{Content: "1"})
	m.Append("b", Entry{Content: "2"})
	m.AppendBriefing(Entry{Content: "3"})

	if m.Len() != 3 {
		t.Fatalf("len = %d, want 3", m.Len())
	}
}
