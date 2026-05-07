# md-view

A lightweight markdown viewer daemon that renders `.md` files as beautiful HTML and opens them in your browser.

## Quick Start

```bash
# Install
go install github.com/go-go-golems/md-view/cmd/md-view@latest

# View a markdown file (starts daemon automatically)
md-view view ./README.md

# Check daemon status
md-view status

# Stop the daemon
md-view stop
```

## Commands

| Command | Description |
|---------|-------------|
| `md-view view <FILE>` | View a markdown file in your browser |
| `md-view serve` | Start the server in foreground (for debugging) |
| `md-view status` | Show daemon status (PID, port, uptime) |
| `md-view stop` | Stop the running daemon |

## Features

- **Single command**: `md-view view file.md` — that's it
- **Auto-daemon**: Server starts automatically, runs in background
- **GitHub-flavored rendering**: Tables, task lists, fenced code blocks, strikethrough
- **Syntax highlighting**: Server-side via Chroma (no JS required)
- **Live reload**: Auto-refreshes the page when the file changes (SSE)
- **Unix socket IPC**: CLI communicates with daemon over a local socket
- **Zero config**: Random port, XDG state directory, respects `$BROWSER`

## How It Works

1. `md-view view file.md` checks if the daemon is running
2. If not, starts it in the background
3. Sends a `view` command over a Unix domain socket
4. Daemon opens your browser on `http://localhost:PORT/render?file=...`
5. CLI exits immediately — the daemon keeps running

## Architecture

```
┌──────────────┐    Unix Socket     ┌──────────────────┐    HTTP      ┌─────────┐
│  md-view CLI │ ──── JSON cmd ───► │  md-view server  │ ──────────► │ Firefox │
│  (ephemeral) │                    │  (daemon)        │             │         │
└──────────────┘                    │  - Renders .md   │             └─────────┘
                                    │  - Serves HTML   │
                                    │  - Opens browser │
                                    └──────────────────┘
```

## Options

```
md-view view <FILE> [options]

Options:
  --browser BROWSER   Override browser command
  --no-reload         Disable live reload
  --port PORT         HTTP port for daemon (0 = random)

md-view serve [options]

Options:
  --port PORT         HTTP port (0 = random)
```

## State Files

Runtime state lives in `~/.local/state/md-view/`:

- `md-view.pid` — Daemon PID
- `md-view.port` — HTTP port
- `md-view.sock` — Unix domain socket

## Dependencies

- [goldmark](https://github.com/yuin/goldmark) — Markdown rendering
- [chroma](https://github.com/alecthomas/chroma) — Syntax highlighting
- [fsnotify](https://github.com/fsnotify/fsnotify) — File watching for live reload
- [Glazed](https://github.com/go-go-golems/glazed) — CLI framework
