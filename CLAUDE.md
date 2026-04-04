# Djinn — AI Agent Industrial Complex

## Philosophy: Log(n) Complexity

Djinn achieves logarithmic time and token cost as work grows. Three mechanisms:

1. **Compose** — reduce total work. Steps compose into Tasks, Tasks into Plans. The "Do" artifact family (Current State → Desired State). Better plans = fewer steps. The Proposal Loop invests tokens in planning to eliminate rework.
2. **Decompose** — parallelize remaining work. Steps decompose into Jobs scheduled to N agents. Agent Space isolation prevents interference. More agents = work / N.
3. **Tooling** — reduce cost per step. Agent Shell enforces safety (less rework). Agent Substrate caches reads (less I/O). Enrichment adds context (fewer wrong edits).

Observable result: Flat Token Curve. Each additional unit of work costs less than the previous one.

## Architecture: Four Facades + Seven Components

### Four Facades (inside djinnd)

Every tool call flows through four layers. No direct host access.

- **Agent Uniform** — who you are. Role, clearance, budget, model. Issued at spawn.
- **Agent Shell** — what you can do. Intercepts every tool call. Enforces Uniform.
- **Agent Space** — what you can see. Chrooted overlay FS per agent. Mirage (Day 1) / Misbah (Day 2).
- **Agent Substrate** — how it executes. Shared cache, Tool Envelope (Gate/Enrich/Execute/Record), router to executors.

### Seven Components

| Component | Role | Type |
|---|---|---|
| **Macro** | TUI / CLI cockpit | Application |
| **djinnd** | Client-side daemon (sessions, substrate, auth) | Daemon |
| **Troupe** | Agent mesh library (interfaces, ACP, local Broker) | Library |
| **Olympiad** | Agent mesh service (discovery, publishing, routing) | Service |
| **Mirage** | Overlay FS library (logical isolation) | Library |
| **Misbah** | Compute daemon (containers, physical isolation) | Service |
| **tesseractui** | Shared TUI primitives | Library |

### Agent Provider

```go
type AgentProvider interface {
    Acquire(ctx context.Context, prefs Preferences) (Agent, error)
}
// Adapters: LocalCLIProvider (existing drivers), JerichoProvider (Troupe), MockProvider
```

### Built Systems
- **Tool Envelope** (SPC-118): Gate/Enrich/Execute/Record pipeline — SecurityGate, PolicyGate, SymbolEnricher, WasteRecorder, HookRunner
- **CellSight** (SPC-85): bidirectional TUI state in agent prompt
- **SymbolGraph** (SPC-109): pre-edit caller impact
- **WasteClassifier** (SPC-105): 7 Lean waste types
- **MountTable**: VFS path translation (exists, not wired into tools)
- **CompositeExecutor**: 3-tier tool routing (override → builtin → MCP)

## Day 0 / Day 1 / Day 2

- **Day 0**: binary boots, no config, no external tools
- **Day 1**: built-in tools work standalone (batteries included). Every feature MUST work Day 1.
- **Day 2**: MCP enrichment — Locus, Lex, Scribe upgrade built-ins. MCPs enhance, never gate.

## Dependency Rules

- Djinn → Troupe (agent mesh library). Jericho codebase becomes Troupe.
- Djinn → Mirage (overlay FS library). Agent Space isolation.
- Djinn NEVER imports Origami — use Olympiad mesh for shared agent pool.
- Dependency direction: `Origami → Olympiad ← Djinn` (both are mesh clients)
- Library defines contract. Daemon provides distributed implementation.

```
Library (contract)       Daemon (distributed)     Domain
Mirage                   Misbah                   Isolation
Troupe                   Olympiad                 Agent Mesh
tesseractui              Macro                    Presentation
```

## Manufacturing Principles

Djinn is influenced by Toyota Production System, Lean Manufacturing, 5S, Kaizen, and Agile:
- **JIT** (Just-in-Time): SupportScheduler spawns agents on demand, MCP tools load on connect
- **Jidoka** (stop on defect): QualityGate + HookRunner + Sovereign override
- **Andon** (visual signal): signal.Bus + watchdog + dashboard blinker
- **Kanban** (visual scheduling): KanbanPanel for artifact lifecycle
- **Kaizen** (continuous improvement): Flywheel Gate proves each sprint makes the next easier
- **Gemba** (go and see): CellSight — agent sees real code, operator sees agent thinking
- **Nemawashi** (consensus): Proposal Loop — debate before implementation

## Working with Djinn

```bash
# Build
go build ./...

# Test
go test ./... -count=1

# Lint
golangci-lint run --new-from-rev=HEAD ./...

# Architecture
mcp__locus__analysis preset=architecture_review
mcp__locus__analysis solid_scan
mcp__locus__analysis drift
```

Consult Scribe for task details:
```
mcp__scribe__artifact list --scope djinn --kind task --status draft
mcp__scribe__artifact get --id DJN-TSK-619
```

## Logging (ROGYB)

Every boundary-crossing function must have structured logging:
- **Orange** (before Green): `slog.Warn` on errors, denials, failures — "What went wrong?"
- **Yellow** (after Green): `slog.Info/Debug` on success, decisions, metrics — "What happened? Are we healthy?"

Use `djinnlog.KeyX` constants for ALL slog field keys. `djinnlog.For(log, "component")` for scoped loggers.
