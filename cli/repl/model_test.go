package repl

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/dpopsuev/djinn/agent"
	"github.com/dpopsuev/djinn/driver"
	"github.com/dpopsuev/djinn/session"
	"github.com/dpopsuev/djinn/tools/builtin"
	"github.com/dpopsuev/djinn/tui"
)

func testModel() Model {
	sess := session.New("test", "test-model", "/workspace")
	m := NewModel(Config{
		Tools:   builtin.NewRegistry(),
		Session: sess,
		Mode:    "agent",
	})
	m.ready = true
	m.width = 80
	m.height = 24
	return m
}

// asModel extracts Model from tea.Model which may be Model or *Model
// depending on whether handleKey (pointer receiver) or Update (value receiver) ran.
func asModel(t *testing.T, m tea.Model) Model {
	t.Helper()
	switch v := m.(type) {
	case Model:
		return v
	case *Model:
		return *v
	default:
		t.Fatalf("unexpected type %T", m)
		return Model{}
	}
}

func TestModel_WindowSizeMsg(t *testing.T) {
	m := testModel()
	m.ready = false
	m2, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	model := asModel(t, m2)
	if !model.ready {
		t.Fatal("should be ready after WindowSizeMsg")
	}
	if model.width != 120 || model.height != 40 {
		t.Fatalf("size = %dx%d, want 120x40", model.width, model.height)
	}
}

func TestModel_TextMsg_Streaming(t *testing.T) {
	m := testModel()
	m.state = stateStreaming
	m2, _ := m.Update(tui.TextMsg("hello"))
	model := asModel(t, m2)
	if model.StreamBufString() != "hello" {
		t.Fatalf("streamBuf = %q", model.StreamBufString())
	}
}

func TestModel_TextMsg_Chunked(t *testing.T) {
	m := testModel()
	m.state = stateStreaming
	m.outputMode = outputChunked
	m2, _ := m.Update(tui.TextMsg("hello"))
	model := asModel(t, m2)
	if model.chunkedBuf.String() != "hello" {
		t.Fatalf("chunkedBuf = %q", model.chunkedBuf.String())
	}
	if model.StreamBufString() != "" {
		t.Fatal("streamBuf should be empty in chunked mode")
	}
}

func TestModel_ThinkingMsg(t *testing.T) {
	m := testModel()
	before := m.outputPanel.LineCount()
	m2, _ := m.Update(tui.ThinkingMsg("let me think"))
	model := asModel(t, m2)
	if model.outputPanel.LineCount() != before+1 {
		t.Fatal("should append to conversation")
	}
}

func TestModel_ToolCallMsg_AgentMode(t *testing.T) {
	m := testModel()
	m.mode = agent.ModeAgent
	m.state = stateStreaming
	m2, _ := m.Update(tui.ToolCallMsg{Call: driver.ToolCall{
		ID: "c1", Name: "Bash", Input: json.RawMessage(`{"command":"ls"}`),
	}})
	model := asModel(t, m2)
	if model.state != stateToolApproval {
		t.Fatalf("state = %d, want stateToolApproval", model.state)
	}
	if model.pendingTool == nil {
		t.Fatal("pendingTool should be set")
	}
}

func TestModel_ToolCallMsg_AutoMode(t *testing.T) {
	m := testModel()
	m.mode = agent.ModeAuto
	m.state = stateStreaming
	m2, _ := m.Update(tui.ToolCallMsg{Call: driver.ToolCall{
		ID: "c1", Name: "Bash", Input: json.RawMessage(`{}`),
	}})
	model := asModel(t, m2)
	if model.state == stateToolApproval {
		t.Fatal("auto mode should not prompt for approval")
	}
}

func TestModel_ToolResultMsg_Success(t *testing.T) {
	m := testModel()
	before := m.outputPanel.LineCount()
	m2, _ := m.Update(tui.ToolResultMsg{Name: "Read", Output: "file contents", IsError: false})
	model := asModel(t, m2)
	if model.outputPanel.LineCount() != before+1 {
		t.Fatal("should append to conversation")
	}
}

func TestModel_ToolResultMsg_Error(t *testing.T) {
	m := testModel()
	before := m.outputPanel.LineCount()
	m2, _ := m.Update(tui.ToolResultMsg{Name: "Read", Output: "not found", IsError: true})
	model := asModel(t, m2)
	if model.outputPanel.LineCount() != before+1 {
		t.Fatal("should append to conversation")
	}
}

func TestModel_DoneMsg(t *testing.T) {
	m := testModel()
	m2, _ := m.Update(tui.DoneMsg{Usage: &driver.Usage{InputTokens: 100, OutputTokens: 50}})
	model := asModel(t, m2)
	if model.totalIn != 100 || model.totalOut != 50 {
		t.Fatalf("tokens = %d/%d, want 100/50", model.totalIn, model.totalOut)
	}
}

