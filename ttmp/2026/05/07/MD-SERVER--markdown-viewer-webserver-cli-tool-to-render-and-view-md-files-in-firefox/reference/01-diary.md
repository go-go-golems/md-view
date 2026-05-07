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

---

## Step 3: Polish — Frontmatter Formatting, Error Pages, Browser Window, Tests

Added UX polish: frontmatter as key-value table, page title from frontmatter Title field, styled error pages, --new-window browser launch, and unit tests.

### Prompt Context

**User prompt (verbatim):** (see Step 2) — continued working on polish items

**Assistant interpretation:** Continue improving the tool based on the design guide's Step 8 (polish & edge cases).

**Inferred user intent:** Make the viewer feel professional and handle edge cases well.

**Commit (code):** `002c5e3` — frontmatter as collapsible details
**Commit (code):** `38da3a9` — --new-window browser launch
**Commit (code):** `bad68fd` — frontmatter key-value table, error pages, tests

### What I did

- Stripped YAML frontmatter from rendered body, displayed as collapsible `<details>` element
- Upgraded frontmatter display from raw YAML to formatted key-value table (Title, Ticket, Status, etc. as separate rows)
- Used frontmatter `Title` field as the page `<title>` when available (falls back to filename)
- Changed browser launch from `--new-tab` to `--new-window` for i3 floating window support
- Added styled HTML error pages (400, 404, 500) with large status code, heading, and contextual message
- Added favicon.ico handler (204 No Content)
- Added `md-view:` prefix to all page titles for i3 window matching
- Added Makefile with build/test/lint/dev/install targets
- Added unit tests: renderer (frontmatter extraction, rendering, frontmatter title), daemon (StateDir, IsAlive), protocol (Command, Response)

### Why

The frontmatter was rendering as an ugly blob. Error pages were plain text. Browser opened in a tab instead of a window. These are all quality-of-life improvements for daily use.

### What worked

- Frontmatter parsing with simple YAML key-value extraction is sufficient for docmgr frontmatter
- The key-value table display is much more readable than raw YAML
- `--new-window` for Firefox works perfectly with i3 floating rules
- Live reload verified working (edited file, browser auto-refreshed within 2 seconds)

### What didn't work

- Initial frontmatter display was raw YAML — too noisy, too tall
- `--new-tab` opened in existing Firefox window, couldn't be floated in i3
- Page title was always the filename, ignoring the human-readable Title in frontmatter

### What I learned

- Simple YAML parsing (colon-delimited key: value at top level only) is enough for docmgr frontmatter — no need for a full YAML parser dependency
- Firefox `--new-window` flag creates a separate window that i3 can manage independently
- `html.EscapeString` isn't in the stdlib — wrote a simple `htmlEscape` helper

### What was tricky to build

- Frontmatter nested values (lists like `Topics:`, maps like `RelatedFiles:`) — solved by collecting indented lines as the value when the top-level value is empty
- Making the page title come from frontmatter without overriding explicit `--title` flags — solved by leaving `opts.Title` empty and letting the renderer fall through frontmatter → filename

### What warrants a second pair of eyes

- The YAML parser is very simple — won't handle complex nesting or multi-line scalars beyond `key:\n  - item`
- The SSE `CloseNotify` deprecation warning in newer Go — may need updating

### What should be done in the future

- Add golangci-lint config
- Add GitHub Actions CI
- Consider adding a dark theme toggle
- Add `--title` flag to the view command for custom titles

### Code review instructions

- `pkg/renderer/renderer.go` — frontmatter extraction and formatting, title from frontmatter
- `pkg/server/server.go` — styled error pages, --new-window
- `pkg/renderer/renderer_test.go` — frontmatter and rendering tests
- `Makefile` — build targets

### Technical details

- Frontmatter keys rendered as `<span class="md-view-fm-key">`, values as `<span class="md-view-fm-value">`
- Error pages use inline CSS (not embedded base.css) for simplicity
- i3 rule: `for_window [title="^md-view:.*"] floating enable`

---

## Step 4: Documentation — README, Getting Started, User Guide

Wrote three documentation pieces: a concise README, a getting-started guide, and a comprehensive user guide.

### Prompt Context

**User prompt (verbatim):** "write a nice getting-started and a nice full user-guide, as well as a nice README.md for this project."

**Assistant interpretation:** Write three polished docs: README (project overview), getting-started (first steps), user guide (complete reference).

**Inferred user intent:** The project needs proper documentation for users to discover, install, and use md-view effectively.

**Commit (code):** `e184d1c` — "docs: add README, getting-started guide, and full user guide"

### What I did

- Rewrote README.md as a concise project overview: install, 30-second quick start, commands table, key features, architecture diagram, links to docs
- Created docs/getting-started.md: install, first view, live reload, multiple files, browser choice, status/stop, quick reference
- Created docs/user-guide.md: comprehensive guide with table of contents covering all 4 commands (view/serve/status/stop), rendering (GFM, syntax highlighting, frontmatter, page titles), live reload, daemon management, browser integration (i3/Sway floating rules), HTTP API reference, Unix socket protocol, security model, troubleshooting

