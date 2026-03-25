package sandbox

import (
	"context"
	"errors"
	"testing"
)

// mockSandbox implements Sandbox for testing.
type mockSandbox struct {
	name string
}

func (m *mockSandbox) Create(_ context.Context, _ string, _ []string) (Handle, error) {
	return Handle("mock-123"), nil
}
func (m *mockSandbox) Destroy(_ context.Context, _ Handle) error { return nil }
func (m *mockSandbox) Exec(_ context.Context, _ Handle, cmd []string, _ int64) (ExecResult, error) {
	return ExecResult{Stdout: "mock output"}, nil
}
func (m *mockSandbox) Name() string { return m.name }

func TestRegisterAndGet(t *testing.T) {
	// Clean up global state after test.
	defer func() {
		mu.Lock()
		delete(backends, "test-backend")
		mu.Unlock()
	}()

	Register("test-backend", func() (Sandbox, error) {
		return &mockSandbox{name: "test-backend"}, nil
	})

	sb, err := Get("test-backend")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if sb.Name() != "test-backend" {
		t.Fatalf("Name = %q, want test-backend", sb.Name())
	}
}

func TestGet_NotFound(t *testing.T) {
	_, err := Get("nonexistent-backend")
	if err == nil {
		t.Fatal("should return error for unknown backend")
	}
	if !errors.Is(err, ErrBackendNotFound) {
		t.Fatalf("err = %v, want ErrBackendNotFound", err)
	}
}

func TestGet_FactoryError(t *testing.T) {
	defer func() {
		mu.Lock()
		delete(backends, "broken")
		mu.Unlock()
	}()

	Register("broken", func() (Sandbox, error) {
		return nil, ErrBackendFailed
	})

	_, err := Get("broken")
	if err == nil {
		t.Fatal("should return factory error")
	}
	if !errors.Is(err, ErrBackendFailed) {
		t.Fatalf("err = %v, want ErrBackendFailed", err)
	}
}

func TestAvailable(t *testing.T) {
	defer func() {
		mu.Lock()
		delete(backends, "avail-a")
		delete(backends, "avail-b")
		mu.Unlock()
	}()

	Register("avail-a", func() (Sandbox, error) { return &mockSandbox{}, nil })
	Register("avail-b", func() (Sandbox, error) { return &mockSandbox{}, nil })

	names := Available()
	found := map[string]bool{}
	for _, n := range names {
		found[n] = true
	}
	if !found["avail-a"] || !found["avail-b"] {
		t.Fatalf("Available = %v, missing registered backends", names)
	}
}

func TestHandle_StringConversion(t *testing.T) {
	h := Handle("sandbox-456")
	if string(h) != "sandbox-456" {
		t.Fatalf("Handle = %q", h)
	}
}

func TestLevelConstants(t *testing.T) {
	levels := []string{LevelNone, LevelNamespace, LevelContainer, LevelKata}
	for _, l := range levels {
		if l == "" {
			t.Fatal("level constant should not be empty")
		}
	}
}

func TestMockSandbox_Create(t *testing.T) {
	sb := &mockSandbox{name: "mock"}
	h, err := sb.Create(context.Background(), LevelContainer, []string{"/workspace"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if h == "" {
		t.Fatal("handle should not be empty")
	}
}

func TestMockSandbox_Destroy(t *testing.T) {
	sb := &mockSandbox{name: "mock"}
	if err := sb.Destroy(context.Background(), Handle("test")); err != nil {
		t.Fatalf("Destroy: %v", err)
	}
}
