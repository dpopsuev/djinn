package store

import (
	"errors"
	"testing"
)

// contractTest runs the same test suite against any KeyStore implementation.
// Liskov: both MemStore and FileStore must behave identically.
func contractTest(t *testing.T, s KeyStore) {
	t.Helper()

	// Put + Get roundtrip.
	if err := s.Put("key1", []byte("value1")); err != nil {
		t.Fatalf("put: %v", err)
	}
	got, err := s.Get("key1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if string(got) != "value1" {
		t.Errorf("get = %q, want value1", got)
	}

	// Overwrite.
	if err := s.Put("key1", []byte("updated")); err != nil {
		t.Fatalf("put overwrite: %v", err)
	}
	got, err = s.Get("key1")
	if err != nil {
		t.Fatalf("get after overwrite: %v", err)
	}
	if string(got) != "updated" {
		t.Errorf("get after overwrite = %q, want updated", got)
	}

	// List.
	if err := s.Put("key2", []byte("value2")); err != nil {
		t.Fatalf("put key2: %v", err)
	}
	keys, err := s.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(keys) != 2 {
		t.Errorf("list len = %d, want 2", len(keys))
	}

	// Delete.
	if err := s.Delete("key1"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	_, err = s.Get("key1")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("get after delete: want ErrNotFound, got %v", err)
	}

	// Delete missing.
	err = s.Delete("nonexistent")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("delete missing: want ErrNotFound, got %v", err)
	}

	// Get missing.
	_, err = s.Get("nonexistent")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("get missing: want ErrNotFound, got %v", err)
	}

	// List after delete.
	keys, err = s.List()
	if err != nil {
		t.Fatalf("list after delete: %v", err)
	}
	if len(keys) != 1 {
		t.Errorf("list after delete len = %d, want 1", len(keys))
	}
}

func TestMemStore_Contract(t *testing.T) {
	contractTest(t, NewMemStore())
}

func TestFileStore_Contract(t *testing.T) {
	s, err := NewFileStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	contractTest(t, s)
}

func TestFileStore_SecretPermissions(t *testing.T) {
	s, err := NewFileStore(t.TempDir(), WithPermissions(0o600), WithExtension(".yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Put("secret", []byte("sensitive")); err != nil {
		t.Fatal(err)
	}
	got, err := s.Get("secret")
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "sensitive" {
		t.Errorf("got %q, want sensitive", got)
	}
}
