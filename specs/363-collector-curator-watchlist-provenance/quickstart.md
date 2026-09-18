# Quickstart: Feature 363 Validation

All acceptance uses controlled fixtures. Do not depend on live mutable dealer,
auction or attribution sources.

## Prerequisites and rollout

1. Go API, Vue app and Python agent service are running.
2. Two normal owners plus an administrator exist.
3. Existing portfolio/gap, wishlist availability/search-alert, auction,
   Deep Analysis and Coin Copilot suites pass as a baseline.
4. Enable flags in order: profile, curator, evaluation, risk, wishlist action.
5. Keep Feature 362 disabled for baseline scenarios, then enable its controlled
   projection only for enriched-risk scenarios.

## Scenario 1 — Private collector profile

1. As owner A, open **Settings > Collector Profile** on desktop and installed
   PWA. Verify missing storage displays neutral defaults.
2. Save boundary-valid budgets, lists and 50 goals; verify one atomic version
   increment.
3. Submit negative/non-finite/reversed/oversized budgets, invalid currency,
   duplicate normalized list entries, 51 goals, oversized text, unknown enums
   and stale version; verify field errors and no partial change.
4. Race two updates from the same version; exactly one succeeds.
5. As owner B and admin, request A's guessed profile/goal ids; verify unknown-
   equivalent response and zero data/log leakage.
6. Disable the flag; verify profile is inert/recoverable and no admin setting
   or other user field changed.

## Scenario 2 — Read-only curator

1. Seed known collection facts, gaps and conflicting/sparse metadata.
2. Run curator with profile version N; edit profile during execution.
3. Verify output cites only the N snapshot, separates observations from
   recommendations, and shows strengths/themes/gaps/acquisition ideas,
   confidence, why-it-matters, profile effects and limitations.
4. Verify disliked/budget-conflicting results are excluded or visibly
   deprioritized; empty profile invents nothing.
5. View, rerun, replay, dismiss and cancel; compare database state and verify
   zero profile/collection/wishlist/auction mutation beyond normal run records.

## Scenario 3 — Watchlist evaluation

1. Evaluate owned wishlist items, active goals, and retained Feature 361 dealer/
   auction results containing duplicates, missing prices, incomparable
   currencies, stale states and provider degradation.
2. Verify goal/budget/preference/dealer fit, owned/wishlist duplicates,
   collection coverage, source quality, listing state and price comparability
   are closed typed checks with explicit unknowns.
3. Verify availability and auction state link to canonical records and are not
   rewritten.
4. Submit a direct URL, saved search, foreign coin/goal/run/result, altered
   digest and raw provider payload; all fail closed with no provider call.
5. Replay a completed run; provider call counts remain unchanged.

## Scenario 4 — Tiered risk and Feature 362

1. Exercise every allowed finding kind at low/medium/high confidence.
2. Verify every finding contains evidence or a missing-field fact, tier,
   confidence, why-it-matters, limitations and visible “needs review.”
3. Feed accusation/fraud/authenticity text, unsupported claims, invalid URLs,
   missing citations and low confidence. Verify unsafe output rejects; valid
   low confidence remains visible in `review_low`.
4. Provide duplicate-image evidence with exact existing method/basis and
   limitations, then omit that record; only the first may yield a finding.
5. Disable Feature 362. Baseline findings work and attribution-dependent
   behavior returns `attribution_evidence_unavailable`.
6. Enable a validated Feature 362 projection. Verify its confidence, conflicts,
   provider coverage, URLs and limitations survive unchanged.

## Scenario 5 — Explicit stage, review and confirmation

1. Render retained eligible dealer and auction cards. Verify only those typed
   items have an accessible **Add to Wishlist** button; prose/non-market/stale/
   cancelled/unverified items do not.
2. Click the button. Verify Go creates only an immutable stage and the review
   shows provider/source, evidence, observation, confidence, listing state,
   price/currency, mapped/omitted fields, limitations and duplicates.
3. Verify mapping sets `IsWishlist`, uses `CurrentValue` rather than
   `PurchasePrice`, and does not populate purchase fields, invoice/SKU, sold,
   storage, visibility, images, notes or unsupported references.
4. Cancel/close/wait for expiry; zero coins and outcomes exist.
5. Confirm with the exact stage version/fingerprint and fresh idempotency key.
   Verify one owner-scoped wishlist coin, immutable outcome and assisted-create
   coin journal record commit atomically.
6. Replay same request/key; receive the same result. Reuse key with changed
   fingerprint; receive `409`. Race two tabs/keys for the same source; at most
   one coin exists.
7. Alter owner, stage, version, URL, provider, provenance, result digest,
   price/currency, mapping, expiry or confirmation literal; fail closed.
8. Verify Python has no matching route/tool/DTO and cannot self-approve.

## Scenario 6 — Threat and tamper matrix

- **Leakage**: private profile, prices, images, facts and recommendations never
  reach owner B, follower/public DTOs, telemetry, browser storage or logs.
- **Prompt injection**: provider/goal text asking for tools, prompts, secrets or
  writes remains inert data; no allowlist or output safety changes.
- **False accusations**: banned fraud/inauthentic/person/dealer assertions
  reject the whole finding, not merely hide the label.
- **SSRF**: file/non-HTTPS/user-info/localhost/private/link-local/metadata,
  DNS rebinding, unsafe redirect and unregistered host fixtures are rejected.
- **Duplicate images**: transformed/cropped/partial matches disclose the
  method and uncertainty; no match identifies an “original.”
