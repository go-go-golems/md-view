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
