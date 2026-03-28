package app

import "errors"

// Sentinel errors for the app layer.
var (
	ErrUnknownDriver     = errors.New("unknown driver")
	ErrDriverNotImpl     = errors.New("driver not yet implemented")
	ErrNoSessions        = errors.New("no sessions found")
	ErrUnknownImport     = errors.New("unsupported import source")
	ErrMissingArgs       = errors.New("missing required arguments")
	ErrSocketRequired    = errors.New("--socket is required for backend mode")
	ErrUsageDebugSession = errors.New("usage: djinn debug session <file-or-name>")
	ErrUnknownDebugCmd   = errors.New("unknown debug command")
	ErrNoFrames          = errors.New("no frames in file")
	ErrNoComponentData   = errors.New("no component data in frame (run with debug.tap_file enabled)")
	ErrNoDriverDetected  = errors.New("no agent CLI found on PATH")
	ErrUsageKill         = errors.New("kill requires a session name (usage: djinn kill <name>)")
	ErrUsageImport       = errors.New("import requires source and file (usage: djinn import claude <session.jsonl> [-s name])")
	ErrUsageConfigDump   = errors.New("usage: djinn config dump")
	ErrUnknownConfigCmd  = errors.New("unknown config command")
	ErrUsageWsCreate     = errors.New("usage: djinn workspace create <name> --repo <path> [--repo <path>...]")
	ErrUnknownWsCmd      = errors.New("unknown workspace command")
	ErrUsageRun          = errors.New("run requires a prompt (usage: djinn run <prompt>)")
	ErrConfigExists      = errors.New("djinn.yaml already exists")
)
