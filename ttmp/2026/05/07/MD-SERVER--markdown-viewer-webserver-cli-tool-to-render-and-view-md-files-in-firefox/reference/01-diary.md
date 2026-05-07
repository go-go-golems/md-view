---
Title: Diary
Ticket: MD-SERVER
Status: active
Topics:
    - markdown
    - cli
    - web
    - unix-socket
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ttmp/2026/05/07/MD-SERVER--markdown-viewer-webserver-cli-tool-to-render-and-view-md-files-in-firefox/design-impl-guide/01-design-and-implementation-guide.md
      Note: Design doc is the primary artifact tracked by this diary
ExternalSources: []
Summary: Implementation diary for md-view markdown viewer daemon.
LastUpdated: 2026-05-07T00:00:00Z
WhatFor: Tracking design decisions, progress, and lessons learned
WhenToUse: Read before resuming work on MD-SERVER to understand prior context
---


# Diary

## Goal

Capture the design and implementation journey for md-view, a CLI-driven markdown viewer daemon that renders .md files as beautiful HTML and opens them in Firefox.

---

## Step 1: Ticket Creation & Design Document

Created the docmgr ticket MD-SERVER and wrote a comprehensive design + implementation guide. This step established the full architecture before any code was written.

### Prompt Context

**User prompt (verbatim):** "Create a new docmgr ticket to create a webserver that serves markdown files nicely rendered, running in the background and you pass it a path over unix socket or so and it opens firefox on the rendered md file. Basically a little markdown viewer I can use from the CLI to just "view" a MD file. Keep a diary as you work. Create a design + implementation guide and store it in the ticket."

**Assistant interpretation:** Create a docmgr ticket for a background markdown viewer webserver with CLI/Unix-socket control, write a full design+implementation guide, and maintain a diary.

**Inferred user intent:** Want a simple CLI tool (`md-view <file>`) that opens a nicely rendered markdown file in Firefox, powered by a background daemon. Design should be thought through before coding begins.

### What I did

- Added vocabulary entries: `markdown`, `cli`, `web`, `unix-socket`, `design-impl-guide`
- Created ticket `MD-SERVER` with topics `markdown,cli,web,unix-socket`
- Created two docs: "Design and Implementation Guide" (design-impl-guide) and "Diary" (reference)
- Wrote the full design doc covering:
  - Problem statement and high-level architecture
  - Component breakdown (daemon + CLI client)
  - Unix socket NDJSON protocol
  - HTTP routes (`/render`, `/raw`, `/events`)
  - Rendering pipeline (Python markdown + pymdown-extensions + Pygments)
  - Live reload via SSE
  - Daemonization (double-fork)
  - CLI interface (`md-view FILE`, `--stop`, `--status`)
  - Security model (localhost-only, path validation)
  - 8-step implementation checklist
  - Dependency summary and future enhancements

### Why

Design-first approach prevents rework and ensures the Unix-socket IPC, daemon lifecycle, and live reload are coherently designed before implementation.

### What worked

- docmgr vocabulary + ticket creation flow was smooth
- The architecture naturally splits into independent layers (renderer → HTTP server → socket IPC → daemon → CLI), enabling step-by-step implementation and testing

### What didn't work

- Nothing blocked. Clean design phase.

### What I learned

- The combination of Unix socket (for CLI↔daemon IPC) + HTTP (for browser access) + SSE (for live reload) is a clean three-protocol stack where each protocol serves exactly one purpose
- Python's stdlib covers almost everything needed — only 4 external packages required

### What was tricky to build

- Balancing simplicity vs. features: live reload could be left out of V1, but it's a natural fit for a markdown viewer and SSE is simpler than WebSocket
- Daemon lifecycle edge cases: stale PID files, socket cleanup on crash, port conflicts

### What warrants a second pair of eyes

- The path validation model (allow-by-default, block traversal) — is it too permissive for a local tool?
- Whether the random-port strategy needs a fallback range for firewall users

### What should be done in the future

- Implement the 8 steps from the design guide
- Add integration tests that exercise the full CLI→socket→HTTP→browser pipeline
- Consider adding a `--pipe` mode that reads stdin for pipeline use

### Code review instructions

- Start with `design-impl-guide/01-design-and-implementation-guide.md` — review the architecture, protocol, and implementation steps
- No code yet — pure design phase

### Technical details

- State dir: `~/.local/state/md-view/` (PID, socket, port files)
- Socket protocol: NDJSON over Unix domain socket (SOCK_STREAM)
- HTTP bind: `127.0.0.1` only, random port
- CSS: GitHub-flavored markdown theme (inline)
- Live reload: SSE with `inotify_simple` (Linux) or polling fallback
