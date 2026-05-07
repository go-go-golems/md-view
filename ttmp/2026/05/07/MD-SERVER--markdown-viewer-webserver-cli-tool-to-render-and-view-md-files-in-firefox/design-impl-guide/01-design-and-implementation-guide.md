---
Title: Design and Implementation Guide
Ticket: MD-SERVER
Status: active
Topics:
    - markdown
    - cli
    - web
    - unix-socket
    - go
DocType: design-impl-guide
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ttmp/2026/05/07/MD-SERVER--markdown-viewer-webserver-cli-tool-to-render-and-view-md-files-in-firefox/reference/01-diary.md
      Note: Diary tracks design evolution and implementation progress
ExternalSources: []
Summary: "Architecture, design decisions, and step-by-step implementation guide for md-view, a Go CLI-driven markdown viewer daemon using the Glazed command framework."
LastUpdated: 2026-05-07
WhatFor: "Building the md-view markdown viewer from scratch"
WhenToUse: "Reference this doc when implementing, debugging, or extending md-view"
---

# md-view — Design & Implementation Guide

## 1. Problem Statement

You have markdown files scattered across the filesystem. You want to quickly view one rendered nicely in a browser — no copying, no ad-hoc `python -m http.server`, no opening VS Code just to preview. You want:

```
md-view ./notes/meeting.md
```

And Firefox opens on a beautifully rendered page. Done.

## 2. High-Level Design

**md-view** is a Go daemon + CLI combo built on the Glazed command framework:

| Component | Role |
|-----------|------|
| **Server (daemon)** | Background process. Serves rendered markdown over HTTP on `localhost`. Listens on a Unix domain socket for IPC commands. |
| **CLI client** | Glazed-powered Cobra commands. Sends a `view` command (file path) over the Unix socket. Auto-starts the daemon if not running. |

### 2.1 Lifecycle

```
┌──────────────┐    Unix Socket     ┌──────────────────┐    HTTP      ┌─────────┐
│  md-view CLI │ ──── JSON cmd ───► │  md-view server  │ ──────────► │ Firefox │
│  (ephemeral) │                    │  (daemon)        │             │         │
└──────────────┘                    │                  │             └─────────┘
                                    │  - Renders .md   │
                                    │  - Serves HTML   │
                                    │  - Opens browser │
                                    └──────────────────┘
```

1. User runs `md-view view ./notes/meeting.md`
2. CLI checks if daemon is alive (via PID file + socket probe)
3. If not alive → start daemon in background, wait for socket to appear
4. CLI sends `{"command": "view", "path": "/abs/path/to/file.md"}` over Unix socket
5. Server receives command, resolves path, opens Firefox at `http://localhost:PORT/render?file=...`
6. CLI exits immediately (fire-and-forget)

### 2.2 Key Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Language | **Go** | Single binary, fast startup, excellent stdlib for HTTP + Unix sockets. |
| CLI framework | **Glazed + Cobra** | Structured output, consistent flag handling, help system, logging. go-go-golems standard. |
| Markdown renderer | **goldmark** (with goldmark-highlighting) | Fast, CommonMark compliant, extensible. Go-native. |
| Syntax highlighting | **chroma** (via goldmark-highlighting) | Go-native Pygments alternative. Wide language support. |
| CSS theme | **GitHub-flavored** (inline CSS) | Familiar, readable, zero-dependency. |
| IPC | **Unix domain socket** (SOCK_STREAM) | Simple, local-only, no port conflicts, natural for CLI↔daemon. |
| Browser launch | **`exec.Command` → `xdg-open` / `$BROWSER`** | Simple, cross-desktop Linux. |
| Port | **Random available port** (or `--port`) | Avoids conflicts. Daemon writes port to a file next to PID file. |
| State directory | **`~/.local/state/md-view/`** | PID file, socket path, port file. XDG-compliant. |
| Live reload | **Server-Sent Events (SSE)** | Server watches the file with `fsnotify`. On change, pushes `reload` event. Client JS reconnects automatically. No WebSocket complexity. |

## 3. Architecture Deep-Dive

### 3.1 Directory Layout