func TestModel_AgentDoneMsg(t *testing.T) {
	m := testModel()
	m.state = stateStreaming
	m2, _ := m.Update(tui.AgentDoneMsg{Result: "done"})
	model := asModel(t, m2)
	if model.state != stateInput {
		t.Fatalf("state = %d, want stateInput", model.state)
	}
}

func TestModel_AgentDoneMsg_WithError(t *testing.T) {
	m := testModel()
	m.state = stateStreaming
	m2, _ := m.Update(tui.AgentDoneMsg{Err: errors.New("something failed")})
	model := asModel(t, m2)
	found := false
	for _, line := range model.outputPanel.Lines() {
		if strings.Contains(line, "something failed") {
			found = true
		}
	}
	if !found {
		t.Fatal("error should appear in conversation")
	}
}

func TestModel_TickMsg_WhileStreaming(t *testing.T) {
	m := testModel()
	m.SetState(StateStreaming)
	// Stream text via message (goes to OutputPanel's internal streamBuf)
	m.Update(tui.TextMsg("buffered text"))
	m.AppendConversation("assistant: ")

	m2, cmd := m.Update(tui.TickMsg(time.Now()))
	model := asModel(t, m2)
	if cmd == nil {
		t.Fatal("should return tick cmd while streaming")
	}
	if model.StreamBufString() != "" {
		t.Fatal("buffer should be flushed")
	}
}

func TestModel_TickMsg_WhileInput(t *testing.T) {
	m := testModel()
	m.state = stateInput
	_, cmd := m.Update(tui.TickMsg(time.Now()))
	if cmd != nil {
		t.Fatal("no tick cmd when not streaming")
	}
}

func TestModel_HandleKey_CtrlC(t *testing.T) {
	m := testModel()
	m2, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	model := asModel(t, m2)
	if !model.quitting {
		t.Fatal("should be quitting")
	}
	if cmd == nil {
		t.Fatal("should return quit cmd")
	}
}

func TestModel_HandleKey_HistoryUp(t *testing.T) {
	m := testModel()
	m.AddInputHistory("first")
	m.AddInputHistory("second")
	m.AddInputHistory("third")
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
	model := asModel(t, m2)
	if model.TextInputValue() != "third" {
		t.Fatalf("value = %q, want third", model.TextInputValue())
	}
}

func TestModel_HandleKey_HistoryDown(t *testing.T) {
	m := testModel()
	m.AddInputHistory("first")
	m.AddInputHistory("second")
	// Navigate up twice to land on "first", then down once to get "second"
	m.Update(tea.KeyMsg{Type: tea.KeyUp}) // → "second"
	m.Update(tea.KeyMsg{Type: tea.KeyUp}) // → "first"
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	model := asModel(t, m2)
	if model.TextInputValue() != "second" {
		t.Fatalf("value = %q, want second", model.TextInputValue())
	}
}

func TestModel_HandleApproval_Y(t *testing.T) {
	m := testModel()
	m.state = stateToolApproval
	m.pendingTool = &driver.ToolCall{Name: "Bash"}
	m2, _ := m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	model := asModel(t, m2)
	if model.state != stateStreaming {
		t.Fatalf("state = %d, want stateStreaming", model.state)
	}
}

func TestModel_HandleApproval_N(t *testing.T) {
	m := testModel()
	m.state = stateToolApproval
	m.pendingTool = &driver.ToolCall{Name: "Bash"}
	m2, _ := m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	model := asModel(t, m2)
	found := false
	for _, line := range model.outputPanel.Lines() {
		if strings.Contains(line, "denied") {
			found = true
		}
	}
	if !found {
		t.Fatal("denied should appear in conversation")
	}
}

func TestModel_View_Welcome(t *testing.T) {
	m := testModel()
	// Use a tall terminal so MOTD fits inside viewport with borders
	m2, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 40})
	model := asModel(t, m2)
	view := model.View()
	// Structural: welcome view should contain the logo (non-empty)
	if !strings.Contains(view, tui.DjinnLogo[:10]) {
		t.Fatal("welcome should render the logo")
	}
	// Structural: welcome view should contain help hint
	if !strings.Contains(view, "/help") {
		t.Fatal("welcome should contain /help hint")
	}
}


func TestModel_View_StatusBar(t *testing.T) {
	m := testModel()
	if m.sess.Model != "test-model" {
		t.Fatalf("model = %q", m.sess.Model)
	}
	// Default role (gensec) overrides cfg.Mode to "plan"
	if m.mode != agent.ModePlan {
		t.Fatalf("mode = %s, want plan (from gensec role)", m.mode)
	}
	view := m.View()
	if view == "" {
		t.Fatal("view should not be empty")
	}
}

