package claude

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/dpopsuev/djinn/djinnlog"
	"github.com/dpopsuev/djinn/driver"
	"github.com/dpopsuev/djinn/tools/builtin"
)

// API endpoints and defaults.
const (
	defaultAPIURL     = "https://api.anthropic.com/v1/messages"
	vertexAPITemplate = "https://%s-aiplatform.googleapis.com/v1/projects/%s/locations/%s/publishers/anthropic/models/%s:streamRawPredict"

	headerAPIKey       = "x-api-key"
	headerContentType2 = "content-type"
	headerAnthropicVer = "anthropic-version"
	anthropicVersion   = "2023-06-01"

	envAPIKey        = "ANTHROPIC_API_KEY" //nolint:gosec // env var name, not a credential
	envVertexProject = "ANTHROPIC_VERTEX_PROJECT_ID"
	envVertexRegion  = "CLOUD_ML_REGION"

	defaultModel     = "claude-sonnet-4-6"
	defaultMaxTokens = 8192

	sseDataPrefix = "data: "
	sseDoneMarker = "[DONE]"

	contentTypeToolUse = "tool_use"
)

// Sentinel errors.
var (
	ErrNoAPIKey   = errors.New("no API key: set ANTHROPIC_API_KEY or ANTHROPIC_VERTEX_PROJECT_ID")
	ErrAPIError   = errors.New("claude API error")
	ErrAuthFailed = errors.New("authentication failed")
	ErrEmptyToken = errors.New("empty token from gcloud")
)

// APIDriver implements driver.ChatDriver by calling the Claude Messages API
// directly with streaming SSE support.
var _ driver.ChatDriver = (*APIDriver)(nil)

type APIDriver struct {
	config       driver.DriverConfig
	tools        *builtin.Registry
	systemPrompt string
	apiURL       string
	apiKey       string
	useVertex    bool
	log          *slog.Logger
	client       *http.Client

	mu       sync.Mutex
	messages []apiMessage
	started  bool
	stopped  bool
}

// APIDriverOption configures an APIDriver.
type APIDriverOption func(*APIDriver)

// WithAPISystemPrompt sets the system prompt.
func WithAPISystemPrompt(prompt string) APIDriverOption {
	return func(d *APIDriver) { d.systemPrompt = prompt }
}

// WithTools registers built-in tools for the driver.
func WithTools(reg *builtin.Registry) APIDriverOption {
	return func(d *APIDriver) { d.tools = reg }
}

// WithAPIURL overrides the API endpoint (for testing).
func WithAPIURL(url string) APIDriverOption {
	return func(d *APIDriver) { d.apiURL = url }
}

// SetSystemPrompt updates the system prompt at runtime (thread-safe).
// Takes effect on the next Chat() call.
func (d *APIDriver) SetSystemPrompt(prompt string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.systemPrompt = prompt
}

// WithLogger sets the logger for the driver.
func WithLogger(log *slog.Logger) APIDriverOption {
	return func(d *APIDriver) { d.log = log }
}

// WithHTTPClient overrides the HTTP client (for testing).
func WithHTTPClient(c *http.Client) APIDriverOption {
	return func(d *APIDriver) { d.client = c }
}

// NewAPIDriver creates a Claude Messages API driver.
func NewAPIDriver(config driver.DriverConfig, opts ...APIDriverOption) (*APIDriver, error) {
	d := &APIDriver{
		config: config,
		log:    djinnlog.Nop(),
	}
	for _, opt := range opts {
		opt(d)
	}

	// Resolve API endpoint and key (don't override if already set by option)
	if key := os.Getenv(envAPIKey); key != "" {
		d.apiKey = key
		if d.apiURL == "" {
			d.apiURL = defaultAPIURL
		}
	} else if project := os.Getenv(envVertexProject); project != "" {
		region := os.Getenv(envVertexRegion)
		if region == "" {
			region = "us-east5"
		}
		model := d.resolveModel()
		d.apiURL = fmt.Sprintf(vertexAPITemplate, region, project, region, model)
		d.useVertex = true
	} else {
		return nil, ErrNoAPIKey
	}

	if d.config.Model == "" {
		d.config.Model = defaultModel
	}

	if d.client == nil {
		d.client = &http.Client{
			Transport: newRetryTransport(http.DefaultTransport, DefaultRetryConfig(), d.log),
			// No Timeout — SSE streaming can run for minutes. Context handles cancellation.
		}
	}

	return d, nil
}

func (d *APIDriver) Start(ctx context.Context, sandbox driver.SandboxHandle) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.started {
		return ErrAlreadyStarted
	}

	// Fail-fast: verify auth before REPL prompt appears
	if d.useVertex {
		token, err := getGCPAccessToken()
		if err != nil {
			return fmt.Errorf("%w: gcloud auth failed: %w (run: gcloud auth login)", ErrAuthFailed, err)
		}
		d.apiKey = token // reuse apiKey field for bearer token
	}

	d.started = true
	d.messages = nil
	return nil
}

