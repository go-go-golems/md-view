# Changelog

## 2026-09-24

- Initial workspace created

## 2026-09-24

Analysis-only: identified two independent root causes for macOS Command-C failure — (A) custom menu omits native Edit/App roles, so macOS never translates ⌘C into copy:; (B) JS copy buttons depend on navigator.clipboard, unavailable in WKWebView's non-secure wails:// scheme. Wrote intern design/implementation guide and captured sources.

### Related Files

- /Users/manuel.odendahl/code/go-go-golems/md-view/frontend/dist/augment.js — Bug B; navigator.clipboard.writeText with no catch.
- /Users/manuel.odendahl/code/go-go-golems/md-view/menu.go — Bug A root cause; buildMenu adds only File/View.

## 2026-09-24

Uploaded the design/implementation guide + diary + sources index to reMarkable as a ToC bundle: OK: uploaded MDV-CLIPBOARD-001 macOS Command-C and Clipboard Guide.pdf -> /ai/2026/09/24/MDV-CLIPBOARD-001

## 2026-09-24

Implemented both fixes: (A) native App+Edit menu roles on darwin (commit ab36db9), (B) bound App.CopyText via runtime.ClipboardSetText and shared window.MDSCopyText routing (commit d70229b). Verified on macOS: menu bar shows File/Edit/View and Cmd-A+Cmd-C copies 7108 bytes of rendered text under a sentinel test. make test/gofmt/go vet clean; make lint blocked by an unrelated Go 1.27/golangci-lint export-data mismatch. Updated docs/user-guide.md and re-uploaded the guide.

## 2026-09-24

Re-uploaded the updated guide+diary+sources bundle to reMarkable: OK: uploaded MDV-CLIPBOARD-001 macOS Command-C and Clipboard Guide v2 implemented.pdf -> /ai/2026/09/24/MDV-CLIPBOARD-001

## 2026-09-24

Bumped .golangci-lint-version v2.11.2 -> v2.14.0 to fix the Go 1.27 export-data incompatibility (x/tools v0.50.0 reads V4/V5). make lint now passes with 0 issues; CI uses the same version-file. Verified make test still passes.
