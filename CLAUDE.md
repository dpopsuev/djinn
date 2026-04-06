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

## Architecture: Substrate + Libraries

### Substrate (djinnd) — The Only Component

The daemon that manages everything. Agents act freely inside their Mirage.

- **Uniform** — spawn config DATA. Defines what tools exist in the agent's Space. Not blocking — absence.
- **Shell** — DISSOLVED. No interception. Agents act freely inside Space.
- **Space** — containment boundary. Agent works freely inside it. Mirage (Day 1) / Misbah (Day 2).
- **Substrate** — THE DAEMON (djinnd). Enrichment + observation, NOT interception. Caching, symbols, rules, planning, spawning.

### Libraries (module imports, not services)

| Library | Domain | Status |
|---|---|---|
| **Parchment** | Artifact graph engine | v0.1.0 published |
| **Ordo** | Rule resolution engine | v0.1.0 published |
| **Oculus** | Symbol/architecture analysis | Pending extraction from Locus |
| **Troupe** | Agent mesh (Actor/Broker/Driver/ACP) | Direct import |
| **Mirage** | Isolation facade (overlay/Kata/K8s sandbox) | v0.1 overlay, v0.2 planned |

### Built Systems
- **Tool Envelope** (SPC-118): Gate/Enrich/Execute/Record pipeline
- **CellSight** (SPC-85): bidirectional TUI state in agent prompt
- **SymbolGraph** (SPC-109): pre-edit caller impact
- **WasteClassifier** (SPC-105): 7 Lean waste types
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

## Flywheel Forge Philosophy

Build the forge before the sword. Every DX investment compounds.

**Rules:**
- **Stub with implementation.** Every new interface ships with a testkit stub in the same PR. Not after, not later — together. `var _ Interface = (*StubImpl)(nil)` is the first line written.
- **Red first.** Write the failing test using the stub before implementing the real code. If you can't write a test, the interface is wrong.
- **E2E skeleton before features.** Wire stubs end-to-end to prove interfaces compose. The skeleton runs before any real backend exists.
- **Stubs at every boundary.** Mirage has MockBuilder. Troupe has MockActor. Substrate has StubSubstrate. No exceptions.
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

Use `djinnlog.KeyX` constants for ALL slog field keys. `djinnlog.For(log, "component")` for scoped loggers.
