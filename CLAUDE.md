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

## Architecture

### Canonical Names

| Name | Role | Description |
|---|---|---|
| **Vezir** | Control Plane | Supervisor daemon. Reconciliation loop, socket relay, builder/watcher. Stateless. |
| **Substrate** | Node Mediator | Wires all node-local services. Revived name (was Miraged, dissolved for SRP). |
| **Vessel** | Agent Harness | Tools, envelope, space, budget. LLM + Vessel = Djinn Agent. |
| **Djinn** | Agent | LLM + Vessel. Ephemeral. Does the work. |
| **Terminal** | System Interface | Programmatic facade. AuthN/AuthZ. All actors interact through Terminal. |
| **TUI** | Human Interface | Operator's visual client of Terminal. |
| **Mirage** | Space Isolation | Overlays, containment, snapshots. Library. |
| **Lector** | File Understanding | File + symbol hot cache. Observes tool I/O, wraps Oculus. Substrate service. |
| **EventLog** | Event Stream | Append-only. Troupe signal.EventLog. CQRS write side. |
| **MutationTree** | Undo/Redo | Persistent projection over EventLog + Workspace snapshots. |
| **Workspace** | Working Dir | Mirage overlay. Two-tier manifest (ecosystem → project). |
| **Discourse** | Planning | Natural language deliberation. Program. |
| **Assignment** | Execution | Structured work unit downstream of Discourse. Process. |
| **Crucible** | Test Harness | Scenarios + referee. Was: Arena. |

### Knowledge Services (Relic ecosystem)

| Name | Role | Description |
|---|---|---|
| **Relic** | Contract Library | Types (Node, Edge, Section), interfaces (RelicStore, TypedDAG[T]). |
| **Reliquary** | Storage Engine | Dumb CRUD + structural invariants. SQLite per Realm. |
| **Hierophant** | Context Enrichment | Context sandwich maker. Fans out to Seers, scores, budgets. Envelope Enrich phase. |
| **Bishop** | Rules Seer | Scores rules by context. Reliquary(rules.db). |
| **Pontiff** | Requirements Seer | Lifecycle, guards, cascade. Reliquary(requirements.db). |
| **Abbot** | Plans Seer | Topo-sort, phase progress. Reliquary(plans.db). |
| **Deacon** | Execution Seer | Assignment state machine. Reliquary(execution.db). |

### Relic Layers & Kinds (5 layers, 12 kinds)

```
Governance: rule
Problem:    need, defect, vulnerability
Solution:   decision, spec, story
Effort:     campaign, phase, goal, task
Execution:  assignment
```

### RBAC Three-Entity Model (DJN-REF-77)

Role HAS capabilities. Tool REQUIRES capabilities. ToolClearance filters where requires ⊆ capabilities.

### Domain Services (Battery pattern: Service = Observer + Controller + Data)

Each domain package contains an Observer (watches, emits signals) and a Controller (decides, mutates). Stubs ship with the domain (Forge rule).

### Runtime Topology

```
Vezir (Control Plane — always running, supervised by OS init)
  ├── Socket Relay: operator terminal connects here (permanent endpoint)
  ├── Supervisor: Erlang OTP-style restart strategies
  ├── Reconciler: desired state vs actual state loop
  └── Builder: watch source → compile → trigger restart

  Substrate (Node Mediator — supervised by Vezir)
  ├── Mirage (space isolation — overlays, containment)
  ├── Lector (file + symbol understanding — hot cache, observes tool I/O)
  ├── Troupe (EventLog, actor mesh, signals)
  ├── Hierophant connection (context enrichment from Seers)
  └── Vessels (one per agent session)
        └── Djinn(s) (agents, interact through Terminal)
            General Staff (root scope /):
              Human Operator, GenSec (PID 1), 2Sec
            Project Staff (project scope):
              Executor, Auditor, Inspector
```

### Storage Tiers

```
Hot:   Lector cache (files, symbols) + Hierophant cache (assembled context)
Cold:  Reliquary (knowledge Relics, SQLite per Realm)
Log:   Troupe EventLog (temporal facts, append-only)
```

