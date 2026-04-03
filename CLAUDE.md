# Djinn — AI Agent Industrial Complex

## Philosophy: Flat Token Curve

Djinn invests in planning to eliminate rework. Every other agent CLI optimizes for speed to first edit — fast start, exponential rework, high waste. Djinn optimizes for cost of the tenth edit — slower start, flat waste rate, lower total cost.

The **Proposal Loop**: agent proposes (understanding, plan, risks), operator reviews via TUI, annotates, amends, accepts. The upfront negotiation IS the product. The code is just the output of a good plan.

Higher initial token cost. Flat waste rate. Lower total tokens over a longer time horizon.

## Architecture (66 components, 0 cycles, 170 edges)

### Three Spaces
- **Worker** = LLM agent (untrusted, stateless, teleportable, CAN DIE)
- **Vehicle** = Workstation (boots without LLM, persists across agent death, carries tools + scratch paper + claims)
- **Space** = Misbah jail (physical isolation — namespaces, containers, Kata)

### Key Systems (built in Sprints 1-4)
- **CellSight** (SPC-85): agent sees what operator sees — 6 panels implement Sighted
- **SightManager**: operator controls what agent can see — :sight on/off/reveal/hide
- **SymbolGraph** (SPC-109): pre-edit caller impact — RegexProvider (Day 1a), ASTProvider (Day 1b Go), LSPProvider (Day 2)
- **HookRunner**: shell command interception at tool boundaries (pre/post_tool_use)
- **WasteClassifier** (SPC-105): 7 Lean waste types for agent tool calls
- **Andon** (SPC-106): TPS stop-the-line with AlertQueue + auto-cordon on critical
- **Workstation** (SPC-111): persistent agent-independent execution environment with scratch paper
- **VFS** (SPC-95): virtual filesystem with MountTable + path translation + escape detection
- **MCP Wiring**: auto-connect ecosystem tools, CompositeExecutor with tool upgrade path
- **AgentAccountability**: extensible compliance metrics per MetricKind

### Pending Systems (designed, not built)
- **Dynamic Gating** (SPC-101): three-stamp lifecycle — Deterministic > Sovereign > Stochastic
- **Proposal Loop**: agent-operator negotiation before execution (the flat curve mechanism)
- **File Kanban** (SPC-104): agent coordination via intent declaration + worktree isolation
- **Kanban Board** (SPC-103): TUI panel for artifact lifecycle visualization

## Day 0 / Day 1 / Day 2

- **Day 0**: binary boots, no config, no external tools
- **Day 1**: built-in tools work standalone (batteries included). Every feature MUST work Day 1.
- **Day 2**: MCP enrichment — Locus, Lex, Scribe upgrade built-ins. MCPs enhance, never gate.

## Dependency Rules

- Djinn -> Jericho (via jerichoport/ adapter). This is the ONLY Jericho import path.
- Djinn NEVER imports origami/ — use Bugle Protocol for circuit execution.
- Dependency direction: `Origami -> Jericho <- Djinn` (lateral via Bugle Protocol)
- Jericho v0.2.0 pinned. Misbah v0.15.0 pinned. No replace directives.

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
