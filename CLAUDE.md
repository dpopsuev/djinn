# Djinn — AI Agent Industrial Complex

## Philosophy: Log(n) Complexity

Djinn achieves logarithmic time and token cost as work grows. Three mechanisms:

1. **Compose** — reduce total work. Steps compose into Tasks, Tasks into Plans. The "Do" artifact family (Current State → Desired State). Better plans = fewer steps. The Proposal Loop invests tokens in planning to eliminate rework.
2. **Decompose** — parallelize remaining work. Steps decompose into Jobs scheduled to N agents. Agent Space isolation prevents interference. More agents = work / N.
3. **Tooling** — reduce cost per step. Agent Shell enforces safety (less rework). Agent Substrate caches reads (less I/O). Enrichment adds context (fewer wrong edits).

Observable result: Flat Token Curve. Each additional unit of work costs less than the previous one.

## Pipeline: Prompt → Intent → Problem → Classify → Solution

The philosophical core (DJN-NED-31). Every operator interaction follows this flow:

```
Prompt   → raw text ("fix the auth timeout")
Intent   → parsed action + context (in memory, not an artifact)
Problem  → Need artifact (problem domain, apex of pyramid)
Classify → Taxonomer at each layer (Oculus/Parchment/Ordo modules)
Solution → Spec → Goal → Task → Code → Doc (pyramid descent)
```

Each layer runs Decompose → Taxonomy → Compose. The pyramid IS Parchment artifact kinds.

## Architecture: Three-Tier Runtime

### Canonical Names

| Name | Role | Description |
|---|---|---|
| **Vezir** | Control Plane | Supervisor daemon. Reconciliation loop, socket relay, builder/watcher. Stateless. |
| **Miraged** | Data Plane | Node daemon. Workspace, EventLog, MCP routing. One per node. |
| **Vessel** | Agent Harness | Tools, envelope, space, budget. One per agent/LLM session. |
| **Djinn** | Agent | LLM + Vessel. Ephemeral. Does the work. |
| **Terminal** | System Interface | Programmatic facade. AuthN/AuthZ. All actors interact through Terminal. |
| **TUI** | Human Interface | Operator's visual client of Terminal. |
| **EventLog** | Event Stream | Append-only. Troupe signal.EventLog. CQRS write side. |
| **TraceProjection** | Read Model | Bounded query cache. CQRS read side. Was: Ring. |
| **MutationTree** | Undo/Redo | Branching tree over EventLog + Workspace snapshots. |
| **Workspace** | Working Dir | Mirage overlay. Agent reads/writes here. |
| **Discourse** | Planning | Natural language deliberation. Program. |
| **Assignment** | Execution | Structured work unit downstream of Discourse. Process. |
| **Crucible** | Test Harness | Scenarios + referee. Was: Arena. |

### Domain Services (Battery pattern: Service = Observer + Controller + Data)

Each domain package contains an Observer (watches, emits signals) and a Controller (decides, mutates). Stubs ship with the domain (Forge rule).

### Runtime Topology

```
Vezir (Control Plane — always running, supervised by OS init)
  ├── Socket Relay: operator terminal connects here (permanent endpoint)
  ├── Supervisor: Erlang OTP-style restart strategies
  ├── Reconciler: desired state vs actual state loop
  └── Builder: watch source → compile → trigger restart

  Miraged (Data Plane — supervised by Vezir)
  ├── EventLog (mission-critical, persists across restarts)
  ├── Workspace (Mirage overlay)
  ├── Tool Envelope + MCP routing
  └── Vessels (one per agent session)
        └── Djinn(s) (agents, interact through Terminal)
            General Staff (root scope /):
              Human Operator, GenSec (PID 1), 2Sec
            Project Staff (project scope):
              Executor, Auditor, Inspector
```

### Agents & Staff

- **General Staff** (root scope `/`): Human Operator + GenSec + 2Sec. General Discourse forum.
- **Project Staff** (project scope `/djinn/djinn`): Executor (Pos 1), Auditor (Pos 3), Inspector (Pos 4).
- GenSec stewards General Discourse. 2Sec handles planning and scheduling.

### Libraries

| Library | Domain | Version |
|---|---|---|
| **Troupe** | Agent mesh (World, Identity, Signal, Broker) | v0.7.1 |
| **Mirage** | Isolation (overlay, Snapshot/Restore) | v0.3.0 |
| **Battery** | Tools Contract Library (tool, policy, middleware, service) | v0.3.0 |
| **Parchment** | Artifact graph engine | v0.1.0 |
| **Ordo** | Rule resolution engine | v0.1.0 |
| **Oculus** | Symbol/architecture analysis | v1.0.0 |

```
Library (contract)       Daemon              Domain
Mirage                   Miraged             Isolation / Data Plane
Troupe                   Olympiad            Agent Mesh
—                        Vezir               Control Plane (top of stack)
```

### Built Systems
- **Tool Envelope**: Gate/Enrich/Execute/Record pipeline
- **CompositeExecutor**: 3-tier tool routing (override → builtin → MCP)
- **TraceProjection**: bounded CQRS read model over EventLog
- **MutationTree**: checkpoint/rollback (undo/ package)

