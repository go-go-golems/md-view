---
title: "Secure-context compat pack for non-secure asset origins · Issue #592 · water-rs/waterui · GitHub"
author: "lexoliu"
site: "GitHub - water-rs/waterui"
published: 2026-09-12T23:45:47.000Z
source: "https://github.com/water-rs/waterui/issues/592"
domain: "github.com"
language: "en"
description: "Problem On WKWebView the bundled-asset origin (#586) is a custom scheme, which is not a secure context. Every Web platform API gated on isSe"
word_count: 461
---

## Problem

On WKWebView the bundled-asset origin ([#586](https://github.com/water-rs/waterui/issues/586)) is a custom scheme, which is not a secure context. Every Web platform API gated on `isSecureContext` is then unavailable to the page — `crypto.subtle`, `navigator.clipboard`, `navigator.share`, `Notification`, WebAuthn/passkeys, OPFS, `SharedArrayBuffer`, and more. A migrated web frontend discovers each one as a runtime failure.

## Proposal

A **secure-context compat pack**: a set of `DocumentStart` scripts, installed only when `window.isSecureContext === false` (so secure engines pay nothing), that fills each missing API at the best fidelity available and fails fast on the ones that cannot be filled.

Three tiers, and the tier is the design constraint:

### Faithful or near-faithful — implement

- `crypto.subtle` — pure-JS/WASM implementation (e.g. a noble-style audited impl). Hashing (PKCE code challenges!), sign/verify, derive all work. For `generateKey({ extractable: false })` the key material can live in the platform keychain via bridge (`waterkit-secret`) and be referenced by id — strictly better than the real API's in-heap keys.
- `crypto.randomUUID` — trivial over `getRandomValues` (which is not gated).
- `navigator.clipboard` — bridge to `waterkit-clipboard`.
- `navigator.share` — bridge to `waterkit-share`.
- `Notification` + `requestPermission` — bridge to `waterkit-notification`.
- `CacheStorage` (`caches`) — over IndexedDB or a bridge-backed KV; `match`/`put`/`delete` fidelity is achievable.

### Feasible, real work — implement deliberately

- `navigator.geolocation` → `waterkit-location`.
- File System Access (`showOpenFilePicker`, async `FileHandle`) → `waterkit-dialog` + `waterkit-fs`. Async surface only.
- `navigator.credentials` (WebAuthn `create`/`get`) → `waterkit-passkey`. Signatures come from the real platform authenticator, so credentials are genuine — but the ceremony JSON (`clientDataJSON`, attestation handling) must be constructed exactly.
- OPFS async surface → bridge-backed fs. Sync access handles in workers are excluded: they need `Atomics.wait` over `SharedArrayBuffer`, which is itself gated — document this, don't approximate it.

### Cannot be polyfilled — never fake

- `SharedArrayBuffer` / `crossOriginIsolated` / WASM threads — an engine-level memory primitive.
- Service Worker interception semantics — engine-level; moot for bundled assets anyway.
- WebRTC, `getUserMedia` as a real `MediaStream` — at best an approximation via `canvas.captureStream()`; mark unsupported first.
- SRI (`integrity=` enforcement) and `isSecureContext` itself — do not redefine the property and lie to feature detection.

## Rules

- Installed only where `isSecureContext` is false; identical code path on every engine, gated by the runtime check rather than per-engine lists.
- A member the pack does not implement throws a clear error naming the gap — never a silent partial.
- The pack widens nothing: it uses the handlers/`#[js_api]` surface the application already exposes, under the same `OriginPolicy`.

## Acceptance

- A page calling `crypto.subtle.digest('SHA-256', …)` under the Apple asset origin resolves correctly.
- `navigator.clipboard.writeText` under the same origin round-trips through the system clipboard.
- `typeof SharedArrayBuffer` remains `"undefined"` and the pack's support matrix reports it as unsupported.