// getGCPAccessToken obtains an OAuth2 token from gcloud CLI.
func getGCPAccessToken() (string, error) {
	out, err := exec.Command("gcloud", "auth", "print-access-token").Output()
	if err != nil {
		return "", fmt.Errorf("gcloud auth print-access-token: %w", err)
	}
	token := strings.TrimSpace(string(out))
	if token == "" {
		return "", ErrEmptyToken
	}
	return token, nil
}

func (d *APIDriver) Stop(ctx context.Context) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.stopped = true
	return nil
}

// Send sends a message and streams the response. The response may contain
// tool calls — the caller (agent loop) handles executing tools and calling
// Send again with tool results.
func (d *APIDriver) Send(ctx context.Context, msg driver.Message) error {
	d.mu.Lock()
	if !d.started || d.stopped {
		d.mu.Unlock()
		return ErrNotRunning
	}
	d.mu.Unlock()

	d.appendMessage(apiMessage{Role: msg.Role, Content: msg.Content})
	return nil
}

// SendRich sends a message with structured content blocks (for tool results).
func (d *APIDriver) SendRich(ctx context.Context, msg driver.RichMessage) error {
	d.mu.Lock()
	if !d.started || d.stopped {
		d.mu.Unlock()
		return ErrNotRunning
	}
	d.mu.Unlock()

	am := apiMessage{Role: msg.Role}
	if len(msg.Blocks) > 0 {
		am.ContentBlocks = richBlocksToAPI(msg.Blocks)
	} else {
		am.Content = msg.Content
	}
	d.appendMessage(am)
	return nil
}

