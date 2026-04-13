// Package canon provides VCS state caching for the Djinn Substrate.
//
// Canon caches: content hashes per file (SHA256 + mtime verification),
// dirty files list, recent commits, blame per file. Other Substrate
// services (Lector, Litmus) depend on Canon's content hashes as cache keys.
//
// Stigmergic: agents read/write files → Canon observes → cache updates.
// Next agent gets hashes from cache instead of re-computing.
//
// GOL-174, CMP-24
package canon