```
md-view/
├── cmd/
│   └── md-view/
│       └── main.go              # Root cobra command, Glazed init, help system
├── pkg/
│   ├── commands/
│   │   ├── view.go              # `md-view view <file>` — Glazed command
│   │   ├── serve.go             # `md-view serve` — foreground server (Glazed command)
│   │   ├── stop.go              # `md-view stop` — stop the daemon
│   │   └── status.go            # `md-view status` — show daemon status
│   ├── server/
│   │   ├── server.go            # HTTP server + Unix socket listener
│   │   ├── handler.go           # HTTP routes (/render, /raw, /static/*, /events)
│   │   └── socket.go            # Unix socket handler (view, ping, stop)
│   ├── renderer/
│   │   ├── renderer.go          # Markdown → HTML (goldmark + chroma)
│   │   └── static.go            # Embed base.css + reload.js
│   ├── daemon/
│   │   └── daemon.go            # Daemon start/stop/status, PID file, state dir
│   ├── protocol/
│   │   └── protocol.go          # Socket message types (JSON wire format)
│   └── watcher/
│       └── watcher.go            # File watcher (fsnotify)
├── pkg/static/
│   ├── base.css                 # GitHub-flavored markdown CSS
│   └── reload.js                # SSE client for live reload
├── doc/
│   └── ...                      # Embedded help docs for Glazed help system
├── go.mod
├── go.sum
├── pyproject.toml               # (removed — was Python)
└── README.md
```

### 3.2 State Files

All runtime state lives in `~/.local/state/md-view/`:

| File | Purpose |
|------|---------|
| `md-view.pid` | Daemon PID (for `stop` and liveness check) |
| `md-view.sock` | Unix domain socket path |
| `md-view.port` | HTTP port the daemon bound to |

### 3.3 Unix Socket Protocol

Messages are newline-delimited JSON (NDJSON). Each message is one line.

**Client → Server:**

```json
{"command": "view", "path": "/abs/path/to/file.md"}
{"command": "stop"}
{"command": "ping"}
```

**Server → Client:**

```json
{"status": "ok", "url": "http://localhost:42137/render?file=/abs/path/to/file.md"}
{"status": "error", "message": "File not found: /nope.md"}
{"status": "pong"}
```

### 3.4 HTTP Routes

| Route | Purpose |
|-------|---------|
| `GET /render?file=<abs_path>` | Render a markdown file as styled HTML |
| `GET /raw?file=<abs_path>` | Serve the raw `.md` source (for debugging) |
| `GET /static/base.css` | GitHub-flavored CSS |
| `GET /static/reload.js` | SSE client script |
| `GET /events?file=<abs_path>` | SSE endpoint for live reload on file changes |

### 3.5 Security Model

- **Bind to localhost only** (`127.0.0.1`). No external access.
- **Unix socket** is filesystem-local. Permissions: `0600` (owner only).
- **Path validation**: Resolve to absolute path via `filepath.Abs`. Reject if resolved path contains traversal or points to non-regular files.
- **No authentication** — this is a single-user local tool.

### 3.6 Rendering Pipeline

```
/path/to/file.md
    │
    ▼
Read file (UTF-8, os.ReadFile)
    │
    ▼
goldmark.New(
    goldmark.WithExtensions(
        extension.GFM,
        extension.TOC,
        highlighting.NewHighlighting(
            highlighting.WithStyle("github"),
            highlighting.WithFormatOptions(chromaHTML.WithClasses()),
        ),
    ),
)
    │
    ▼
HTML body
    │
    ▼
Wrap in <html><head>...</head><body>...</body></html>
  - Inline <style> with base.css content (embedded)
  - Inline <script> with reload.js content (embedded)
  - Inline chroma CSS classes for syntax highlighting
    │
    ▼
Full HTML page → HTTP response
```

### 3.7 Live Reload (SSE)

1. Client JS (injected into every rendered page) opens `EventSource` to `/events?file=<path>`.
2. Server-side watcher monitors the file using `fsnotify`.
3. On `Write` event, server pushes `data: reload\n\n` to all connected SSE clients for that file.
4. Client JS calls `location.reload()`.

### 3.8 Daemonization

Go doesn't have native `fork()` — we use a simpler approach:

1. `exec.Command(os.Args[0], "serve", ...)` with `Start()` (not `Run`) — child process runs in background.
2. Parent writes child PID to state file.
3. Child sets `os.Stdout`, etc. to `/dev/null` via `os.DevNull`.
4. On `stop`, send SIGTERM to the PID.

Alternative: use `cmd.ExtraFiles` to pass the socket FD, but for simplicity the child creates its own socket after the old one is cleaned up.

### 3.9 CLI Interface (Glazed Commands)