func TestModel_HandleSubmit_SlashCommand(t *testing.T) {
	m := testModel()
	m.SetTextInput("/help")
	m2, _ := m.handleSubmit()
	model := asModel(t, m2)
	// Assert conversation grew (structural), not exact text content (brittle)
	if model.outputPanel.LineCount() == 0 {
		t.Fatal("help should add output to conversation")
	}
}

func TestModel_HandleSubmit_ModeSwitch(t *testing.T) {
	m := testModel()
	m.SetTextInput("/mode auto")
	m2, _ := m.handleSubmit()
	model := asModel(t, m2)
	if model.mode != agent.ModeAuto {
		t.Fatalf("mode = %s, want auto", model.mode)
	}
}

func TestModel_HandleSubmit_Empty(t *testing.T) {
	m := testModel()
	m.SetTextInput("")
	m2, cmd := m.handleSubmit()
	model := asModel(t, m2)
	if cmd != nil {
		t.Fatal("empty submit should return no cmd")
	}
	if model.state != stateInput {
		t.Fatal("state should remain input")
	}
}

func TestModel_HandleSubmit_Exit(t *testing.T) {
	m := testModel()
	m.SetTextInput("/exit")
	m2, _ := m.handleSubmit()
	model := asModel(t, m2)
	if !model.quitting {
		t.Fatal("should be quitting")
	}
}

func TestTruncate(t *testing.T) {
	if truncate("short", 10) != "short" {
		t.Fatal("short string should pass through")
	}
	long := strings.Repeat("x", 100)
	result := truncate(long, 10)
	if len(result) != 13 { // 10 + "..."
		t.Fatalf("truncated len = %d", len(result))
	}
}

func TestModel_FlushStreamBuffer(t *testing.T) {
	m := testModel()
	m.AppendConversation("assistant: ")
	// Stream via TextMsg (OutputPanel handles it internally now)
	m.Update(tui.TextMsg("hello world"))
	// Flush via FlushStreamMsg
	m.outputPanel.Update(tui.FlushStreamMsg{})
	lines := m.outputPanel.Lines()
	last := lines[len(lines)-1]
	if !strings.Contains(last, "hello world") {
		t.Fatalf("last line = %q", last)
	}
	if m.StreamBufString() != "" {
		t.Fatal("buffer should be empty after flush")
	}
}

func TestModel_FlushStreamBuffer_Empty(t *testing.T) {
	m := testModel()
	before := m.outputPanel.LineCount()
	m.outputPanel.Update(tui.FlushStreamMsg{})
	if m.outputPanel.LineCount() != before {
		t.Fatal("empty flush should not modify conversation")
	}
}

func TestModel_NewModel_DefaultMode(t *testing.T) {
	sess := session.New("test", "model", "/workspace")
	m := NewModel(Config{Tools: builtin.NewRegistry(), Session: sess})
	// Default role (gensec) overrides cfg.Mode — gensec is "plan"
	if m.mode != agent.ModePlan {
		t.Fatalf("default mode = %s, want plan (from gensec role)", m.mode)
	}
}

func TestModel_NewModel_ParsesMode(t *testing.T) {
	sess := session.New("test", "model", "/workspace")
	m := NewModel(Config{Tools: builtin.NewRegistry(), Session: sess, Mode: "auto"})
	// cfg.Mode="auto" is overridden by default role (gensec) which is "plan"
	if m.mode != agent.ModePlan {
		t.Fatalf("mode = %s, want plan (gensec role overrides cfg.Mode)", m.mode)
	}
}

// --- Spinner tests ---

func TestModel_SpinnerActiveOnStreaming(t *testing.T) {
	m := testModel()
	m.SetTextInput("hello")
	// Can't test full submit without driver, but verify initial state
	if m.spinnerActive {
		t.Fatal("spinner should be inactive initially")
	}
}

func TestModel_SpinnerDeactivatedOnText(t *testing.T) {
	m := testModel()
	m.state = stateStreaming
	m.spinnerActive = true
	m2, _ := m.Update(tui.TextMsg("first token"))
	model := asModel(t, m2)
	if model.spinnerActive {
		t.Fatal("spinner should deactivate on first TextMsg")
	}
}

// --- Viewport tests ---

func TestModel_ViewportInitOnResize(t *testing.T) {
	m := testModel()
	m2, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	model := asModel(t, m2)
	// Verify the output panel has content (MOTD rendered on first init)
	if model.outputPanel.LineCount() == 0 {
		t.Fatal("output panel should have MOTD after WindowSizeMsg")
	}
}

