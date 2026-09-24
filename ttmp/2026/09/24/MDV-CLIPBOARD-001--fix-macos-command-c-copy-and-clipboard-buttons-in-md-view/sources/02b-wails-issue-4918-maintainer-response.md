---
title: "Wails issue #4918 — maintainer response (GitHub API capture)"
source: "https://github.com/wailsapp/wails/issues/4918"
captured: "2026-09-24 via api.github.com"
---

# [v2] Custom menu disables clipboard shortcuts in text inputs (macOS)

**State:** closed  **Created:** 2026-01-28T13:18:45Z

### Description

When a custom menu is set via `Menu: appMenu` in `main.go`, standard text input shortcuts stop working. In this repro app, `Cmd/Ctrl+V`, `Cmd/Ctrl+A`, and `Cmd/Ctrl+C` do not work in the input. If I comment out `Menu: appMenu`, the shortcuts work again.

### To Reproduce

1. `wails init -n wails-menu-input-bug -t vanilla`
2. Add a custom menu and set `Menu: appMenu` in `main.go` ([see repro repo](https://github.com/beam-transfer/wails-menu-bug/blob/main/main.go#L60-L62)).
3. `wails dev`
4. Click the input field.
5. Try `Cmd/Ctrl+V`, `Cmd/Ctrl+A`, `Cmd/Ctrl+C`.
6. Comment out `Menu: appMenu`, restart, and try again.

### Expected behaviour

Clipboard shortcuts should work in text inputs even when a custom menu is set.

### Screenshots

N/A (text input shortcut issue).

### Attempted Fixes

- Commented out `Menu: appMenu` → shortcuts work.
- Created minimal repro app and simplified menu structure.
- Read the Wails troubleshooting guide.

### System Details

```shell
❯ wails doctor


          Wails Doctor



# Wails
Version | v2.11.0


# System
┌────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┐
| OS           | MacOS                                                                                                                       |
| Version      | 15.7.3                                                                                                                      |
| ID           | 24G419                                                                                                                      |
| Branding     |                                                                                                                             |
| Go Version   | go1.24.7                                                                                                                    |
| Platform     | darwin                                                                                                                      |
| Architecture | arm64                                                                                                                       |
| CPU 1        | Apple M1 Max                                                                                                                |
| CPU 2        | Apple M1 Max                                                                                                                |
| GPU          | Chipset Model: Apple M1 Max Type: GPU Bus: Built-In Total Number of Cores: 24 Vendor: Apple (0x106b) Metal Support: Metal 3 |
| Memory       | 64GB                                                                                                                        |
└────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┘

# Dependencies
┌─────────────────────────────────────────────────────────────────────┐
| Dependency                | Package Name | Status    | Version      |
| Xcode command line tools  | N/A          | Installed | 2410         |
| Nodejs                    | N/A          | Installed | 22.17.0      |
| npm                       | N/A          | Installed | 10.9.2       |
| *Xcode                    | N/A          | Installed | 26.2 (17C52) |
| *upx                      | N/A          | Installed | upx 5.1.0    |
| *nsis                     | N/A          | Installed | v3.11        |
|                                                                     |
└────────────────────── * - Optional Dependency ──────────────────────┘

# Diagnosis
 SUCCESS  Your system is ready for Wails development!

 ♥   If Wails is useful to you or your company, please consider sponsoring the project:
https://github.com/sponsors/leaanthony
```

### Additional context

- Minimal repro repo: https://github.com/beam-transfer/wails-menu-bug
- The menu line in question: https://github.com/beam-transfer/wails-menu-bug/blob/main/main.go#L60-L62
- The menu definition is in `main.go`. The issue goes away when `Menu: appMenu` is removed.


## Comments

### leaanthony — 2026-02-24T09:33:30Z

This is a known macOS behaviour — when you set a custom menu, the default Edit menu (which provides the clipboard shortcut handlers) is removed by the OS.

The fix is to add an Edit menu with the standard items to your custom menu:

```go
editMenu := menu.NewMenu()
editMenu.AddText("Cut", keys.CmdOrCtrl("x"), func(_ *menu.CallbackData) {
    // handled by macOS
})
editMenu.AddText("Copy", keys.CmdOrCtrl("c"), func(_ *menu.CallbackData) {})
editMenu.AddText("Paste", keys.CmdOrCtrl("v"), func(_ *menu.CallbackData) {})
editMenu.AddText("Select All", keys.CmdOrCtrl("a"), func(_ *menu.CallbackData) {})

appMenu.Append(&menu.MenuItem{
    Label:   "Edit",
    SubMenu: editMenu,
})
```

The callbacks can be empty — macOS intercepts these shortcuts natively as long as the menu items exist with the correct key bindings.

This is handled more gracefully in v3. If you're able to migrate, that's the long-term solution.

### pavelbinar — 2026-02-24T14:56:37Z

**Thanks, Lea! That worked perfectly.**

Since you mentioned v3 — I'm keen to upgrade. How's the stability looking on macOS, particularly around drag-and-drop? That's a core feature for us. Would you feel confident using it in production at this point?

### leaanthony — 2026-02-25T20:18:43Z

V3 has the best DnD support 

