package staff

import (
	"sync"
	"testing"
)

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

// --- GOL-16 RoleMemoryStore tests ---

func TestRoleMemoryStore_AppendAndGet(t *testing.T) {
	m := NewRoleMemory()
	m.Append("executor", Entry{Role: "assistant", Content: "first task"})
	m.Append("executor", Entry{Role: "assistant", Content: "second task"})

	got := m.GetSlice("executor")
	if len(got) != 2 {
		t.Fatalf("GetSlice len = %d, want 2", len(got))
	}
	if got[0].Content != "first task" {
		t.Fatalf("got[0].Content = %q, want %q", got[0].Content, "first task")
	}
	if got[1].Content != "second task" {
		t.Fatalf("got[1].Content = %q, want %q", got[1].Content, "second task")
	}

	// GetSlice returns a copy — mutating it must not affect the store.
	got[0].Content = "mutated"
	fresh := m.GetSlice("executor")
	if fresh[0].Content != "first task" {
		t.Fatal("GetSlice must return a copy, not a reference")
	}
}

func TestRoleMemoryStore_IsolatedSlices(t *testing.T) {
	m := NewRoleMemory()
	m.Append("executor", Entry{Role: "assistant", Content: "exec work A"})
	m.Append("executor", Entry{Role: "assistant", Content: "exec work B"})
	m.Append("inspector", Entry{Role: "assistant", Content: "inspect result"})

	execSlice := m.GetSlice("executor")
	inspSlice := m.GetSlice("inspector")

	if len(execSlice) != 2 {
		t.Fatalf("executor slice len = %d, want 2", len(execSlice))
	}
	if len(inspSlice) != 1 {
		t.Fatalf("inspector slice len = %d, want 1", len(inspSlice))
	}
	if inspSlice[0].Content != "inspect result" {
		t.Fatalf("inspector content = %q", inspSlice[0].Content)
	}
}

func TestRoleMemoryStore_SharedBriefing(t *testing.T) {
	m := NewRoleMemory()
	m.AppendBriefing(Entry{Content: "mission brief"})

	// Briefing visible regardless of role.
	b := m.Briefing()
	if len(b) != 1 || b[0].Content != "mission brief" {
		t.Fatalf("briefing = %v", b)
	}

	// Context for any role includes briefing.
	ctxExec := m.Context("executor")
	ctxInsp := m.Context("inspector")
	if len(ctxExec) < 1 || ctxExec[0].Content != "mission brief" {
		t.Fatal("executor context should include briefing")
	}
	if len(ctxInsp) < 1 || ctxInsp[0].Content != "mission brief" {
		t.Fatal("inspector context should include briefing")
	}
}

func TestRoleMemoryStore_SeedSlice(t *testing.T) {
	m := NewRoleMemory()
	seed := []Entry{
		{Role: "assistant", Content: "bootstrap 1"},
		{Role: "assistant", Content: "bootstrap 2"},
	}
	m.SeedSlice("manager", seed)

	got := m.GetSlice("manager")
	if len(got) != 2 {
		t.Fatalf("seeded slice len = %d, want 2", len(got))
	}
	if got[0].Content != "bootstrap 1" || got[1].Content != "bootstrap 2" {
		t.Fatalf("seeded content mismatch: %v", got)
	}
	if got[0].Speaker != "manager" {
		t.Fatalf("speaker = %q, want manager", got[0].Speaker)
	}
	if got[0].ID == "" {
		t.Fatal("seeded entry should have an assigned ID")
	}

	// Seeded entries are findable by Get.
	e, ok := m.Get(got[0].ID)
	if !ok {
		t.Fatal("should find seeded entry by ID")
	}
	if e.Content != "bootstrap 1" {
		t.Fatalf("Get content = %q", e.Content)
	}

	// SeedSlice replaces existing entries.
	m.SeedSlice("manager", []Entry{{Role: "user", Content: "replaced"}})
	got2 := m.GetSlice("manager")
	if len(got2) != 1 || got2[0].Content != "replaced" {
		t.Fatalf("re-seed failed: %v", got2)
	}
}

func TestRoleMemoryStore_Clear(t *testing.T) {
	m := NewRoleMemory()
	m.Append("executor", Entry{Content: "exec data"})
	m.Append("auditor", Entry{Content: "audit data"})

	m.Clear("executor")

	if m.SliceLen("executor") != 0 {
		t.Fatal("executor slice should be empty after Clear")
	}
	if m.SliceLen("auditor") != 1 {
		t.Fatal("auditor slice should be unaffected by clearing executor")
	}

	// Cleared entries should not be findable by Get.
	results := m.Search("exec data")
	if len(results) != 0 {
		t.Fatal("cleared entries should not appear in Search")
	}
}

func TestRoleMemoryStore_ConcurrentAccess(t *testing.T) {
	m := NewRoleMemory()
	roles := []string{"executor", "inspector", "auditor", "manager"}
	const perRole = 100

	var wg sync.WaitGroup
	for _, role := range roles {
		wg.Add(1)
		go func(r string) {
			defer wg.Done()
			for i := 0; i < perRole; i++ {
				m.Append(r, Entry{Content: r + " work"})
			}
		}(role)
	}
	wg.Wait()

	for _, role := range roles {
		if m.SliceLen(role) != perRole {
			t.Fatalf("%s slice len = %d, want %d", role, m.SliceLen(role), perRole)
		}
	}

	total := m.Len()
	if total != len(roles)*perRole {
		t.Fatalf("total len = %d, want %d", total, len(roles)*perRole)
	}
}
