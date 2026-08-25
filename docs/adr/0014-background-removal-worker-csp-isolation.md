# ADR 0014: Scoped-Worker CSP Isolation for Background-Removal `unsafe-eval`

Date: 2026-08-24
Status: Accepted

## Context

`@imgly/background-removal` 1.7.0 vendors `ndarray`, whose
`compileConstructor()` calls JavaScript `new Function(...)` at runtime to
build a typed-array constructor. Under the SPA's strict `script-src` this
throws `EvalError: Evaluating a string as JavaScript violates the following
Content Security Policy directive: "script-src ..."`, reproduced and
diagnosed experimentally (`.squad/agents/aurelia/history.md`, commits
`706435fe` and `9d8272dd`). `'wasm-unsafe-eval'` — already granted for
compiling the library's ONNX model to WebAssembly — is a distinct grant from
plain JavaScript `'unsafe-eval'` and does not cover this call; the failure is
specifically the JS `new Function()` path, not WASM compilation.

The SPA stores its JWT access token in `localStorage` (Constitution
Principle V) and is rendered by a single `index.html` document whose CSP
(`appCSP` in `src/api/middleware/csp.go`) is the primary control keeping an
injected script from reaching that token. `script-src` is the directive that
matters most for that threat model.

### Forces

- The failing call is inside a vendored, minified third-party bundle
  (`@imgly/background-removal` → `ndarray`), not application code.
- The SPA document and the background-removal code share no functional need
  to be in the same script-src trust boundary: the library only ever
  receives an image `Blob` and returns a result `Blob` or a plain
  `{name, message}` error (`src/web/src/workers/backgroundRemovalWorker.ts`).
  It has no legitimate reason to touch `window`, `document`, or
  `localStorage`.

## Decision

**Isolate the `eval`-needing code inside a dedicated same-origin ES-module
Worker, and grant `'unsafe-eval'` only to that worker's own script response —
not to the SPA document.**

### What was rejected, and why

1. **App-wide `'unsafe-eval'` in `appCSP`.** Rejected. The SPA holds the JWT
   in `localStorage`, reachable from any script executing in the document's
   origin/realm. Adding `'unsafe-eval'` to the document-level `script-src`
   would let any future XSS payload — not just this vendored call —
   eval arbitrary JavaScript in that same realm, materially weakening the
   one control (Principle V) that keeps an injected script from reaching the
   token. This is a permanent, standing weakening of the whole app for one
   third-party library's one internal call.
2. **Patch the vendored `ndarray`/`@imgly/background-removal` bundle to avoid
   `new Function()`.** Rejected. Hand-patching a transitively vendored,
   minified numerical bundle to remove a `new Function()` call it relies on
   for constructing typed-array views is a correctness risk (subtle
   numerical/typed-array behavior is exactly the kind of code most likely to
   regress silently) and a standing maintenance burden every time the
   dependency is upgraded — patches would need to be re-derived and
   re-verified against a new minified build on every bump.
3. **Run background removal in an `<iframe>`, or replace the library
   entirely.** Rejected as disproportionate. An iframe boundary would still
   need its own CSP/sandboxing design work and still shares more surface
   (a document, potential postMessage misuse) than a dedicated Worker does;
   replacing a working, otherwise-correct library over one CSP directive
   fails Principle IV's proportionality requirement.

### What was selected

- **Vite** (`src/web/vite.config.ts`, `worker.rollupOptions.output`) emits
  the background-removal module worker and everything it statically imports
  — `backgroundRemovalWorker.ts`, the vendored `@imgly/background-removal`
  and `onnxruntime-web` chunks — under a stable path prefix,
  `assets/workers/[name]-[hash].js`, distinct from the SPA's own
  `assets/[name]-[hash].js` output.
