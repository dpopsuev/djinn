package troupe

import (
	"context"
	"sync"

	anyllm "github.com/mozilla-ai/any-llm-go/providers"
)

var _ anyllm.Provider = (*StubProvider)(nil)

// StubProvider implements anyllm.Provider for testing.
// Returns pre-configured responses and records all calls.
type StubProvider struct {
	mu        sync.Mutex
	responses []*anyllm.ChatCompletion
	current   int
	CallLog   []anyllm.CompletionParams
	// Error to return from Completion (if set, overrides responses).
	Error error
}

// NewStubProvider creates a provider that cycles through canned responses.
func NewStubProvider(responses ...*anyllm.ChatCompletion) *StubProvider {
	return &StubProvider{responses: responses}
}

func (p *StubProvider) Name() string { return "stub" }

// Completion returns the next canned response, recording the call.
func (p *StubProvider) Completion(_ context.Context, params anyllm.CompletionParams) (*anyllm.ChatCompletion, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.CallLog = append(p.CallLog, params)

	if p.Error != nil {
		return nil, p.Error
	}

	if p.current >= len(p.responses) {
		return &anyllm.ChatCompletion{
			Choices: []anyllm.Choice{{
				Message:      anyllm.Message{Role: "assistant", Content: "(no more responses)"},
				FinishReason: anyllm.FinishReasonStop,
			}},
			Usage: &anyllm.Usage{},
		}, nil
	}

	resp := p.responses[p.current]
	p.current++
	return resp, nil
}

// CompletionStream wraps Completion into a single-chunk stream.
func (p *StubProvider) CompletionStream(ctx context.Context, params anyllm.CompletionParams) (<-chan anyllm.ChatCompletionChunk, <-chan error) { //nolint:gocritic // interface compliance
	chunks := make(chan anyllm.ChatCompletionChunk, 1)
	errs := make(chan error, 1)

	resp, err := p.Completion(ctx, params)
	if err != nil {
		errs <- err
		close(chunks)
		close(errs)
		return chunks, errs
	}

	chunk := anyllm.ChatCompletionChunk{
		ID:    resp.ID,
		Model: resp.Model,
		Usage: resp.Usage,
	}
	for _, c := range resp.Choices {
		chunk.Choices = append(chunk.Choices, anyllm.ChunkChoice{
			Index: c.Index,
			Delta: anyllm.ChunkDelta{
				Role:      c.Message.Role,
				Content:   c.Message.ContentString(),
				ToolCalls: c.Message.ToolCalls,
			},
			FinishReason: c.FinishReason,
		})
	}
	chunks <- chunk
	close(chunks)
	close(errs)
	return chunks, errs
}

// TextResponse is a helper to build a text-only ChatCompletion.
func TextResponse(text string, inputTokens, outputTokens int) *anyllm.ChatCompletion {
	return &anyllm.ChatCompletion{
		Choices: []anyllm.Choice{{
			Message:      anyllm.Message{Role: anyllm.RoleAssistant, Content: text},
			FinishReason: anyllm.FinishReasonStop,
		}},
		Usage: &anyllm.Usage{
			PromptTokens:     inputTokens,
			CompletionTokens: outputTokens,
		},
	}
}

// ToolCallResponse is a helper to build a tool-call ChatCompletion.
func ToolCallResponse(text string, calls []anyllm.ToolCall, inputTokens, outputTokens int) *anyllm.ChatCompletion {
	return &anyllm.ChatCompletion{
		Choices: []anyllm.Choice{{
			Message: anyllm.Message{
				Role:      anyllm.RoleAssistant,
				Content:   text,
				ToolCalls: calls,
			},
			FinishReason: anyllm.FinishReasonToolCalls,
		}},
		Usage: &anyllm.Usage{
			PromptTokens:     inputTokens,
			CompletionTokens: outputTokens,
		},
	}
}
