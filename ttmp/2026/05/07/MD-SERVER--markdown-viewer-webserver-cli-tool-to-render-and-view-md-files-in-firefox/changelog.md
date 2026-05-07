# Changelog

## 2026-05-07

- Initial workspace created


## 2026-05-07

Created ticket, wrote design+implementation guide, started diary. Architecture: Python daemon + CLI client, Unix socket IPC, HTTP rendering, SSE live reload.


## 2026-05-07

Implemented full md-view Go binary with Glazed CLI, goldmark renderer, HTTP+socket server, daemon management, and live reload. All 8 design steps completed and tested.

### Related Files

- /home/manuel/code/wesen/2026-05-07--md-server/cmd/md-view/main.go — Root command with Glazed wiring
- /home/manuel/code/wesen/2026-05-07--md-server/pkg/renderer/renderer.go — Markdown rendering with goldmark + chroma
- /home/manuel/code/wesen/2026-05-07--md-server/pkg/server/server.go — HTTP + Unix socket server


## 2026-05-07

Added frontmatter key-value table, page title from frontmatter, styled error pages, --new-window browser launch, unit tests, Makefile


## 2026-05-07

Wrote README, getting-started guide, and comprehensive user guide (15KB, 14 sections)


## 2026-05-07

Added Mermaid diagram rendering (mermaid.js CDN) and dark theme (CSS + chroma dracula + toggle button + --dark flag + ?theme=dark param)

