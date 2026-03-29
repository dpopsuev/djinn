// cascade.go — Cascade propagation + component overlap detection (SPC-79).
//
// When a segment changes, cascade finds all affected segments via:
// 1. Explicit dependency edges (depends_on)
// 2. Spatial overlap (shared files in ComponentMap)
package plan

// Cascade finds all segments transitively affected by a change to the given segment.
// Returns IDs of affected segments (excluding the changed segment itself).
// Marks affected segments as invalidated.
func (pg *PlanGraph) Cascade(changedID string) []string {
	pg.mu.Lock()
	defer pg.mu.Unlock()

	changed, ok := pg.segments[changedID]
	if !ok {
		return nil
	}

	affected := make(map[string]bool)
	pg.cascadeDeps(changedID, affected)
	pg.cascadeOverlaps(changed, affected)

	// Mark affected as invalidated.
	result := make([]string, 0, len(affected))
	for id := range affected {
		if id == changedID {
			continue
		}
		s := pg.segments[id]
		if s.Status == StatusInProgress || s.Status == StatusClaimed || s.Status == StatusReady {
			s.Status = StatusInvalidated
			s.Version++
		}
		result = append(result, id)
	}
	return result
}

// cascadeDeps finds transitive dependents via depends_on edges.
func (pg *PlanGraph) cascadeDeps(sourceID string, affected map[string]bool) {
	for _, s := range pg.segments {
		if affected[s.ID] {
			continue
		}
		for _, dep := range s.DependsOn {
			if dep == sourceID || affected[dep] {
				affected[s.ID] = true
				pg.cascadeDeps(s.ID, affected) // recurse for transitive
				break
			}
		}
	}
}

// cascadeOverlaps finds segments with shared files in ComponentMap.
func (pg *PlanGraph) cascadeOverlaps(changed *Segment, affected map[string]bool) {
	changedFiles := fileSet(changed.Components)
	if len(changedFiles) == 0 {
		return
	}

	for _, s := range pg.segments {
		if s.ID == changed.ID || affected[s.ID] {
			continue
		}
		otherFiles := fileSet(s.Components)
		for f := range changedFiles {
			if otherFiles[f] {
				affected[s.ID] = true
				break
			}
		}
	}
}

// Overlaps returns file paths shared between two segments' ComponentMaps.
func (pg *PlanGraph) Overlaps(a, b string) []string {
	pg.mu.RLock()
	defer pg.mu.RUnlock()

	segA, okA := pg.segments[a]
	segB, okB := pg.segments[b]
	if !okA || !okB {
		return nil
	}

	filesA := fileSet(segA.Components)
	filesB := fileSet(segB.Components)

	var shared []string
	for f := range filesA {
		if filesB[f] {
			shared = append(shared, f)
		}
	}
	return shared
}

func fileSet(cm ComponentMap) map[string]bool {
	m := make(map[string]bool, len(cm.Files)+len(cm.Directories))
	for _, f := range cm.Files {
		m[f] = true
	}
	for _, d := range cm.Directories {
		m[d] = true
	}
	return m
}
