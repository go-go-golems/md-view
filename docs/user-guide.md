# md-view User Guide

Everything you need to know about md-view — commands, flags, rendering, integration, and troubleshooting.

---

## Table of Contents

- [Overview](#overview)
- [Commands](#commands)
  - [view](#view)
  - [serve](#serve)
  - [status](#status)
  - [stop](#stop)
- [Rendering](#rendering)
  - [Markdown Features](#markdown-features)
  - [Syntax Highlighting](#syntax-highlighting)
  - [YAML Frontmatter](#yaml-frontmatter)
  - [Page Titles](#page-titles)
- [Live Reload](#live-reload)
- [Daemon Management](#daemon-management)
  - [How the Daemon Starts](#how-the-daemon-starts)
  - [State Files](#state-files)
  - [Stale PID Files](#stale-pid-files)
- [Browser Integration](#browser-integration)
  - [Browser Selection](#browser-selection)
  - [New Window Behavior](#new-window-behavior)
  - [i3 / Sway Integration](#i3--sway-integration)
- [HTTP API](#http-api)
  - [Render Endpoint](#render-endpoint)
  - [Raw Endpoint](#raw-endpoint)
  - [Static Assets](#static-assets)
  - [SSE Events Endpoint](#sse-events-endpoint)
- [Unix Socket Protocol](#unix-socket-protocol)
- [Security](#security)
- [Troubleshooting](#troubleshooting)

---

## Overview

md-view is a background daemon + CLI combo. The daemon serves rendered Markdown over HTTP on `localhost`. The CLI sends commands to the daemon over a Unix domain socket. You typically only interact with the CLI — the daemon starts and stops automatically.

```
┌──────────────┐   Unix Socket    ┌──────────────────┐    HTTP     ┌─────────┐
│  md-view CLI │ ─── JSON cmd ──► │  md-view server  │ ─────────► │ Browser │
│  (ephemeral) │                  │  (daemon)        │            │         │
└──────────────┘                  │                  │            └─────────┘
                                  │  - Renders .md   │
                                  │  - Serves HTML   │
                                  │  - Watches files │
                                  └──────────────────┘
```

---

## Commands

### view

```bash
md-view view <FILE> [flags]
```

The primary command. Opens a Markdown file in your browser as rendered HTML.

**What it does:**

1. Resolves the file path to an absolute path
2. Checks if the daemon is running
3. If not, starts the daemon in the background and waits for it to be ready
4. Sends a `view` command over the Unix socket
5. The daemon opens a new browser window on the rendered page
6. The CLI exits — you're done

**Arguments:**

| Argument | Required | Description |
|----------|----------|-------------|
| `FILE` | Yes | Path to the Markdown file to view (relative or absolute) |

**Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--browser` | string | `$BROWSER` or auto-detect | Override the browser command |
| `--no-reload` | bool | false | Disable live reload for this view |
| `--port` | int | 0 | HTTP port for the daemon (0 = random available) |

**Examples:**

```bash
# View a file (simplest usage)
md-view view ./README.md

# View without live reload
md-view view --no-reload ./notes.md

# Force Firefox
md-view view --browser firefox ./doc.md

# Use a specific port (useful for firewalls)
md-view view --port 8080 ./doc.md
```

**Output:**

The CLI outputs the rendered URL in a Glazed table:

```
+---------------------------------------------------------+----------------------+
| url                                                     | file                 |
+---------------------------------------------------------+----------------------+
| http://localhost:42213/render?file=/home/you/README.md  | /home/you/README.md  |
+---------------------------------------------------------+----------------------+
```

You can use Glazed output flags to change the format:

```bash
md-view view --output json ./README.md
md-view view --select url ./README.md
```

---

### serve

```bash
md-view serve [flags]
```

Start the HTTP server in the foreground. Normally called internally by the daemon, but useful for debugging.

**Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--port` | int | 0 | HTTP port (0 = random available) |

**Examples:**

```bash
# Start on a random port
md-view serve

# Start on a fixed port
md-view serve --port 8080
```

Press `Ctrl+C` to stop the server gracefully.

---

### status

```bash
md-view status
```

Show whether the daemon is running, its PID, HTTP port, uptime, and socket path.

**Output:**

```
+---------+-------+-------+--------+------------------------------------------------+
| running | pid   | port  | uptime | socket                                         |
+---------+-------+-------+--------+------------------------------------------------+
| true    | 23461 | 42213 | 3s     | /home/you/.local/state/md-view/md-view.sock    |
+---------+-------+-------+--------+------------------------------------------------+
```

When the daemon is not running:

```
+---------+-----+------+--------+------------------------------------------------+
| running | pid | port | uptime | socket                                         |
+---------+-----+------+--------+------------------------------------------------+
| false   | 0   | 0    |        | /home/you/.local/state/md-view/md-view.sock    |
+---------+-----+------+--------+------------------------------------------------+
```

---

### stop

```bash
md-view stop
```

Stop the daemon by sending SIGTERM. If the process doesn't exit within 5 seconds, it's force-killed. All state files (PID, socket, port) are cleaned up.

---

## Rendering

### Markdown Features

md-view uses [goldmark](https://github.com/yuin/goldmark) with the GFM (GitHub-Flavored Markdown) extension. Supported features:

| Feature | Syntax | Example |
|---------|--------|---------|
| Tables | GFM pipe tables | `| A | B |` |
| Task lists | `- [ ]` / `- [x]` | `- [x] Done` |
| Strikethrough | `~~text~~` | `~~removed~~` |
| Fenced code blocks | ` ``` ` with language hint | ` ```go ` |
| Autolinks | Bare URLs | `https://example.com` |
| Hard wraps | End line with two spaces | Text↵↵next line |

### Syntax Highlighting

Code blocks are syntax-highlighted server-side using [Chroma](https://github.com/alecthomas/chroma) with the `github` style. Over 200 languages are supported — just add the language name after the backticks:

```
```python
def hello():
    print("Hello, md-view!")
```

```go
func main() {
    fmt.Println("Hello, md-view!")
```

```javascript
console.log("Hello, md-view!");
```
```

No JavaScript is required — highlighting is done entirely on the server.

### YAML Frontmatter

If your Markdown file starts with YAML frontmatter (delimited by `---`), md-view:

1. **Strips it** from the rendered body
2. **Displays it** as a collapsible key-value table at the top of the page
3. **Uses the `Title` field** as the browser tab title (if present)

Example frontmatter:

```yaml
---
Title: API Reference
Status: draft
Topics:
  - backend
  - api
---
```

The frontmatter appears as a collapsed `▶ Frontmatter` section. Click it to expand. Each key is on the left; each value is on the right. Nested values (lists, maps) are displayed as formatted text.

### Page Titles

The browser tab title is determined in this order:

1. **Frontmatter `Title`** — if the file has a `Title:` field in its frontmatter
2. **Filename** — the basename of the file (e.g., `README.md`)

All titles are prefixed with `md-view: ` for window manager matching. Examples:

| File | Frontmatter Title | Browser Tab Title |
|------|------------------|-------------------|
| `README.md` | (none) | `md-view: README.md` |
| `01-diary.md` | `Diary` | `md-view: Diary` |
| `api.md` | `API Reference` | `md-view: API Reference` |

---

## Live Reload

When you view a file, md-view watches it for changes using [fsnotify](https://github.com/fsnotify/fsnotify). When the file is saved, the browser page reloads automatically within about a second.

**How it works:**

1. The rendered page includes a small JavaScript snippet that opens a Server-Sent Events (SSE) connection to `/events?file=<path>`
2. The server watches the file via fsnotify
3. On a `Write` event, the server pushes a `reload` event to all connected SSE clients
4. The client JavaScript calls `location.reload()`

**Disable live reload:**

```bash
md-view view --no-reload ./final-draft.md
```

**Limitations:**

- If the file is deleted and recreated (e.g., `git checkout`), the watch may be lost. Stop and restart the daemon to re-establish it.
- The watcher monitors the file itself, not the directory. Some editors that write to a temp file and rename may not trigger a reload.

---

## Daemon Management

### How the Daemon Starts

When you run `md-view view file.md`:

1. The CLI checks for a PID file at `~/.local/state/md-view/md-view.pid`
2. If the PID file exists and the process is alive, the daemon is already running
3. If not, the CLI starts a new daemon: `exec.Command(os.Args[0], "serve", ...)`
4. The daemon starts in the background (detached from the terminal via `Setpgid`)
5. The CLI waits for the socket file to appear (up to 5 seconds)
6. The CLI sends the `view` command and exits

### State Files

All runtime state lives in `~/.local/state/md-view/` (respects `$XDG_STATE_HOME`):

| File | Purpose |
|------|---------|
| `md-view.pid` | Daemon process ID |
| `md-view.port` | HTTP port the daemon is listening on |
| `md-view.sock` | Unix domain socket for CLI↔daemon IPC |

You can safely delete these files when the daemon is not running.

### Stale PID Files

If the daemon crashes without cleaning up (e.g., `kill -9`), the PID file may be stale. md-view handles this automatically:

- When `view` or `status` detects a stale PID (process not alive), it removes the state files and starts a fresh daemon
- You can also manually clean up: `rm ~/.local/state/md-view/md-view.*`

---

## Browser Integration

### Browser Selection

md-view selects the browser in this order:

1. `--browser` flag (if specified)
2. `$BROWSER` environment variable
3. Auto-detection: tries `xdg-open`, `firefox`, `google-chrome`, `chromium`

### New Window Behavior

md-view always opens a **new browser window** (not a tab). This is important for window manager integration — a new window can be floated, moved, or assigned to a workspace independently.

For Firefox, this uses `--new-window`. For Chrome/Chromium, this uses `--new-window`.

### i3 / Sway Integration

All md-view browser windows have titles starting with `md-view:`. Add this to your i3 or Sway config:

```
for_window [title="^md-view:.*"] floating enable
```

After `i3-msg reload` (or `swaymsg reload`), every `md-view view` will open as a floating window.

**Advanced: resize and position**

```
for_window [title="^md-view:.*"] floating enable, resize set 960 800, move position center
```

**Advanced: assign to a specific workspace**

```
for_window [title="^md-view:.*"] move container to workspace $ws3
```

---

## HTTP API

The daemon serves HTTP on `http://127.0.0.1:<PORT>/`. All endpoints are localhost-only.

### Render Endpoint

```
GET /render?file=<absolute_path>
```

Render a Markdown file as styled HTML. Returns `text/html`.

**Example:**

```bash
curl "http://localhost:42213/render?file=/home/you/README.md"
```

**Error responses:**

- `400` — Missing `file` parameter, invalid path, or not a regular file
- `404` — File not found
- `500` — Render error (malformed Markdown, etc.)

Error pages are styled HTML with a large status code, heading, and contextual message.

### Raw Endpoint

```
GET /raw?file=<absolute_path>
```

Serve the raw Markdown source. Returns `text/plain`.

**Example:**

```bash
curl "http://localhost:42213/raw?file=/home/you/README.md"
```

### Static Assets

```
GET /static/base.css      — GitHub-flavored Markdown CSS
GET /static/reload.js      — SSE client for live reload
GET /favicon.ico           — Returns 204 No Content
```

### SSE Events Endpoint

```
GET /events?file=<absolute_path>
```

Server-Sent Events endpoint for live reload. The server pushes `event: reload` when the file changes.

**Event format:**

```
event: reload
data: reload
```

**Browser client (built into md-view):**

```javascript
var es = new EventSource("/events?file=/path/to/file.md");
es.addEventListener("reload", function() { location.reload(); });
```

---

## Unix Socket Protocol

The daemon listens on a Unix domain socket at `~/.local/state/md-view/md-view.sock`. Messages are newline-delimited JSON (NDJSON).

**Client → Server:**

| Command | JSON | Description |
|---------|------|-------------|
| `view` | `{"command": "view", "path": "/abs/path/to/file.md"}` | Open a file in the browser |
| `ping` | `{"command": "ping"}` | Check if the daemon is alive |
| `stop` | `{"command": "stop"}` | Shut down the daemon |

**Server → Client:**

| Status | JSON | Description |
|--------|------|-------------|
| `ok` | `{"status": "ok", "url": "http://localhost:PORT/render?file=..."}` | Success response to `view` |
| `pong` | `{"status": "pong"}` | Response to `ping` |
| `error` | `{"status": "error", "message": "..."}` | Error response |

**Manual socket interaction (debugging):**

```bash
echo '{"command":"ping"}' | socat - UNIX-CONNECT:$HOME/.local/state/md-view/md-view.sock
```

---

## Security

md-view is designed as a single-user local tool. Security measures:

- **Localhost only** — HTTP server binds to `127.0.0.1`. No external access.
- **Socket permissions** — Unix socket is `0600` (owner only).
- **Path validation** — Only regular files can be rendered. No directory traversal, no symlinks to `/proc`.
- **No authentication** — This is intentional. If you can access localhost, you can view files. Don't expose the port.

---

## Troubleshooting

### "daemon did not start"

The CLI waited 5 seconds for the socket to appear but it didn't. Check:

```bash
# Is the process running?
ps aux | grep md-view

# Is there a stale PID file?
cat ~/.local/state/md-view/md-view.pid
# If the PID doesn't match a running process, remove it:
rm ~/.local/state/md-view/md-view.*

# Try starting in foreground to see errors:
md-view serve --port 18765
```

### "bind: address already in use"

A previous daemon didn't shut down cleanly. Stop it and restart:

```bash
md-view stop
md-view view ./README.md
```

### Browser doesn't open

Check that `$BROWSER` is set or that one of the supported browsers is installed:

```bash
which firefox xdg-open google-chrome chromium
```

Try explicitly:

```bash
md-view view --browser firefox ./README.md
```

### Live reload doesn't work

Some editors (especially those that write to a temp file and rename) may not trigger fsnotify. Try:

- Saving again
- Stopping and restarting the daemon
- Using `--no-reload` and manually refreshing

### Port conflicts

If you're running multiple instances, use `--port`:

```bash
md-view view --port 8080 ./README.md
```

### Multiple daemons

Only one daemon should be running at a time. Check with `md-view status`. If you see unexpected behavior, stop and restart:

```bash
md-view stop
md-view view ./README.md
```

---

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `$BROWSER` | (auto-detect) | Browser command to use |
| `$XDG_STATE_HOME` | `~/.local/state` | Base directory for state files |

---

## Dependencies

| Package | Purpose |
|---------|---------|
| [goldmark](https://github.com/yuin/goldmark) | Markdown → HTML conversion |
| [chroma](https://github.com/alecthomas/chroma) | Syntax highlighting |
| [fsnotify](https://github.com/fsnotify/fsnotify) | File watching for live reload |
| [Glazed](https://github.com/go-go-golems/glazed) | CLI framework (Cobra + structured output) |
