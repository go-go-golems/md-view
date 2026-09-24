---
title: "[v2] Custom menu disables clipboard shortcuts in text inputs (macOS) · Issue #4918 · wailsapp/wails · GitHub"
author: "pavelbinar"
site: "GitHub - wailsapp/wails"
published: 2026-01-28T13:18:45.000Z
source: "https://github.com/wailsapp/wails/issues/4918"
domain: "github.com"
language: "en"
description: "Description When a custom menu is set via Menu: appMenu in main.go, standard text input shortcuts stop working. In this repro app, Cmd/Ctrl+"
word_count: 366
---

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

- Minimal repro repo: [https://github.com/beam-transfer/wails-menu-bug](https://github.com/beam-transfer/wails-menu-bug)
- The menu line in question: [https://github.com/beam-transfer/wails-menu-bug/blob/main/main.go#L60-L62](https://github.com/beam-transfer/wails-menu-bug/blob/main/main.go#L60-L62)
- The menu definition is in `main.go`. The issue goes away when `Menu: appMenu` is removed.