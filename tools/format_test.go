package tools

import (
	"errors"
	"strings"
	"testing"
)

func TestConvert_JSONToYAML(t *testing.T) {
	input := `{"name": "djinn", "version": 1}`
	out, err := Convert([]byte(input), FormatJSON, FormatYAML)
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	if !strings.Contains(string(out), "name: djinn") {
		t.Fatalf("output = %q, expected YAML with 'name: djinn'", string(out))
	}
}

func TestConvert_YAMLToJSON(t *testing.T) {
	input := "name: djinn\nversion: 1\n"
	out, err := Convert([]byte(input), FormatYAML, FormatJSON)
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	if !strings.Contains(string(out), `"name": "djinn"`) {
		t.Fatalf("output = %q, expected JSON with name field", string(out))
	}
}

func TestConvert_SameFormat(t *testing.T) {
	input := `{"key": "val"}`
	out, err := Convert([]byte(input), FormatJSON, FormatJSON)
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	if string(out) != input {
		t.Fatalf("same format should return input unchanged, got %q", string(out))
	}
}

func TestConvert_InvalidJSON(t *testing.T) {
	_, err := Convert([]byte("{bad"), FormatJSON, FormatYAML)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestConvert_InvalidYAML(t *testing.T) {
	_, err := Convert([]byte(":\n  :\n    - [invalid"), FormatYAML, FormatJSON)
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestConvert_UnsupportedFrom(t *testing.T) {
	_, err := Convert([]byte("data"), "toml", FormatJSON)
	if !errors.Is(err, ErrUnsupportedFormat) {
		t.Fatalf("err = %v, want ErrUnsupportedFormat", err)
	}
}

func TestConvert_UnsupportedTo(t *testing.T) {
	_, err := Convert([]byte(`{"k":"v"}`), FormatJSON, "xml")
	if !errors.Is(err, ErrUnsupportedFormat) {
		t.Fatalf("err = %v, want ErrUnsupportedFormat", err)
	}
}
