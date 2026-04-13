package substrate

import (
	"context"
	"errors"
	"testing"

	"github.com/dpopsuev/troupe/director"
	"github.com/dpopsuev/troupe/signal"

	"github.com/dpopsuev/djinn/agent"
	"github.com/dpopsuev/djinn/cortex"
	"github.com/dpopsuev/djinn/driver"
)

// TestUnifiedDirector_ImplementsDirectorInterface proves troupe/director.Director compliance.
func TestUnifiedDirector_ImplementsDirectorInterface(t *testing.T) {
	var _ director.Director = (*UnifiedDirector)(nil)

	dir := NewUnifiedDirector(&stubChatDriver{text: "ok"}, nil)

	ch, err := dir.Direct(context.Background(), nil)
	if err != nil {
		t.Fatalf("Direct: %v", err)
	}

	var events int
	for range ch {
		events++
	}
	if events == 0 {
		t.Fatal("expected at least one event from Direct()")
	}
}

// TestUnifiedDirector_Run proves the single-agent REPL path works.
func TestUnifiedDirector_Run(t *testing.T) {
	eventLog := signal.NewMemLog()

	dir := NewUnifiedDirector(&stubChatDriver{text: "hello from director"}, nil,
		WithSession(cortex.New("test", "test-model", t.TempDir())),
		WithMaxTurns(5),
		WithEventLogForDirector(eventLog),
	)

	output, err := dir.Run(context.Background(), "say hello", agent.ModeAuto, nil, &nopHandler{}, "gensec")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if output == "" {
		t.Fatal("expected non-empty output")
	}

	// Verify lifecycle events.
	events := eventLog.Since(-1)
	var started, done bool
	for _, e := range events {
		switch e.Kind {
		case "agent.run.start":
			started = true
		case "agent.run.done":
			done = true
		}
	}
	if !started {
		t.Error("expected agent.run.start event")
	}
	if !done {
		t.Error("expected agent.run.done event")
	}
}

// TestUnifiedDirector_ErrorEmitsErrorEvent proves error path emits agent.run.error.
func TestUnifiedDirector_ErrorEmitsErrorEvent(t *testing.T) {
	eventLog := signal.NewMemLog()

	dir := NewUnifiedDirector(&stubChatDriver{err: errors.New("model down")}, nil,
		WithSession(cortex.New("test", "test-model", t.TempDir())),
		WithEventLogForDirector(eventLog),
	)

	_, err := dir.Run(context.Background(), "will fail", agent.ModeAuto, nil, &nopHandler{}, "gensec")
	if err == nil {
		t.Fatal("expected error")
	}

	events := eventLog.Since(-1)
	var gotError bool
	for _, e := range events {
		if e.Kind == "agent.run.error" {
			gotError = true
		}
	}
	if !gotError {
		t.Error("expected agent.run.error event")
	}
}

// TestUnifiedDirector_WithScheduler proves role resolution works.
func TestUnifiedDirector_WithScheduler(t *testing.T) {
	dir := NewUnifiedDirector(&stubChatDriver{text: "ok"}, nil,
		WithSchedulerForDirector(DefaultScheduler()),
	)
	if dir.Scheduler() == nil {
		t.Fatal("Scheduler should not be nil")
	}
}

// --- Test doubles ---

// stubChatDriver is a minimal ChatDriver for testing. Returns canned text.
type stubChatDriver struct {
	text string
	err  error
}

func (s *stubChatDriver) Start(_ context.Context, _ string) error                { return nil }
func (s *stubChatDriver) Stop(_ context.Context) error                           { return nil }
func (s *stubChatDriver) SetSystemPrompt(_ string)                               {}
func (s *stubChatDriver) ContextWindow() int                                     { return 100000 }
func (s *stubChatDriver) Send(_ context.Context, _ driver.Message) error         { return nil }
func (s *stubChatDriver) SendRich(_ context.Context, _ driver.RichMessage) error { return nil }
func (s *stubChatDriver) AppendAssistant(_ driver.RichMessage)                   {}

func (s *stubChatDriver) Chat(_ context.Context) (<-chan driver.StreamEvent, error) {
	if s.err != nil {
		return nil, s.err
	}
	ch := make(chan driver.StreamEvent, 3) //nolint:mnd // test stub
	ch <- driver.StreamEvent{Type: driver.EventText, Text: s.text}
	ch <- driver.StreamEvent{Type: driver.EventDone, Usage: &driver.Usage{InputTokens: 10, OutputTokens: 5}}
	close(ch)
	return ch, nil
}

// nopHandler is a no-op EventHandler for testing.
type nopHandler struct{}

func (*nopHandler) OnText(_ string)                     {}
func (*nopHandler) OnThinking(_ string)                 {}
func (*nopHandler) OnToolCall(_ driver.ToolCall)        {}
func (*nopHandler) OnToolResult(_, _, _ string, _ bool) {}
func (*nopHandler) OnDone(_ *driver.Usage)              {}
func (*nopHandler) OnError(_ error)                     {}
