---
Title: 'macOS Command-C and clipboard: analysis, design, and intern implementation guide'
Ticket: MDV-CLIPBOARD-001
Status: draft
Topics:
    - md-view
    - wails
    - desktop
    - frontend
    - go
DocType: design-impl-guide
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://app.go
      Note: App struct; add CopyText via runtime.ClipboardSetText; ctx saved in Startup
    - Path: repo://frontend/dist/augment.js
      Note: Bug B code-block copy; route through MDSCopyText
    - Path: repo://frontend/dist/buttons.js
      Note: Bug B toolbar copy-path/copy-article
    - Path: repo://frontend/wailsjs/go/main/App.d.ts
      Note: Generated bindings; CopyText appears after rebuild
    - Path: repo://main.go
      Note: runDesktop wires options.App.Menu to buildMenu; replaces Wails defaults
    - Path: repo://menu.go
      Note: 'Bug A root cause: custom menu lacks Edit/App roles'
ExternalSources:
    - https://github.com/wailsapp/wails/issues/4918
    - https://github.com/water-rs/waterui/issues/592
    - https://wails.io/docs/reference/menus/
Summary: |
    A two-cause root-cause analysis and phased implementation guide for macOS Command-C and all in-app copy affordances in the md-view Wails v2 desktop app: '(1) the custom menu omits the native Edit menu, so macOS has no copy: key equivalent; (2) the in-page copy buttons rely on navigator.clipboard, which a non-secure WKWebView custom scheme does not expose.'
LastUpdated: 2026-09-24T00:00:00Z
WhatFor: ""
WhenToUse: ""
---

# macOS Command-C and clipboard — analysis, design, and intern implementation guide

## 0. How to read this document

This guide is written for an intern who has never worked on `md-view` (or on Wails) before.
It is deliberately over-explained. Read it in this order:

1. **§1–§3** to understand what the app is and what the bug actually is.
2. **§4–§6** to learn the two independent mechanisms that make "copy" work on macOS.
3. **§7** for the one-page root-cause verdict.
4. **§8–§10** to implement the fix (design, exact code, tests).
5. **§11–§14** as reference: API tables, file map, glossary, reading list.

Everything that is a *claim about upstream behavior* is backed by a captured source in
`ttmp/2026/09/24/MDV-CLIPBOARD-001--fix-macos-command-c-copy-and-clipboard-buttons-in-md-view/sources/`.
If you disagree with a conclusion, open the corresponding source file first.

The guiding rule for this ticket: **there are two different "copy" features and the bug
report only names one of them.** A good fix repairs both without regressing the other.

---

## 1. What `md-view` is