```
md-view view <FILE>             View a markdown file in Firefox (default command)
md-view serve [--port PORT]     Start the server in foreground
md-view stop                    Stop the running daemon
md-view status                  Show daemon status (PID, port, uptime)
```

The `view` command is the primary verb. The `serve` command is used internally for daemonization but can also be run directly for debugging.

## 4. Implementation Steps

### Step 1: Project Scaffold

- [ ] `go mod init github.com/go-go-golems/md-view`
- [ ] Create `cmd/md-view/main.go` with Glazed root command wiring
- [ ] Add dependencies: goldmark, chroma, cobra, glazed, fsnotify
- [ ] Verify: `go build ./...` and `md-view --help` works

### Step 2: Markdown Renderer

- [ ] Implement `pkg/renderer/renderer.go`: read `.md`, convert to styled HTML using goldmark + chroma
- [ ] Create `pkg/static/base.css` (GitHub-flavored theme)
- [ ] Embed CSS + JS via `go:embed`
- [ ] Test: render a sample `.md` file and verify HTML output

### Step 3: HTTP Server

- [ ] Implement `pkg/server/server.go` with `net/http`
- [ ] Routes: `/render`, `/raw`, `/static/*`
- [ ] Bind to `127.0.0.1` on random port
- [ ] Write port to state file
- [ ] Test: `curl "http://localhost:PORT/render?file=/tmp/test.md"` returns HTML

### Step 4: Unix Socket IPC

- [ ] Implement `pkg/protocol/protocol.go` with message types
- [ ] Add Unix socket listener in `pkg/server/socket.go` (goroutine, alongside HTTP)
- [ ] Handle `view`, `ping`, `stop` commands
- [ ] On `view`: resolve path → open browser via `xdg-open` / `$BROWSER`
- [ ] Test: send JSON over socket, verify browser opens

### Step 5: Daemon Management

- [ ] Implement `pkg/daemon/daemon.go` (start/stop/status, PID file, state dir)
- [ ] Start daemon via `exec.Command(os.Args[0], "serve", ...)`
- [ ] Graceful shutdown: trap SIGTERM/SIGINT, clean up socket + PID file
- [ ] Test: start daemon, verify PID file, `stop` cleans up

### Step 6: CLI Commands (Glazed)

- [ ] Implement `view` command: auto-start daemon, send view command over socket
- [ ] Implement `serve` command: foreground server (Glazed command)
- [ ] Implement `stop` command
- [ ] Implement `status` command
- [ ] Wire all commands in `main.go`
- [ ] Test: full end-to-end `md-view view ./test.md`

### Step 7: Live Reload (SSE)

- [ ] Implement `pkg/watcher/watcher.go` using `fsnotify`
- [ ] Add `/events` SSE route
- [ ] Create `pkg/static/reload.js` (EventSource + reload)
- [ ] Inject `<script>` into rendered pages
- [ ] Test: edit a viewed file, verify browser auto-refreshes

### Step 8: Polish & Edge Cases

- [ ] Path validation (reject traversal, non-regular files)
- [ ] Error pages (404 for missing file, 500 for render errors)
- [ ] Multi-file view support (multiple `view` commands → multiple tabs)
- [ ] Graceful handling of daemon crash (stale PID file → auto-restart)
- [ ] README with installation and usage

## 5. Dependency Summary

| Package | Purpose |
|---------|---------|
| `github.com/yuin/goldmark` | Core markdown → HTML conversion |
| `github.com/yuin/goldmark/extension` | GFM tables, fenced code, TOC |
| `github.com/yuin/goldmark/renderer/html` | HTML renderer |
| `github.com/alecthomas/chroma` | Syntax highlighting (Pygments-compatible) |
| `github.com/alecthomas/chroma/formatters/html` | HTML formatter for chroma |
| `github.com/alecthomas/chroma/styles` | Chroma style registry |
| `github.com/fsnotify/fsnotify` | Cross-platform file change watching |
| `github.com/go-go-golems/glazed` | CLI framework (Cobra + structured output) |
| `github.com/spf13/cobra` | Command framework (via Glazed) |

## 6. Future Enhancements (Out of Scope for V1)

- **Mermaid diagram rendering** (client-side JS)
- **PDF export** (via `wkhtmltopdf` or browser print)
- **Directory index** (browse folders of `.md` files)
- **Search** (full-text search across viewed files)
- **Multiple themes** (switchable via `?theme=dark`)
- **Remote browser control** (Chrome DevTools Protocol for tab management)
