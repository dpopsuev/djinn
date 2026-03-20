package stubs

import (
	"context"
	"errors"
	"testing"

	"github.com/dpopsuev/djinn/driver"
)

func TestStubDriver_InterfaceSatisfaction(t *testing.T) {
	var _ driver.Driver = (*StubDriver)(nil)
}

func TestStubDriver_MessageDelivery(t *testing.T) {
	d := NewStubDriver(
		driver.Message{Role: "assistant", Content: "hello"},
		driver.Message{Role: "assistant", Content: "done"},
	)

	ctx := context.Background()
	if err := d.Start(ctx, "sandbox-1"); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if !d.Started() {
		t.Fatal("Started() = false after Start")
	}
	if d.Sandbox() != "sandbox-1" {
		t.Fatalf("Sandbox() = %q, want %q", d.Sandbox(), "sandbox-1")
	}

	if err := d.Send(ctx, driver.Message{Role: "user", Content: "hi"}); err != nil {
		t.Fatalf("Send: %v", err)
	}
	log := d.SendLog()
	if len(log) != 1 {
		t.Fatalf("SendLog() = %d, want 1", len(log))
	}

	ch := d.Recv(ctx)
	msg := <-ch
	if msg.Content != "hello" {
		t.Fatalf("first message = %q, want %q", msg.Content, "hello")
	}
	msg = <-ch
	if msg.Content != "done" {
		t.Fatalf("second message = %q, want %q", msg.Content, "done")
	}

	if err := d.Stop(ctx); err != nil {
		t.Fatalf("Stop: %v", err)
	}
}

func TestStubDriver_ErrorInjection(t *testing.T) {
	d := NewStubDriver()
	ctx := context.Background()

	injected := errors.New("injected")

	d.SetStartErr(injected)
	if err := d.Start(ctx, "s"); !errors.Is(err, injected) {
		t.Fatalf("Start err = %v, want injected", err)
	}

	d.SetStartErr(nil)
	d.Start(ctx, "s")

	d.SetSendErr(injected)
	if err := d.Send(ctx, driver.Message{}); !errors.Is(err, injected) {
		t.Fatalf("Send err = %v, want injected", err)
	}

	d.SetStopErr(injected)
	if err := d.Stop(ctx); !errors.Is(err, injected) {
		t.Fatalf("Stop err = %v, want injected", err)
	}
}