- **Stale listings**: freshness expiry or authoritative state conflict blocks
  staging/confirmation or becomes an explicit unknown.
- **Cancellation/replay**: cancel before capability/stage/commit yields no
  late result/write; committed confirmation remains idempotently readable.
- **Cross-user ids**: every profile/goal/coin/run/result/stage id is
  unknown-equivalent across owners.

## Focused automated tests

### Go

- repository: owner scopes, unique indexes, migration order, neutral defaults,
  profile/goal transaction and stage/outcome immutability;
- service: every validation boundary, snapshot consistency, composition reuse,
  mapping allowlist, manual-field preservation, eligibility/freshness,
  duplicate/idempotency and cancellation linearization;
- handler/contract: strict JSON, body cap, auth, foreign-equals-unknown,
  sanitized errors and Swagger;
- integration: stage→confirm atomicity, two-tab race, transaction rollback,
  journal linkage and existing workflow regression;
- architecture: no GORM in handlers/services, no internal mutation callback,
  no Python write route.

```powershell
Push-Location src/api
go test ./repository ./services ./handlers ./integration -run 'Collector|Curator|WatchlistAction|Risk|Wishlist' -count=1
go test . -run 'TestArchitecture|TestRegisteredAPIRoutesAreDocumentedInOpenAPI' -count=1
Pop-Location
```

### Python

- strict request/result fixtures and every enum/bound;
- prompt-injection/token text, unsafe citations, raw provider keys and unknown
  fields;
- risk language, low-confidence visibility and Feature 362 gate;
- cancellation/replay and payload truncation;
- architecture import/allowlist tests proving read-only inference.

```powershell
Push-Location src/agent
uv run ruff check app/ tests/
uv run pytest tests/test_coin_copilot_contract.py `
  tests/test_coin_copilot_harness.py `
  tests/test_coin_copilot_security.py `
  tests/test_coin_copilot_architecture.py `
  tests/test_coin_copilot_specialists.py `
  tests/test_collector_workflows.py -v
Pop-Location
```

### Vue

- profile form boundaries/version conflict/clear behavior;
- specialist-card eligibility and absence on other cards;
- button keyboard/touch event, review focus trap/restore, ARIA/live status,
  confirmation separation, cancel/expiry and retry;
- stage rendering never trusts raw card JSON and never calls Python;
- 320px mobile, installed PWA, dark/high-contrast/reduced-motion and offline
  unavailable states.

```powershell
Push-Location src/web
npx vitest run `
  src/components/settings/__tests__/CollectorProfile.test.ts `
  src/components/chat/__tests__/CopilotRunProgress.collector.test.ts `
  src/components/wishlist/__tests__/WishlistActionReview.test.ts `
  src/composables/__tests__/useCoinCopilot.test.ts
npm run type-check
npm run test:browser
Pop-Location
```

## Existing-workflow regression gate

Before and after implementation, run targeted suites proving unchanged:

- collection create/update/portfolio/gap and Feature 012 tools;
- wishlist create plus availability Features 337/353 and alert-candidate
  conversion;
- auction tracking;
- Deep Analysis Features 344/351/352 and Feature 362 when present;
- Coin Copilot harness/specialists, replay/cancel/fallback;
- legacy `/api/agent/chat`.

Database assertions must compare affected row sets, not only HTTP results.

## Full Quality Gate

```powershell
Push-Location src/api
go build ./...
go vet ./...
go test ./...
$env:CGO_ENABLED='1'; go test -race ./...; Remove-Item Env:CGO_ENABLED
Pop-Location

Push-Location src/agent
uv sync --locked --extra dev
uv run ruff check app/ tests/
uv run pytest tests/ -v
Pop-Location

Push-Location src/web
npm ci
npm run lint
npm run type-check
npm run test
npm run build
npm run test:browser
Pop-Location

task openapi
git diff --exit-code -- src/api/docs/docs.go src/api/docs/swagger.json src/api/docs/swagger.yaml docs/openapi.json
```

Also require:

1. `.github/workflows/ci.yml` green, including Go race, Vue and Python jobs.
2. `.github/workflows/security-scan.yml` green: Gitleaks, Govulncheck,
   `npm audit`, `pip-audit`, and agent runtime-image pip/health checks.
3. Repository CodeQL/default code-scanning checks green. There is no checked-in
   CodeQL workflow today; do not invent or bypass one—verify the repository
   branch-protection check or add a separately reviewed pinned-SHA workflow if
   maintainers require it.
4. Docker image workflows/builds green, including
   `docker-publish-beta.yml`, `docker-publish.yml` when release-targeted, and
   the agent runtime image check; no push/deploy occurs during implementation
   validation without explicit release authorization.
5. `.github/workflows/ai-browser-exploration.yml` and critical browser
   workflows green with controlled provider fixtures and privacy-safe
   artifacts.
6. All actions remain SHA-pinned; no new dependency/provider is introduced.
7. Run the `post-major-work-qc-audit` skill after the implementation is
   complete. Resolve all High/Critical findings and document lower-severity
   dispositions before merge.

## Expected result

Each phase can be enabled and rolled back independently. Pre-362 profile,
curator, watchlist, baseline risk and confirmed wishlist action remain usable;
only attribution-rich findings wait for Feature 362. No recommendation,
evaluation, risk result, Python/model action, cancellation or replay can write.
Only the explicit Vue event plus separate Go confirmation creates one audited,
owner-scoped wishlist item.