// Chat sends the accumulated messages to Claude and returns streaming events.
// This is the main method the agent loop calls.
func (d *APIDriver) Chat(ctx context.Context) (<-chan driver.StreamEvent, error) {
	d.mu.Lock()
	if !d.started || d.stopped {
		d.mu.Unlock()
		return nil, ErrNotRunning
	}
	messages := make([]apiMessage, len(d.messages))
	copy(messages, d.messages)
	d.mu.Unlock()

	if err := d.validateRequest(messages); err != nil {
		return nil, err
	}

	req, err := d.buildRequest(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	d.log.Debug("sending request", "model", d.resolveModel(), "messages", len(messages))
	reqStart := time.Now()

	resp, err := d.client.Do(req)
	if err != nil {
		d.log.Error("API call failed", "error", err)
		return nil, fmt.Errorf("api call: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		d.log.Error("API error", slog.Int("status", resp.StatusCode), slog.Duration("rtt", time.Since(reqStart)))
		return nil, &driver.DriverError{
			StatusCode: resp.StatusCode,
			Retryable:  driver.ClassifyRetryable(resp.StatusCode),
			Provider:   d.providerName(),
			RequestID:  driver.ExtractRequestID(body),
			Message:    string(body),
		}
	}

	d.log.Debug("response received", slog.Int("status", resp.StatusCode), slog.Duration("rtt", time.Since(reqStart)))

	ch := make(chan driver.StreamEvent, 100)
	go d.streamResponse(resp.Body, ch)
	return ch, nil
}

// AppendAssistant adds the assistant's response to the message history
// so the next Chat call includes it.
func (d *APIDriver) AppendAssistant(msg driver.RichMessage) {
	am := apiMessage{Role: driver.RoleAssistant}
	if len(msg.Blocks) > 0 {
		am.ContentBlocks = richBlocksToAPI(msg.Blocks)
	} else {
		am.Content = msg.Content
	}
	d.appendMessage(am)
}

// Recv is provided for backward compatibility with the Driver interface.
// For the REPL, use Chat() instead.
func (d *APIDriver) Recv(ctx context.Context) <-chan driver.Message {
	ch := make(chan driver.Message, 1)
	go func() {
		defer close(ch)
		events, err := d.Chat(ctx)
		if err != nil {
			ch <- driver.Message{Role: driver.RoleAssistant, Content: fmt.Sprintf("error: %v", err)}
			return
		}
		var text strings.Builder
		for evt := range events {
			switch evt.Type {
			case driver.EventText:
				text.WriteString(evt.Text)
			case driver.EventError:
				text.WriteString("\nerror: " + evt.Error)
			}
		}
		ch <- driver.Message{Role: driver.RoleAssistant, Content: text.String()}
	}()
	return ch
}

func (d *APIDriver) appendMessage(m apiMessage) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.messages = append(d.messages, m)
}

func (d *APIDriver) providerName() string {
	if d.useVertex {
		return "claude-vertex"
	}
	return "claude-direct"
}

// ContextWindow returns the model's context window in tokens.
// Opus models have 1M context; all others default to 200K.
func (d *APIDriver) ContextWindow() int {
	model := d.resolveModel()
	if strings.Contains(model, "opus") {
		return 1_000_000
	}
	if strings.Contains(model, "haiku") {
		return 200_000
	}
	return 200_000
}

func (d *APIDriver) resolveModel() string {
	if d.config.Model != "" {
		return d.config.Model
	}
	return defaultModel
}

// --- API request/response types ---

type apiMessage struct {
	Role          string
	Content       string
	ContentBlocks []apiContent
}

// We need custom marshaling because Content can be string or array.
func (m apiMessage) MarshalJSON() ([]byte, error) {
	type raw struct {
		Role    string `json:"role"`
		Content any    `json:"content"`
	}
	r := raw{Role: m.Role}
	if len(m.ContentBlocks) > 0 {
		r.Content = m.ContentBlocks
	} else {
		r.Content = m.Content
	}
	return json.Marshal(r)
}

type apiContent struct {
	Type      string          `json:"type"`
	Text      string          `json:"text,omitempty"`
	ID        string          `json:"id,omitempty"`
	Name      string          `json:"name,omitempty"`
	Input     json.RawMessage `json:"input,omitempty"`
	ToolUseID string          `json:"tool_use_id,omitempty"`
	Content   string          `json:"content,omitempty"`
	IsError   bool            `json:"is_error,omitempty"`
}

// MarshalJSON ensures tool_use blocks always have input:{} not null.
func (c apiContent) MarshalJSON() ([]byte, error) {
	type raw apiContent // avoid recursion
	r := raw(c)
	if r.Type == contentTypeToolUse && r.Input == nil {
		r.Input = json.RawMessage(`{}`)
	}
	return json.Marshal(r)
}

type apiRequest struct {
	Model            string       `json:"model,omitempty"`
	MaxTokens        int          `json:"max_tokens"`
	System           string       `json:"system,omitempty"`
	Messages         []apiMessage `json:"messages"`
	Stream           bool         `json:"stream"`
	Tools            []apiTool    `json:"tools,omitempty"`
	AnthropicVersion string       `json:"anthropic_version,omitempty"` // required for Vertex
}

type apiTool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"input_schema"`
}

func (d *APIDriver) buildRequest(ctx context.Context, messages []apiMessage) (*http.Request, error) {
	maxTokens := d.config.MaxTokens
	if maxTokens == 0 {
		maxTokens = defaultMaxTokens
	}

	body := apiRequest{
		MaxTokens: maxTokens,
		System:    d.systemPrompt,
		Messages:  messages,
		Stream:    true,
	}

	if d.useVertex {
		body.AnthropicVersion = "vertex-2023-10-16"
		// Vertex puts model in URL, not body — omit model field
	} else {
		body.Model = d.resolveModel()
	}

	// Add tools if registry is set
	if d.tools != nil {
		for _, t := range d.tools.All() {
			body.Tools = append(body.Tools, apiTool{
				Name:        t.Name(),
				Description: t.Description(),
				InputSchema: t.InputSchema(),
			})
		}
	}

	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, d.apiURL, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	req.Header.Set(headerContentType2, "application/json")
	if d.useVertex {
		req.Header.Set("Authorization", "Bearer "+d.apiKey)
	} else {
		req.Header.Set(headerAnthropicVer, anthropicVersion)
		if d.apiKey != "" {
			req.Header.Set(headerAPIKey, d.apiKey)
		}
	}

	return req, nil
}

// --- SSE streaming parser ---

type sseEvent struct {
	Event string
	Data  string
}

// streamResponse parses the Claude SSE stream and emits typed events.
//
// Tool input accumulation: Claude streams tool_use input as multiple
// input_json_delta events. We buffer these per content-block index and
// emit the complete EventToolUse only when content_block_stop arrives.
// This ensures ToolCall.Input is always a complete JSON object, never nil.
func (d *APIDriver) streamResponse(body io.ReadCloser, ch chan<- driver.StreamEvent) {
	defer close(ch)
	defer body.Close()

	scanner := bufio.NewScanner(body)
	var currentEvent sseEvent

	// Per content-block-index accumulators for streamed tool input JSON.
	toolInputBufs := make(map[int]*strings.Builder)
	toolCalls := make(map[int]*driver.ToolCall)

	for scanner.Scan() {
		line := scanner.Text()

		if line == "" {
			if currentEvent.Data != "" {
				d.processSSEEventWithInput(currentEvent, ch, toolInputBufs, toolCalls)
			}
			currentEvent = sseEvent{}
			continue
		}

		if after, ok := strings.CutPrefix(line, "event: "); ok {
			currentEvent.Event = after
		} else if data, ok := strings.CutPrefix(line, sseDataPrefix); ok {
			if data == sseDoneMarker {
				return
			}
			currentEvent.Data = data
		}
	}
}

type sseContentBlockStart struct {
	Type         string `json:"type"`
	Index        int    `json:"index"`
	ContentBlock struct {
		Type  string          `json:"type"`
		ID    string          `json:"id,omitempty"`
		Name  string          `json:"name,omitempty"`
		Input json.RawMessage `json:"input,omitempty"`
		Text  string          `json:"text,omitempty"`
	} `json:"content_block"`
}

type sseContentBlockDelta struct {
	Type  string `json:"type"`
	Index int    `json:"index"`
	Delta struct {
		Type        string `json:"type"`
		Text        string `json:"text,omitempty"`
		Thinking    string `json:"thinking,omitempty"`
		PartialJSON string `json:"partial_json,omitempty"`
	} `json:"delta"`
}

type sseMessageDelta struct {
	Type  string `json:"type"`
	Delta struct {
		StopReason string `json:"stop_reason"`
	} `json:"delta"`
	Usage *driver.Usage `json:"usage,omitempty"`
}

func (d *APIDriver) processSSEEventWithInput(evt sseEvent, ch chan<- driver.StreamEvent, toolInputBufs map[int]*strings.Builder, toolCalls map[int]*driver.ToolCall) {
	switch evt.Event {
	case "content_block_start":
		var block sseContentBlockStart
		if json.Unmarshal([]byte(evt.Data), &block) != nil {
			return
		}
		if block.ContentBlock.Type == contentTypeToolUse {
			tc := &driver.ToolCall{
				ID:   block.ContentBlock.ID,
				Name: block.ContentBlock.Name,
			}
			toolCalls[block.Index] = tc
			toolInputBufs[block.Index] = &strings.Builder{}
			// Don't emit EventToolUse yet — wait for input accumulation.
		}

	case "content_block_delta":
		var delta sseContentBlockDelta
		if json.Unmarshal([]byte(evt.Data), &delta) != nil {
			return
		}
		switch delta.Delta.Type {
		case "text_delta":
			ch <- driver.StreamEvent{Type: driver.EventText, Text: delta.Delta.Text}
		case "thinking_delta":
			ch <- driver.StreamEvent{Type: driver.EventThinking, Thinking: delta.Delta.Thinking}
		case "input_json_delta":
			// Accumulate tool input JSON chunks (DJN-BUG-19).
			if buf, ok := toolInputBufs[delta.Index]; ok {
				buf.WriteString(delta.Delta.PartialJSON)
			}
		}

	case "content_block_stop":
		// Emit tool_use with accumulated input when the block ends.
		var stopBlock struct {
			Index int `json:"index"`
		}
		if json.Unmarshal([]byte(evt.Data), &stopBlock) != nil {
			return
		}
		if tc, ok := toolCalls[stopBlock.Index]; ok {
			if buf, ok := toolInputBufs[stopBlock.Index]; ok && buf.Len() > 0 {
				tc.Input = json.RawMessage(buf.String())
			} else {
				tc.Input = json.RawMessage(`{}`)
			}
			ch <- driver.StreamEvent{
				Type:     driver.EventToolUse,
				ToolCall: tc,
			}
			delete(toolCalls, stopBlock.Index)
			delete(toolInputBufs, stopBlock.Index)
		}

	case "message_delta":
		var md sseMessageDelta
		if json.Unmarshal([]byte(evt.Data), &md) != nil {
			return
		}
		ch <- driver.StreamEvent{
			Type:  driver.EventDone,
			Usage: md.Usage,
		}

	case "message_stop":
		// Final event — channel will be closed by streamResponse

	case "error":
		ch <- driver.StreamEvent{Type: driver.EventError, Error: evt.Data}
	}
}

func richBlocksToAPI(blocks []driver.ContentBlock) []apiContent {
	var out []apiContent
	for _, b := range blocks {
		switch b.Type {
		case driver.BlockText:
			out = append(out, apiContent{Type: "text", Text: b.Text})
		case driver.BlockToolUse:
			if b.ToolCall != nil {
				out = append(out, apiContent{
					Type:  contentTypeToolUse,
					ID:    b.ToolCall.ID,
					Name:  b.ToolCall.Name,
					Input: b.ToolCall.Input,
				})
			}
		case driver.BlockToolResult:
			if b.ToolResult != nil {
				out = append(out, apiContent{
					Type:      "tool_result",
					ToolUseID: b.ToolResult.ToolCallID,
					Content:   b.ToolResult.Output,
					IsError:   b.ToolResult.IsError,
				})
			}
		}
	}
	return out
}