- **Go** (`src/api/middleware/csp.go`) matches on that exact prefix,
  `backgroundRemovalWorkerPathPrefix = "/assets/workers/"` (trailing slash
  load-bearing, so `/assets/workers-evil/...` cannot match), and serves a
  narrow `workerScriptCSP` for responses under it:
  `default-src 'none'; script-src 'self' 'unsafe-eval' 'wasm-unsafe-eval'
  blob:; connect-src 'self' blob:; worker-src 'self' blob:; object-src
  'none'`. Every other response path — including the SPA document and every
  other static asset — keeps the unmodified, strict `appCSP` (`script-src
  'self' 'wasm-unsafe-eval' blob:`, no `'unsafe-eval'`).
- **`src/web/src/utils/backgroundRemoval.ts`** is the only caller-facing
  surface: it lazily constructs a page-level singleton `Worker` and
  communicates over a small, structured-clone-safe request/response protocol
  (`{id, image}` in; `{id, ok, result}` or `{id, ok:false, error:{name,
  message}}` out). No DOM object, cookie, token, or `window`/`document`
  reference ever crosses into or out of the worker.

### Why this is a real isolation boundary, not just a narrower blast radius

A dedicated same-origin module `Worker` has no `window`, no `document`, no
DOM, and cannot reach `localStorage`/`sessionStorage` (not part of
`WorkerGlobalScope`). It never receives the SPA's JWT — the only things
posted into it are image `Blob`s, and the only things posted out are result
`Blob`s or a plain error shape. So even with `'unsafe-eval'` granted inside
that context, there is no token, no cookie jar, and no document for an
eval'd payload to act on. `workerScriptCSP` additionally omits every
document-oriented directive (`style-src`, `img-src`, `font-src`, `base-uri`,
`form-action`, `frame-ancestors`, `manifest-src`) rather than inheriting them
from `appCSP`, so `default-src 'none'` is the true fallback for anything not
explicitly listed — least privilege, not merely a smaller version of the
same grant.

## Validation

- **Go, static boundary:** `TestBackgroundRemovalWorkerAssetIsReachableWithWorkerCSP`
  (`src/api/main_static_test.go`) wires the real `ContentSecurityPolicy()`
  middleware ahead of the real `configureStaticRoutes`, and asserts a file
  under `wwwroot/assets/workers/` carries `'unsafe-eval'` while a sibling
  file directly under `wwwroot/assets/` does not — exercising the real static
  handler, not a mock. Prefix-confusion paths (`/assets/workers-evil/...`)
  are separately asserted to fall through to `appCSP`.
- **Frontend, worker client:** `src/web/src/utils/__tests__/backgroundRemoval.test.ts`
  unit-tests request/response correlation under concurrency, the fatal
  `error`/`messageerror` paths (reject in-flight requests, terminate and
  recreate the singleton worker on the next call, and — added during the
  Strict Lockout revision below — that a late/stale event from an
  already-replaced worker cannot corrupt a newer worker's in-flight
  requests).
