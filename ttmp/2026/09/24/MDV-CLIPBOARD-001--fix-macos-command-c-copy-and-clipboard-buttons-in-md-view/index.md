---
Title: Fix macOS Command-C copy and clipboard buttons in md-view
Ticket: MDV-CLIPBOARD-001
Status: active
Topics:
    - md-view
    - wails
    - desktop
    - frontend
    - go
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://menu.go
      Note: Root cause of broken Command-C: custom menu has no Edit role.
    - Path: repo://frontend/dist/augment.js
      Note: JS copy buttons depend on navigator.clipboard.
ExternalSources:
    - "https://github.com/wailsapp/wails/issues/4918"
    - "https://github.com/water-rs/waterui/issues/592"
Summary: >
  Analysis-only ticket: two independent root causes for macOS Command-C and
  in-page copy failures, plus a phased intern implementation guide.
LastUpdated: 2026-09-24T00:00:00Z
WhatFor: "Understand and fix macOS clipboard behavior in md-view."
WhenToUse: "Read before touching menu construction or any copy/code-block UI."
---

# Fix macOS Command-C copy and clipboard buttons in md-view

## Overview

`md-view` is a Wails v2 single-binary desktop Markdown viewer. On macOS, ⌘C on selected
text does nothing and the in-page copy buttons (code block, copy path, copy article) do not
work. Investigation found **two independent causes**:

1. **Bug A — no native Edit menu.** `menu.go` builds a custom menu with only File and View.
   macOS implements ⌘C by matching an Edit-menu item whose action is the `copy:` selector and
   routing it through the responder chain to WKWebView. With no Edit menu, ⌘C is never
   translated into a copy. Verified against Wails v2.12.0 source and upstream issue #4918.
2. **Bug B — non-secure WebView origin.** The JS copy buttons call
   `navigator.clipboard.writeText`. Wails serves the frontend from the custom `wails://`
   scheme, which WKWebView does not treat as a secure context, so `navigator.clipboard` is
   unavailable. Fix: a bound Go method using `runtime.ClipboardSetText`.

This is an analysis-only ticket; no production code has been changed. This ticket is the
brief and the guide for the follow-up implementation work.

## Key Links

- [Intern analysis, design, and implementation guide](design-impl-guide/01-macos-command-c-and-clipboard-analysis-design-and-intern-implementation-guide.md)
- [Chronological diary](reference/01-diary.md)
- [Curated sources and provenance index](sources/README.md)
- Wails issue #4918 (upstream root cause)
- waterui issue #592 (custom scheme is not a secure context)

## Status

Current status: **active** (both fixes implemented, committed, and macOS-verified; one
environment limitation on `make lint`).

- Bug A root cause: **verified**.
- Fix A: menu bar now `Apple, md-view, File, Edit, View`; ⌘C copied 7108 bytes of rendered
  README text under an end-to-end test.
- Bug B root cause: **strongly supported**; Fix B implemented (bound `App.CopyText` +
  `MDSCopyText` route), build-verified. Copy-button *clicks* were not automated.

Commits on `main`:

| Commit | Contents |
|--------|----------|
| `5a4876b` | Ticket docs, guide, diary, sources |
| `ab36db9` | Fix A: native App/Edit menus on darwin + structural test |
| `d70229b` | Fix B: `App.CopyText` + `MDSCopyText` routing |

## Topics

- md-view, wails, desktop, frontend, go

## Tasks

See [tasks.md](./tasks.md) for the current task list.

## Changelog

See [changelog.md](./changelog.md) for recent changes and decisions.

## Structure

- design-impl-guide/ - Analysis, design, and intern implementation guide
- reference/ - Chronological diary
- sources/ - Captured upstream issues and verbatim Wails source excerpts
- scripts/ - Temporary code and tooling