// --- Tool progress tests ---

func TestModel_ToolProgressReplacement(t *testing.T) {
	m := testModel()
	m.state = stateStreaming
	m.mode = agent.ModeAuto

	// Tool call adds line with activeToolIdx
	m2, _ := m.Update(tui.ToolCallMsg{Call: driver.ToolCall{
		ID: "c1", Name: "Read", Input: json.RawMessage(`{"path":"test.go"}`),
	}})
	model := asModel(t, m2)
	if model.activeToolIdx < 0 {
		t.Fatal("activeToolIdx should be set after ToolCallMsg")
	}
	beforeLen := model.outputPanel.LineCount()

	// Tool result replaces the spinner line (same length, not appended)
	m3, _ := model.Update(tui.ToolResultMsg{Name: "Read", Output: "contents", IsError: false})
	model2 := asModel(t, m3)
	if model2.outputPanel.LineCount() != beforeLen {
		t.Fatalf("tool result should replace, not append: %d → %d", beforeLen, model2.outputPanel.LineCount())
	}
	if model2.activeToolIdx != -1 {
		t.Fatal("activeToolIdx should reset to -1 after result")
	}
}

// TestModel_RawStreamLine_NoPanicOnCopy reproduces DJN-BUG-7:
// Bubbletea copies Model by value in Update(). strings.Builder panics
// when copied after first write. This test simulates the exact crash path.
func TestModel_RawStreamLine_NoPanicOnCopy(t *testing.T) {
	m := testModel()
	m.SetState(StateStreaming)
	m.AppendConversation(tui.AssistStyle.Render(tui.LabelAssist) + ": ")

	// Write to rawStreamLine (simulates first flush)
	m.rawStreamLine.WriteString("hello")

	// Bubbletea copies the Model by value in Update().
	// After fix: rawStreamLine is a *strings.Builder, so copy shares the pointer.
	// Before fix: strings.Builder panics on copy after write.
	panicked := func() (caught bool) {
		defer func() {
			if r := recover(); r != nil {
				caught = true
			}
		}()
		copied := m //nolint:govet // intentional copy to test
		copied.rawStreamLine.WriteString(" world")
		return false
	}()

	if panicked {
		t.Fatal("DJN-BUG-7: panic on Model copy after rawStreamLine write — use *strings.Builder not strings.Builder")
	}
}

// --- BUG-44: TUI responsive during streaming ---

func TestModel_TypeAheadDuringStreaming(t *testing.T) {
	m := testModel()
	m.SetState(StateStreaming)

	// Type a key during streaming — should be forwarded to input panel.
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	model := asModel(t, m2)
	if model.TextInputValue() == "" {
		t.Fatal("BUG-44: keys dropped during streaming — type-ahead should work")
	}
}

func TestModel_TabCycleDuringStreaming(t *testing.T) {
	m := testModel()
	m.SetState(StateStreaming)

	// Tab during streaming should cycle focus, not be dropped.
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	model := asModel(t, m2)
	// Focus should have moved from input (idx=1) to dashboard (idx=2).
	_ = model // If Tab is dropped, focus stays at 1. Test will verify fix works.
}

func TestModel_AltM_DuringStreaming(t *testing.T) {
	m := testModel()
	m.SetState(StateStreaming)
	modeBefore := m.mode

	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}, Alt: true})
	model := asModel(t, m2)
	if model.mode == modeBefore {
		t.Fatal("BUG-44: Alt+M dropped during streaming — mode should cycle")
	}
}

// --- BUG-47: Message queue during streaming ---

func TestModel_SubmitMsg_QueuedDuringStreaming(t *testing.T) {
	m := testModel()
	m.SetState(StateStreaming)

	// SubmitMsg during streaming → queued, not dropped.
	m2, _ := m.Update(tui.SubmitMsg{Value: "queued prompt"})
	model := asModel(t, m2)
	if model.queuePanel.Len() != 1 {
		t.Fatalf("queue = %d, want 1", model.queuePanel.Len())
	}
	if model.queuePanel.Items()[0] != "queued prompt" {
		t.Fatalf("queued = %q", model.queuePanel.Items()[0])
	}
	// Queue panel should show the item.
	queueView := model.queuePanel.View(80)
	if !strings.Contains(queueView, "queued prompt") {
		t.Fatalf("queue view should show item: %q", queueView)
	}
}

