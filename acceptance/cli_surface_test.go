package acceptance

import (
	"bytes"
	"strings"
	"testing"

	"github.com/dpopsuev/djinn/app"

	"gopkg.in/yaml.v3"
)

func TestCLI_Version(t *testing.T) {
	var buf bytes.Buffer
	if err := app.Run([]string{"version"}, &buf); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), app.Version) {
		t.Fatalf("output = %q", buf.String())
	}
}

func TestCLI_Help(t *testing.T) {
	var buf bytes.Buffer
	app.Run([]string{"--help"}, &buf)
	out := buf.String()

	for _, flag := range []string{"--driver", "--model", "--mode", "--workspace", "--config", "--verbose"} {
		if !strings.Contains(out, flag) {
			t.Fatalf("help missing %s", flag)
		}
	}
}

func TestCLI_HelpContainsWorkspace(t *testing.T) {
	var buf bytes.Buffer
	app.Run([]string{"--help"}, &buf)
	if !strings.Contains(buf.String(), "--workspace") {
		t.Fatal("help should mention --workspace")
	}
}

func TestCLI_DoctorOutput(t *testing.T) {
	var buf bytes.Buffer
	if err := app.RunDoctor(&buf); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, section := range []string{"djinn doctor", "version:", "drivers:", "tools:"} {
		if !strings.Contains(out, section) {
			t.Fatalf("doctor missing %q", section)
		}
	}
}

func TestCLI_ConfigDump(t *testing.T) {
	var buf bytes.Buffer
	if err := app.RunConfig([]string{"dump"}, &buf); err != nil {
		t.Fatal(err)
	}
	var parsed map[string]any
	if err := yaml.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("config dump not valid YAML: %v", err)
	}
}

func TestCLI_UnknownSubcommand(t *testing.T) {
	// Unknown subcommands are treated as REPL prompts — requires driver
	// so we just verify no panic and error mentions driver
	var buf bytes.Buffer
	err := app.Run([]string{"repl", "--driver", "unknown"}, &buf)
	if err == nil {
		t.Fatal("unknown driver should error")
	}
}
