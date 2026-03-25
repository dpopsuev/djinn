# TUI Reference Dump — Claude Code, Codex, Gemini CLI

## Installed Versions
- Claude Code: 2.1.81
- Codex CLI: 0.98.0
- Gemini CLI: 0.27.3

## Claude Code TUI (from DJN-DOC-10)
```
╭─── Claude Code v2.1.81 ──────────────────────────────╮
│                                    │ Tips             │
│        Welcome back!               │ Run /init        │
│           ▐▛███▜▌                  │ ────────────     │
│          ▝▜█████▛▘                 │ Recent activity  │
│            ▘▘ ▝▝                   │ No recent        │
│ Opus 4.6 (1M) · API Billing       │                  │
│       /home/dpopsuev               │                  │
╰──────────────────────────────────────────────────────╯

────────────────────────────────────────────────────────
❯ Try "refactor <filepath>"
────────────────────────────────────────────────────────
 -- INSERT --                          ● high · /effort
```

Key features:
- Rounded border (╭╮╰╯) with version in top border
- Two-column MOTD: logo left, tips right
- ❯ chevron prompt with placeholder text
- Thin separator lines above/below input
- Vim mode indicator bottom-left
- Effort selector bottom-right
- No colors on borders — clean monochrome structure

## Codex CLI Key Features (from --help)
- --no-alt-screen: inline mode for terminal multiplexers
- --sandbox: read-only, workspace-write, danger-full-access
- --full-auto: -a on-request + --sandbox workspace-write
- Subcommands: exec, review, resume, fork, cloud, debug, apply
- Config: TOML-based (~/.codex/config.toml), dotted path overrides
- MCP: experimental, both client and server mode

## Gemini CLI Key Features (from --help + DJN-DOC-11)
- React + Ink framework (JSX for terminal)
- --approval-mode: default, auto_edit, yolo, plan
- --screen-reader: accessibility mode
- Multi-size ASCII logo (responsive to terminal width)
- Themes: custom via settings.json
- Extensions + skills system
- Debug: F12 opens debug console
- --raw-output: allow ANSI escape codes (with security warning)

## Djinn TUI Current State
```
╭────────────── Djinn v0.2.0 ──────────────────────────╮
│                                                       │
│  ████  █████          ecosystem: aeon                 │
│ ███████████████       model:     claude-sonnet-4-6    │
│█████████████████      mode:      plan                 │
│ █████████████████     role:      gensec               │
│██████████████████     tools:     6 built-in           │
│  █████████████████                                    │
│   ████████████████    /help for commands              │
│         ████████████  Tab to complete slash commands   │
│                                                       │
╰──────────────────────────────────────────────────────╯
❯ Hello
djinn: Hello! What do you need?
[tokens: 0 in, 15 out]

╭────────────────────────────────────╮
│❯ Try "explain this codebase"      │
╰────────────────────────────────────╯
  -- GENSEC --                  ws:aeon │ model:claude-sonnet-4-6 │ mode:plan
```

## Gap Analysis: Djinn vs Claude Code vs Codex vs Gemini

| Feature | Claude | Codex | Gemini | Djinn |
|---------|--------|-------|--------|-------|
| Welcome banner | bordered two-col | minimal | multi-size logo | bordered two-col ✓ |
| Prompt character | ❯ | > | ❯ | ❯ ✓ |
| Placeholder | "Try refactor" | none | none | "Try explain" ✓ |
| Mode indicator | -- INSERT -- | none | none | -- GENSEC -- ✓ |
| Effort/mode toggle | /effort | --full-auto | --approval-mode | /role ✓ |
| Tab completion | slash commands | none | slash commands | slash commands ✓ |
| Alt-screen | yes | --no-alt-screen opt | yes | yes |
| Markdown streaming | yes | yes | buggy | completion-only |
| Theme system | none | TOML config | JSON settings | staff.yaml |
| Debug console | --debug | debug subcommand | F12 | --debug-tap + --live-debug ✓ |
| Session resume | --continue | resume/fork | --resume | -c / -s ✓ |
| MCP support | built-in | experimental | extensions | tool capabilities |
| Sandbox | none (hooks) | read-only/write/full | --sandbox | misbah (future) |
| Screen reader | none | none | --screen-reader | none |
| Role system | none | none | none | staff roles ✓ (unique) |