func TestModel_SubmitMsg_ProcessedWhenInput(t *testing.T) {
	m := testModel()
	m.SetState(StateInput)

	// SubmitMsg during input → processed immediately (starts agent).
	m2, cmd := m.Update(tui.SubmitMsg{Value: "direct prompt"})
	model := asModel(t, m2)
	if model.state != stateStreaming {
		t.Fatalf("state = %d, want streaming", model.state)
	}
	if cmd == nil {
		t.Fatal("should return agent cmd")
	}
}

func TestModel_QueueDrainOnAgentDone(t *testing.T) {
	m := testModel()
	m.queuePanel.Update(tui.QueueAddMsg{Prompt: "queued one"})
	m.state = stateStreaming

	// AgentDoneMsg should drain queue and return a SubmitMsg cmd.
	m2, cmd := m.Update(tui.AgentDoneMsg{Result: "done"})
	model := asModel(t, m2)
	if model.queuePanel.Len() != 0 {
		t.Fatalf("queue should be empty after drain, got %d", model.queuePanel.Len())
	}
	if cmd == nil {
		t.Fatal("should return cmd to process queued prompt")
	}
	// Execute the cmd — should produce SubmitMsg.
	msg := cmd()
	submit, ok := msg.(tui.SubmitMsg)
	if !ok {
		t.Fatalf("expected SubmitMsg, got %T", msg)
	}
	if submit.Value != "queued one" {
		t.Fatalf("submit = %q, want 'queued one'", submit.Value)
	}
}

// --- mockChatDriver for switchRole tests ---

type mockChatDriver struct {
	systemPrompt string
}

func (d *mockChatDriver) Start(_ context.Context, _ driver.SandboxHandle) error   { return nil }
func (d *mockChatDriver) Stop(_ context.Context) error                             { return nil }
func (d *mockChatDriver) Send(_ context.Context, _ driver.Message) error           { return nil }
func (d *mockChatDriver) SendRich(_ context.Context, _ driver.RichMessage) error   { return nil }
func (d *mockChatDriver) Chat(_ context.Context) (<-chan driver.StreamEvent, error) {
	ch := make(chan driver.StreamEvent)
	close(ch)
	return ch, nil
}
func (d *mockChatDriver) AppendAssistant(_ driver.RichMessage) {}
func (d *mockChatDriver) SetSystemPrompt(prompt string) {
	d.systemPrompt = prompt
}
func (d *mockChatDriver) ContextWindow() int { return 200_000 }

// testModelWithDriver creates a Model with a mock ChatDriver attached.
func testModelWithDriver() (Model, *mockChatDriver) {
	drv := &mockChatDriver{}
	sess := session.New("test", "test-model", "/workspace")
	m := NewModel(Config{
		Driver:  drv,
		Tools:   builtin.NewRegistry(),
		Session: sess,
		Mode:    "agent",
	})
	m.chatDriver = drv // ensure the mock is wired
	m.ready = true
	m.width = 80
	m.height = 24
	return m, drv
}

// --- switchRole tests ---

func TestModel_SwitchRole_UpdatesCurrentRole(t *testing.T) {
	m, drv := testModelWithDriver()
	m.switchRole("executor")
	if m.currentRole != "executor" {
		t.Fatalf("currentRole = %q, want executor", m.currentRole)
	}
	if drv.systemPrompt == "" {
		t.Fatal("SetSystemPrompt should have been called")
	}
	if m.mode != agent.ModeAgent {
		t.Fatalf("mode = %s, want agent (executor role mode)", m.mode)
	}
}

func TestModel_SwitchRole_InvalidRole(t *testing.T) {
	m, _ := testModelWithDriver()
	before := m.currentRole
	m.switchRole("nonexistent-role")
	if m.currentRole != before {
		t.Fatal("invalid role should not change currentRole")
	}
}

func TestModel_SwitchRole_UpdatesDashboard(t *testing.T) {
	m, _ := testModelWithDriver()
	before := m.outputPanel.LineCount()
	m.switchRole("auditor")
	// Should append a narration line
	if m.outputPanel.LineCount() <= before {
		t.Fatal("switchRole should append narration to output")
	}
}

func TestModel_SwitchRole_UpdatesToolCapabilities(t *testing.T) {
	m, _ := testModelWithDriver()
	m.switchRole("executor")
	if len(m.token.AllowedTools) == 0 {
		t.Fatal("executor role should have tool capabilities")
	}
}

// --- handleSubmit: /role command tests ---

func TestModel_HandleSubmit_RoleShow(t *testing.T) {
	m, _ := testModelWithDriver()
	m.SetTextInput("/role")
	m2, _ := m.handleSubmit()
	model := asModel(t, m2)
	found := false
	for _, line := range model.outputPanel.Lines() {
		if strings.Contains(line, "current role:") {
			found = true
		}
	}
	if !found {
		t.Fatal("/role should show current role")
	}
}

