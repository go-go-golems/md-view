---
Title: Diary
Ticket: MD-VIEW-BARE
Status: active
Topics:
    - go
    - glazed
    - cli
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pkg/commands/view.go
      Note: Converted to BareCommand, added --browser and --no-browser flags
    - Path: pkg/commands/serve.go
      Note: Converted to BareCommand, removed Processor param
    - Path: pkg/commands/stop.go
      Note: Converted to BareCommand, prints to stdout
    - Path: pkg/commands/status.go
      Note: Converted to BareCommand, uses RunStatus()
    - Path: pkg/commands/run.go
      Note: Passes browser/no-browser through protocol, RunStatus moved here
    - Path: pkg/protocol/protocol.go
      Note: Added Browser field to Command
    - Path: pkg/server/server.go
      Note: Added openBrowserWith(), uses protocol browser command
ExternalSources: []
Summary: Diary for BareCommand conversion and default browser flag changes
LastUpdated: 2026-05-07T00:00:00Z
WhatFor: Tracking design decisions, progress, and lessons learned
WhenToUse: Read before resuming work on MD-VIEW-BARE
---

# Diary

## Goal

Convert all md-view commands from GlazeCommand (structured output) to BareCommand (no structured output), and change the default browser to `firefox --new-window`.

---

## Step 1: BareCommand Conversion + Browser Defaults

All four commands (`view`, `serve`, `stop`, `status`) were converted from GlazeCommand to BareCommand. The glazed section (which adds `--output`, `--fields`, `--filter`, etc.) was removed from all commands since none produce tabular data. The default browser was changed from auto-detect to `firefox --new-window`, and a `--no-browser` flag was added.

### Prompt Context

**User prompt (verbatim):** "Update md-view so that it only uses glazed barecommand, not structured output, since none is necessary. (see glazed command skill). Also have the default md-view view be with --browser, and the default browser to be 'firefox --new-window'. Create a new ticket for these improvements, add implementation guide, tasks, and then implement, keeping a diary. Committing at appropriate intervals."

**Assistant interpretation:** Convert all md-view GlazeCommand implementations to BareCommand, set default browser to `firefox --new-window`, add `--no-browser` flag, create docmgr ticket with guide/tasks/diary, commit.

**Inferred user intent:** Remove unnecessary structured output complexity from md-view commands (they don't produce tables), and make the tool default to opening Firefox in a new window since that's the primary use case on this system.

**Commit (code):** `757802b` — "refactor: convert all commands to BareCommand, default browser to firefox --new-window"

### What I did

- Created docmgr ticket MD-VIEW-BARE with implementation guide, tasks, and diary
- Converted `ViewCommand` from `RunIntoGlazeProcessor` to `Run()`: removed glazed section, kept command settings section, prints URL to stdout
- Converted `ServeCommand` similarly: removed glazed section, removed `middlewares.Processor` parameter from `RunServe()`
- Converted `StopCommand`: removed glazed section, prints "Daemon stopped." to stdout
- Converted `StatusCommand`: removed glazed section, delegates to `RunStatus()` which prints to stdout
- Changed `--browser` default from `""` (auto-detect) to `"firefox --new-window"`
- Added `--no-browser` flag (bool, default false) to suppress browser opening
- Added `Browser` field to `protocol.Command` so CLI can pass browser command to daemon
- Added `openBrowserWith()` method to server that splits the browser command string (e.g. "firefox --new-window") into executable + args
- Updated socket handler to use `cmd.Browser` when provided, falling back to auto-detect
- Moved `RunStatus()` from (deleted) status.go to run.go

### Why

None of the md-view commands produce tabular data — they open browsers, run servers, stop daemons, or print status lines. The Glazed structured output pipeline adds `--output`, `--fields`, `--filter` etc. flags that are noise. BareCommand removes all that, giving a clean CLI. The `firefox --new-window` default is what the user needs for i3 floating window management.

### What worked

- The BareCommand interface in glazed is exactly right for this use case — just `Run(ctx, vals) error`
- `BuildCobraCommandFromCommand` auto-detects `BareCommand` and skips glaze mode
- Removing glazed sections removes ~8 unnecessary flags per command
- The `openBrowserWith()` method handles splitting "firefox --new-window" into `exec.Command("firefox", "--new-window", url)` cleanly

### What didn't work

- Initially forgot that `RunStatus()` was defined in the old `status.go` which I overwrote — had to add it to `run.go`

### What I learned

- When converting from GlazeCommand to BareCommand, you can keep `commandSettingsSection` (adds `--print-parsed-fields`, `--print-schema`, etc.) — it's useful for debugging and doesn't add output-formatting noise
- The `BareCommand` interface just needs `Run(ctx, vals) error` — no `middlewares.Processor` parameter
- For browser commands that include flags (like `firefox --new-window`), you need to split the string before passing to `exec.Command`

### What was tricky to build

- The `openBrowserWith()` method needs to split the browser command string into parts before creating the `exec.Command`. Using `strings.Fields()` handles this correctly for "firefox --new-window" → ["firefox", "--new-window"], then appending the URL.

### What warrants a second pair of eyes

- The protocol change (adding `Browser` field) is backward-compatible since it's `omitempty` in JSON — old daemons will just ignore it
- The `--no-browser` flag is checked client-side: if set, `cmd.Browser` is left empty, so the daemon won't open a browser at all

### What should be done in the future

- Update the user guide to reflect the new `--browser` default and `--no-browser` flag
- Consider adding a `--browser` flag to the `serve` command for the case where someone runs it directly
- Add integration test that verifies `--no-browser` suppresses browser opening

### Code review instructions

- Start with `pkg/commands/view.go` — the main BareCommand conversion + new flags
- Then `pkg/commands/run.go` — protocol command now carries browser, RunStatus moved here
- Then `pkg/server/server.go` — `openBrowserWith()` + socket handler change
- Then `pkg/protocol/protocol.go` — Browser field
- Verify with: `go build ./...` and `go test ./... -count=1`
- Manual test: `md-view view --help` (should show --browser default as "firefox --new-window")

### Technical details

- BareCommand interface: `Run(ctx context.Context, parsedValues *values.Values) error`
- `BuildCobraCommandFromCommand` checks `cmds.BareCommand` interface and uses `Run()` directly
- Browser command splitting: `strings.Fields("firefox --new-window")` → `["firefox", "--new-window"]`
- Protocol backward compatibility: `Browser` field has `json:"browser,omitempty"` — old daemons ignore it
