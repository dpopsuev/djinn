package discourse

import "testing"

// RunForumContract runs the Liskov contract test suite against any Forum.
func RunForumContract(t *testing.T, factory func(t *testing.T) Forum) {
	t.Helper()

	t.Run("Post_creates_thread", func(t *testing.T) {
		f := factory(t)
		id, err := f.Post("topic-1", Message{From: "operator", Content: "hello"})
		if err != nil {
			t.Fatalf("Post: %v", err)
		}
		if id == "" {
			t.Fatal("ThreadID should not be empty")
		}
	})

	t.Run("Reply_adds_message", func(t *testing.T) {
		f := factory(t)
		id, _ := f.Post("topic-1", Message{From: "operator", Content: "question"})
		err := f.Reply(id, Message{From: "gensec", Content: "answer"})
		if err != nil {
			t.Fatalf("Reply: %v", err)
		}

		threads := f.Threads("topic-1")
		if len(threads) != 1 {
			t.Fatalf("threads = %d, want 1", len(threads))
		}
		if len(threads[0].Messages) != 2 {
			t.Fatalf("messages = %d, want 2", len(threads[0].Messages))
		}
		if threads[0].Messages[0].From != "operator" {
			t.Errorf("msg[0].From = %q, want operator", threads[0].Messages[0].From)
		}
		if threads[0].Messages[1].From != "gensec" {
			t.Errorf("msg[1].From = %q, want gensec", threads[0].Messages[1].From)
		}
	})

	t.Run("Reply_to_unknown_thread", func(t *testing.T) {
		f := factory(t)
		err := f.Reply("nonexistent", Message{From: "x", Content: "y"})
		if err == nil {
			t.Fatal("expected error for unknown thread")
		}
	})

	t.Run("Threads_returns_empty_for_unknown_topic", func(t *testing.T) {
		f := factory(t)
		threads := f.Threads("nonexistent")
		if len(threads) != 0 {
			t.Fatalf("expected 0, got %d", len(threads))
		}
	})

	t.Run("Multiple_threads_in_topic", func(t *testing.T) {
		f := factory(t)
		f.Post("design", Message{From: "operator", Content: "idea 1"}) //nolint:errcheck // test
		f.Post("design", Message{From: "operator", Content: "idea 2"}) //nolint:errcheck // test

		threads := f.Threads("design")
		if len(threads) != 2 {
			t.Fatalf("threads = %d, want 2", len(threads))
		}
	})

	t.Run("Author_preserved_across_replies", func(t *testing.T) {
		f := factory(t)
		id, _ := f.Post("chat", Message{From: "operator", Content: "start"})
		f.Reply(id, Message{From: "gensec", Content: "ack"})      //nolint:errcheck // test
		f.Reply(id, Message{From: "coder-1", Content: "on it"})   //nolint:errcheck // test
		f.Reply(id, Message{From: "operator", Content: "thanks"}) //nolint:errcheck // test

		threads := f.Threads("chat")
		msgs := threads[0].Messages
		if len(msgs) != 4 {
			t.Fatalf("messages = %d, want 4", len(msgs))
		}
		expected := []string{"operator", "gensec", "coder-1", "operator"}
		for i, want := range expected {
			if msgs[i].From != want {
				t.Errorf("msg[%d].From = %q, want %q", i, msgs[i].From, want)
			}
		}
	})
}

func TestStubForum_Contract(t *testing.T) {
	RunForumContract(t, func(_ *testing.T) Forum {
		return NewStubForum()
	})
}
