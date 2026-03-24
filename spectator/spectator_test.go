package spectator

import "testing"

func TestReadOnlySpectator_InterfaceSatisfaction(t *testing.T) {
	var _ Spectator = (*ReadOnlySpectator)(nil)
}

func TestReadOnlySpectator_AttachReturnsNotImplemented(t *testing.T) {
	s := &ReadOnlySpectator{}
	err := s.Attach(nil)
	if err == nil {
		t.Fatal("stub should return not-implemented error")
	}
}

func TestReadOnlySpectator_DetachIsIdempotent(t *testing.T) {
	s := &ReadOnlySpectator{}
	if err := s.Detach(); err != nil {
		t.Fatal(err)
	}
	if err := s.Detach(); err != nil {
		t.Fatal("double detach should not error")
	}
}

// --- Test skeletons: future implementation ---

func TestSpectator_ReadOnlyObservation(t *testing.T) { t.Skip("not implemented — requires multi-client socket") }
func TestSpectator_Intervention(t *testing.T)         { t.Skip("not implemented — requires command channel") }
func TestSpectator_PauseResume(t *testing.T)          { t.Skip("not implemented") }
func TestIsolationControl_Mount(t *testing.T)         { t.Skip("not implemented — requires sandbox runtime") }
func TestIsolationControl_Network(t *testing.T)       { t.Skip("not implemented") }
func TestIsolationControl_Resources(t *testing.T)     { t.Skip("not implemented") }
