package staff

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/dpopsuev/djinn/driver"
)

// mockChatDriver is a minimal ChatDriver for pool tests.
type mockChatDriver struct {
	started atomic.Bool
	stopped atomic.Bool
	model   string
}

func (d *mockChatDriver) Start(_ context.Context, _ driver.SandboxHandle) error {
	d.started.Store(true)
	return nil
}

func (d *mockChatDriver) Stop(_ context.Context) error {
	d.stopped.Store(true)
	return nil
}

func (d *mockChatDriver) Send(_ context.Context, _ driver.Message) error        { return nil }
func (d *mockChatDriver) SendRich(_ context.Context, _ driver.RichMessage) error { return nil }

func (d *mockChatDriver) Chat(_ context.Context) (<-chan driver.StreamEvent, error) {
	ch := make(chan driver.StreamEvent)
	close(ch)
	return ch, nil
}

func (d *mockChatDriver) AppendAssistant(_ driver.RichMessage)  {}
func (d *mockChatDriver) SetSystemPrompt(_ string)              {}
func (d *mockChatDriver) ContextWindow() int                    { return 100_000 }

var _ driver.ChatDriver = (*mockChatDriver)(nil)

func TestDriverPool_CreateAndReuse(t *testing.T) {
	var createCount atomic.Int32
	factory := func(driverName, model, prompt string) (driver.ChatDriver, error) {
		createCount.Add(1)
		return &mockChatDriver{model: model}, nil
	}

	pool := NewDriverPool(factory)
	ctx := context.Background()

	d1, err := pool.GetOrCreate(ctx, "claude", "opus-4", "system prompt")
	if err != nil {
		t.Fatalf("first GetOrCreate: %v", err)
	}

	d2, err := pool.GetOrCreate(ctx, "claude", "opus-4", "system prompt")
	if err != nil {
		t.Fatalf("second GetOrCreate: %v", err)
	}

	if d1 != d2 {
		t.Fatal("expected same driver instance on reuse")
	}
	if createCount.Load() != 1 {
		t.Fatalf("factory called %d times, want 1", createCount.Load())
	}
	if pool.Count() != 1 {
		t.Fatalf("pool count = %d, want 1", pool.Count())
	}
}

func TestDriverPool_DifferentModels(t *testing.T) {
	factory := func(driverName, model, prompt string) (driver.ChatDriver, error) {
		return &mockChatDriver{model: model}, nil
	}

	pool := NewDriverPool(factory)
	ctx := context.Background()

	d1, err := pool.GetOrCreate(ctx, "claude", "opus-4", "")
	if err != nil {
		t.Fatalf("opus: %v", err)
	}

	d2, err := pool.GetOrCreate(ctx, "claude", "sonnet-4", "")
	if err != nil {
		t.Fatalf("sonnet: %v", err)
	}

	if d1 == d2 {
		t.Fatal("different models should get different drivers")
	}
	if pool.Count() != 2 {
		t.Fatalf("pool count = %d, want 2", pool.Count())
	}
}

func TestDriverPool_Active(t *testing.T) {
	factory := func(driverName, model, prompt string) (driver.ChatDriver, error) {
		return &mockChatDriver{model: model}, nil
	}

	pool := NewDriverPool(factory)
	ctx := context.Background()

	if pool.Active() != nil {
		t.Fatal("expected nil active before any creation")
	}

	d1, _ := pool.GetOrCreate(ctx, "claude", "opus-4", "")
	if pool.Active() != d1 {
		t.Fatal("active should be opus after first create")
	}

	d2, _ := pool.GetOrCreate(ctx, "claude", "sonnet-4", "")
	if pool.Active() != d2 {
		t.Fatal("active should be sonnet after second create")
	}

	// Re-access opus — should become active again.
	d1Again, _ := pool.GetOrCreate(ctx, "claude", "opus-4", "")
	if pool.Active() != d1Again {
		t.Fatal("active should be opus after re-access")
	}
}

func TestDriverPool_StopAll(t *testing.T) {
	var drivers []*mockChatDriver
	var mu sync.Mutex

	factory := func(driverName, model, prompt string) (driver.ChatDriver, error) {
		d := &mockChatDriver{model: model}
		mu.Lock()
		drivers = append(drivers, d)
		mu.Unlock()
		return d, nil
	}

	pool := NewDriverPool(factory)
	ctx := context.Background()

	pool.GetOrCreate(ctx, "claude", "opus-4", "")   //nolint:errcheck
	pool.GetOrCreate(ctx, "claude", "sonnet-4", "") //nolint:errcheck

	pool.StopAll(ctx)

	if pool.Count() != 0 {
		t.Fatalf("pool count after StopAll = %d, want 0", pool.Count())
	}
	if pool.Active() != nil {
		t.Fatal("active should be nil after StopAll")
	}
	for i, d := range drivers {
		if !d.stopped.Load() {
			t.Fatalf("driver[%d] not stopped", i)
		}
	}
}

func TestDriverPool_ConcurrentAccess(t *testing.T) {
	var createCount atomic.Int32
	factory := func(driverName, model, prompt string) (driver.ChatDriver, error) {
		createCount.Add(1)
		return &mockChatDriver{model: model}, nil
	}

	pool := NewDriverPool(factory)
	ctx := context.Background()

	const goroutines = 20
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			_, err := pool.GetOrCreate(ctx, "claude", "opus-4", "")
			if err != nil {
				t.Errorf("concurrent GetOrCreate: %v", err)
			}
		}()
	}

	wg.Wait()

	if pool.Count() != 1 {
		t.Fatalf("pool count = %d, want 1 (all goroutines same model)", pool.Count())
	}

	// Factory may be called more than once due to the double-check pattern,
	// but the pool should only contain one driver.
	if pool.Active() == nil {
		t.Fatal("expected non-nil active driver")
	}
}
