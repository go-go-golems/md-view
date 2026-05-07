---
Title: 'Implementation Guide: BareCommand + Default Browser'
Ticket: MD-VIEW-BARE
Status: active
Topics:
    - go
    - glazed
    - cli
DocType: design-impl-guide
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pkg/commands/run.go
      Note: Run functions for all commands
    - Path: pkg/commands/view.go
      Note: Primary command converted to BareCommand
    - Path: pkg/protocol/protocol.go
      Note: Browser field in protocol
    - Path: pkg/server/server.go
      Note: Browser handling in server
ExternalSources: []
Summary: Convert md-view commands from GlazeCommand to BareCommand and set default browser to firefox --new-window
LastUpdated: 2026-05-07T00:00:00Z
WhatFor: Reference for implementing BareCommand conversion and browser defaults
WhenToUse: Read before starting implementation of this ticket
---


# Implementation Guide: BareCommand + Default Browser

## Goal

Convert all four md-view commands (`view`, `serve`, `stop`, `status`) from `GlazeCommand` (structured output via `RunIntoGlazeProcessor`) to `BareCommand` (plain `Run()` with no structured output). Also change the default `view` behavior to always open the browser, with `firefox --new-window` as the default browser command.

## Context

Currently all four commands implement `RunIntoGlazeProcessor`, which forces them through Glazed's table output pipeline (with `--output`, `--fields`, etc.). But none of these commands produce tabular data — `view` opens a browser, `serve` runs a foreground server, `stop` stops a daemon, and `status` prints a simple line. The Glazed structured output adds unnecessary flags and complexity for no benefit.

The `BareCommand` interface in glazed (`cmds.BareCommand`) requires just a `Run(ctx, parsedValues) error` method. When `BuildCobraCommandFromCommand` detects a `BareCommand`, it skips the glaze processor pipeline entirely.

## Current State

### view.go
- Implements `GlazeCommand` (`RunIntoGlazeProcessor`)
- Creates a `glazedSection` and `commandSettingsSection`
- Emits a row with `url` and `file` fields
- `--browser` flag default is `""` (empty), meaning auto-detect

### serve.go
- Implements `GlazeCommand` (`RunIntoGlazeProcessor`)
- Creates glazed + command settings sections
- Calls `RunServe()` with a `Processor` parameter (never used for output)

### stop.go
- Implements `GlazeCommand` (`RunIntoGlazeProcessor`)
- Emits a row with `status: "stopped"`

### status.go
- Implements `GlazeCommand` (`RunIntoGlazeProcessor`)
- Emits rows with daemon info

## Implementation Steps

### Step 1: Convert `view` command to BareCommand

**File**: `pkg/commands/view.go`

1. Remove import of `middlewares`, `settings`, `types`
2. Keep import of `cmds`, `fields`, `values`, `schema`, `cli`
3. Change `ViewCommand` to implement `BareCommand`:
   - Replace `RunIntoGlazeProcessor` with `Run(ctx context.Context, vals *values.Values) error`
4. Remove the `glazedSection` from the constructor (no more `settings.NewGlazedSchema()`)
5. Keep `commandSettingsSection` (useful for `--print-parsed-fields` debugging)
6. In `Run()`, decode settings, call `RunView()`, and print the URL to stdout with `fmt.Println`
7. Change `--browser` default from `""` to `"firefox --new-window"`
8. Add a `--no-browser` flag (bool, default false) to suppress browser opening
9. Pass the browser command and no-browser flag through the protocol to the server

**Key change**: The `view` command should always open a browser by default. The browser command defaults to `firefox --new-window` instead of empty auto-detect.

### Step 2: Convert `serve` command to BareCommand

**File**: `pkg/commands/serve.go`

1. Remove `glazedSection` from constructor
2. Remove `middlewares.Processor` parameter from `RunServe()` signature
3. Replace `RunIntoGlazeProcessor` with `Run()`
4. Keep `commandSettingsSection`

### Step 3: Convert `stop` command to BareCommand

**File**: `pkg/commands/stop.go`

1. Remove `glazedSection`, `types` imports
2. Replace `RunIntoGlazeProcessor` with `Run()`
3. Print "Daemon stopped." to stdout instead of emitting a row
4. Keep `commandSettingsSection`

### Step 4: Convert `status` command to BareCommand

**File**: `pkg/commands/status.go`

1. Remove `glazedSection`, `types` imports
2. Replace `RunIntoGlazeProcessor` with `Run()`
3. Use the existing `RunStatus()` helper (which already prints to stdout)
4. Keep `commandSettingsSection`

### Step 5: Update `RunView` to handle browser opening client-side

**File**: `pkg/commands/run.go`

Currently, `RunView` sends a `view` command to the daemon via Unix socket, the daemon opens the browser, and returns the URL. This is fine — the daemon should still open the browser. But we need to pass the browser command from the CLI to the daemon.

**File**: `pkg/protocol/protocol.go`

Add a `Browser` field to the `Command` struct so the CLI can tell the daemon which browser to use.

**File**: `pkg/server/server.go`

Update `handleSocketConn` to use `cmd.Browser` when provided, falling back to the server's default.

### Step 6: Update root command in main.go

**File**: `cmd/md-view/main.go`

No changes needed — `BuildCobraCommandFromCommand` already handles `BareCommand` via `BuildCobraCommandFromCommand`. The cobra builder detects the `BareCommand` interface and uses `Run()` directly.

### Step 7: Build and test

1. `go build ./...`
2. `go test ./...`
3. Manual test: `md-view view some.md` — should open Firefox in a new window

## Browser Default Rationale

The default `firefox --new-window` is chosen because:
- The user (Manuel) uses i3/Sway and wants floating windows
- `--new-window` creates a separate window that i3 can float independently
- Firefox is the primary browser on this system
- Users who prefer auto-detection can set `--browser ""` or remove the default

## Files to Modify

1. `pkg/commands/view.go` — BareCommand, new defaults
2. `pkg/commands/serve.go` — BareCommand
3. `pkg/commands/stop.go` — BareCommand
4. `pkg/commands/status.go` — BareCommand
5. `pkg/commands/run.go` — browser flag handling, print URL
6. `pkg/protocol/protocol.go` — add Browser field
7. `pkg/server/server.go` — use browser from protocol