### Deployment: Monolith First

Restructure + PoC + MVP: everything in one binary, all services as Go packages, function calls not network. v0.1.0: start decoupling Seers and Hierophant into standalone services.

### RBAC (DJN-REF-77)

```yaml
# Capabilities → tools
read:        [Read, Glob, Grep]
write:       [Write, Edit]
code:        [Symbol, Build, Test, Lint]
vcs:         [VCS]
observe:     [Observe]
coordinate:  [Assignment]       # push — managers only
work:        [Task]             # pull — universal
shell:       [Bash]
communicate: [Discourse, Notes] # universal

# Base role (every role composes this)
agent: [communicate, work]

# Roles (composable)
developer:  [agent, reader, writer, coder] + vcs
architect:  [agent, reader, coder, observer, coordinator]
qa:         [agent, reader, coder]
operations: [agent, reader, observer]
manager:    [agent, observer, coordinator]
director:   [agent, manager] + shell
operator:   [agent, developer, architect, qa, operations] + shell
```

### Task + Assignment (two-sided work model)

- **Assignment** (push, coordinator cap): `assign`, `unassign`, `reassign`, `list` + opts
- **Task** (pull, universal): `current`, `next`, `submit`, `board` + opts
- **Gate on submit** (Jidoka): submit triggers gate (build/test/lint). Fail = stay. Pass = advance.
- **Submit statuses**: done (gate judges), blocked (escalate), needs_review (want approval)
- **Required comments** on ALL state-changing actions (Write, Edit, VCS, Task submit, Assignment)
- System prompt lists ONLY available tools. No mention of unavailable tools.

### Agents & Staff

**PoC: General Staff only (scope `/`, flat, everyone sees all)**

| Persona | Role | Purpose |
|---|---|---|
| Human Operator | operator | You. All capabilities. |
| GenSec | director | Root agent (PID 1). Shell access. Stewards Discourse. |
| 2Sec | manager | Plans, schedules, tracks assignments. |
| Coder(s) | developer | Writes code. Plural — spawn multiple. |

MVP adds Project Staff (scoped to `/djinn/troupe`) with Auditor (qa) and Inspector (operations).

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
Library (contract)       Service             Domain
Relic                    Seers               Knowledge graph
Mirage                   —                   Isolation
Troupe                   Olympiad            Agent Mesh
—                        Reliquary           Dumb storage (SQLite)
—                        Substrate           Node mediator
—                        Vezir               Control Plane
```

### Built Systems
- **Tool Envelope**: Gate/Enrich/Execute/Record pipeline. Hierophant enriches here.
- **CompositeExecutor**: 3-tier tool routing (override → builtin → MCP)
- **MutationTree**: persistent projection over EventLog (undo/ package)
- **Observe Tool**: introspection facade over EventLog (collapses TraceProjection)

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
cmd/vezir/     — Control Plane daemon binary
config/        — Config loading
cortex/        — Agent working memory: context window, warming, compaction, anchoring
discourse/     — Planning service (natural language deliberation)
driver/        — LLM driver (driver/troupe/)
hotswap/       — Socket protocol (Vezir pre-work)
lector/        — NEW: File + symbol understanding, hot cache, observes tool I/O
mcp/           — MCP client
observe/       — NEW: Introspection facade over EventLog (collapses TraceProjection)
policy/        — Battery policy bridge
repl/          — Interactive composition root
review/        — Code review, LSP
sandbox/       — Space adapter (Mirage bridge)
scripts/       — Build scripts
substrate/     — NEW: Node mediator (wires Mirage + Lector + Troupe + Vessels). Was: miraged/
telemetry/     — Pipe only (keys, logging setup). TraceProjection collapsed into observe/
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

Changes from current: `miraged/`→`substrate/`, `cmd/miraged/` deleted. New: `lector/`, `observe/`, `substrate/`. TraceProjection moved from `telemetry/` to `observe/`.

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