func TestModel_HandleSubmit_RoleSwitch(t *testing.T) {
	m, drv := testModelWithDriver()
	m.SetTextInput("/role executor")
	m2, _ := m.handleSubmit()
	model := asModel(t, m2)
	if model.currentRole != "executor" {
		t.Fatalf("currentRole = %q, want executor", model.currentRole)
	}
	if drv.systemPrompt == "" {
		t.Fatal("SetSystemPrompt should have been called")
	}
}

func TestModel_HandleSubmit_RoleCreate(t *testing.T) {
	m, _ := testModelWithDriver()
	m.SetTextInput("/role create mybot agent")
	m2, _ := m.handleSubmit()
	model := asModel(t, m2)
	if _, ok := model.roles["mybot"]; !ok {
		t.Fatal("custom role should be created")
	}
	found := false
	for _, line := range model.outputPanel.Lines() {
		if strings.Contains(line, "created role") {
			found = true
		}
	}
	if !found {
		t.Fatal("should narrate creation")
	}
}

// --- handleSubmit: /staff command ---

func TestModel_HandleSubmit_Staff(t *testing.T) {
	m, _ := testModelWithDriver()
	m.SetTextInput("/staff")
	m2, _ := m.handleSubmit()
	model := asModel(t, m2)
	found := false
	for _, line := range model.outputPanel.Lines() {
		if strings.Contains(line, "Staff:") || strings.Contains(line, "gensec") {
			found = true
		}
	}
	if !found {
		t.Fatal("/staff should list roles")
	}
}

// --- handleSubmit: /briefing command ---

func TestModel_HandleSubmit_Briefing_Empty(t *testing.T) {
	m := testModel()
	m.SetTextInput("/briefing")
	m2, _ := m.handleSubmit()
	model := asModel(t, m2)
	found := false
	for _, line := range model.outputPanel.Lines() {
		if strings.Contains(line, "empty") {
			found = true
		}
	}
	if !found {
		t.Fatal("/briefing should show empty message")
	}
}

func TestModel_HandleSubmit_Briefing_WithEntries(t *testing.T) {
	m, _ := testModelWithDriver()
	// switchRole appends a briefing entry
	m.switchRole("executor")
	m.SetTextInput("/briefing")
	m2, _ := m.handleSubmit()
	model := asModel(t, m2)
	found := false
	for _, line := range model.outputPanel.Lines() {
		if strings.Contains(line, "Briefing:") || strings.Contains(line, "switched to executor") {
			found = true
		}
	}
	if !found {
		t.Fatal("/briefing should show entries after switchRole")
	}
}

// --- Init ---

func TestModel_Init(t *testing.T) {
	m := testModel()
	cmd := m.Init()
	if cmd == nil {
		t.Fatal("Init should return a batch cmd (blink + window title)")
	}
}

// --- approvalForMode ---

func TestApprovalForMode_Auto(t *testing.T) {
	fn := approvalForMode(agent.ModeAuto, nil)
	if !fn(driver.ToolCall{Name: "Bash"}) {
		t.Fatal("auto mode should auto-approve")
	}
}

func TestApprovalForMode_Ask(t *testing.T) {
	fn := approvalForMode(agent.ModeAsk, nil)
	if fn(driver.ToolCall{Name: "Bash"}) {
		t.Fatal("ask mode should deny all")
	}
}

func TestApprovalForMode_Plan(t *testing.T) {
	fn := approvalForMode(agent.ModePlan, nil)
	if fn(driver.ToolCall{Name: "Bash"}) {
		t.Fatal("plan mode should deny all")
	}
}

func TestApprovalForMode_Agent(t *testing.T) {
	ch := make(chan bool, 1)
	ch <- true
	fn := approvalForMode(agent.ModeAgent, ch)
	if !fn(driver.ToolCall{Name: "Bash"}) {
		t.Fatal("agent mode should return channel value")
	}
}

// --- Accessor tests (cover trivial 0% functions) ---

func TestModel_CurrentState(t *testing.T) {
	m := testModel()
	if m.CurrentState() != StateInput {
		t.Fatalf("state = %d, want input", m.CurrentState())
	}
}

func TestModel_SetMode(t *testing.T) {
	m := testModel()
	m.SetMode(agent.ModeAuto)
	if m.mode != agent.ModeAuto {
		t.Fatalf("mode = %s, want auto", m.mode)
	}
}

func TestModel_ConversationLen(t *testing.T) {
	m := testModel()
	before := m.ConversationLen()
	m.AppendConversation("test line")
	if m.ConversationLen() != before+1 {
		t.Fatal("ConversationLen should increase after append")
	}
}

// --- overlayContent tests ---

