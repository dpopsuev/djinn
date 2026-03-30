// graph.go — ArtifactGraph: DAG operations, cascade, claims, HITL, persistence (GOL-59).
//
// Generalizes plan.PlanGraph and tools.TaskStore into one graph that works
// with any artifact kind. Template validation on Add. Thread-safe.
package artifact

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// Sentinel errors.
var (
	ErrNotFound          = errors.New("artifact: not found")
	ErrAlreadyClaimed    = errors.New("artifact: already claimed")
	ErrNotClaimed        = errors.New("artifact: not claimed")
	ErrInvalidTransition = errors.New("artifact: invalid status transition")
	ErrInvalidStatus     = errors.New("artifact: invalid status for kind")
)

// Graph is a DAG of Artifacts with template validation, cascade, claims, and persistence.
type Graph struct {
	Title    string `json:"title"`
	mu       sync.RWMutex
	items    map[string]*Artifact
	counters map[string]*atomic.Int64 // per-kind ID generation
	registry *TemplateRegistry
}

// NewGraph creates an empty graph with the given template registry.
func NewGraph(title string, registry *TemplateRegistry) *Graph {
	return &Graph{
		Title:    title,
		items:    make(map[string]*Artifact),
		counters: make(map[string]*atomic.Int64),
		registry: registry,
	}
}

// --- CRUD ---

// Add validates an artifact against its template and stores it.
// Generates an ID using the template's IDFormat if ID is empty.
// Normalizes Content ↔ Sections["content"].
func (g *Graph) Add(a Artifact) (string, error) { //nolint:gocritic // value copy intentional — caller retains ownership
	// Normalize Content → Sections["content"].
	if a.Content != "" && (a.Sections == nil || a.Sections["content"] == "") {
		if a.Sections == nil {
			a.Sections = make(map[string]string)
		}
		a.Sections["content"] = a.Content
	}
	if a.Sections != nil && a.Sections["content"] != "" && a.Content == "" {
		a.Content = a.Sections["content"]
	}

	// Validate against template.
	if err := g.registry.Check(&a); err != nil {
		return "", err
	}

	g.mu.Lock()
	defer g.mu.Unlock()

	// Generate ID if empty.
	if a.ID == "" {
		tmpl, _ := g.registry.Get(a.Kind) // already validated above
		counter, ok := g.counters[a.Kind]
		if !ok {
			counter = &atomic.Int64{}
			g.counters[a.Kind] = counter
		}
		a.ID = fmt.Sprintf(tmpl.IDFormat, counter.Add(1))
	}

	// Set defaults.
	if a.Status == "" {
		a.Status = StatusDraft
	}
	if a.Version == 0 {
		a.Version = 1
	}
	now := time.Now()
	if a.Created.IsZero() {
		a.Created = now
	}
	if a.Updated.IsZero() {
		a.Updated = now
	}

	g.items[a.ID] = &a
	return a.ID, nil
}

// Get returns an artifact by ID.
func (g *Graph) Get(id string) (*Artifact, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	a, ok := g.items[id]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrNotFound, id)
	}
	return a, nil
}

// All returns all artifacts as values.
func (g *Graph) All() []Artifact {
	g.mu.RLock()
	defer g.mu.RUnlock()
	out := make([]Artifact, 0, len(g.items))
	for _, a := range g.items {
		out = append(out, *a)
	}
	return out
}

// --- Status transitions ---

// Claim gives exclusive ownership of an artifact to an agent.
func (g *Graph) Claim(id, owner string) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	a, ok := g.items[id]
	if !ok {
		return fmt.Errorf("%w: %s", ErrNotFound, id)
	}
	if a.Status != StatusReady {
		return fmt.Errorf("%w: cannot claim in %s state", ErrInvalidTransition, a.Status)
	}
	if a.Owner != "" {
		return fmt.Errorf("%w: owned by %s", ErrAlreadyClaimed, a.Owner)
	}
	a.Owner = owner
	a.Status = StatusClaimed
	a.Updated = time.Now()
	return nil
}

