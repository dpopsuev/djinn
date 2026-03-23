package claude

import (
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
