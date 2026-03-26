package staff

import (
	"sync"
	"testing"
)

func TestRoundTripRegistry_Register(t *testing.T) {
	r := NewRoundTripRegistry()

	id1 := r.Register("hello")
	id2 := r.Register("world")

	if id1 == id2 {
		t.Fatal("IDs should be unique")
	}
	if id1 != "msg-1" {
		t.Fatalf("first ID = %q, want msg-1", id1)
	}
	if id2 != "msg-2" {
		t.Fatalf("second ID = %q, want msg-2", id2)
	}
}

func TestRoundTripRegistry_Pending(t *testing.T) {
	r := NewRoundTripRegistry()
	r.Register("first")
	r.Register("second")
	r.Register("third")

	if c := r.PendingCount(); c != 3 {
		t.Fatalf("PendingCount = %d, want 3", c)
	}

	pending := r.Pending()
	if len(pending) != 3 {
		t.Fatalf("Pending = %d, want 3", len(pending))
	}
}

func TestRoundTripRegistry_AcknowledgeAndRespond(t *testing.T) {
	r := NewRoundTripRegistry()
	id := r.Register("test message")

	// Before acknowledge.
	pending := r.Pending()
	if len(pending) != 1 {
		t.Fatal("should have 1 pending")
	}
	if pending[0].Acknowledged {
		t.Fatal("should not be acknowledged yet")
	}

	// Acknowledge.
	r.Acknowledge(id)
	pending = r.Pending()
	if len(pending) != 1 {
		t.Fatal("acknowledge should not remove from pending")
	}
	if !pending[0].Acknowledged {
		t.Fatal("should be acknowledged")
	}

	// Respond.
	r.Respond(id)
	if c := r.PendingCount(); c != 0 {
		t.Fatalf("PendingCount after respond = %d, want 0", c)
	}
}

func TestRoundTripRegistry_RespondRemovesFromPending(t *testing.T) {
	r := NewRoundTripRegistry()
	id1 := r.Register("first")
	r.Register("second")

	r.Respond(id1)
	if c := r.PendingCount(); c != 1 {
		t.Fatalf("PendingCount = %d, want 1", c)
	}

	pending := r.Pending()
	if len(pending) != 1 || pending[0].Content != "second" {
		t.Fatalf("should only have 'second' pending, got %v", pending)
	}
}

func TestRoundTripRegistry_AcknowledgeUnknownID(t *testing.T) {
	r := NewRoundTripRegistry()
	// Should not panic.
	r.Acknowledge("nonexistent")
	r.Respond("nonexistent")
}

func TestRoundTripRegistry_ConcurrentAccess(t *testing.T) {
	r := NewRoundTripRegistry()
	var wg sync.WaitGroup

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			id := r.Register("concurrent message")
			r.Acknowledge(id)
			_ = r.PendingCount()
			_ = r.Pending()
			r.Respond(id)
		}()
	}
	wg.Wait()

	if c := r.PendingCount(); c != 0 {
		t.Fatalf("PendingCount after all responded = %d, want 0", c)
	}
}

func TestRoundTripRegistry_PendingReturnsCopies(t *testing.T) {
	r := NewRoundTripRegistry()
	r.Register("test")

	pending1 := r.Pending()
	pending1[0].Content = "mutated"

	pending2 := r.Pending()
	if pending2[0].Content == "mutated" {
		t.Fatal("Pending should return copies, not references")
	}
}