// Start transitions a claimed artifact to in_progress.
func (g *Graph) Start(id string) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	a, ok := g.items[id]
	if !ok {
		return fmt.Errorf("%w: %s", ErrNotFound, id)
	}
	if a.Status != StatusClaimed {
		return fmt.Errorf("%w: cannot start in %s state", ErrInvalidTransition, a.Status)
	}
	a.Status = StatusInProgress
	a.Updated = time.Now()
	return nil
}

// Complete marks an artifact as done.
func (g *Graph) Complete(id string) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	a, ok := g.items[id]
	if !ok {
		return fmt.Errorf("%w: %s", ErrNotFound, id)
	}
	if a.Status != StatusInProgress {
		return fmt.Errorf("%w: cannot complete in %s state", ErrInvalidTransition, a.Status)
	}
	a.Status = StatusComplete
	a.Version++
	a.Updated = time.Now()
	return nil
}

// UpdateStatus sets an artifact's status with template validation.
func (g *Graph) UpdateStatus(id string, status Status) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	a, ok := g.items[id]
	if !ok {
		return fmt.Errorf("%w: %s", ErrNotFound, id)
	}
	tmpl, err := g.registry.Get(a.Kind)
	if err != nil {
		return err
	}
	if !tmpl.HasStatus(status) {
		return fmt.Errorf("%w: %q not valid for kind %q", ErrInvalidStatus, status, a.Kind)
	}
	a.Status = status
	a.Updated = time.Now()
	return nil
}

// --- Queries ---

// Ready returns artifacts whose dependencies are all complete.
func (g *Graph) Ready() []Artifact {
	g.mu.RLock()
	defer g.mu.RUnlock()
	var ready []Artifact
	for _, a := range g.items {
		if a.Status != StatusReady {
			continue
		}
		if g.depsComplete(a) {
			ready = append(ready, *a)
		}
	}
	return ready
}

// DraftGaps returns artifacts in draft status.
func (g *Graph) DraftGaps() []Artifact {
	g.mu.RLock()
	defer g.mu.RUnlock()
	var gaps []Artifact
	for _, a := range g.items {
		if a.Status == StatusDraft {
			gaps = append(gaps, *a)
		}
	}
	return gaps
}

// ByKind returns all artifacts of the given kind.
func (g *Graph) ByKind(kind string) []Artifact {
	g.mu.RLock()
	defer g.mu.RUnlock()
	var out []Artifact
	for _, a := range g.items {
		if a.Kind == kind {
			out = append(out, *a)
		}
	}
	return out
}

// ByStatus returns all artifacts with the given status.
func (g *Graph) ByStatus(s Status) []Artifact {
	g.mu.RLock()
	defer g.mu.RUnlock()
	var out []Artifact
	for _, a := range g.items {
		if a.Status == s {
			out = append(out, *a)
		}
	}
	return out
}

