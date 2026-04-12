package cache

import "testing"

// contractTest runs the same suite against any Cache implementation (Liskov).
func contractTest(t *testing.T, c Cache) {
	t.Helper()

	// Put + Get roundtrip.
	c.Put("agent-1", "main.go", []byte("package main"))
	got, ok := c.Get("agent-1", "main.go")
	if !ok {
		t.Fatal("expected hit after put")
	}
	if string(got) != "package main" {
		t.Fatalf("got %q, want 'package main'", got)
	}

	// Scope isolation.
	_, ok = c.Get("agent-2", "main.go")
	if ok {
		t.Fatal("agent-2 should not see agent-1's entry")
	}

	// Overwrite.
	c.Put("agent-1", "main.go", []byte("updated"))
	got, _ = c.Get("agent-1", "main.go")
	if string(got) != "updated" {
		t.Fatalf("got %q after overwrite, want 'updated'", got)
	}

	// Keys.
	c.Put("agent-1", "go.mod", []byte("module"))
	keys := c.Keys("agent-1")
	if len(keys) != 2 {
		t.Fatalf("keys = %d, want 2", len(keys))
	}

	// Empty scope keys.
	keys = c.Keys("agent-99")
	if len(keys) != 0 {
		t.Fatalf("empty scope keys = %d, want 0", len(keys))
	}

	// Evict.
	c.Evict("agent-1", "main.go")
	_, ok = c.Get("agent-1", "main.go")
	if ok {
		t.Fatal("expected miss after evict")
	}

	// EvictScope.
	c.Put("agent-1", "a.go", []byte("a"))
	c.Put("agent-1", "b.go", []byte("b"))
	c.EvictScope("agent-1")
	if len(c.Keys("agent-1")) != 0 {
		t.Fatal("expected empty after evict scope")
	}

	// Scopes.
	c.Put("x", "f1", []byte("1"))
	c.Put("y", "f2", []byte("2"))
	scopes := c.Scopes()
	if len(scopes) < 2 {
		t.Fatalf("scopes = %d, want >= 2", len(scopes))
	}

	// Len.
	if c.Len() < 2 {
		t.Fatalf("len = %d, want >= 2", c.Len())
	}
}

func TestMemCache_Contract(t *testing.T) {
	contractTest(t, NewMemCache())
}

func TestWriteThrough_BasicOps(t *testing.T) {
	l1 := NewMemCache()
	l2 := NewMemCache()
	wt := NewWriteThrough(l1, l2)

	// Put + Get.
	wt.Put("agent-1", "main.go", []byte("package main"))
	got, ok := wt.Get("agent-1", "main.go")
	if !ok || string(got) != "package main" {
		t.Fatalf("get = %q, %v", got, ok)
	}

	// Scope isolation.
	_, ok = wt.Get("agent-2", "main.go")
	if ok {
		t.Fatal("agent-2 should not see agent-1's entry")
	}

	// Keys.
	wt.Put("agent-1", "go.mod", []byte("module"))
	keys := wt.Keys("agent-1")
	if len(keys) != 2 {
		t.Fatalf("keys = %d, want 2", len(keys))
	}

	// Note: Evict on WriteThrough only evicts L1. Get still hits L2.
	// This is by design — L2 retains for recovery.
}

func TestWriteThrough_L2Promotion(t *testing.T) {
	l1 := NewMemCache()
	l2 := NewMemCache()
	wt := NewWriteThrough(l1, l2)

	// Write through — both L1 and L2 have it.
	wt.Put("agent-1", "main.go", []byte("data"))
	if _, ok := l1.Get("agent-1", "main.go"); !ok {
		t.Fatal("L1 should have entry after put")
	}
	if _, ok := l2.Get("agent-1", "main.go"); !ok {
		t.Fatal("L2 should have entry after put")
	}

	// Evict from L1 — L2 retains.
	wt.Evict("agent-1", "main.go")
	if _, ok := l1.Get("agent-1", "main.go"); ok {
		t.Fatal("L1 should NOT have entry after evict")
	}
	if _, ok := l2.Get("agent-1", "main.go"); !ok {
		t.Fatal("L2 should STILL have entry after L1 evict")
	}

	// Get promotes from L2 back to L1.
	got, ok := wt.Get("agent-1", "main.go")
	if !ok {
		t.Fatal("expected L2 fallback hit")
	}
	if string(got) != "data" {
		t.Fatalf("got %q, want 'data'", got)
	}
	// L1 should now have it again (promoted).
	if _, ok := l1.Get("agent-1", "main.go"); !ok {
		t.Fatal("L1 should have entry after L2 promotion")
	}
}

func TestWriteThrough_AgentRecovery(t *testing.T) {
	l1Old := NewMemCache()
	l2 := NewMemCache()
	wtOld := NewWriteThrough(l1Old, l2)

	// Agent writes 3 files.
	wtOld.Put("coder-1", "main.go", []byte("package main"))
	wtOld.Put("coder-1", "go.mod", []byte("module djinn"))
	wtOld.Put("coder-1", "agent/loop.go", []byte("package agent"))

	// Agent dies — L1 is gone.
	l1Old = nil
	_ = l1Old

	// New agent spawns — fresh L1, same L2.
	l1New := NewMemCache()
	wtNew := NewWriteThrough(l1New, l2)

	// Pre-warm: L2 has coder-1's entries. New agent can read them.
	for _, key := range l2.Keys("coder-1") {
		data, ok := wtNew.Get("coder-1", key)
		if !ok {
			t.Fatalf("expected L2 recovery for %s", key)
		}
		if len(data) == 0 {
			t.Fatalf("expected non-empty data for %s", key)
		}
	}

	// Verify L1 was pre-warmed via promotion.
	if l1New.Len() != 3 {
		t.Fatalf("new L1 should have 3 entries after recovery, got %d", l1New.Len())
	}
}
