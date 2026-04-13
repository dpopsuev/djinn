// Package litmus caches build and test results per package.
//
// Invalidates when source files change (Canon provides content hashes).
// Eliminates repeated go build/test/vet/lint runs on unchanged code.
// Bridges to Limes for failure tracking, flake detection, and RCA.
//
// GOL-175, CMP-24
package litmus
