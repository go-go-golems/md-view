# Sources — MDV-CLIPBOARD-001

Reference material collected for the macOS Command-C / clipboard investigation.
All external pages were converted to Markdown with `defuddle`; Wails internals are
copied verbatim from the pinned module cache (`github.com/wailsapp/wails/v2@v2.12.0`).

## External captures (defuddle)

| File | Source | Why it matters |
|------|--------|----------------|
| `01-wails-issue-4918-macos-clipboard-shortcuts.md` | https://github.com/wailsapp/wails/issues/4918 | Original bug report: a custom `Menu:` removes macOS clipboard shortcuts. |
| `02b-wails-issue-4918-maintainer-response.md` | Same issue, captured via GitHub REST API (comments are not in the rendered HTML) | Maintainer Leaanthony confirms the root cause and the Edit-menu fix. |
| `04-secure-context-navigator-clipboard.md` | https://github.com/water-rs/waterui/issues/592 | Custom WebView schemes are not secure contexts, so `navigator.clipboard` disappears. |

`wails.io` and `stackoverflow.com` returned HTTP 403 to `defuddle`; their relevant
facts are covered by the captures above plus the verbatim Wails source excerpts.

## Wails v2.12.0 source excerpts (verbatim)

| File | Upstream path | Why it matters |
|------|---------------|----------------|
| `06-wails-source-excerpts/menu-menuroles.go.txt` | `pkg/menu/menuroles.go` | Defines the exported `AppMenu()` / `EditMenu()` / `WindowMenu()` role helpers. |
| `06-wails-source-excerpts/menu-menu.go.txt` | `pkg/menu/menu.go` | `NewMenu`, `Append`, `AddSubmenu`, `Prepend` — the API `buildMenu` uses. |
| `06-wails-source-excerpts/darwin-window-UpdateApplicationMenu.go.txt` | `internal/frontend/desktop/darwin/window.go` | Proves the main menu is rebuilt from *only* the user-supplied menu; no defaults are injected. |
| `06-wails-source-excerpts/darwin-menu-processMenu.go.txt` | `internal/frontend/desktop/darwin/menu.go` | `processMenuItem` dispatches a non-zero `Role` to `AppendRole`; gaps in role handling. |
| `06-wails-source-excerpts/darwin-WailsMenu.m.txt` | `internal/frontend/desktop/darwin/WailsMenu.m` | The native Edit menu items use `@selector(copy:)` / `cut:` / `paste:` / `selectAll:` — the responder-chain actions macOS needs. |
| `06-wails-source-excerpts/darwin-clipboard.go.txt` | `internal/frontend/desktop/darwin/clipboard.go` | Native `NSPasteboard` read/write used by `runtime.ClipboardSetText`. |
| `06-wails-source-excerpts/runtime-clipboard.go.txt` | `pkg/runtime/clipboard.go` | Public bound-safe API `runtime.ClipboardGetText` / `ClipboardSetText`. |