## Day 0 / Day 1 / Day 2

- **Day 0**: binary boots, no config, no external tools
- **Day 1**: built-in tools work standalone (batteries included). Every feature MUST work Day 1.
- **Day 2**: MCP enrichment — Locus, Lex, Scribe upgrade built-ins. MCPs enhance, never gate.

## Dependency Rules

- Djinn → Troupe (agent mesh). Djinn → Mirage (isolation). Djinn → Battery (tool contracts).
- Djinn NEVER imports Origami — use Olympiad mesh for shared agent pool.
- Dependency direction: `Origami → Olympiad ← Djinn` (both are mesh clients)

## Workspace Scopes

```
/                  — General. General Staff. General Discourse.
/djinn             — Djinn ecosystem (all projects)
/djinn/djinn       — Djinn the system (this repo)
/djinn/troupe      — Troupe
/djinn/mirage      — Mirage
/djinn/battery     — Battery
/djinn/origami     — Origami
```

## Desired Root Directory

```
agent/         — Agent loop (ReAct cycle)
app/           — Composition root
artifact/      — Parchment bridge
assets/        — Logo, static files
assignment/    — NEW: Execution service (structured work, downstream of Discourse)
budget/        — NEW: Budget service (Observer + Controller, from telemetry/wd_budget)
cmd/djinn/     — TUI + CLI binary
cmd/miraged/   — Data Plane daemon binary (rename from cmd/djinnd/)
cmd/vezir/     — NEW: Control Plane daemon binary
config/        — Config loading
cortex/        — Agent working memory: context window, warming, compaction, anchoring (was contextmgr/)
discourse/     — NEW: Planning service (natural language deliberation)
driver/        — LLM driver (driver/troupe/)
hotswap/       — Socket protocol (Vezir pre-work)
mcp/           — MCP client
miraged/       — Data Plane internals (merge substrate/ + daemon/)
policy/        — Battery policy bridge
repl/          — Interactive composition root
review/        — Code review, LSP
sandbox/       — Space adapter (Mirage bridge)
scripts/       — Build scripts
telemetry/     — Pipe only (keys, TraceProjection, logging setup)
terminal/      — System interface (AuthN/AuthZ facade)
test/          — Integration / E2E tests
testkit/       — Test infra
  crucible/    — Test harness, scenarios, referee (rename from arena/)
  stubs/       — Stubs
tools/         — Tool implementations (builtin, composite)
tui/           — Operator's visual client of Terminal
undo/          — MutationTree (checkpoint/rollback)
uniform/       — Spawn config, clearance, persona
vessel/        — NEW: Agent harness interface + stub
vezir/         — NEW: Control Plane internals (supervisor, reconciler, relay, builder)
workspace/     — Workspace scope, bus, git
```

Changes from current: `broker/` deleted, `contextmgr/`→`context/`, `daemon/`+`substrate/`→`miraged/`, `cmd/djinnd/`→`cmd/miraged/`, `testkit/arena/`→`testkit/crucible/`. New: `budget/`, `discourse/`, `assignment/`, `vessel/`, `vezir/`, `cmd/vezir/`.

## Manufacturing Principles

Djinn is influenced by Toyota Production System, Lean Manufacturing, 5S, Kaizen, and Agile:
- **JIT** (Just-in-Time): SupportScheduler spawns agents on demand, MCP tools load on connect
- **Jidoka** (stop on defect): QualityGate + HookRunner + Sovereign override
- **Andon** (visual signal): telemetry.SignalBus + Observers + dashboard blinker
- **Kanban** (visual scheduling): KanbanPanel for artifact lifecycle
- **Kaizen** (continuous improvement): Flywheel Gate proves each sprint makes the next easier
- **Gemba** (go and see): CellSight — agent sees real code, operator sees agent thinking
- **Nemawashi** (consensus): Proposal Loop — debate before implementation

## Flywheel Forge Philosophy

Build the forge before the sword. Every DX investment compounds.

**Rules:**
- **Stub with implementation.** Every new interface ships with a testkit stub in the same PR. Not after, not later — together. `var _ Interface = (*StubImpl)(nil)` is the first line written.
- **Red first.** Write the failing test using the stub before implementing the real code. If you can't write a test, the interface is wrong.
- **E2E skeleton before features.** Wire stubs end-to-end to prove interfaces compose. The skeleton runs before any real backend exists.
- **Stubs at every boundary.** Mirage has StubSpace. Troupe has MockActor + StubProvider + StubEventLog. Miraged has StubSubstrate. No exceptions.
- **Observable by default.** Every stub records call history. Every boundary logs. No "add tracing later."

The forge grows with the swords — not as a separate phase.

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

Use `telemetry.KeyX` constants for ALL slog field keys. `telemetry.For(log, "component")` for scoped loggers.

## Package Boundaries (depguard enforced)

- Only `tui/` may import bubbletea/lipgloss
- `telemetry/` is a leaf — must NOT import domain packages (fan-in=19, cycle risk)
- Tool envelope types aliased from `battery/middleware` — single source of truth
- Policy types aliased from `battery/policy` — single source of truth