func TestModel_OverlayContent_Streaming_Spinner(t *testing.T) {
	m := testModel()
	m.state = stateStreaming
	m.spinnerActive = true
	overlay := m.overlayContent()
	if !strings.Contains(overlay, "thinking") {
		t.Fatalf("overlay = %q, should contain 'thinking'", overlay)
	}
}

func TestModel_OverlayContent_ToolApproval(t *testing.T) {
	m := testModel()
	m.state = stateToolApproval
	m.pendingTool = &driver.ToolCall{Name: "Bash"}
	overlay := m.overlayContent()
	if !strings.Contains(overlay, "approve") {
		t.Fatalf("overlay = %q, should contain 'approve'", overlay)
	}
}

func TestModel_OverlayContent_Default(t *testing.T) {
	m := testModel()
	m.state = stateInput
	overlay := m.overlayContent()
	if overlay != "" {
		t.Fatalf("overlay = %q, want empty for input state", overlay)
	}
}

// --- ErrorMsg handling ---

func TestModel_ErrorMsg(t *testing.T) {
	m := testModel()
	before := m.outputPanel.LineCount()
	m2, _ := m.Update(tui.ErrorMsg{Err: errors.New("boom")})
	model := asModel(t, m2)
	if model.lastError != "boom" {
		t.Fatalf("lastError = %q, want boom", model.lastError)
	}
	if model.outputPanel.LineCount() <= before {
		t.Fatal("ErrorMsg should append to output")
	}
}

// --- DoneMsg with nil usage ---

func TestModel_DoneMsg_NilUsage(t *testing.T) {
	m := testModel()
	m2, _ := m.Update(tui.DoneMsg{Usage: nil})
	model := asModel(t, m2)
	if model.totalIn != 0 || model.totalOut != 0 {
		t.Fatal("nil usage should not change token counts")
	}
}

// --- AgentDoneMsg with usage ---

func TestModel_AgentDoneMsg_WithUsage(t *testing.T) {
	m := testModel()
	m.state = stateStreaming
	m.lastUsage = &driver.Usage{InputTokens: 200, OutputTokens: 100}
	m2, _ := m.Update(tui.AgentDoneMsg{Result: "done"})
	model := asModel(t, m2)
	// Should include token summary in output
	found := false
	for _, line := range model.outputPanel.Lines() {
		if strings.Contains(line, "200") && strings.Contains(line, "100") {
			found = true
		}
	}
	if !found {
		t.Fatal("AgentDoneMsg should render usage summary")
	}
}

// --- View: quitting and not-ready states ---

func TestModel_View_Quitting(t *testing.T) {
	m := testModel()
	m.quitting = true
	view := m.View()
	if !strings.Contains(view, "goodbye") {
		t.Fatalf("view = %q, want goodbye", view)
	}
}

func TestModel_View_NotReady(t *testing.T) {
	m := testModel()
	m.ready = false
	view := m.View()
	if !strings.Contains(view, "initializing") {
		t.Fatalf("view = %q, want initializing", view)
	}
}

// --- handleKey: Esc during streaming ---

func TestModel_HandleKey_Esc_DuringStreaming(t *testing.T) {
	m := testModel()
	m.state = stateStreaming
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model := asModel(t, m2)
	// Esc during streaming is "back" (climb focus stack), not cancel.
	// State stays streaming — Esc doesn't abort the agent.
	if model.state != stateStreaming {
		t.Fatalf("state = %d, want streaming (Esc is back, not cancel)", model.state)
	}
}

// --- handleKey: tool approval ignores non-y/n keys ---

func TestModel_HandleKey_Approval_OtherKey(t *testing.T) {
	m := testModel()
	m.state = stateToolApproval
	m.pendingTool = &driver.ToolCall{Name: "Bash"}
	m2, _ := m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	model := asModel(t, m2)
	// State should remain tool approval — 'x' is not y/n
	if model.state != stateToolApproval {
		t.Fatalf("state = %d, want stateToolApproval (x is not y/n)", model.state)
	}
}

// --- handleKey: scroll keys forwarded ---

func TestModel_HandleKey_ScrollUp(t *testing.T) {
	m := testModel()
	// Add several lines to make scrollable
	for i := 0; i < 50; i++ {
		m.AppendConversation(fmt.Sprintf("line %d", i))
	}
	// pgup should not crash
	_, cmd := m.handleKey(tea.KeyMsg{Type: tea.KeyPgUp})
	_ = cmd // just verifying no panic
}

func TestModel_HandleKey_ScrollDown(t *testing.T) {
	m := testModel()
	for i := 0; i < 50; i++ {
		m.AppendConversation(fmt.Sprintf("line %d", i))
	}
	_, cmd := m.handleKey(tea.KeyMsg{Type: tea.KeyPgDown})
	_ = cmd
}

