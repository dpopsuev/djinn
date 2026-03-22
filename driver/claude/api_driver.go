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

	envAPIKey          = "ANTHROPIC_API_KEY"
	envVertexProject   = "ANTHROPIC_VERTEX_PROJECT_ID"
	envVertexRegion    = "CLOUD_ML_REGION"

	defaultModel      = "claude-sonnet-4-6"
	defaultMaxTokens  = 8192

	sseDataPrefix = "data: "
	sseDoneMarker = "[DONE]"
)

// Sentinel errors.
var (
	ErrNoAPIKey     = errors.New("no API key: set ANTHROPIC_API_KEY or ANTHROPIC_VERTEX_PROJECT_ID")
	ErrAPIError     = errors.New("claude API error")
	ErrAuthFailed   = errors.New("authentication failed")
)

// APIDriver implements driver.ChatDriver by calling the Claude Messages API
// directly with streaming SSE support.
var _ driver.ChatDriver = (*APIDriver)(nil)
type APIDriver struct {
	config     driver.DriverConfig
	tools      *builtin.Registry
	systemPrompt string
	apiURL     string
	apiKey     string
	useVertex  bool
	log        *slog.Logger

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

// WithLogger sets the logger for the driver.
func WithLogger(log *slog.Logger) APIDriverOption {
	return func(d *APIDriver) { d.log = log }
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
			return fmt.Errorf("%w: gcloud auth failed: %v (run: gcloud auth login)", ErrAuthFailed, err)
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
		return "", fmt.Errorf("empty token from gcloud")
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

	req, err := d.buildRequest(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	d.log.Debug("sending request", "model", d.resolveModel(), "messages", len(messages))
	reqStart := time.Now()

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		d.log.Error("API call failed", "error", err)
		return nil, fmt.Errorf("api call: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		d.log.Error("API error", "status", resp.StatusCode, slog.Duration("rtt", time.Since(reqStart)))
		return nil, fmt.Errorf("%w: %d: %s", ErrAPIError, resp.StatusCode, string(body))
	}

	d.log.Debug("response received", "status", resp.StatusCode, slog.Duration("rtt", time.Since(reqStart)))

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
	Type       string          `json:"type"`
	Text       string          `json:"text,omitempty"`
	ID         string          `json:"id,omitempty"`
	Name       string          `json:"name,omitempty"`
	Input      json.RawMessage `json:"input,omitempty"`
	ToolUseID  string          `json:"tool_use_id,omitempty"`
	Content    string          `json:"content,omitempty"`
	IsError    bool            `json:"is_error,omitempty"`
}

type apiRequest struct {
	Model             string       `json:"model"`
	MaxTokens         int          `json:"max_tokens"`
	System            string       `json:"system,omitempty"`
	Messages          []apiMessage `json:"messages"`
	Stream            bool         `json:"stream"`
	Tools             []apiTool    `json:"tools,omitempty"`
	AnthropicVersion  string       `json:"anthropic_version,omitempty"` // required for Vertex
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
		Model:     d.resolveModel(),
		MaxTokens: maxTokens,
		System:    d.systemPrompt,
		Messages:  messages,
		Stream:    true,
	}

	if d.useVertex {
		body.AnthropicVersion = "vertex-2023-10-16"
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

func (d *APIDriver) streamResponse(body io.ReadCloser, ch chan<- driver.StreamEvent) {
	defer close(ch)
	defer body.Close()

	scanner := bufio.NewScanner(body)
	var currentEvent sseEvent

	for scanner.Scan() {
		line := scanner.Text()

		if line == "" {
			// Empty line = end of event
			if currentEvent.Data != "" {
				d.processSSEEvent(currentEvent, ch)
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
		Type           string          `json:"type"`
		Text           string          `json:"text,omitempty"`
		Thinking       string          `json:"thinking,omitempty"`
		PartialJSON    string          `json:"partial_json,omitempty"`
	} `json:"delta"`
}

type sseMessageDelta struct {
	Type  string `json:"type"`
	Delta struct {
		StopReason string `json:"stop_reason"`
	} `json:"delta"`
	Usage *driver.Usage `json:"usage,omitempty"`
}

func (d *APIDriver) processSSEEvent(evt sseEvent, ch chan<- driver.StreamEvent) {
	switch evt.Event {
	case "content_block_start":
		var block sseContentBlockStart
		if json.Unmarshal([]byte(evt.Data), &block) != nil {
			return
		}
		if block.ContentBlock.Type == "tool_use" {
			ch <- driver.StreamEvent{
				Type: driver.EventToolUse,
				ToolCall: &driver.ToolCall{
					ID:   block.ContentBlock.ID,
					Name: block.ContentBlock.Name,
				},
			}
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
			// Accumulate tool input JSON — handled by caller
			ch <- driver.StreamEvent{Type: driver.EventText, Text: ""} // keep-alive
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
					Type:  "tool_use",
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
