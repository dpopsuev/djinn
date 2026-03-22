package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// stubConfig is a test Configurable.
type stubConfig struct {
	key string
	val string
}

func (c *stubConfig) ConfigKey() string { return c.key }
func (c *stubConfig) Snapshot() any     { return c.val }
func (c *stubConfig) Apply(v any) error {
	s, ok := v.(string)
	if !ok {
		return ErrConfigApply
	}
	c.val = s
	return nil
}

func TestRegistry_RegisterAndDump(t *testing.T) {
	r := NewRegistry()
	r.Register(&stubConfig{key: "test", val: "hello"})
	dump := r.Dump()
	if dump["test"] != "hello" {
		t.Fatalf("dump[test] = %v", dump["test"])
	}
}

func TestRegistry_Keys(t *testing.T) {
	r := NewRegistry()
	r.Register(&stubConfig{key: "b", val: "1"})
	r.Register(&stubConfig{key: "a", val: "2"})
	keys := r.Keys()
	if len(keys) != 2 || keys[0] != "b" || keys[1] != "a" {
		t.Fatalf("keys = %v, want [b, a]", keys)
	}
}

func TestRegistry_Get(t *testing.T) {
	r := NewRegistry()
	s := &stubConfig{key: "test", val: "hello"}
	r.Register(s)
	c, ok := r.Get("test")
	if !ok {
		t.Fatal("should find test")
	}
	if c.Snapshot() != "hello" {
		t.Fatal("wrong value")
	}
	_, ok = r.Get("missing")
	if ok {
		t.Fatal("should not find missing")
	}
}

func TestRegistry_Load(t *testing.T) {
	r := NewRegistry()
	s := &stubConfig{key: "test", val: "old"}
	r.Register(s)
	err := r.Load(map[string]any{"test": "new"})
	if err != nil {
		t.Fatal(err)
	}
	if s.val != "new" {
		t.Fatalf("val = %q, want new", s.val)
	}
}

func TestRegistry_Load_UnknownKeyIgnored(t *testing.T) {
	r := NewRegistry()
	err := r.Load(map[string]any{"unknown": "value"})
	if err != nil {
		t.Fatal("unknown key should be silently ignored")
	}
}

func TestRegistry_Load_Error(t *testing.T) {
	r := NewRegistry()
	r.Register(&stubConfig{key: "test", val: "old"})
	err := r.Load(map[string]any{"test": 42}) // wrong type
	if err == nil {
		t.Fatal("should return error for invalid type")
	}
}

func TestRegistry_DumpYAML(t *testing.T) {
	r := NewRegistry()
	r.Register(&stubConfig{key: "test", val: "hello"})
	data, err := r.DumpYAML()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "test:") {
		t.Fatalf("YAML should contain key: %s", data)
	}
}

func TestRegistry_LoadYAML(t *testing.T) {
	r := NewRegistry()
	s := &stubConfig{key: "mode", val: "agent"}
	r.Register(s)
	err := r.LoadYAML([]byte("mode: auto\n"))
	if err != nil {
		t.Fatal(err)
	}
	if s.val != "auto" {
		t.Fatalf("val = %q, want auto", s.val)
	}
}

func TestRegistry_LoadYAML_Invalid(t *testing.T) {
	r := NewRegistry()
	err := r.LoadYAML([]byte("{{{{invalid"))
	if err == nil {
		t.Fatal("should return error for invalid YAML")
	}
}

func TestRegistry_LoadFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.yaml")
	os.WriteFile(path, []byte("mode: plan\n"), 0644)

	r := NewRegistry()
	s := &stubConfig{key: "mode", val: "agent"}
	r.Register(s)
	if err := r.LoadFile(path); err != nil {
		t.Fatal(err)
	}
	if s.val != "plan" {
		t.Fatalf("val = %q, want plan", s.val)
	}
}

func TestRegistry_LoadFile_NotFound(t *testing.T) {
	r := NewRegistry()
	err := r.LoadFile("/nonexistent/file.yaml")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRegistry_Roundtrip(t *testing.T) {
	r1 := NewRegistry()
	r1.Register(&stubConfig{key: "mode", val: "auto"})
	data, err := r1.DumpYAML()
	if err != nil {
		t.Fatal(err)
	}

	r2 := NewRegistry()
	s := &stubConfig{key: "mode", val: "agent"}
	r2.Register(s)
	if err := r2.LoadYAML(data); err != nil {
		t.Fatal(err)
	}
	if s.val != "auto" {
		t.Fatalf("roundtrip: val = %q, want auto", s.val)
	}
}
