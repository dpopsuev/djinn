// Package lector provides file and symbol understanding for the Substrate.
// Lector observes tool I/O (Read, Write, Edit), maintains a hot cache of
// files, symbols, and AST data. Tools do I/O, Lector does understanding.
// Wraps Oculus for symbol parsing. NOT a Seer — no Reliquary, no persistence.
package lector

// FileEntry describes a file known to Lector.
type FileEntry struct {
	Path         string   // absolute path
	Package      string   // Go package name (empty for non-Go)
	Language     string   // detected language
	Size         int64    // file size in bytes
	Hash         string   // content hash (SHA-256)
	Imports      []string // package imports (Go-specific)
	LastModified int64    // unix timestamp
}

// Symbol describes a code symbol indexed by Lector.
type Symbol struct {
	Name      string // symbol name
	Kind      string // func, type, var, const, method, interface
	Package   string // owning package
	File      string // file path
	Line      int    // line number
	Exported  bool   // starts with uppercase
	Signature string // type signature (e.g. "func(ctx context.Context) error")
}

// Index is the read interface for querying Lector's hot cache.
type Index interface {
	// FileInfo returns metadata for a cached file. Returns false if not indexed.
	FileInfo(path string) (FileEntry, bool)

	// Symbols returns all symbols in the given package scope.
	Symbols(scope string) []Symbol

	// Imports returns all packages that the given package imports.
	Imports(pkg string) []string

	// Dependents returns all packages that import the given package.
	Dependents(pkg string) []string
}

// Observer is the write interface — Lector watches tool I/O through this.
type Observer interface {
	// OnFileRead indexes a file after it's been read.
	OnFileRead(path string)

	// OnFileWrite invalidates and re-indexes a file after it's been written.
	OnFileWrite(path string)

	// OnFileDelete removes a file from the index.
	OnFileDelete(path string)
}

// Lector combines Index (read) and Observer (write) interfaces.
type Lector interface {
	Index
	Observer
}
