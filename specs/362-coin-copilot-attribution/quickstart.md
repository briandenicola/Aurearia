# Quickstart: Feature 362 Verification

## 1. Mandatory pre-implementation gates

1. Record that the ADR 0017 acceptance gate is satisfied.
2. Build/archive the Phase 0 compatibility-guard Release A artifact and pass
   all required local or hosted guard/rollback evidence.
3. Stop at an explicit external/manual checkpoint and obtain separate user
   approval before deploying Release A. CI success, merge, artifact creation,
   or registry/image publication cannot be treated as deployment.
4. Deploy Release A everywhere and record the deployed version/commit,
   environment, approval reference, verification run URLs/ids, and results.
   No migration, `source_draft_id`, source recognition, or `copilot_draft` row
   work may begin until this record exists.
5. `CoinCopilotAttributionEnabled` exists, defaults false, and admits zero work
   through `models/appsetting.go` and constants in
   `services/settings_service.go`.
6. Deterministic Go race tests for cancellation-first, admission-first,
   changed-target idempotency, and duplicate admission are red before
   orchestration implementation and green before the callback is registered.

No local environment limitation waives a gate. If a command cannot run locally,
the equivalent hosted job must pass and its URL/run id must be recorded in the
PR before merge or release.

## 2. Compatibility and rollback matrix

Before Release A deployment, its hosted guard gate runs only the pre-schema
unknown-source/default-off checks and archives the exact guard artifact. It
must not run a Feature 362 migration or create `source_draft_id`,
`copilot_draft`, or handoff rows.

After the separately approved Release A deployment is recorded, the
implementation must run the full executable matrix (for example
`scripts/compat/feature362-rollback.ps1`) in a hosted job:

```powershell
pwsh scripts/compat/feature362-rollback.ps1 `
  -GuardBinary artifacts/feature362-guard/aurearia `
  -FeatureBinary artifacts/feature362-current/aurearia
```

It must:

1. start the guard binary on a copied pre-feature database;
2. raw-seed an unknown source and prove list/get/status/stream/retry/apply and
   worker adoption reject it without mutation;
3. run the Feature 362 migration and verify the flag is default off;
4. enable the flag, create `saved_coin` and `copilot_draft` handoffs, and let
   representative jobs settle;
5. disable new handoffs, drain/cancel accepted work, then start the guard binary
   against a copy of the upgraded database;
6. prove `copilot_draft` adoption/apply fails closed and rows/reports/proposals
   remain byte-for-byte;
7. restart the Feature 362 binary and prove status/review/apply access returns;
8. prove known `intake` and `saved_coin` direct workflows remain compatible.

Rollback to a binary older than the compatibility guard is a failed test and is
not an approved operational path.

## 3. Core workflow

1. Enable Coin Copilot, Deep Analysis, and Feature 362 after migration.
2. Ask Copilot to attribute an exact collection coin, wishlist coin, and active
   Quick Capture draft with distinct valid faces.
3. Repeat/replay each request; verify one handoff binding and one equivalent
   active Deep job, with no repeated provider calls.
4. Change target kind/id, route app context, checkpoint version, face image,
   notes/context, provider generation, or draft state while reusing the same
   handoff key; verify HTTP 409 and zero new jobs.
5. For `rerun`, reuse the same handoff key with a different prior job id;
   verify HTTP 409, zero new handoff/job rows, and zero worker/provider calls.
6. Complete/partially complete jobs and request them again; verify retained
   result reuse and an explicit rerun choice.
7. Open `/deep-analysis/{jobId}` from the existing drawer and use the existing
   proposal editor.

## 4. Apply matrix

For each destination:

- seed manual scalar values, manual notes, images, acquisition/value/storage/
  privacy/status data, relationships, and existing mixed-case equivalent
  references;
- accept one valid scalar, notes, and one new plus one equivalent reference;
- reject all other fields and confirm in the existing review page;
- verify the selected scalar alone changed;
- verify exactly one dated/job-id/source notes block was appended and replay
  updates that block without duplication;
- verify new references append, equivalent references dedupe
  case-insensitively, and existing references remain;
- submit one unsupported/stale field and verify the whole apply rolls back.

Exact valid scalars:

- collection/wishlist: `denomination`, `ruler`, `era`, `dateRange`, `mint`,
  `material`, `weightGrams`, `diameterMm`, `obverseInscription`,
  `reverseInscription`, `obverseDescription`, `reverseDescription`,
  `coin_type`;
- existing draft: `workingTitle`, `era`, `dateRange`.

For a draft, verify accepted references remain staged in the bound proposal and
join the validated promotion transaction without replacing the draft's existing
selected reference.

## 5. Eligibility, cancellation, flags, and bounds

- Unbound legacy intake, unknown/foreign/unbound job, unknown source,
  previously bound deleted coin, and promoted/discarded/deleted bound draft
  all return canonical public bytes
  `{"reason":null,"status":"not_eligible"}` with no metadata.
- Verify privacy-safe internal diagnostic codes, if emitted, cannot change
  public status/body/headers/event projection or appear in public logs/events.
- Cancel-first race: zero handoff/job rows.
- Admission-first race: exactly one binding/job; cancellation requests Deep
  cancel and late report/proposal/event settlement loses.
- Disable any gate before admission: no work.
- Disable after admission: job may finish; status/events/cancel/review/edit/
  confirmed review-page apply work; rerun/new handoff does not.
- Canonical request/public event at 65,536 bytes passes and 65,537 fails.
- Persisted result never exceeds 32,768 bytes; repeated projection produces
  identical bytes, counts, omission order, and full-result digest.
- Final answer and Vue card disclose omitted evidence.
- A user JWT, wrong execution/tool token, expired/revoked token, or request body
  owner is rejected.

## 6. Mandatory quality commands

```powershell
# Go: build, vet, architecture, unit/integration, migration/rollback
Set-Location src/api
go build ./...
go vet ./...
go test -run TestArchitecture ./...
go test ./...
go test ./integration -run 'Feature362|RollbackCompatibility|AttributionHandoff' -count=1

# Python: dependency installation, syntax/build, strict contract typing, lint, tests
Set-Location ../agent
uv sync --extra dev
uv run python -m compileall -q app tests
uv run ruff check app tests
uv run pytest tests/test_coin_copilot_contract.py tests/test_coin_copilot_harness.py -v
uv run pytest tests -v

# Web: clean install, lint, strict type/build, tests
Set-Location ../web
npm ci
npm run lint
npx vue-tsc --build
npm run test -- --run
npm run build

# Root hosted compatibility gate
Set-Location ../..
pwsh scripts/compat/feature362-rollback.ps1 `
  -GuardBinary artifacts/feature362-guard/aurearia `
  -FeatureBinary artifacts/feature362-current/aurearia
```

If the repository's actual scripts differ when implementation starts, tasks
must use the authoritative equivalent commands; they may not omit the gate.

## 7. Mobile/PWA and regressions

- Resume both Copilot and Deep SSE streams from their independent sequence.
- Verify the drawer handoff card and existing Deep page at narrow viewport,
  keyboard navigation, and 44 px touch targets.
- Confirm no proposal editor or apply button appears in chat.
- Re-run Fast Identify, direct Deep Analysis, legacy chat fallback, four
  collection callbacks, specialist tools, provider attribution/license, and
  collection/wishlist/draft preservation suites.
