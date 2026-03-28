package tools

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestDiscourse_CreateAndGetTopic(t *testing.T) {
	store := NewDiscourseStore("")
	topic := store.CreateTopic("/aeon/djinn", "Implement TUI panels", "feature")

	if topic.ID != "D-001" {
		t.Fatalf("ID = %q, want D-001", topic.ID)
	}
	if topic.Title != "Implement TUI panels" {
		t.Fatalf("Title = %q", topic.Title)
	}
	if topic.Kind != "feature" {
		t.Fatalf("Kind = %q", topic.Kind)
	}
	if topic.Status != TopicDraft {
		t.Fatalf("Status = %q, want draft", topic.Status)
	}

	got, ok := store.GetTopic("/aeon/djinn", topic.ID)
	if !ok {
		t.Fatal("GetTopic returned false")
	}
	if got.Title != "Implement TUI panels" {
		t.Fatalf("GetTopic Title = %q", got.Title)
	}
}

func TestDiscourse_CreateThreadAndAppendMessage(t *testing.T) {
	store := NewDiscourseStore("")
	topic := store.CreateTopic("/aeon/djinn", "Bug fix", "bug")

	thread, err := store.CreateThread("/aeon/djinn", topic.ID)
	if err != nil {
		t.Fatalf("CreateThread: %v", err)
	}
	if thread.Status != ThreadActive {
		t.Fatalf("thread Status = %q, want active", thread.Status)
	}

	err = store.AppendMessage("/aeon/djinn", topic.ID, thread.ID, "user", "What's broken?")
	if err != nil {
		t.Fatalf("AppendMessage: %v", err)
	}
	err = store.AppendMessage("/aeon/djinn", topic.ID, thread.ID, "assistant", "The panel layout.")
	if err != nil {
		t.Fatalf("AppendMessage: %v", err)
	}

	if len(thread.Messages) != 2 {
		t.Fatalf("messages = %d, want 2", len(thread.Messages))
	}
	if thread.Messages[0].Role != "user" || thread.Messages[0].Content != "What's broken?" {
		t.Fatalf("msg[0] = %+v", thread.Messages[0])
	}
	if thread.Messages[1].Role != "assistant" || thread.Messages[1].Content != "The panel layout." {
		t.Fatalf("msg[1] = %+v", thread.Messages[1])
	}
}

func TestDiscourse_StaleTopics(t *testing.T) {
	store := NewDiscourseStore("")

	// Create two topics.
	t1 := store.CreateTopic("/aeon", "Old feature", "feature")
	store.CreateTopic("/aeon", "New feature", "feature")

	// Backdate the first topic.
	t1.Updated = time.Now().Add(-48 * time.Hour)

	stale := store.StaleTopics("/aeon", 24*time.Hour)
	if len(stale) != 1 {
		t.Fatalf("stale = %d, want 1", len(stale))
	}
	if stale[0].ID != t1.ID {
		t.Fatalf("stale topic = %q, want %q", stale[0].ID, t1.ID)
	}
}

func TestDiscourse_StaleTopicsSkipsCompleted(t *testing.T) {
	store := NewDiscourseStore("")
	topic := store.CreateTopic("/aeon", "Done feature", "feature")
	topic.Updated = time.Now().Add(-48 * time.Hour)
	store.UpdateTopicStatus("/aeon", topic.ID, TopicComplete) //nolint:errcheck // test setup, error not relevant

	// Backdate again after status update.
	topic.Updated = time.Now().Add(-48 * time.Hour)

	stale := store.StaleTopics("/aeon", 24*time.Hour)
	if len(stale) != 0 {
		t.Fatalf("stale = %d, want 0 (completed topics excluded)", len(stale))
	}
}

func TestDiscourse_OpenTopicCount(t *testing.T) {
	store := NewDiscourseStore("")
	store.CreateTopic("/aeon", "Feature 1", "feature")
	store.CreateTopic("/aeon", "Feature 2", "feature")
	t3 := store.CreateTopic("/aeon", "Feature 3", "feature")

	if count := store.OpenTopicCount("/aeon"); count != 3 {
		t.Fatalf("open = %d, want 3", count)
	}

	store.UpdateTopicStatus("/aeon", t3.ID, TopicComplete) //nolint:errcheck // test setup, error not relevant
	if count := store.OpenTopicCount("/aeon"); count != 2 {
		t.Fatalf("open = %d, want 2 after completing one", count)
	}

	// Non-existent scope returns 0.
	if count := store.OpenTopicCount("/nonexistent"); count != 0 {
		t.Fatalf("open = %d, want 0 for missing scope", count)
	}
}