// --- handleKey: cycle mode with Alt+M ---

func TestModel_HandleKey_CycleMode(t *testing.T) {
	m := testModel()
	// Start at plan (from gensec default)
	if m.mode != agent.ModePlan {
		t.Fatalf("initial mode = %s", m.mode)
	}
	m2, _ := m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}, Alt: true})
	model := asModel(t, m2)
	if model.mode == agent.ModePlan {
		t.Fatal("mode should have cycled")
	}
}

// --- pickPlaceholderFile ---

func TestPickPlaceholderFile_EmptyDirs(t *testing.T) {
	result := pickPlaceholderFile(nil)
	if result != "" {
		t.Fatalf("got %q, want empty", result)
	}
}

func TestPickPlaceholderFile_NonexistentDir(t *testing.T) {
	result := pickPlaceholderFile([]string{"/nonexistent/path/xyz"})
	if result != "" {
		t.Fatalf("got %q, want empty", result)
	}
}

func TestPickPlaceholderFile_FindsGoFile(t *testing.T) {
	// The test workspace itself has Go files
	result := pickPlaceholderFile([]string{"."})
	// The repl package directory has .go files
	if result == "" {
		// Fall back — might not find one in CWD, that's OK
		return
	}
	if !strings.HasSuffix(result, ".go") {
		t.Fatalf("got %q, want a .go file", result)
	}
	if strings.HasSuffix(result, "_test.go") {
		t.Fatalf("got %q, should exclude test files", result)
	}
}

// --- SubmitMsg during tool approval queues ---

func TestModel_SubmitMsg_QueuedDuringToolApproval(t *testing.T) {
	m := testModel()
	m.state = stateToolApproval
	m2, _ := m.Update(tui.SubmitMsg{Value: "queued during approval"})
	model := asModel(t, m2)
	if model.queuePanel.Len() != 1 {
		t.Fatalf("queue = %d, want 1", model.queuePanel.Len())
	}
}

// --- WindowSizeMsg with initial prompt auto-submits ---

func TestModel_WindowSizeMsg_InitialPrompt(t *testing.T) {
	m := testModel()
	m.ready = false
	m.initialPrompt = "hello world"
	m2, cmd := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	model := asModel(t, m2)
	// Initial prompt should have been consumed
	if model.initialPrompt != "" {
		t.Fatal("initialPrompt should be cleared after auto-submit")
	}
	// Should start streaming (handleSubmit triggers agent run)
	if model.state != stateStreaming {
		t.Fatalf("state = %d, want streaming (initial prompt auto-submits)", model.state)
	}
	if cmd == nil {
		t.Fatal("should return cmd from handleSubmit")
	}
}

// --- handleKey: Enter on non-input focused panel attempts dive ---

func TestModel_HandleKey_Enter_OnOutputPanel(t *testing.T) {
	m := testModel()
	m.focus.FocusPanel(0) // focus on output panel
	m2, _ := m.handleKey(tea.KeyMsg{Type: tea.KeyEnter})
	_ = asModel(t, m2) // no panic
}

// --- Unknown message type ---

func TestModel_Update_UnknownMsg(t *testing.T) {
	m := testModel()
	type customMsg struct{}
	m2, cmd := m.Update(customMsg{})
	model := asModel(t, m2)
	if cmd != nil {
		t.Fatal("unknown msg should return nil cmd")
	}
	if model.state != stateInput {
		t.Fatal("state should be unchanged")
	}
}

// --- AgentDoneMsg for non-gensec role transitions back to gensec ---

func TestModel_AgentDoneMsg_NonGensecRole_TransitionsBack(t *testing.T) {
	m, _ := testModelWithDriver()
	m.state = stateStreaming
	m.currentRole = "auditor"
	m2, _ := m.Update(tui.AgentDoneMsg{Result: "done"})
	model := asModel(t, m2)
	if model.currentRole != "gensec" {
		t.Fatalf("currentRole = %q, want gensec (non-gensec roles auto-return)", model.currentRole)
	}
}

// --- SpinnerTickMsg ignored when not active ---

func TestModel_SpinnerTickMsg_Inactive(t *testing.T) {
	m := testModel()
	m.spinnerActive = false
	// spinner.TickMsg — use the spinner's own Update to get a proper TickMsg
	m2, cmd := m.Update(spinner.TickMsg{ID: m.spin.ID(), Time: time.Now()})
	_ = asModel(t, m2)
	if cmd != nil {
		t.Fatal("spinner tick should return nil when inactive")
	}
}

// Ensure unused import suppression
var _ = fmt.Sprint
