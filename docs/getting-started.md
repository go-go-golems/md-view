# Getting Started

Welcome to md-view — the fastest way to view a Markdown file in your browser.

## Install

**Option 1: Go install (recommended)**

```bash
go install github.com/go-go-golems/md-view/cmd/md-view@latest
```

**Option 2: Build from source**

```bash
git clone https://github.com/go-go-golems/md-view.git
cd md-view
make build
sudo cp md-view /usr/local/bin/
```

**Verify it works:**

```bash
md-view --help
```

## Your First View

```bash
md-view view ./README.md
```

What happens:

1. md-view checks if the background daemon is running
2. If not, it starts one automatically
3. Your browser opens a new window showing the rendered Markdown
4. The CLI exits — the daemon keeps running in the background

The browser title is `md-view: README.md` — handy for window manager matching.

## Live Reload

Keep the browser open. Edit the file in your editor. Save it.

The page refreshes automatically within a second. No reload button, no manual refresh — just edit and watch.

To disable live reload for a specific file:

```bash
md-view view --no-reload ./README.md
```

## View Multiple Files

Each `md-view view` opens a new browser window:

```bash
md-view view ./README.md
md-view view ./CHANGELOG.md
md-view view ./docs/api.md
```

All served by the same daemon. No extra processes.

## Choose Your Browser

md-view respects the `$BROWSER` environment variable. If that's not set, it tries `xdg-open`, `firefox`, `google-chrome`, and `chromium` in order.

Override it for a single command:

```bash
md-view view --browser firefox ./notes.md
```

Or set it globally:

```bash
export BROWSER=firefox
md-view view ./notes.md
```

## Check What's Running

```bash
md-view status
```

Output:

```
+---------+-------+-------+--------+------------------------------------------------+
| running | pid   | port  | uptime | socket                                         |
+---------+-------+-------+--------+------------------------------------------------+
| true    | 23461 | 42213 | 3s     | /home/you/.local/state/md-view/md-view.sock    |
+---------+-------+-------+--------+------------------------------------------------+
```

## Stop the Daemon

```bash
md-view stop
```

The daemon cleans up its PID file, socket, and port file on exit.

## What's Next?

- Read the **[User Guide](user-guide.md)** for all commands, flags, i3/Sway integration, and troubleshooting
- Try viewing a file with YAML frontmatter — md-view parses it into a collapsible table
- Set up a floating window rule in your window manager

## Quick Reference

```
md-view view <FILE>           # View a file (auto-starts daemon)
md-view view --no-reload FILE # View without live reload
md-view view --browser firefox FILE  # Use Firefox
md-view view --port 8080 FILE # Use a specific port
md-view serve                  # Start server in foreground
md-view status                 # Show daemon status
md-view stop                   # Stop the daemon
```
