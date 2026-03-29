// register.go — registers Aeon Shell tools into the builtin registry.
//
// RegisterAeonShellTools creates shared instances of the underlying Go
// libraries and registers all 7 shell tools: plan, test, git, arch,
// discourse, reconcile, latency. Call after NewRegistry().
package builtin

import (
	"path/filepath"

	"github.com/dpopsuev/djinn/debug"
	"github.com/dpopsuev/djinn/tools"
	"github.com/dpopsuev/djinn/trace"
)

// RegisterAeonShellTools registers the 8 Aeon Shell tools into the registry.
// workDir is the primary workspace directory (used for git, arch, test, etc.).
// dataDir is the data persistence directory (used for plan, discourse JSON files).
func RegisterAeonShellTools(reg *Registry, workDir, dataDir string) {
	planStore := tools.NewTaskStore(filepath.Join(dataDir, "tasks.json"))
	discourse := tools.NewDiscourseStore(filepath.Join(dataDir, "discourse.json"))
	gitRepo := tools.NewGitRepo(workDir)
	tracker := tools.NewToolLatencyTracker()

	reg.Register(&PlanTool{Store: planStore})
	reg.Register(&TestTool{WorkDir: workDir})
	reg.Register(&GitTool{Repo: gitRepo})
	reg.Register(&ArchTool{WorkDir: workDir})
	reg.Register(&DiscourseTool{Store: discourse})
	reg.Register(&ReconcileTool{PlanStore: planStore, WorkDir: workDir})
	reg.Register(&LatencyTool{Tracker: tracker})
	reg.Register(&RenderTool{})
}

// RegisterDebugTrace registers the djinn_trace builtin tool for self-debugging.
// Call after RegisterAeonShellTools if trace ring is available.
func RegisterDebugTrace(reg *Registry, ring *trace.Ring) {
	if ring == nil {
		return
	}
	reg.Register(&DebugTraceTool{Server: debug.NewServer(ring)})
}
