package claude

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/dpopsuev/djinn/djinnlog"
	"github.com/dpopsuev/djinn/driver"
)

func TestValidate_EmptyMessages(t *testing.T) {
	d := &APIDriver{apiURL: "http://test", apiKey: "key", log: djinnlog.Nop()}
	err := d.validateRequest(nil)

	var ve *driver.ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected ValidationError, got %T: %v", err, err)
	}
	if ve.Field != "messages" {
		t.Fatalf("Field = %q, want messages", ve.Field)
	}
}

func TestValidate_EmptyAPIURL(t *testing.T) {
	d := &APIDriver{apiKey: "key", log: djinnlog.Nop()}
	msgs := []apiMessage{{Role: "user", Content: "hi"}}
	err := d.validateRequest(msgs)

	var ve *driver.ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected ValidationError, got %T: %v", err, err)
	}
	if ve.Field != "api_url" {
		t.Fatalf("Field = %q, want api_url", ve.Field)
	}
}

func TestValidate_EmptyAPIKey(t *testing.T) {
	d := &APIDriver{apiURL: "http://test", log: djinnlog.Nop()}
	msgs := []apiMessage{{Role: "user", Content: "hi"}}
	err := d.validateRequest(msgs)

	var ve *driver.ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected ValidationError, got %T: %v", err, err)
	}
	if ve.Field != "api_key" {
		t.Fatalf("Field = %q, want api_key", ve.Field)
	}
}

func TestValidate_EmptyModel_Direct(t *testing.T) {
	d := &APIDriver{
		config: driver.DriverConfig{Model: ""},
		apiURL: "http://test",
		apiKey: "key",
		log:    djinnlog.Nop(),
	}
	msgs := []apiMessage{{Role: "user", Content: "hi"}}
	err := d.validateRequest(msgs)

	// resolveModel() returns defaultModel when config.Model is empty,
	// so this should NOT fail validation.
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidate_ValidDirect(t *testing.T) {
	d := &APIDriver{
		config: driver.DriverConfig{Model: "claude-sonnet-4-6"},
		apiURL: "http://test",
		apiKey: "key",
		log:    djinnlog.Nop(),
	}
	msgs := []apiMessage{{Role: "user", Content: "hi"}}
	if err := d.validateRequest(msgs); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidate_ValidVertex(t *testing.T) {
	d := &APIDriver{
		config:    driver.DriverConfig{Model: ""},
		apiURL:    "http://vertex",
		apiKey:    "token",
		useVertex: true,
		log:       djinnlog.Nop(),
	}
	msgs := []apiMessage{{Role: "user", Content: "hi"}}
	if err := d.validateRequest(msgs); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidate_EmptyToolUseInput(t *testing.T) {
	d := &APIDriver{
		config: driver.DriverConfig{Model: "claude-sonnet-4-6"},
		apiURL: "http://test",
		apiKey: "key",
		log:    djinnlog.Nop(),
	}
	// Message with tool_use content block that has nil input
	msgs := []apiMessage{
		{Role: "user", Content: "hi"},
		{Role: "assistant", ContentBlocks: []apiContent{
			{Type: "tool_use", ID: "call-1", Name: "Bash", Input: nil},
		}},
		{Role: "user", ContentBlocks: []apiContent{
			{Type: "tool_result", ToolUseID: "call-1", Content: "output"},
		}},
	}

	err := d.validateRequest(msgs)
	if err == nil {
		t.Fatal("BUG-12: should reject messages with nil tool_use.input")
	}

	var ve *driver.ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected ValidationError, got %T: %v", err, err)
	}
	if ve.Field != "tool_use.input" {
		t.Fatalf("field = %q, want tool_use.input", ve.Field)
	}
}

func TestAPIDriver_ToolUseInputNeverNil(t *testing.T) {
	msg := apiMessage{
		Role: "assistant",
		ContentBlocks: []apiContent{
			{Type: "tool_use", ID: "call-1", Name: "Read", Input: nil},
		},
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatal(err)
	}

	content, ok := parsed["content"].([]any)
	if !ok || len(content) == 0 {
		t.Fatal("content should be an array")
	}

	block, ok := content[0].(map[string]any)
	if !ok {
		t.Fatal("content[0] should be an object")
	}

	input := block["input"]
	if input == nil {
		t.Fatal("BUG-12: tool_use.input serialized as null — Vertex will reject this")
	}
}
