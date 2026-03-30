package repl

import "testing"

func TestClassifyInput(t *testing.T) {
	tests := []struct {
		input    string
		wantKind InputKind
		wantBody string
	}{
		// Shell mode
		{"! ls -la", InputShell, "ls -la"},
		{"!ls", InputShell, "ls"},
		{"!  spaced", InputShell, "spaced"},

		// Command mode
		{":g E2", InputCommand, "g E2"},
		{":quit", InputCommand, "quit"},
		{":ac+", InputCommand, "ac+"},
		{": spaced", InputCommand, "spaced"},

		// Slash commands (legacy compat)
		{"/help", InputSlash, "help"},
		{"/mode agent", InputSlash, "mode agent"},

		// Prompt (default)
		{"fix the login bug", InputPrompt, "fix the login bug"},
		{"what does this do?", InputPrompt, "what does this do?"},

		// Edge cases
		{"", InputPrompt, ""},
		{"  ", InputPrompt, ""},
		{"  ! ls", InputShell, "ls"},      // leading spaces stripped
		{"  :quit", InputCommand, "quit"}, // leading spaces stripped
	}

	for _, tt := range tests {
		kind, body := ClassifyInput(tt.input)
		if kind != tt.wantKind {
			t.Errorf("ClassifyInput(%q): kind = %v, want %v", tt.input, kind, tt.wantKind)
		}
		if body != tt.wantBody {
			t.Errorf("ClassifyInput(%q): body = %q, want %q", tt.input, body, tt.wantBody)
		}
	}
}

func TestInputKindString(t *testing.T) {
	tests := []struct {
		kind InputKind
		want string
	}{
		{InputPrompt, "prompt"},
		{InputShell, "shell"},
		{InputCommand, "command"},
		{InputSlash, "slash"},
		{InputKind(99), "prompt"}, // unknown defaults to prompt
	}
	for _, tt := range tests {
		if got := tt.kind.String(); got != tt.want {
			t.Errorf("InputKind(%d).String() = %q, want %q", tt.kind, got, tt.want)
		}
	}
}
