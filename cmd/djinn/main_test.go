package main

import (
	"bytes"
	"testing"

	"github.com/dpopsuev/djinn/app"
)

func TestVersion(t *testing.T) {
	var buf bytes.Buffer
	if err := app.Run([]string{"version"}, &buf); err != nil {
		t.Fatalf("version: %v", err)
	}
	if got := buf.String(); got != "djinn "+app.Version+"\n" {
		t.Fatalf("output = %q", got)
	}
}

func TestHelp(t *testing.T) {
	var buf bytes.Buffer
	if err := app.Run([]string{"--help"}, &buf); err != nil {
		t.Fatalf("help: %v", err)
	}
	if buf.Len() == 0 {
		t.Fatal("help output should not be empty")
	}
}
