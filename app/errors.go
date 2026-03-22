package app

import "errors"

// Sentinel errors for the app layer.
var (
	ErrUnknownDriver    = errors.New("unknown driver")
	ErrDriverNotImpl    = errors.New("driver not yet implemented")
	ErrNoSessions       = errors.New("no sessions found")
	ErrUnknownImport    = errors.New("unsupported import source")
	ErrMissingArgs      = errors.New("missing required arguments")
)