// ListSorted returns all artifacts sorted by ID.
func (g *Graph) ListSorted() []Artifact {
	g.mu.RLock()
	defer g.mu.RUnlock()
	out := make([]Artifact, 0, len(g.items))
	for _, a := range g.items {
		out = append(out, *a)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func (g *Graph) depsComplete(a *Artifact) bool {
	for _, depID := range a.DependsOn {
		dep, ok := g.items[depID]
		if !ok || dep.Status != StatusComplete {
			return false
		}
	}
	return true
}

// --- Cascade ---

// Cascade finds all artifacts transitively affected by a change to the given artifact.
// Uses two-phase detection: explicit dependency edges + spatial overlap (ComponentMap).
// Returns the IDs of affected artifacts (marked as invalidated).
func (g *Graph) Cascade(changedID string) []string {
	g.mu.Lock()
	defer g.mu.Unlock()

	changed, ok := g.items[changedID]
	if !ok {
		return nil
	}
	changed.Version++
	changed.Updated = time.Now()

	affected := make(map[string]bool)
	g.cascadeDeps(changedID, affected)
	g.cascadeOverlaps(changedID, affected)

	ids := make([]string, 0, len(affected))
	for id := range affected {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func (g *Graph) cascadeDeps(changedID string, affected map[string]bool) {
	for _, a := range g.items {
		if affected[a.ID] || a.ID == changedID {
			continue
		}
		for _, dep := range a.DependsOn {
			if dep == changedID || affected[dep] {
				affected[a.ID] = true
				if a.Status == StatusInProgress || a.Status == StatusClaimed ||
					a.Status == StatusReady {
					a.Status = StatusInvalidated
					a.Updated = time.Now()
				}
				g.cascadeDeps(a.ID, affected) // recurse
				break
			}
		}
	}
}

func (g *Graph) cascadeOverlaps(changedID string, affected map[string]bool) {
	changed := g.items[changedID]
	changedFiles := fileSet(changed)
	if len(changedFiles) == 0 {
		return
	}
	for _, a := range g.items {
		if a.ID == changedID || affected[a.ID] {
			continue
		}
		aFiles := fileSet(a)
		for f := range changedFiles {
			if aFiles[f] {
				affected[a.ID] = true
				if a.Status == StatusInProgress || a.Status == StatusClaimed ||
					a.Status == StatusReady {
					a.Status = StatusInvalidated
					a.Updated = time.Now()
				}
				break
			}
		}
	}
}

// Overlaps returns files shared between two artifacts' ComponentMaps.
func (g *Graph) Overlaps(idA, idB string) []string {
	g.mu.RLock()
	defer g.mu.RUnlock()
	a, okA := g.items[idA]
	b, okB := g.items[idB]
	if !okA || !okB {
		return nil
	}
	aFiles := fileSet(a)
	bFiles := fileSet(b)
	var shared []string
	for f := range aFiles {
		if bFiles[f] {
			shared = append(shared, f)
		}
	}
	sort.Strings(shared)
	return shared
}

func fileSet(a *Artifact) map[string]bool {
	s := make(map[string]bool, len(a.Components.Files))
	for _, f := range a.Components.Files {
		s[f] = true
	}
	return s
}

// --- HITL ---

// FillDraft transitions a draft artifact to ready with the given content.
func (g *Graph) FillDraft(id, content string) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	a, ok := g.items[id]
	if !ok {
		return fmt.Errorf("%w: %s", ErrNotFound, id)
	}
	if a.Status != StatusDraft {
		return fmt.Errorf("%w: cannot fill non-draft (status=%s)", ErrInvalidTransition, a.Status)
	}
	a.Content = content
	if a.Sections == nil {
		a.Sections = make(map[string]string)
	}
	a.Sections["content"] = content
	a.Status = StatusReady
	a.Version++
	a.Updated = time.Now()
	return nil
}

// Annotate adds operator feedback to an artifact.
func (g *Graph) Annotate(id string, ann Annotation) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	a, ok := g.items[id]
	if !ok {
		return fmt.Errorf("%w: %s", ErrNotFound, id)
	}
	a.Annotations = append(a.Annotations, ann)
	a.Updated = time.Now()
	return nil
}

// Inject adds a new artifact linked to an existing one.
func (g *Graph) Inject(parentID string, child Artifact) (string, error) { //nolint:gocritic // value copy intentional
	if _, ok := g.items[parentID]; !ok {
		return "", fmt.Errorf("%w: %s", ErrNotFound, parentID)
	}
	child.DependsOn = append(child.DependsOn, parentID)
	return g.Add(child)
}

// Reorder changes an artifact's dependencies.
func (g *Graph) Reorder(id string, newDeps []string) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	a, ok := g.items[id]
	if !ok {
		return fmt.Errorf("%w: %s", ErrNotFound, id)
	}
	a.DependsOn = newDeps
	a.Updated = time.Now()
	return nil
}

// --- TopoSort ---

// TopoSort returns artifacts in dependency order (Kahn's algorithm).
// Deterministic: sorted by ID within each frontier. Cycle-safe: appends remaining.
func (g *Graph) TopoSort() []Artifact {
	g.mu.RLock()
	defer g.mu.RUnlock()

	inDegree := make(map[string]int, len(g.items))
	dependents := make(map[string][]string) // dep → artifacts that depend on it

	for id, a := range g.items {
		if _, ok := inDegree[id]; !ok {
			inDegree[id] = 0
		}
		for _, dep := range a.DependsOn {
			if _, ok := g.items[dep]; ok {
				inDegree[id]++
				dependents[dep] = append(dependents[dep], id)
			}
		}
	}

	var queue []string
	for id, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, id)
		}
	}
	sort.Strings(queue)

	var sorted []Artifact
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		sorted = append(sorted, *g.items[id])
		for _, child := range dependents[id] {
			inDegree[child]--
			if inDegree[child] == 0 {
				queue = append(queue, child)
			}
		}
		sort.Strings(queue)
	}

	// Append remaining (cycles) in sorted order.
	if len(sorted) < len(g.items) {
		inSorted := make(map[string]bool, len(sorted))
		for i := range sorted {
			inSorted[sorted[i].ID] = true
		}
		var remaining []string
		for id := range g.items {
			if !inSorted[id] {
				remaining = append(remaining, id)
			}
		}
		sort.Strings(remaining)
		for _, id := range remaining {
			sorted = append(sorted, *g.items[id])
		}
	}

	return sorted
}

