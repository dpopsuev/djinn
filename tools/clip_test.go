package tools

import (
	"errors"
	"testing"
)

func TestClipboard_SetAndGet(t *testing.T) {
	cb, err := NewClipboard(t.TempDir())
	if err != nil {
		t.Fatalf("NewClipboard: %v", err)
	}

	if err := cb.Set("key1", "value1"); err != nil {
		t.Fatalf("Set: %v", err)
	}

	val, err := cb.Get("key1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if val != "value1" {
		t.Fatalf("val = %q, want %q", val, "value1")
	}
}

func TestClipboard_GetNotFound(t *testing.T) {
	cb, _ := NewClipboard(t.TempDir())

	_, err := cb.Get("nonexistent")
	if !errors.Is(err, ErrClipNotFound) {
		t.Fatalf("err = %v, want ErrClipNotFound", err)
	}
}

func TestClipboard_Delete(t *testing.T) {
	cb, _ := NewClipboard(t.TempDir())
	cb.Set("key1", "val")

	if err := cb.Delete("key1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err := cb.Get("key1")
	if !errors.Is(err, ErrClipNotFound) {
		t.Fatal("key should be deleted")
	}
}

func TestClipboard_DeleteNotFound(t *testing.T) {
	cb, _ := NewClipboard(t.TempDir())
	err := cb.Delete("ghost")
	if !errors.Is(err, ErrClipNotFound) {
		t.Fatalf("err = %v, want ErrClipNotFound", err)
	}
}

func TestClipboard_List(t *testing.T) {
	cb, _ := NewClipboard(t.TempDir())
	cb.Set("beta", "2")
	cb.Set("alpha", "1")
	cb.Set("gamma", "3")

	keys, err := cb.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(keys) != 3 {
		t.Fatalf("len = %d, want 3", len(keys))
	}
	if keys[0] != "alpha" || keys[1] != "beta" || keys[2] != "gamma" {
		t.Fatalf("keys = %v, want [alpha beta gamma]", keys)
	}
}

func TestClipboard_EmptyKey(t *testing.T) {
	cb, _ := NewClipboard(t.TempDir())

	if err := cb.Set("", "val"); !errors.Is(err, ErrEmptyKey) {
		t.Fatalf("Set empty key err = %v", err)
	}
	if _, err := cb.Get(""); !errors.Is(err, ErrEmptyKey) {
		t.Fatalf("Get empty key err = %v", err)
	}
	if err := cb.Delete(""); !errors.Is(err, ErrEmptyKey) {
		t.Fatalf("Delete empty key err = %v", err)
	}
}

func TestClipboard_Overwrite(t *testing.T) {
	cb, _ := NewClipboard(t.TempDir())
	cb.Set("key", "old")
	cb.Set("key", "new")

	val, _ := cb.Get("key")
	if val != "new" {
		t.Fatalf("val = %q, want %q", val, "new")
	}
}

func TestClipboard_SanitizesKey(t *testing.T) {
	cb, _ := NewClipboard(t.TempDir())
	cb.Set("path/to/key", "value")

	val, _ := cb.Get("path/to/key")
	if val != "value" {
		t.Fatalf("sanitized key lookup failed: %q", val)
	}
}