### Why

A tool is only as good as its documentation. The README is the landing page. The getting-started gets people productive in under a minute. The user guide is the reference they come back to.

### What worked

- Writing the user guide forced me to document all the edge cases and features clearly
- The i3/Sway integration section is practical — includes floating, resizing, and workspace assignment
- The HTTP API and Unix socket protocol sections serve as both user docs and developer reference

### What didn't work

- Nothing blocked

### What I learned

- The user guide naturally grew to cover all the implicit knowledge that was only in the design doc

### What warrants a second pair of eyes

- The troubleshooting section — are the common issues actually common?

### What should be done in the future

- Add a `md-view help` topic for the getting-started and user-guide content (embed via Glazed help system)
- Add screenshots/GIFs to the README

### Code review instructions

- README.md, docs/getting-started.md, docs/user-guide.md
- Verify links between docs work

### Technical details

- User guide is ~15KB, covers 14 sections with table of contents
- All three docs use consistent formatting and examples

---

## Step 5: Mermaid Diagrams + Dark Theme

Added client-side Mermaid diagram rendering and a full dark theme with three activation methods.

### Prompt Context

**User prompt (verbatim):** "add mermaid diagram support, dark theme. Add tasks, then implement, commit at appropriate intervals"

**Assistant interpretation:** Add Mermaid rendering (client-side) and a dark theme (CSS + toggle) to md-view. Add docmgr tasks, implement, and commit.

**Inferred user intent:** Make md-view useful for technical documentation that uses Mermaid diagrams, and add dark mode for users who prefer it.

**Commit (code):** `edfb81a` — "feat: add Mermaid diagram rendering and dark theme"

### What I did

- Added mermaid.js (CDN) for client-side rendering of ```mermaid code blocks
- Created mermaid-init.js that detects `<code class="language-mermaid">` blocks, wraps them in `<div class="mermaid">`, and initializes mermaid
- Added full dark CSS (GitHub dark style) for body, code, tables, blockquotes, frontmatter, and theme toggle
- Added chroma dark overrides (dracula-style token colors)
- Added three ways to activate dark theme:
  - `--dark` CLI flag (appends `&theme=dark` to URL)
  - `?theme=dark` query parameter
  - Toggle button in top-right corner (🌙 Dark / ☀ Light)
- Theme persisted in localStorage so it survives page reloads
- Updated protocol to carry `dark` boolean in view commands
- Updated server to read `?theme=` query param and pass it to renderer
- Renderer now selects chroma style based on theme: "github" for light, "dracula" for dark
- Added `Dark` field to `ViewSettings`, `Options`, and `Command`

### Why

Technical documentation commonly uses Mermaid for architecture diagrams. Dark mode is a quality-of-life feature many developers expect.

### What worked

- Mermaid integration was straightforward — goldmark renders ```mermaid as `<code class="language-mermaid">`, mermaid.js detects these and renders SVG
- The dark CSS was largely a color inversion of the light theme
- The toggle button with localStorage persistence is a nice UX touch
- Three activation methods cover all use cases: CLI flag for scripting, URL param for direct links, toggle for ad-hoc switching

### What didn't work

- The chroma dark overrides are manual CSS — not auto-generated from the "dracula" style. This means some token types might be missing colors. A better approach would be to generate two full chroma CSS files (one for light, one for dark) and include both, selecting via `[data-theme="dark"]` prefix.

### What I learned

- mermaid.js needs `<div class="mermaid">` not `<code class="language-mermaid">` — the init script converts one to the other
- The dark theme is best implemented as CSS overrides on `[data-theme="dark"]` selectors, not a separate CSS file. This way the page only needs one CSS bundle.

### What was tricky to build

- The chroma style selection happens at render time, not at toggle time. When you toggle dark mode, the code highlighting colors don't change because chroma CSS was generated for the initial theme. A future improvement would be to include both chroma stylesheets and switch via CSS selectors.

### What warrants a second pair of eyes

- The mermaid.js CDN dependency — should we embed it? CDN is simpler but requires network. Embedding adds ~1MB to the binary.
- The chroma style switching at toggle time — currently only works on initial page load, not on toggle

### What should be done in the future

- Include both light and dark chroma CSS and switch via `[data-theme]` selectors so code highlighting changes on toggle
- Embed mermaid.js in the binary (or make it optional)
- Test Mermaid dark theme rendering (mermaid.initialize with theme: 'dark')

### Code review instructions

- `pkg/renderer/renderer.go` — theme-aware rendering, mermaid script injection
- `pkg/renderer/static/dark.css` — dark theme CSS
- `pkg/renderer/static/mermaid-init.js` — mermaid initialization
- `pkg/commands/view.go` — --dark flag
- `pkg/protocol/protocol.go` — dark field in Command

### Technical details

- Mermaid CDN: `https://cdn.jsdelivr.net/npm/mermaid@11/dist/mermaid.min.js`
- Dark chroma style: "dracula"
- Light chroma style: "github"
- Theme toggle: `<button class="md-view-theme-toggle">` with localStorage persistence
