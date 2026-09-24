---
Title: Diary — macOS Command-C and clipboard investigation
Ticket: MDV-CLIPBOARD-001
Status: active
Topics:
    - md-view
    - wails
    - desktop
    - frontend
    - go
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://frontend/dist/augment.js
      Note: Bug B copy path
    - Path: repo://menu.go
      Note: Root cause of Bug A
ExternalSources:
    - https://github.com/wailsapp/wails/issues/4918
Summary: Chronological record of the analysis-only investigation for the macOS Command-C bug.
LastUpdated: 2026-09-24T00:00:00Z
WhatFor: ""
WhenToUse: ""
---

# Diary — macOS Command-C and clipboard investigation

Analysis-only ticket. No production code was changed.

## Step 1 — User request (verbatim)

> "On macosx, copy Command-C doesn't seem to work. Analyze, create docmgr ticke, Create a detailed analysis / design / implementation guide that is for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet points and pseudocode and diagrams and api references and file references. It should be very clear and technical. Store in the ticket and upload to remarkable."

A follow-up instruction also asked to save relevant resources into the ticket's `sources/` directory, using `defuddle` where needed.

## Step 2 — Orientation in the repository

Read `AGENT.md` and `README.md`. Established that `md-view` is a Wails v2 single-binary desktop app, not the older daemon/server. Identified the copy surface:

- Native macOS menu: `menu.go` (`buildMenu`).
- Wails app wiring: `main.go` (`runDesktop`, `options.App.Menu`).
- Bound backend: `app.go` (`App`, `Startup` saving `ctx`).
- Frontend copy buttons: `frontend/dist/augment.js`, `frontend/dist/buttons.js`, legacy `frontend/dist/copy-button.js`.

## Step 3 — Formed the two-root-cause hypothesis

Hypothesis A: the custom menu has no Edit menu, so macOS has no `copy:` key equivalent for ⌘C.
Hypothesis B: the JS copy buttons use `navigator.clipboard`, unavailable in WKWebView's
non-secure `wails://` custom scheme.

## Step 4 — Verified against Wails v2.12.0 source

Read the pinned module in `$GOMODCACHE/github.com/wailsapp/wails/v2@v2.12.0`:

- `internal/frontend/desktop/darwin/window.go` — `UpdateApplicationMenu` rebuilds the main menu
  from only the user-supplied menu; defaults are not injected. Confirms Hypothesis A.
- `internal/frontend/desktop/darwin/WailsMenu.m` — the Edit role builds items with
  `@selector(copy:)` etc. and no target. Confirms the mechanism.
- `pkg/menu/menuroles.go` — `AppMenu()`/`EditMenu()` exported; `pkg/menu/mac.go`'s
  `DefaultMacMenu()` is commented out in v2.12.0.
- `pkg/runtime/clipboard.go` + `internal/frontend/desktop/darwin/clipboard.go` — native
  clipboard bridge available to fix Hypothesis B.

## Step 5 — Verified upstream issue

Retrieved Wails issue #4918 via `defuddle` (rendered page) and the GitHub REST API (to capture
comments, which `defuddle` omits). Maintainer Leaanthony confirms the behavior and the Edit-menu
fix. Captured under `sources/01-…` and `sources/02b-…`. Also captured waterui issue #592 for the
non-secure custom scheme fact (`sources/04-…`).

`sources/03-wkwebview-custom-scheme-not-secure.md` is incomplete: StackOverflow returned HTTP
403 to `defuddle` and to `curl` with a browser UA, and `wails.io` likewise returned 403. The
facts are covered by `sources/04-…` and the verbatim Wails scheme source, and this limitation is
recorded here rather than papered over.

## Step 6 — Wrote the guide

Created `design-impl-guide/01-macos-command-c-and-clipboard-analysis-design-and-intern-implementation-guide.md`
covering: app/Wails architecture, the responder chain, the Wails menu pipeline, the secure-context
clipboard problem, a comparison of the two bugs, a phased implementation plan with real code and
pseudocode, API tables, a file map, risks, glossary, and definition of done.

## Step 7 — Sources and delivery

Copied verbatim Wails v2.12.0 source excerpts into `sources/06-wails-source-excerpts/` and wrote
`sources/README.md` as the provenance index. Uploaded the guide (and sources where useful) to
reMarkable.

## Outcome / evidence level

- Bug A root cause: **verified** (Wails source + upstream maintainer).
- Bug B root cause: **strongly supported, not run-verified** — no macOS runtime check of
  `window.isSecureContext` / `typeof navigator.clipboard` was performed in this analysis-only pass.
  Phase 0 of the implementation plan captures that check as the first action.

---

# Implementation (2026-09-24, continued)

## Step 8 — User request to implement, commit, and keep the diary

> "commit it all, btw commit at appropriate intervals and keep a detailed diary as you work (using the diary format from the skill). Actually, before starting, how are you going to address a and B"

Plan stated before coding:

