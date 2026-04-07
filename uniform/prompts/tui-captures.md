# Live TUI Captures — Claude Code, Codex, Gemini CLI
# Captured via tmux pane capture at 120x40 terminal

## Claude Code v2.1.81
```
 ▐▛███▜▌   Claude Code v2.1.81
▝▜█████▛▘  Opus 4.6 (1M context) with high effort · API Usage Billing
  ▘▘ ▝▝    ~/Workspace/djinn

────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────
❯ Try "refactor andon_test.go"
────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────
  -- INSERT --                                                                                            ● high · /effort
```

Notes:
- NO bordered welcome box (contrary to what I captured earlier — that was from a wider terminal)
- Compact: logo + version + model + billing on 3 lines
- CWD shown below logo
- Full-width thin separators (─) above and below input
- ❯ chevron with file-specific placeholder ("refactor andon_test.go" — context-aware!)
- vim mode "-- INSERT --" bottom left
- Effort indicator "● high · /effort" bottom right
- NO two-column layout at 120 chars — single column, left-aligned

## Codex CLI v0.98.0
```
  ✨ Update available! 0.98.0 -> 0.114.0

  Release notes: https://github.com/openai/codex/releases/latest

› 1. Update now (runs `npm install -g @openai/codex`)
  2. Skip
  3. Skip until next version

  Press enter to continue
```

Notes:
- Shows update prompt before TUI (blocks interaction)
- › arrow selector for menu items
- Minimal chrome — no borders, no logo
- Could not capture main TUI (blocked by update prompt)

## Gemini CLI v0.27.3
```
 ███            █████████  ██████████ ██████   ██████ █████ ██████   █████ █████
░░░███         ███░░░░░███░░███░░░░░█░░██████ ██████ ░░███ ░░██████ ░░███ ░░███
  ░░░███      ███     ░░░  ░███  █ ░  ░███░█████░███  ░███  ░███░███ ░███  ░███
    ░░░███   ░███          ░██████    ░███░░███ ░███  ░███  ░███░░███░███  ░███
     ███░    ░███    █████ ░███░░█    ░███ ░░░  ░███  ░███  ░███ ░░██████  ░███
   ███░      ░░███  ░░███  ░███ ░   █ ░███      ░███  ░███  ░███  ░░█████  ░███
 ███░         ░░█████████  ██████████ █████     █████ █████ █████  ░░█████ █████
░░░            ░░░░░░░░░  ░░░░░░░░░░ ░░░░░     ░░░░░ ░░░░░ ░░░░░    ░░░░░ ░░░░░

Tips for getting started:
1. Ask questions, edit files, or run commands.
2. Be specific for the best results.
3. Create GEMINI.md files to customize your interactions with Gemini.
4. /help for more information.

╭──────────────────────────────────────────────────────────────────────────────────────────────────────────────────────╮
│ Gemini CLI update available! 0.27.3 → 0.34.0                                                                       │
│ Installed with npm. Attempting to automatically update now...                                                        │
╰──────────────────────────────────────────────────────────────────────────────────────────────────────────────────────╯

╭──────────────────────────────────────────────────────────────────────────────────────────────────────────────────────╮
│ ? Get started                                                                                                        │
│   How would you like to authenticate for this project?                                                               │
│   ● 1. Login with Google                                                                                             │
│     2. Use Gemini API Key                                                                                            │
│     3. Vertex AI                                                                                                     │
│   (Use Enter to select)                                                                                              │
╰──────────────────────────────────────────────────────────────────────────────────────────────────────────────────────╯
```

Notes:
- HUGE ASCII art logo (8 lines, full width)
- Tips section: numbered list, not bordered
- Rounded bordered boxes (╭╮╰╯) for notifications and auth
- Multiple bordered sections stacked vertically
- Auth picker with ● radio buttons
- Update notification in bordered box
- No status bar visible (auth blocks it)

## Key Takeaways for Djinn

1. Claude Code is MORE minimal than I thought — no bordered welcome at 120 chars
2. Gemini is the most visually heavy — huge logo, multiple bordered sections
3. Codex blocks on update prompts — bad UX pattern to avoid
4. Context-aware placeholder ("refactor andon_test.go") is Claude's killer detail
5. All three have different philosophies:
   - Claude: minimal, vim-like, information-dense
   - Codex: invisible chrome, let the terminal be the UI
   - Gemini: React-style bordered components, visual hierarchy
6. Djinn's current TUI is closest to Claude Code's style
