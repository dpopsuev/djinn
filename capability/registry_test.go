package capability

import "testing"

type mockTestRunner struct{}

func (m *mockTestRunner) RunTests(_ interface{}, _ string) (TestResult, error) {
	return TestResult{Passed: 10}, nil
}

func TestRegistry_RegisterAndGet(t *testing.T) {
	r := NewRegistry()
	r.Register(PortTestRunner, &mockTestRunner{})

	p, ok := r.Get(PortTestRunner)
	if !ok {
		t.Fatal("expected port to be registered")
	}
	if p == nil {
		t.Fatal("port should not be nil")
	}
}

func TestRegistry_GetUnregistered(t *testing.T) {
	r := NewRegistry()
	_, ok := r.Get(PortMetricsSource)
	if ok {
		t.Fatal("unregistered port should return false")
	}
}

func TestRegistry_MustGet_Panics(t *testing.T) {
	r := NewRegistry()
	defer func() {
		if recover() == nil {
			t.Fatal("MustGet should panic for unregistered port")
		}
	}()
	r.MustGet(PortArtifactBuild)
}

func TestRegistry_Has(t *testing.T) {
	r := NewRegistry()
	if r.Has(PortSourceControl) {
		t.Fatal("should not have unregistered port")
	}
	r.Register(PortSourceControl, &mockTestRunner{})
	if !r.Has(PortSourceControl) {
		t.Fatal("should have registered port")
	}
}

func TestRegistry_Names(t *testing.T) {
	r := NewRegistry()
	r.Register(PortSourceControl, nil)
	r.Register(PortTestRunner, nil)

	names := r.Names()
	if len(names) != 2 {
		t.Fatalf("Names = %d, want 2", len(names))
	}
}

func TestPortNameConstants(t *testing.T) {
	ports := []string{
		PortSourceControl, PortTestRunner, PortStaticAnalysis,
		PortMetricsSource, PortArtifactBuild, PortContinuousDelivery,
		PortSecurityScanning,
	}
	seen := make(map[string]bool)
	for _, p := range ports {
		if p == "" {
			t.Fatal("port name constant is empty")
		}
		if seen[p] {
			t.Fatalf("duplicate port name: %q", p)
		}
		seen[p] = true
	}
}