- **Real production build, real browser:** a Chromium session driven against
  the actual Go `SecurityHeaders()` + `ContentSecurityPolicy()` +
  `configureStaticRoutes()` middleware chain, serving the real `npm run
  build` output with real downloaded `@imgly/background-removal` model
  assets, produced a non-empty output PNG end-to-end with zero CSP
  violations logged, and confirmed the nested onnxruntime-web em-pthread
  worker pool (spawned when `crossOriginIsolated` is true, which it is here
  via this app's global COOP/COEP headers) is constructed from `blob:` URLs
  already covered by the existing `worker-src 'self' blob:` grant in both
  `appCSP` and `workerScriptCSP` — not from the two same-named-but-orphaned
  `ort(.webgpu)?.bundle.min-*.mjs` chunks Vite also emits outside
  `/assets/workers/` (those are dead code in a browser: their only reference
  is gated on `import.meta.url.startsWith('file:')`, which is never true for
  a script loaded over http/https). No widening of the CSP boundary was
  required by this finding.
- **Independent reviewer gate:** Brutus's initial review
  (`.squad/decisions/inbox/brutus-background-removal-worker-review.md`)
  confirmed the CSP boundary itself was correct and blocked only on a
  frontend defect (the singleton worker was never recreated after a fatal
  `error`/`messageerror` event, so every request after one crash hung
  forever). That defect was fixed under Strict Lockout by the revision owner
  (commit `3a0d7b04`, `.squad/agents/maximus/history.md`), and Brutus
  subsequently cleared the block and APPROVED the revision.

## Consequences

### Positive

- The SPA document's `script-src` never carries `'unsafe-eval'`. The one
  control protecting the `localStorage`-held JWT (Principle V) is unchanged
  by this feature.
- The blast radius of a hypothetical exploit of the vendored library's own
  `new Function()` call is a worker with no DOM, no storage, and no token —
  not the document.
- The boundary is enforced by an exact path prefix plus a real Go contract
  test, not by convention or comment.

### Negative and trade-offs

- Two CSP branches now exist in `src/api/middleware/csp.go` instead of one
  (`appCSP` and `workerScriptCSP`), plus the pre-existing `swaggerCSP`. Any
  future change to `vite.config.ts`'s `worker.rollupOptions.output` path
  (`assets/workers/[name]-[hash].js`) or to `backgroundRemovalWorkerPathPrefix`
  must keep the two in sync, or the worker script silently reverts to the
  strict `appCSP` and background removal breaks again with the original
  `EvalError`. `TestBackgroundRemovalWorkerAssetIsReachableWithWorkerCSP` is
  the regression guard for that drift; it must be kept green and updated in
  the same PR as any such path change.
- The two orphaned `ort(.webgpu)?.bundle.min-*.mjs` chunks Vite emits outside
  `/assets/workers/` are currently harmless dead weight (never fetched in a
  real browser trace) but are not actively pruned. A future
  `onnxruntime-web` bump that changes the em-pthread bootstrap to reference
  them directly (instead of self-referencing via `import.meta.url`) would be
  a new, observable regression against this ADR's stated real-browser
  finding — not a currently-live gap, but worth re-checking on any
  `onnxruntime-web`/`@imgly/background-removal` upgrade.
- This ADR's real-browser validation was run against a local `httptest`
  instance of the production middleware chain serving a local production
  build, not a deployed environment. **No deployment occurred as part of
  this decision or its validation.** A real coin-photo smoke test in the
  actually deployed environment remains the outstanding deployment-time
  verification step before this feature is considered fully proven in
  production.
- No app-wide CSP relaxation was made or is authorized by this ADR;
  `'unsafe-eval'` remains scoped to exactly the `/assets/workers/` response
  path.

## Related

- [ADR 0001](0001-record-architecture-decisions.md) — ADR process.
- `src/api/middleware/csp.go`, `src/api/main_static_test.go`
- `src/web/vite.config.ts`, `src/web/src/utils/backgroundRemoval.ts`,
  `src/web/src/workers/backgroundRemovalWorker.ts`
- `.squad/agents/aurelia/history.md` — original CSP-failure diagnosis and
  isolation design (commits `fbbe8503`, `706435fe`, `9d8272dd`).
- `.squad/decisions/inbox/brutus-background-removal-worker-review.md` —
  independent reviewer gate; BLOCK on the worker-recreation defect, cleared
  after revision.
- `.squad/agents/maximus/history.md` — Strict Lockout revision record
  (commit `3a0d7b04`) and this ADR's real-browser nested-worker
  investigation.
- Constitution Principle IV (Simple Complete Changes — rejecting the patch
  and iframe/replacement alternatives as disproportionate), Principle V
  (Security, Auth, and Privacy by Default — the `localStorage` JWT threat
  model this boundary protects), Principle VII (CI, Supply Chain, and
  Release Integrity — commit hygiene for `fbbe8503`, `6cdce47d`, `3a0d7b04`),
  §17 (Quality Gate), §21 (Definition of Done, item 12: ADR required for a
  material design choice).
