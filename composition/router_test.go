package composition

import "testing"

func TestFlatStrategy_RoundRobin(t *testing.T) {
	s := &FlatStrategy{}
	backends := []Backend{
		{ID: "b1", Role: "executor"},
		{ID: "b2", Role: "executor"},
		{ID: "b3", Role: "executor"},
	}

	b1, _ := s.Route(WorkUnit{ID: "u1"}, backends)
	b2, _ := s.Route(WorkUnit{ID: "u2"}, backends)
	b3, _ := s.Route(WorkUnit{ID: "u3"}, backends)
	b4, _ := s.Route(WorkUnit{ID: "u4"}, backends)

	if b1.ID != "b1" || b2.ID != "b2" || b3.ID != "b3" || b4.ID != "b1" {
		t.Fatalf("round-robin: %s %s %s %s", b1.ID, b2.ID, b3.ID, b4.ID)
	}
}

func TestFlatStrategy_NoBackends(t *testing.T) {
	s := &FlatStrategy{}
	_, err := s.Route(WorkUnit{ID: "u1"}, nil)
	if err == nil {
		t.Fatal("expected error with no backends")
	}
}

func TestChainStrategy_AlwaysFirst(t *testing.T) {
	s := &ChainStrategy{}
	backends := []Backend{
		{ID: "primary", Role: "executor"},
		{ID: "secondary", Role: "executor"},
	}

	b1, _ := s.Route(WorkUnit{ID: "u1"}, backends)
	b2, _ := s.Route(WorkUnit{ID: "u2"}, backends)

	if b1.ID != "primary" || b2.ID != "primary" {
		t.Fatalf("chain should always route to first: %s %s", b1.ID, b2.ID)
	}
}

// --- Test skeletons: future implementation ---

func TestHubAndSpoke_SpawnSibling(t *testing.T) {
	t.Skip("not implemented — requires Misbah + Clutch integration")
}
func TestHubAndSpoke_BrokerDispatch(t *testing.T) {
	t.Skip("not implemented — requires Broker + multi-backend")
}
func TestRoutingStrategy_Affinity(t *testing.T) {
	t.Skip("not implemented — requires Bugle identity system")
}