func TestDiscourse_SaveLoadRoundtrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "discourse.json")

	// Build state.
	store := NewDiscourseStore(path)
	topic := store.CreateTopic("/aeon/djinn", "Scope tree", "feature")
	store.UpdateTopicStatus("/aeon/djinn", topic.ID, TopicActive) //nolint:errcheck // test setup, error not relevant
	thread, _ := store.CreateThread("/aeon/djinn", topic.ID)
	store.AppendMessage("/aeon/djinn", topic.ID, thread.ID, "user", "Let's build it") //nolint:errcheck // test setup, error not relevant

	if err := store.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Verify file exists.
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("file should exist: %v", err)
	}

	// Load into fresh store.
	store2 := NewDiscourseStore(path)
	if err := store2.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}

	got, ok := store2.GetTopic("/aeon/djinn", topic.ID)
	if !ok {
		t.Fatal("topic not found after load")
	}
	if got.Status != TopicActive {
		t.Fatalf("status = %q, want active", got.Status)
	}
	if len(got.Threads) != 1 {
		t.Fatalf("threads = %d, want 1", len(got.Threads))
	}
	if len(got.Threads[0].Messages) != 1 {
		t.Fatalf("messages = %d, want 1", len(got.Threads[0].Messages))
	}

	// NextIDs preserved — D counter was at 1, so next topic is D-002.
	next := store2.CreateTopic("/aeon/djinn", "Next topic", "chore")
	if next.ID != "D-002" {
		t.Fatalf("ID = %q, want D-002 (nextIDs should be preserved)", next.ID)
	}
}

func TestDiscourse_MultipleForums(t *testing.T) {
	store := NewDiscourseStore("")

	store.CreateTopic("/aeon/djinn", "Djinn feature", "feature")
	store.CreateTopic("/aeon/bugle", "Bugle bug", "bug")
	store.CreateTopic("/aeon/bugle", "Bugle refactor", "refactor")

	djinnForum := store.GetForum("/aeon/djinn")
	bugleForum := store.GetForum("/aeon/bugle")

	if len(djinnForum.Topics) != 1 {
		t.Fatalf("djinn topics = %d, want 1", len(djinnForum.Topics))
	}
	if len(bugleForum.Topics) != 2 {
		t.Fatalf("bugle topics = %d, want 2", len(bugleForum.Topics))
	}

	if store.OpenTopicCount("/aeon/djinn") != 1 {
		t.Fatalf("djinn open = %d", store.OpenTopicCount("/aeon/djinn"))
	}
	if store.OpenTopicCount("/aeon/bugle") != 2 {
		t.Fatalf("bugle open = %d", store.OpenTopicCount("/aeon/bugle"))
	}
}

func TestDiscourse_UpdateTopicStatusLifecycle(t *testing.T) {
	store := NewDiscourseStore("")
	topic := store.CreateTopic("/aeon", "Lifecycle test", "spike")

	// draft -> active -> complete -> archived
	transitions := []string{TopicActive, TopicComplete, TopicArchived}
	for _, status := range transitions {
		if err := store.UpdateTopicStatus("/aeon", topic.ID, status); err != nil {
			t.Fatalf("transition to %s: %v", status, err)
		}
		got, _ := store.GetTopic("/aeon", topic.ID)
		if got.Status != status {
			t.Fatalf("status = %q, want %q", got.Status, status)
		}
	}

	// Invalid status.
	if err := store.UpdateTopicStatus("/aeon", topic.ID, "invalid"); err == nil {
		t.Fatal("expected error for invalid status")
	}

	// Non-existent topic.
	if err := store.UpdateTopicStatus("/aeon", "D-999", TopicActive); err == nil {
		t.Fatal("expected error for missing topic")
	}

	// Non-existent scope.
	if err := store.UpdateTopicStatus("/nonexistent", topic.ID, TopicActive); err == nil {
		t.Fatal("expected error for missing scope")
	}
}

func TestDiscourse_ConcurrentAccess(t *testing.T) {
	store := NewDiscourseStore("")
	topic := store.CreateTopic("/aeon", "Concurrent", "chore")
	thread, _ := store.CreateThread("/aeon", topic.ID)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			store.AppendMessage("/aeon", topic.ID, thread.ID, "user", "msg") //nolint:errcheck // test setup, error not relevant
		}()
	}
	wg.Wait()

	if len(thread.Messages) != 50 {
		t.Fatalf("messages = %d, want 50", len(thread.Messages))
	}
}

func TestDiscourse_ErrorCases(t *testing.T) {
	store := NewDiscourseStore("")

	// CreateThread on non-existent scope.
	_, err := store.CreateThread("/missing", "D-001")
	if err == nil {
		t.Fatal("expected error for missing scope")
	}

	// CreateThread on non-existent topic.
	store.CreateTopic("/aeon", "Topic", "feature")
	_, err = store.CreateThread("/aeon", "D-999")
	if err == nil {
		t.Fatal("expected error for missing topic")
	}

	// AppendMessage on non-existent scope.
	err = store.AppendMessage("/missing", "D-001", "TH-001", "user", "hi")
	if err == nil {
		t.Fatal("expected error for missing scope")
	}

	// AppendMessage on non-existent thread.
	topic := store.CreateTopic("/test", "Test", "bug")
	store.CreateThread("/test", topic.ID) //nolint:errcheck // test setup, error not relevant
	err = store.AppendMessage("/test", topic.ID, "TH-999", "user", "hi")
	if err == nil {
		t.Fatal("expected error for missing thread")
	}

	// GetTopic on non-existent scope.
	_, ok := store.GetTopic("/missing", "D-001")
	if ok {
		t.Fatal("expected false for missing scope")
	}

	// Load non-existent file.
	store2 := NewDiscourseStore("/nonexistent/discourse.json")
	if err := store2.Load(); err == nil {
		t.Fatal("expected error loading nonexistent file")
	}
}
