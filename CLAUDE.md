# Claude Code Instructions for Djinn Development

## Multi-Repository Workspace

This is part of a multi-repository workspace:
- **Misbah** (`/home/dpopsuev/Workspace/misbah`) - Layer 1: Jail creation and namespace isolation
- **Djinn** (`/home/dpopsuev/Workspace/djinn`) - Layer 2+3: Agent runtime and CLI

**Primary instructions are in `/home/dpopsuev/Workspace/misbah/CLAUDE.md`** - read that file for full context.

## Djinn Overview

Djinn is an AI pair programming CLI (inspired by Aider) that runs LLM agents in isolated Misbah jails.

### Goal

DJN-GOL-1: "Djinn PoC - Aider-like AI Pair Programming CLI"

- Conversational coding sessions with LLM agents
- Multi-repository support
- Safe execution in Misbah jails
- Agent Runtime Interface (ARI) for jail orchestration

### Architecture

Djinn consists of:
- **Djinn Runtime** (Layer 2): Jail lifecycle management, ARI HTTP server, agent drivers (Claude, etc.)
- **Djinn CLI** (Layer 3): User-facing conversational interface

### Key Specifications (in Scribe)

- `DJN-SPC-2026-001`: Agent Runtime Interface (ARI) v1.0 - HTTP/gRPC API contract
- `DJN-SPC-2026-004`: Djinnfile Format Specification
- `DJN-SPC-2026-005`: Aider Architecture Case Study - Prior Art

### Current Status

Pre-implementation. Djinn will be built once Misbah Layer 1 PoC is complete.

## Working with Djinn

Use absolute paths to work across both repos:
```bash
# Djinn
/home/dpopsuev/Workspace/djinn/...

# Misbah (dependency)
/home/dpopsuev/Workspace/misbah/...
```

Consult Scribe for task details:
```
mcp__scribe__artifact list --query djinn
mcp__scribe__artifact get --id DJN-SPC-2026-001
```
