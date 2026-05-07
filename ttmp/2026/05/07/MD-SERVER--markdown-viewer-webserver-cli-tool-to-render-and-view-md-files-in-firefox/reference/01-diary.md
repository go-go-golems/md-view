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
    - Path: cmd/md-view/main.go
      Note: Root command
    - Path: pkg/commands/run.go
      Note: View/Serve/Stop implementation
    - Path: pkg/commands/view.go
      Note: View command
    - Path: pkg/daemon/daemon.go
      Note: Daemon management
    - Path: pkg/renderer/renderer.go
      Note: Markdown renderer
    - Path: pkg/server/server.go
      Note: HTTP + socket server
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

Capture the design and implementation journey for md-view, a Go CLI-driven markdown viewer daemon that renders .md files as beautiful HTML and opens them in Firefox.

---

## Step 1: Ticket Creation & Design Document

Created the docmgr ticket MD-SERVER and wrote a comprehensive design + implementation guide. This step established the full architecture before any code was written.

### Prompt Context

**User prompt (verbatim):** "Create a new docmgr ticket to create a webserver that serves markdown files nicely rendered, running in the background and you pass it a path over unix socket or so and it opens firefox on the rendered md file. Basically a little markdown viewer I can use from the CLI to just 'view' a MD file. Keep a diary as you work. Create a design + implementation guide and store it in the ticket."

**Assistant interpretation:** Create a docmgr ticket for a background markdown viewer webserver with CLI/Unix-socket control, write a full design+implementation guide, and maintain a diary.

**Inferred user intent:** Want a simple CLI tool (`md-view <file>`) that opens a nicely rendered markdown file in Firefox, powered by a background daemon. Design should be thought through before coding begins.

### What I did

- Added vocabulary entries: `markdown`, `cli`, `web`, `unix-socket`, `design-impl-guide`
- Created ticket `MD-SERVER` with topics `markdown,cli,web,unix-socket`
- Created two docs: "Design and Implementation Guide" (design-impl-guide) and "Diary" (reference)
- Wrote the full design doc covering architecture, protocol, rendering pipeline, daemonization, and 8 implementation steps

### Why

Design-first approach prevents rework and ensures the Unix-socket IPC, daemon lifecycle, and live reload are coherently designed before implementation.

### What worked

- docmgr vocabulary + ticket creation flow was smooth
- The architecture naturally splits into independent layers

### What didn't work

- Nothing blocked. Clean design phase.

### What I learned

- The combination of Unix socket (for CLI↔daemon IPC) + HTTP (for browser access) + SSE (for live reload) is a clean three-protocol stack

### What was tricky to build

- Balancing simplicity vs. features: live reload could be left out of V1, but it's a natural fit

### What warrants a second pair of eyes

- The path validation model (allow-by-default, block traversal)

### What should be done in the future

- Implement the 8 steps from the design guide

### Code review instructions

- Start with `design-impl-guide/01-design-and-implementation-guide.md`

### Technical details

- State dir: `~/.local/state/md-view/`
- Socket protocol: NDJSON over Unix domain socket (SOCK_STREAM)
- HTTP bind: `127.0.0.1` only, random port

---

## Step 2: Language Switch to Go + Full Implementation

The user requested switching from Python to Go, and using the Glazed command framework for the CLI. I updated the design doc and then implemented all 8 steps in one pass.

### Prompt Context

**User prompt (verbatim):** "actually use golang, update your document. Use glazed command framework to build the CLI verb."

**Assistant interpretation:** Rewrite the design doc for Go + Glazed, then implement the full md-view tool.

**Inferred user intent:** Want a Go binary with Glazed CLI conventions, not a Python script.

**Commit (code):** `0ef83ca` — "feat: scaffold Go project with Glazed CLI, renderer, server, daemon, watcher"

### What I did

- Rewrote the design doc for Go (goldmark, chroma, fsnotify, Glazed)
- Created project scaffold: `go mod init`, `cmd/md-view/main.go`, Glazed root command
- Implemented all 4 Glazed commands: `view`, `serve`, `stop`, `status`
- Built the renderer with goldmark + chroma highlighting (embedded CSS/JS)
- Built the HTTP server with `/render`, `/raw`, `/events` (SSE), `/static/*` routes
- Built the Unix socket IPC with NDJSON protocol (`view`, `ping`, `stop`)
- Built the daemon manager (PID file, state dir, start/stop/status)
- Built the file watcher with fsnotify for live reload
- End-to-end tested: daemon auto-start, curl rendering, socket ping, browser rendering
- Added README.md

### Why

Go gives us a single binary with fast startup. Glazed gives us structured output, logging, and help system out of the box.

### What worked

- Go build and Glazed wiring was straightforward — the skill doc patterns worked exactly
- goldmark + chroma integration produced beautiful rendering on first try
- The daemon auto-start pattern (exec self with "serve" subcommand) works cleanly on Linux
- All three protocols (HTTP, Unix socket, SSE) working correctly

### What didn't work

- Initial Python design had to be fully rewritten for Go
- chroma v1 vs v2 import path conflict with goldmark-highlighting (needed `chroma/v2`)
- goldmark doesn't have a `TOC` extension (removed from design)
- `go:embed` requires explicit `_ "embed"` import even when only using `//go:embed` directives on variables
- SSE client map key type: `Watch()` returns `<-chan struct{}` which can't be used as map key directly (Go channels are comparable but direction matters)

### What I learned

- goldmark-highlighting/v2 uses chroma/v2 internally — must import `github.com/alecthomas/chroma/v2/...` directly
- Go's `//go:embed` on `[]byte` variables requires `_ "embed"` import, even though the directive is on the variable
- `<-chan struct{}` is a valid map key type in Go (channels are comparable)
- `setsid` on Linux is done via `SysProcAttr{Setpgid: true}` not a separate setsid call

### What was tricky to build

- The chroma version mismatch: goldmark-highlighting/v2 depends on chroma/v2, but our direct imports were v1. Had to read the go.mod of the highlighting library to discover this.
- The `go:embed` + `import "embed"` requirement was non-obvious — the compiler error message is clear though.

### What warrants a second pair of eyes

- The daemon start pattern (`exec.Command(os.Args[0], "serve", ...)`) — is it robust across Go builds and installations?
- The SSE event loop — is the `CloseNotify` deprecation in newer Go a concern?
- The fsnotify watcher only monitors the file itself, not the directory — if the file is deleted and recreated (e.g. `git checkout`), the watch is lost.

### What should be done in the future

- Add a Makefile with lint/test/build targets (go-go-golems project setup)
- Add integration tests (start daemon, curl, verify HTML, stop)
- Add path validation (reject traversal, non-regular files)
- Add proper error pages (404, 500)
- Handle stale PID files more robustly
- Consider watching the parent directory instead of just the file for robustness

### Code review instructions

- Start with `cmd/md-view/main.go` — the Glazed root wiring
- Then `pkg/commands/view.go` + `pkg/commands/run.go` — the main user-facing command
- Then `pkg/server/server.go` — the core HTTP + socket server
- Then `pkg/renderer/renderer.go` — the markdown → HTML pipeline
- Verify with: `go build ./...` and `md-view view /tmp/test.md`

### Technical details

- Module: `github.com/go-go-golems/md-view`
- Key deps: goldmark v1.8.2, chroma/v2 v2.16.0, fsnotify v1.10.1, glazed v1.2.7
- Browser detection: checks `$BROWSER`, then tries `xdg-open`, `firefox`, `google-chrome`, `chromium`
- Chroma style: "github" (light theme matching the CSS)