// --- Persistence ---

type graphState struct {
	Title     string           `json:"title"`
	Artifacts []Artifact       `json:"artifacts"`
	Counters  map[string]int64 `json:"counters"`
}

// Save writes the graph to disk as atomic JSON.
func (g *Graph) Save(path string) error {
	g.mu.RLock()
	state := g.marshalState()
	g.mu.RUnlock()

	raw, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("artifact: marshal: %w", err)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("artifact: mkdir: %w", err)
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return fmt.Errorf("artifact: write tmp: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp) //nolint:errcheck // best-effort cleanup
		return fmt.Errorf("artifact: rename: %w", err)
	}
	return nil
}

// Load reads graph data from a JSON file. Existing items are replaced.
func (g *Graph) Load(path string) error {
	data, err := os.ReadFile(path) //nolint:gosec // path from controlled config
	if err != nil {
		return fmt.Errorf("artifact: read: %w", err)
	}

	var state graphState
	if err := json.Unmarshal(data, &state); err != nil {
		return fmt.Errorf("artifact: unmarshal: %w", err)
	}

	g.mu.Lock()
	defer g.mu.Unlock()

	g.Title = state.Title
	g.items = make(map[string]*Artifact, len(state.Artifacts))
	for i := range state.Artifacts {
		a := state.Artifacts[i]
		g.items[a.ID] = &a
	}
	g.counters = make(map[string]*atomic.Int64, len(state.Counters))
	for kind, val := range state.Counters {
		counter := &atomic.Int64{}
		counter.Store(val)
		g.counters[kind] = counter
	}
	return nil
}

func (g *Graph) marshalState() graphState {
	artifacts := make([]Artifact, 0, len(g.items))
	for _, a := range g.items {
		artifacts = append(artifacts, *a)
	}
	sort.Slice(artifacts, func(i, j int) bool { return artifacts[i].ID < artifacts[j].ID })

	counters := make(map[string]int64, len(g.counters))
	for kind, counter := range g.counters {
		counters[kind] = counter.Load()
	}
	return graphState{Title: g.Title, Artifacts: artifacts, Counters: counters}
}