- **A:** `menu.go` — on `runtime.GOOS == "darwin"`, prepend `menu.AppMenu()` (first) and append
  `menu.EditMenu()` after File, before View. No literal Edit submenu on Linux/Windows (a no-op
  Ctrl+C item can swallow the WebView's native shortcut). Add a structural test.
- **B:** `app.go` — bound `CopyText(text) error` → `runtime.ClipboardSetText(a.ctx, text)`;
  `augment.js` — shared `window.MDSCopyText` (Go binding → `navigator.clipboard` →
  `document.execCommand`); route `augment.js` and `buttons.js` call sites through it.
- Commit intervals: docs, Fix A, Fix B, validation/docs.

## Step 9 — Fix A: native App + Edit menus on darwin

Changed `menu.go`:

- Added stdlib `runtime` import.
- Prepended `menu.AppMenu()` when `runtime.GOOS == "darwin"` (also restores About/Hide/Quit ⌘Q).
- Appended `menu.EditMenu()` between the File and View submenus on darwin.

Added `menu_test.go` (`TestBuildMenuIncludesDarwinRoles`) asserting the darwin menu contains
`AppMenuRole` and `EditMenuRole` with the App role first, and that no role items exist off-darwin.

Evidence:

```
$ GOWORK=off go test -tags webkit2_41 . -run 'TestBuildMenuIncludesDarwinRoles|TestParseViewArgs' -v
--- PASS: TestParseViewArgs (0.00s)
--- PASS: TestBuildMenuIncludesDarwinRoles (0.00s)
PASS
ok  github.com/go-go-golems/md-view  0.772s
```

Commit: `fix(MDV-CLIPBOARD-001): add native App and Edit menus on macOS`.
Native ⌘C smoke test still pending; the structural test only proves the roles are present.

## Step 10 — Fix B: native clipboard bridge for the JS copy buttons

Changed `app.go`: added bound `CopyText(text string) error` calling
`runtime.ClipboardSetText(a.ctx, text)`; guards a nil `ctx` with a clean error.

Changed `frontend/dist/augment.js`: added a global `window.MDSCopyText(text)` that returns a
Promise and tries, in order: (1) `window.go.main.App.CopyText` (native), (2) Web Clipboard
API, (3) `document.execCommand('copy')` via a temporary textarea. The code-block copy button
now calls `window.MDSCopyText(text)` and handles rejection (previously it called
`navigator.clipboard` with no `.catch`, so it threw silently when the API was absent).

Changed `frontend/dist/buttons.js`: copy-path and copy-article now call `window.MDSCopyText`.
Copy-path gained an error toast it did not have before.

Left `frontend/dist/copy-button.js` untouched: it is the legacy IIFE superseded by `augment.js`
and is not referenced by `frontend/dist/index.html`.

Regenerated bindings with `make build`; `frontend/wailsjs/` and `build/` are gitignored, so the
committed diff is `app.go`, `augment.js`, `buttons.js`. `CopyText` is present in the generated
`App.d.ts`/`App.js`.

Evidence:

```
$ GOWORK=off go build -tags webkit2_41 .        # BUILD OK
$ make test
ok  github.com/go-go-golems/md-view            0.246s
ok  github.com/go-go-golems/md-view/internal/launch  0.680s
ok  github.com/go-go-golems/md-view/pkg/renderer      0.957s
ok  github.com/go-go-golems/md-view/pkg/watcher       0.889s
$ make build
Built '.../build/bin/md-view.app/Contents/MacOS/md-view' in 3.695s.
$ grep CopyText frontend/wailsjs/go/main/App.d.ts
export function CopyText(arg1:string):Promise<void>;
```

Commit: `fix(MDV-CLIPBOARD-001): route copy buttons through native clipboard`.
Runtime click verification on macOS still pending (next step).

## Step 11 — Native validation on macOS

The user's running `md-view` held the Wails single-instance lock (`com.go-go-golems.md-view`),
so an isolated test launched with a private temp dir to get its own lock file (Wails uses
`$TMPDIR`/`<id>.lock` on darwin):

```
$ TMPDIR=/tmp/mdview-clip-test ./build/bin/md-view.app/Contents/MacOS/md-view view --foreground ./README.md &
```

Menu bar inspection (`osascript` + System Events), proving Fix A's menu exists at the native level:

```
$ osascript -e 'tell application "System Events" to tell process "md-view" to get name of every menu bar item of menu bar 1'
Apple, md-view, File, Edit, View
```

End-to-end ⌘C test: seeded the clipboard with a sentinel, activated the window, sent ⌘A then ⌘C:

```
$ printf 'SENTINEL-NOT-COPIED-123' | pbcopy
$ osascript ... keystroke "a" using command down; delay; keystroke "c" using command down
$ pbpaste | wc -c
7108
$ pbpaste | head -c 80
📂 Open
README.md
🕘 Recent
🌙 Dark
md-view

A markdown viewer that just works...
```

The sentinel was replaced by the rendered page: **⌘C works and copies the DOM selection.**

Test instance killed and `/tmp/mdview-clip-test` removed. Side effect: the system clipboard now
holds a snippet of the README (the sentinel was overwritten by the test).

Build/regeneration evidence:

```
$ make build            # regenerated bindings; Built .../md-view.app/Contents/MacOS/md-view in 3.695s
$ grep CopyText frontend/wailsjs/go/main/App.d.ts
export function CopyText(arg1:string):Promise<void>;
```

Formatting and vet:

```
$ gofmt -l app.go menu.go menu_test.go   # (no output)
$ GOWORK=off go vet -tags webkit2_41 .   # (no output)
```

`make lint` **fails**, but not because of this change: the installed golangci-lint binary cannot
decode Go 1.27.1 export data ("export data version 4 is greater than maximum supported version
2") and reports typecheck errors in untouched packages such as `pkg/watcher`. Recorded as an
environment limitation, not a regression.

## Step 12 — Limitations honestly recorded

- In-page **copy-button clicks** were not exercised end-to-end. macOS System Events exposes only
the native window buttons for a WKWebView, not the HTML buttons, and no Web Inspector automation
was configured. Evidence for Fix B is therefore: the binding is generated (`App.d.ts`), the Go
method compiles and vets clean, and all copy call sites now route through it. This is weaker than
the ⌘C verification and is labeled as such.
- `make lint` environment failure as above.
- `docs/user-guide.md` updated to describe ⌘C and the native clipboard path, including the
  previously undocumented "Copy article" button.

## Step 13 — `make lint` failure: precise root cause and workaround

Investigated the "export data version 4 is greater than maximum supported version 2" failure
instead of leaving it as a vague environment note.

Root cause: "unified IR" export-data format version skew between the Go toolchain and the
`golang.org/x/tools` vendored into golangci-lint.

- Homebrew Go **1.27.1** defines `V0…V4` in `internal/pkgbits/version.go` (`numVersions=5`),
  so it emits/accepts export-data version **4**.
- golangci-lint **v2.11.2** pins `golang.org/x/tools v0.42.0`, whose
  `internal/pkgbits/version.go` stops at `V2` (`numVersions=3`). It therefore panics in
  `internal/pkgbits/decoder.go` for any package whose export data says version 3 or 4 — the
  Go 1.27.1 standard library, including `internal/goarch`, `fmt`, `sync`, `os`, `testing`.
- The failure is toolchain-scoped, not code-scoped: it reproduces on untouched packages
  (`pkg/watcher`) and there are **0 issues** in this repo's code.
- `go.mod` already says `toolchain go1.26.3`, but `GOTOOLCHAIN=auto` only *upgrades*; because
  the local 1.27.1 is newer than 1.26.3, the directive is ignored and 1.27.1 runs.

Version map established from the module proxy and local caches:

| golangci-lint | x/tools | max supported export version | reads Go 1.27? |
|---------------|---------|------------------------------|----------------|
| v2.11.2 (pinned) | v0.42.0 | V2 | no |
| v2.12.0 | v0.44.0 | (V2/V3) | no/unknown |
| v2.13.2 | v0.49.0 | V4 | yes |
| v2.14.0 | v0.50.0 | V5 | yes |

Workaround verified (no repo change): force the cached Go 1.26.6 toolchain, which emits V2
export data that x/tools v0.42.0 can read:

```
$ GOTOOLCHAIN=go1.26.6 GOWORK=off .bin/golangci-lint run --timeout=5m . ./cmd/... ./pkg/...
0 issues.
EXIT=0
```

Two clean fixes, in preference order:

1. Bump `.golangci-lint-version` to `v2.13.2` (pins x/tools v0.49.0; reads Go 1.27). Re-run
   `make lint`; new linter versions may surface new findings to triage.
2. Keep the pin and force the toolchain for lint, e.g. `GOTOOLCHAIN=go1.26.6 make lint` or
   `GOTOOLCHAIN=go1.26.6` in the lint target. This mirrors the repo's existing Wails-CLI
   x/tools workaround.

## Step 14 — Applied fix: bump golangci-lint to v2.14.0

User request: "let's update golangci-lint".

Changed `.golangci-lint-version` from `v2.11.2` to `v2.14.0` (pins `x/tools v0.50.0`, whose
pkgbits decoder knows V5 and therefore reads Go 1.27's V4 export data). The version file is
consumed by both the Makefile (`go install ...@$(GOLANGCI_LINT_VERSION)`) and CI
(`.github/workflows/lint.yml` -> `golangci-lint-action@v9` `version-file: .golangci-lint-version`),
so the bump propagates to both.

Evidence:

```
$ make lint
level=info msg="golangci-lint has version 2.14.0 built with go1.27.1 ..."
[lintersdb] Active 9 linters: [errcheck exhaustive gofmt govet ineffassign nonamedreturns predeclared staticcheck unused]
[linters] 0 issues.
# exit 0
```

The old "export data version 4 ... maximum supported version 2" typecheck failure is gone.
No new findings were introduced, so no source changes were needed. `make test` still passes.

The old workaround (`GOTOOLCHAIN=go1.26.6`) is no longer needed, but the toolchain note is kept
above for the record.