`md-view` is a **single-binary native desktop Markdown viewer** built with
[Wails v2](https://wails.io/). It is *not* a daemon, *not* a web server, and *not* a browser
tab. One process owns one native window.

The technology stack:

- **Go** is the application host. It parses the CLI, builds the native window, renders
  Markdown to HTML, watches the file for changes, and owns the native menu bar.
- **Wails v2** is the bridge. It embeds a platform WebView (WKWebView on macOS, WebKitGTK
  on Linux, WebView2 on Windows) and exposes Go methods to JavaScript.
- **JavaScript/HTML/CSS** is the in-window UI. It is served from `frontend/dist/` (embedded
  into the binary at build time via `//go:embed all:frontend/dist` in `main.go`).

### 1.1 The Wails execution model

Wails gives you two communication channels. Every frontend/backend feature in this app uses
exactly one of them, and you must know which:

```
┌─────────────────────────────── md-view process ───────────────────────────────┐
│                                                                                │
│   Go backend (main package)                    WebView (frontend/dist)        │
│   ─────────────────────────                    ───────────────────────        │
│                                                                                │
│   App struct  ──── bound methods ─────────▶  window.go.main.App.Method()      │
│   (app.go)          (async Promises)          (JS → Go; request/response)      │
│                                                                                │
│   runtime.EventsEmit(event, payload) ─────▶  runtime.EventsOn(event, cb)      │
│   (Go → JS; push, no return value)            (app.js event handlers)          │
│                                                                                │
│   Native menu bar (menu.go) ──┐                                               │
│                              │ callbacks run in Go, then emit an event         │
│                              └──────────────────────────────▶ app.js           │
└────────────────────────────────────────────────────────────────────────────────┘
```

The "golden rule" documented in `menu.go` is: **native menu callbacks run in Go and cannot
touch the DOM.** A menu item that should change the page emits a Wails event; a menu item
that should compute something calls Go directly.

### 1.2 Where copy lives today

There are **three** frontend copy affordances, all implemented in JavaScript:

| Control | File | Function | Backend needed? |
|---------|------|----------|-----------------|
| Copy button on a fenced code block | `frontend/dist/augment.js` (`initCopyButtons`) and a legacy `frontend/dist/copy-button.js` | `navigator.clipboard.writeText(text)` | No |
| Toolbar "copy file path" | `frontend/dist/buttons.js` (`buildRow`) | `navigator.clipboard.writeText(filePath)` | No |
| Toolbar "copy entire article" | `frontend/dist/buttons.js` (`buildRow`) | `App.RawFile(path)` → `navigator.clipboard.writeText(text)` | Yes (for the bytes) |

And there is **one** native capability the bug report is actually about:

| Control | Provided by | Requires |
|---------|-------------|----------|
| Select text with the mouse and press **⌘C** | macOS **Edit menu** → `copy:` responder action | An Edit menu with a `Cmd+C` key equivalent |

These are different code paths that fail for different reasons. Keep them separate in your
head; §4 and §5 explain each.

---

## 2. The reported bug

> "On macosx, copy Command-C doesn't seem to work."

Symptoms and scope that this guide assumes:

- Selecting rendered text (or text in an input) and pressing **⌘C** does nothing: the
  clipboard is not updated. The Edit menu in the macOS menu bar is also **absent** —
  there is no Undo/Redo/Cut/Copy/Paste/Select All.
- The in-page copy buttons (code-block copy, copy-path, copy-article) are a *separate*
  report. On macOS they are very likely also broken because WKWebView does not expose
  `navigator.clipboard` on the custom `wails://` scheme.

What is **not** the bug: the Markdown renderer, the file watcher, the toolbar layout, or the
CLI. Nothing about rendering or input parsing is involved.

---

## 3. Reproduction and the smallest failing case

### 3.1 Manual reproduction (macOS)

```bash
# from the repo root
make build
build/bin/md-view.app/Contents/MacOS/md-view view ./README.md
# In the window: drag-select a paragraph, press ⌘C, paste into TextEdit.
# Observed: nothing was copied.
# Also look at the menu bar: there is no "Edit" menu at all.
```

The menu bar on macOS is expected to look like:

```
┌──────────┬──────┬──────┬───────────────────────────────┐
│ md-view  │ File │ Edit │ View                          │   ← BUG: no "Edit"
└──────────┴──────┴──────┴───────────────────────────────┘
```

### 3.2 Cross-reference: upstream issue

This is a known Wails v2 behavior, not an `md-view` invention. Wails issue **#4918**
("Custom menu disables clipboard shortcuts in text inputs (macOS)", closed) reproduces it
with a minimal app and a custom menu. The captured report is at
`sources/01-wails-issue-4918-macos-clipboard-shortcuts.md` and the maintainer's answer is at
`sources/02b-wails-issue-4918-maintainer-response.md`.

Maintainer Leaanthony (2026-02-24), verbatim:

> "This is a known macOS behaviour — when you set a custom menu, the default Edit menu
> (which provides the clipboard shortcut handlers) is removed by the OS."

> "The fix is to add an Edit menu with the standard items to your custom menu … The callbacks
> can be empty — macOS intercepts these shortcuts natively as long as the menu items exist
> with the correct key bindings."

---

## 4. How macOS turns ⌘C into a copy

To fix this correctly you must understand the Cocoa responder chain. This is the single most
important concept in the ticket.

### 4.1 The responder chain in one diagram

```
   ⌘C key-down
      │
      ▼
┌───────────────────────────┐
│ NSApplication             │
│   mainMenu = [App][Edit]… │   ← installed by Wails
└───────────┬───────────────┘
            │ 1. Is there an item whose keyEquivalent is "c"
            │    with mask ⌘?  →  the "Copy" item in the Edit menu
            ▼
┌───────────────────────────┐
│ NSMenuItem action=copy:   │
│ target = nil              │   ← nil target = "send to first responder"
└───────────┬───────────────┘
            │ 2. Deliver the selector to the key window's first responder
            ▼
┌───────────────────────────┐
│ First responder:          │
│   WKWebView (or an        │   ← WKWebView implements -copy: by copying
│   NSTextField, etc.)      │      the current DOM selection to NSPasteboard
└───────────┬───────────────┘
            ▼
      NSPasteboard updated → ⌘C "works"
```

Key facts:

- macOS does **not** hard-wire ⌘C in the WebView. The shortcut is matched at the
  **application menu** level first, using the menu item's `keyEquivalent` and modifier mask.
- The menu item's action is the Objective-C selector **`copy:`**. The target is `nil`, so
  Cocoa walks the responder chain until something implements `copy:`. `WKWebView` does.
- If the **Edit menu does not exist**, no menu item claims ⌘C, so the shortcut is never
  turned into `copy:`. The key event reaches the WebView but the WebView's own key handling
  does not synthesize a clipboard copy. Result: ⌘C silently does nothing.
- Therefore **the fix is to make sure an Edit menu with a `copy:`/`cut:`/`paste:`/`selectAll:`
  item exists.** The menu item callbacks can be empty; macOS handles them natively.

The same reasoning applies to:

| Shortcut | Selector | Needs menu item |
|----------|----------|-----------------|
| ⌘Z / ⇧⌘Z | `undo:` / `redo:` | yes |
| ⌘X | `cut:` | yes |
| ⌘C | `copy:` | yes |
| ⌘V | `paste:` | yes |
| ⌘A | `selectAll:` | yes |

### 4.2 Evidence from Wails internals

Read these captured files alongside this section:

**`sources/06-wails-source-excerpts/darwin-WailsMenu.m.txt`** — the native builder for the
Edit role. Notice every item has a standard selector and a ⌘ key equivalent, and **no target**
(so it routes through the responder chain):

```objc
case EditMenu:
{
    WailsMenu *editMenu = [[[WailsMenu new] initWithNSTitle:@"Edit"] autorelease];
    [editMenu addItem:[self newMenuItem:@"Undo"       :@selector(undo:)      :@"z" :NSEventModifierFlagCommand]];
    [editMenu addItem:[self newMenuItem:@"Redo"       :@selector(redo:)      :@"z" :(NSEventModifierFlagShift | NSEventModifierFlagCommand)]];
    [editMenu addItem:[NSMenuItem separatorItem]];
    [editMenu addItem:[self newMenuItem:@"Cut"        :@selector(cut:)       :@"x" :NSEventModifierFlagCommand]];
    [editMenu addItem:[self newMenuItem:@"Copy"       :@selector(copy:)      :@"c" :NSEventModifierFlagCommand]];
    [editMenu addItem:[self newMenuItem:@"Paste"      :@selector(paste:)     :@"v" :NSEventModifierFlagCommand]];
    [editMenu addItem:[self newMenuItem:@"Delete"     :@selector(delete:)    :[self accel:@"backspace"] :0]];
    [editMenu addItem:[self newMenuItem:@"Select All" :@selector(selectAll:) :@"a" :NSEventModifierFlagCommand]];
    ...
}
```

**`sources/06-wails-source-excerpts/darwin-menu-processMenu.go.txt`** — the Go→Objective-C
bridge. A menu item with a non-zero `Role` is turned into a native role submenu:

```go
func processMenuItem(parent *NSMenu, menuItem *menu.MenuItem) *MenuItem {
	if menuItem.Hidden { return nil }
	if menuItem.Role != 0 {          // ← AppMenuRole / EditMenuRole / WindowMenuRole
		parent.AppendRole(menuItem.Role)
		return nil
	}
	if menuItem.Type == menu.SeparatorType {
		C.AppendSeparator(parent.nsmenu)
		return nil
	}
	return parent.AddMenuItem(menuItem)
}
```

**`sources/06-wails-source-excerpts/darwin-window-UpdateApplicationMenu.go.txt`** — the
smoking gun. When the app sets a custom menu, Wails rebuilds the *entire* main menu from
**only** that menu object. It does **not** prepend any default App/Edit menu:

```go
func (w *Window) UpdateApplicationMenu() {
	mainMenu := NewNSMenu(w.context, "")
	if w.applicationMenu != nil {
		processMenu(mainMenu, w.applicationMenu)   // ← processes ONLY the user's menu
	}
	C.SetAsApplicationMenu(w.context, mainMenu.nsmenu)
	C.UpdateApplicationMenu(w.context)             // → NSApp setMainMenu:
}
```

> **Intern takeaway:** Wails' `main.m` demo app appends roles 1 and 2, but the real
> `options.App.Menu` path replaces the menu wholesale. That is why `md-view` has no Edit
> menu: it never asked for one.

**`sources/06-wails-source-excerpts/menu-menuroles.go.txt`** — the exported Go helpers you
will use:

```go
func AppMenu() *MenuItem    { return &MenuItem{Role: AppMenuRole} }    // app name, About, Hide, Quit (⌘Q)
func EditMenu() *MenuItem   { return &MenuItem{Role: EditMenuRole} }   // Undo…Select All (see above)
func WindowMenu() *MenuItem { return &MenuItem{Role: WindowMenuRole} }
```

> **Important v2.12.0 detail:** in `pkg/menu/mac.go`, the higher-level helper
> `DefaultMacMenu()` is **inside a block comment and therefore not exported** in v2.12.0.
> `AppMenu()` and `EditMenu()` *are* exported. Build the menu from those two roles; do not
> call `menu.DefaultMacMenu()` or the code will not compile.

### 4.3 What `md-view` does today

`menu.go` builds a menu from scratch and never adds App or Edit:

```go
func buildMenu(app *App) *menu.Menu {
	appMenu := menu.NewMenu()

	fileMenu := appMenu.AddSubmenu("File")
	fileMenu.AddText("Open…", keys.CmdOrCtrl("o"), func(_ *menu.CallbackData) { ... })
	fileMenu.AddSeparator()
	fileMenu.AddText("Close", keys.CmdOrCtrl("w"), func(_ *menu.CallbackData) { ... })

	viewMenu := appMenu.AddSubmenu("View")
	viewMenu.AddText("Toggle Theme", keys.Key("t"), func(_ *menu.CallbackData) { ... })

	return appMenu
}
```

`main.go` then hands that menu to Wails:

```go
return wails.Run(&options.App{
    ...
    Menu: buildMenu(app),   // ← this is the ENTIRE main menu; defaults are gone
    ...
})
```

**This is the root cause of the ⌘C bug.** The fix is to add the App and Edit roles, on
macOS only (see §8 for why "macOS only" matters).

---

## 5. How the in-page copy buttons work (and why they may also fail)

The copy buttons never use the native menu. They use the Web Clipboard API from JavaScript:

```js
navigator.clipboard.writeText(text).then(...)
```

### 5.1 `navigator.clipboard` requires a secure context

Per the Web Platform, `navigator.clipboard` is available only in a **secure context**
(`window.isSecureContext === true`): `https:`, `http://localhost`, or `file:`. Wails loads the
frontend from a **custom scheme**, `wails://wails/` (see `internal/frontend/desktop/darwin/`
`frontend.go` and `WailsContext.m`, which registers the `wails` scheme handler).
Custom schemes registered with `WKURLSchemeHandler` are **not** treated as secure contexts by
WKWebView, so `navigator.clipboard` is `undefined` on macOS.

The captured evidence:

- `sources/04-secure-context-navigator-clipboard.md` (water-rs/waterui issue #592) states the
  precise problem: "On WKWebView the bundled-asset origin (#586) is a custom scheme, which is
  not a secure context. Every Web platform API gated on `isSecureContext` is then unavailable
  to the page — `crypto.subtle`, `navigator.clipboard`, `navigator.share`…"
- `sources/03-wkwebview-custom-scheme-not-secure.md` (attempted capture; StackOverflow
  returned HTTP 403 to `defuddle`, so this one is documented from the waterui capture and the
  Wails scheme source rather than a first-party quote).

### 5.2 What the current code does when `navigator.clipboard` is missing

In `frontend/dist/augment.js` (`initCopyButtons`):

```js
btn.addEventListener('click', function () {
    var text = codeEl.textContent;
    navigator.clipboard.writeText(text).then(function () {
        btn.innerHTML = checkIcon;   // success feedback
        ...
    });
    // NOTE: no .catch() here at all.
});
```

If `navigator.clipboard` is `undefined`, the expression `navigator.clipboard.writeText(text)`
throws a `TypeError` **synchronously**, the click handler aborts, and there is no error
handling — the button just does nothing.

In `frontend/dist/buttons.js` there *is* a `.catch()`, but its "fallback" only **selects** the
text; it does not copy it:

```js
}).catch(function() {
    var range = document.createRange();
    range.selectNodeContents(codeBlock);
    var sel = window.getSelection();
    sel.removeAllRanges();
    sel.addRange(range);
});
```

So on macOS: code-block copy silently no-ops; toolbar copy shows an error toast (copy-article)
or is a no-op (copy-path has no `.catch`).

### 5.3 The reliable alternative: the native clipboard through Go

Wails exposes a native clipboard API that does **not** depend on the WebView's secure context:

```go
// github.com/wailsapp/wails/v2/pkg/runtime/clipboard.go
func ClipboardGetText(ctx context.Context) (string, error)
func ClipboardSetText(ctx context.Context, text string) error
```

On macOS this writes `NSPasteboard.generalPasteboard` directly (see
`sources/06-wails-source-excerpts/darwin-clipboard.go.txt`). The public API is in
`sources/06-wails-source-excerpts/runtime-clipboard.go.txt`.

`App.ctx` is already stored in `app.go` (`Startup` saves the Wails `context.Context`), so a
bound Go method can call `runtime.ClipboardSetText(a.ctx, text)` and be invoked from JS as
`window.go.main.App.CopyText("...")`.

---

## 6. Two bugs, one symptom — comparison table

| Aspect | Bug A: ⌘C on selected text | Bug B: in-page copy buttons |
|--------|---------------------------|-----------------------------|
| Trigger | Keyboard, applied to DOM/native selection | Mouse click on a button |
| Broken layer | macOS menu / responder chain | WKWebView Web Clipboard API |
| Cause | No Edit menu → no `copy:` key equivalent | Non-secure custom scheme → no `navigator.clipboard` |
| Fix location | `menu.go` (Go) | `app.go` (Go) + `augment.js`/`buttons.js` (JS) |
| Upstream evidence | Wails #4918 + `WailsMenu.m`/`window.go` | waterui #592 + Wails scheme source |
| Fix mechanism | Add `menu.AppMenu()` + `menu.EditMenu()` roles | Add bound `CopyText` → `runtime.ClipboardSetText` |

Both must be fixed. Fixing only Bug A leaves the code-block copy button broken; fixing only
Bug B leaves ⌘C broken.

---

## 7. Root-cause verdict (one page)

1. `md-view` passes a hand-built `menu.NewMenu()` (File + View) to `options.App.Menu`.
2. Wails' `Window.UpdateApplicationMenu` rebuilds the macOS main menu from exactly that
   object and injects **no** defaults.
3. macOS Ctrl/Cmd keyboard shortcuts for editing are implemented by **Edit-menu items with
   standard selectors** (`copy:`, `cut:`, `paste:`, `selectAll:`) and `nil` targets routed
   through the responder chain. With no Edit menu, ⌘C is never translated into `copy:`.
4. Therefore ⌘C does nothing. ✅ **Confirmed by upstream issue #4918 and Wails source.**
5. Independently, the JS copy buttons call `navigator.clipboard.writeText`, which WKWebView
   does not expose on the non-secure `wails://` custom scheme. Those buttons fail even after
   Bug A is fixed. ⚠️ **Strongly supported by upstream issues and Wails source; verify on a
   real macOS run during implementation.**

---

## 8. Proposed design

### 8.1 Design goals

- ⌘C / ⌘V / ⌘X / ⌘A / ⌘Z work on selected text on macOS.
- All in-page copy buttons work on **every** platform (macOS, Linux, Windows), independent of
  WebView secure-context quirks.
- No regression on Linux/Windows, where menu **roles are macOS-only**.
- No new dependencies, no new architecture, minimal diff.
- Keep the existing "golden rule" (menu callbacks emit events; DOM work is in JS).

### 8.2 Fix A — add the native menus on macOS

Build the app menu conditionally on `runtime.GOOS == "darwin"`, prepending the App and Edit
roles before the existing File and View submenus:

```go
import "runtime"

func buildMenu(app *App) *menu.Menu {
	appMenu := menu.NewMenu()

	if runtime.GOOS == "darwin" {
		// Native macOS application menu: About/Hide/Quit (⌘Q).
		appMenu.Append(menu.AppMenu())
		// Native Edit menu: Undo/Redo/Cut/Copy/Paste/Select All.
		// This is what gives macOS a ⌘C / ⌘V / ⌘A key equivalent.
		appMenu.Append(menu.EditMenu())
	}

	// ... existing File and View submenus unchanged ...
	return appMenu
}
```

Why this exact shape:

- `menu.AppMenu()` + `menu.EditMenu()` are the exported, non-commented role helpers in
  v2.12.0 (see §4.2).
- Roles are processed only by the Darwin frontend. On Linux, `processMenu` ignores items
  without a `SubMenu`; on Windows, `processMenu` would create an empty submenu for a role item.
  Gating on `runtime.GOOS` avoids both.
- Alternatively (and equivalently on macOS), the maintainer's public workaround builds a
  literal Edit submenu with empty callbacks. Prefer the role helper: it delegates all the
  Cocoa selectors and key equivalents to Wails, so there is less to get wrong. If you need the
  literal version for auditability, here it is:

  ```go
  editMenu := appMenu.AddSubmenu("Edit")
  editMenu.AddText("Undo",        keys.CmdOrCtrl("z"), func(_ *menu.CallbackData) {})
  editMenu.AddText("Redo",        keys.Shift("z"),     func(_ *menu.CallbackData) {})
  editMenu.AddSeparator()
  editMenu.AddText("Cut",         keys.CmdOrCtrl("x"), func(_ *menu.CallbackData) {})
  editMenu.AddText("Copy",        keys.CmdOrCtrl("c"), func(_ *menu.CallbackData) {})
  editMenu.AddText("Paste",       keys.CmdOrCtrl("v"), func(_ *menu.CallbackData) {})
  editMenu.AddText("Select All",  keys.CmdOrCtrl("a"), func(_ *menu.CallbackData) {})
  ```

  > Do **not** add this literal Edit menu on Windows/Linux: a real `Ctrl+C` menu item with a
  > no-op callback can *swallow* the shortcut that WebView2/WebKitGTK would otherwise handle.

Adds a new menu capability (Quit via ⌘Q) that also fixes the missing About/Hide/Quit menu —
call this out in the ticket as a deliberate, related improvement.

### 8.3 Fix B — a native clipboard bridge for the JS buttons

Add one bound Go method to `App`:

```go
// CopyText writes text to the system clipboard using the native Wails
// clipboard (NSPasteboard on macOS, etc.). It exists because the Wails
// WebView loads the frontend from the non-secure wails:// scheme, where
// navigator.clipboard is unavailable. Errors are surfaced to JS.
func (a *App) CopyText(text string) error {
	if a.ctx == nil {
		return errors.New("clipboard unavailable: app not started")
	}
	return runtime.ClipboardSetText(a.ctx, text)
}
```

Then add a single frontend helper so all three call sites share one code path. Put it in
`augment.js` (which already owns content augmentation) and expose it globally:

```js
// window.MDSCopyText(text) -> Promise<void>
// Prefer the native Go clipboard (works in WKWebView's non-secure context),
// fall back to the Web Clipboard API, then to execCommand.
window.MDSCopyText = function (text) {
    var App = window['go'] && window['go']['main'] && window['go']['main']['App'];
    if (App && App.CopyText) {
        return App.CopyText(text);
    }
    if (navigator.clipboard && navigator.clipboard.writeText) {
        return navigator.clipboard.writeText(text);
    }
    return new Promise(function (resolve, reject) {
        var ta = document.createElement('textarea');
        ta.value = text;
        ta.style.position = 'fixed';
        ta.style.opacity = '0';
        document.body.appendChild(ta);
        ta.select();
        try { document.execCommand('copy'); resolve(); }
        catch (e) { reject(e); }
        finally { ta.remove(); }
    });
};
```

Update the three call sites:

```js
// augment.js — code-block copy
window.MDSCopyText(text).then(function () { showSuccess(btn); })
                        .catch(function (e) { toastCopyError(e); });

// buttons.js — copy path
window.MDSCopyText(filePath).then(function () { showSuccess(copyBtn); })
                            .catch(function (e) { toastCopyError(e); });

// buttons.js — copy article (RawFile returns []byte → decode first)
App.RawFile(filePath).then(function (data) {
    return window.MDSCopyText(decodeBytes(data));
});
```

Note: since Wails regenerates `frontend/wailsjs/go/main/App.d.ts` at build time, `CopyText`
will automatically appear in the bindings and thus on `window.go.main.App`. No manual
binding edit is required; just run `make build` (or `make wails-dev`) to regenerate.

### 8.4 Sequence diagram — after the fix

```
⌘C flow (Bug A fixed)
  User ⌘C ─▶ NSApp mainMenu ─▶ Edit menu "Copy" (keyEquivalent c, ⌘)
           ─▶ action copy: ─▶ first responder = WKWebView
           ─▶ WKWebView copies DOM selection to NSPasteboard ✅

Copy-button flow (Bug B fixed)
  User clicks button ─▶ JS click handler
                     ─▶ window.MDSCopyText(text)
                     ─▶ Promise from App.CopyText(text)      (Go→JS binding)
                     ─▶ runtime.ClipboardSetText(ctx, text)
                     ─▶ NSPasteboard setString ✅
```

### 8.5 Alternatives considered

| Alternative | Why rejected |
|-------------|--------------|
| Only add an Edit *submenu* with empty callbacks, on all platforms | Can swallow `Ctrl+C` on Windows/Linux; macOS roles are cleaner. |
| Make `wails://` a secure context | Requires patching WebKit scheme registration in Wails; upstream change, out of scope, fragile. |
| Use `document.execCommand('copy')` only | Deprecated and unreliable; keep it as a last-resort fallback, not the primary path. |
| Have the frontend call `navigator.clipboard` and ignore macOS | Leaves the code-block/toolbar buttons broken on the primary platform. |
| Migrate to Wails v3 | Long-term option mentioned by the maintainer, but out of scope for this fix. |

---

## 9. Implementation plan

Phased so each step is independently reviewable and testable.

### Phase 0 — Baseline evidence

- [ ] On macOS, run the current build and confirm: no Edit menu; ⌘C does not copy; code-block
      copy button does nothing. Record the app version/commit and macOS version.
- [ ] In the running WebView (Safari Web Inspector or a temporary log line), evaluate
      `window.isSecureContext` and `typeof navigator.clipboard`. Record both. This turns the
      "strongly supported" Bug B claim into verified evidence.

### Phase 1 — Fix A (native Edit menu)

- [ ] Edit `menu.go`: add `runtime.GOOS == "darwin"` guard; append `menu.AppMenu()` and
      `menu.EditMenu()`.
- [ ] `make build`.
- [ ] Verify the menu bar shows `md-view | File | Edit | View`.
- [ ] Verify ⌘C/⌘V/⌘X/⌘A/⌘Z on selected text; verify ⌘Q quits; verify File/View items still
      work.
- [ ] Verify Linux/Windows still build (cross-compile or CI) and menus are unchanged.

### Phase 2 — Fix B (native clipboard bridge)

- [ ] Edit `app.go`: add `CopyText(text string) error` calling `runtime.ClipboardSetText`.
- [ ] Edit `augment.js`: add `window.MDSCopyText`, route the code-block button through it.
- [ ] Edit `buttons.js`: route copy-path and copy-article through it; add error toasts.
- [ ] Remove or leave-aligned the legacy `frontend/dist/copy-button.js` (it is superseded by
      `augment.js`; check `index.html` to see whether it is still loaded — it is not in the
      current `<script>` list, so either delete it or leave a note).
- [ ] `make build` (regenerates bindings), `make test`.
- [ ] Verify all three buttons copy the correct content on macOS and Linux.

### Phase 3 — Validation and docs

- [ ] Run `go test -tags webkit2_41 ./...` and `make lint`.
- [ ] Update `docs/user-guide.md` §"reMarkable Upload, Copy, and Download" to state that ⌘C
      works on text and that the buttons use the native clipboard.
- [ ] Add a diary entry with the exact commands and observed results.
- [ ] Upload the final guide to reMarkable under `/ai/2026/09/24/MDV-CLIPBOARD-001`.

### Pseudocode for a menu unit test (optional)

Menu construction is hard to unit test because it is platform-dependent and returns a native
menu. A cheap, useful test is structural:

```go
func TestBuildMenuIncludesEditOnDarwin(t *testing.T) {
    m := buildMenu(NewApp())
    if runtime.GOOS != "darwin" { t.Skip("role menus are macOS-only") }
    var hasEdit, hasApp bool
    for _, it := range m.Items {
        switch it.Role {
        case menu.EditMenuRole: hasEdit = true
        case menu.AppMenuRole:  hasApp = true
        }
    }
    if !hasEdit || !hasApp {
        t.Fatalf("darwin menu must include App+Edit roles (got %+v)", m.Items)
    }
}
```

> Caveat documented in the ticket: this only proves the roles are present; the actual ⌘C
> behavior needs a human/native smoke test on macOS.

---

## 10. Testing strategy

| Layer | What | How |
|-------|------|-----|
| Static | Go compiles and lint clean | `make test`, `make lint` |
| Structural | Darwin menu contains App+Edit roles | unit test in §9 |
| Native manual | ⌘C/⌘V/⌘X/⌘A/⌘Z, ⌘Q, File/View items | run the `.app`, use TextEdit as clipboard witness |
| Frontend manual | code-block copy, copy-path, copy-article | click buttons, paste elsewhere |
| Cross-platform | Linux/Windows build & menu unchanged | CI cross-compile; no role items on those OSes |
| Regression | File > Open/Close, View > Toggle Theme | manual |

A good clipboard witness is TextEdit or `pbpaste` (macOS): `pbpaste` prints the clipboard.

---

## 11. API and type reference

### 11.1 Wails menu API (`github.com/wailsapp/wails/v2/pkg/menu`)

| Symbol | Signature | Notes |
|--------|-----------|-------|
| `NewMenu` | `func NewMenu() *Menu` | New empty menu. |
| `(*Menu).AddSubmenu` | `func (m *Menu) AddSubmenu(label string) *Menu` | Adds and returns a submenu. |
| `(*Menu).AddText` | `func (m *Menu) AddText(label string, accel *keys.Accelerator, cb Callback) *MenuItem` | Adds a text item. |
| `(*Menu).Append` | `func (m *Menu) Append(item *MenuItem)` | Appends a raw item (used for roles). |
| `AppMenu` | `func AppMenu() *MenuItem` | macOS app menu role (`AppMenuRole=1`). |
| `EditMenu` | `func EditMenu() *MenuItem` | macOS Edit menu role (`EditMenuRole=2`). |
| `WindowMenu` | `func WindowMenu() *MenuItem` | macOS Window menu role (`WindowMenuRole=3`). |
| `DefaultMacMenu` | *not exported in v2.12.0* (commented out in `mac.go`) | Do not use. |

`MenuItem` fields relevant here (`pkg/menu/menuitem.go`): `Label string`, `Role Role`,
`Accelerator *keys.Accelerator`, `Type Type`, `SubMenu *Menu`, `Click Callback`.

`keys` helpers (`pkg/menu/keys`): `CmdOrCtrl(key)`, `Shift(key)`, `Control(key)`,
`OptionOrAlt(key)`, `Key(key)`.

### 11.2 Wails runtime clipboard API (`github.com/wailsapp/wails/v2/pkg/runtime`)

| Symbol | Signature | Platform backing |
|--------|-----------|------------------|
| `ClipboardGetText` | `func ClipboardGetText(ctx context.Context) (string, error)` | NSPasteboard / Win32 / GTK |
| `ClipboardSetText` | `func ClipboardSetText(ctx context.Context, text string) error` | NSPasteboard / Win32 / GTK |

The `ctx` must be the one saved in `App.Startup` (`app.go`).

### 11.3 JavaScript binding surface

Generated in `frontend/wailsjs/go/main/App.d.ts` (do not edit; regenerated by `wails build`).
After adding `CopyText` to `App`, the generated binding becomes:

```ts
export function CopyText(arg1:string):Promise<void>;
```

and is callable as `window['go']['main']['App']['CopyText'](text)`.

---

## 12. File reference map

| Path | Role in this ticket |
|------|---------------------|
| `menu.go` | **Fix A.** `buildMenu` builds File/View only; add App+Edit roles on darwin. |
| `main.go` | `runDesktop` passes `Menu: buildMenu(app)` to Wails; confirms defaults are replaced. |
| `app.go` | **Fix B.** `App` struct, `Startup` saves `ctx`; add `CopyText`. |
| `assets.go` | Asset handler; shows the WebView origin is the Wails asset server (context for secure-context discussion). |
| `frontend/dist/augment.js` | Code-block copy buttons; add `MDSCopyText`, route through it. |
| `frontend/dist/buttons.js` | Toolbar copy-path / copy-article; route through `MDSCopyText`. |
| `frontend/dist/copy-button.js` | Legacy code-block copy IIFE; superseded, not currently loaded from `index.html`. |
| `frontend/dist/index.html` | Script load order; proves which copy scripts are active. |
| `frontend/dist/app.js` | Wails event handlers (`file-opened`, `theme-changed`, `file-changed`). |
| `frontend/wailsjs/go/main/App.d.ts` | Generated binding list; `CopyText` will appear after rebuild. |
| `go.mod` | Pins `github.com/wailsapp/wails/v2 v2.12.0` — all internal claims are for this version. |
| `Makefile` | `make build`, `make wails-dev`, `make test`, `make frontend-css`. |
| `AGENT.md` | Build/test/lint commands and project structure. |
| `docs/user-guide.md` | User-facing copy documentation to update. |

Wails internals (verbatim captures under `sources/06-wails-source-excerpts/`):

| Captured file | Upstream path |
|---------------|---------------|
| `runtime-clipboard.go.txt` | `pkg/runtime/clipboard.go` |
| `darwin-clipboard.go.txt` | `internal/frontend/desktop/darwin/clipboard.go` |
| `menu-menuroles.go.txt` | `pkg/menu/menuroles.go` |
| `menu-menu.go.txt` | `pkg/menu/menu.go` |
| `darwin-window-UpdateApplicationMenu.go.txt` | `internal/frontend/desktop/darwin/window.go` |
| `darwin-menu-processMenu.go.txt` | `internal/frontend/desktop/darwin/menu.go` |
| `darwin-WailsMenu.m.txt` | `internal/frontend/desktop/darwin/WailsMenu.m` |

---

## 13. Risks, pitfalls, and open questions

**Risks**

- **Platform gating.** `menu.AppMenu()`/`menu.EditMenu()` are macOS-only. If added
  unconditionally, Windows creates empty submenus and Linux ignores them — visual regressions
  on Windows. Gate with `runtime.GOOS == "darwin"`.
- **Context lifetime.** `runtime.ClipboardSetText` panics or errors if `ctx` is nil. Guard in
  `CopyText` and return a clean error.
- **RawFile decoding.** `App.RawFile` returns `[]byte`, which Wails JSON-marshals as base64;
  the existing `buttons.js` already decodes it. Keep that decoding before calling
  `MDSCopyText`, or move it into a helper.
- **Duplicate copy code.** `augment.js`, `buttons.js`, and `copy-button.js` each implement
  copy logic. Consolidation is the point of `MDSCopyText`; resist adding a fourth variant.
- **Menu ordering.** On macOS the App menu must be first; prepend roles before File/View.

**Open questions to resolve during implementation**

- Does the Wails `EditMenu` role emit menu *clicks* back to Go for items like Paste, or does
  macOS handle them entirely natively? (Evidence: `nil` targets + standard selectors ⇒ native
  handling; confirm with a manual smoke test.)
- Should `md-view` also add a right-click/context menu for Copy on selected text? Wails'
  `defaultContextMenuEnabled` defaults to true, so a context menu may already exist; verify.
- Is `copy-button.js` dead code that should be deleted, or loaded by some build path? Check
  `index.html` and `Makefile` before deleting.

---

## 14. Glossary and further reading

**Glossary**

- **WKWebView** — Apple's embedded WebKit view, used by Wails on macOS.
- **Custom scheme** — a non-`http(s)` URL scheme (here `wails://`) handled by a
  `WKURLSchemeHandler`. Not a secure context.
- **First responder** — the object that currently receives input events; the target of
  responder-chain actions like `copy:`.
- **Responder chain** — the runtime search order Cocoa uses to find an object that handles a
  selector/action.
- **Role** — a Wails `menu.Role` constant that builds a predefined native menu (App/Edit/Window).
- **Bound method** — a Go method on a Wails-bound struct, auto-exposed to JS.

**Further reading**

- Wails menus reference: https://wails.io/docs/reference/menus/
- Wails issue #4918 (this bug): `sources/01-…` and `sources/02b-…`
- Apple: `NSResponder`, `NSMenu` key equivalents, `NSPasteboard`.
- MDN: `navigator.clipboard`, "Secure contexts".
- This ticket's diary: `reference/01-diary.md`.

---

## 15. Definition of done

- [x] macOS menu bar shows App + Edit menus; ⌘C/⌘V/⌘X/⌘A/⌘Z work on selected text.
- [x] Code-block copy, copy-path, and copy-article routed through `App.CopyText`.
- [x] Linux/Windows menus and clipboard behavior unchanged (roles gated on darwin).
- [x] `make test` passes; `gofmt`/`go vet` clean.
- [x] `make lint` passes with golangci-lint v2.14.0 (0 issues).
- [x] `docs/user-guide.md` updated; diary entry recorded with evidence.
- [x] Guide uploaded to reMarkable at `/ai/2026/09/24/MDV-CLIPBOARD-001`.
- [ ] Copy-button click verified on macOS at runtime (menu ⌘C was verified; button clicks were
      not automated because WKWebView web content is not exposed to System Events accessibility).

---

## 16. Implementation status (2026-09-24)

Both fixes are implemented and committed on `main`:

| Commit | Contents |
|--------|----------|
| `5a4876b` | Ticket docs, guide, diary, sources |
| `ab36db9` | Fix A: native App/Edit menus on darwin + structural test |
| `d70229b` | Fix B: `App.CopyText` + `MDSCopyText` routing |

Verified on macOS:

- Menu bar reads `Apple, md-view, File, Edit, View` (queried via `System Events`).
- With the clipboard seeded to a sentinel, `⌘A` then `⌘C` in the window replaced the
  sentinel with 7108 bytes of rendered README text.
- `make build` regenerated bindings; `CopyText` appears in `frontend/wailsjs/go/main/App.d.ts`.
- `make test` passes; `gofmt -l` and `go vet -tags webkit2_41 .` are clean.

Not verified / limitations:

- In-page copy-button *clicks* were not driven end-to-end; WKWebView does not expose its web
  buttons to macOS accessibility, and no inspector automation was set up. The Go binding and
  the shared JS helper are present and build-verified.

Resolved after the initial write-up: `make lint` was failing because golangci-lint v2.11.2
pinned `x/tools v0.42.0` (unified-IR decoder max V2) while Go 1.27.1 emits V4. Bumped
`.golangci-lint-version` to `v2.14.0` (x/tools v0.50.0); `make lint` now reports 0 issues.

