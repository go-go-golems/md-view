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
