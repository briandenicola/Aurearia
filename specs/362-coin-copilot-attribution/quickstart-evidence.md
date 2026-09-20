# Feature 362 Verification Evidence

## Release A compatibility prerequisite

- ADR 0017 is accepted at commit
  `11bf09d60df95a81d0f4af9b51f6706caecdc977`.
- Release A must contain the closed Deep job-source compatibility guard before
  any Feature 362 schema, source value, callback, or row-producing work.
- `CoinCopilotAttributionEnabled` exists and defaults to `false`.
- Minimum compatibility-guard commit:
  `0910bc7b86e6d5a567c62f8b25000c64fca493a2`.

## Local guard evidence

Date: 2026-09-18

- Red test-first run failed because the unknown-source error/validator and
  default-off attribution setting did not exist.
- Focused guard suite:
  `go test ./handlers ./repository ./services -run "UnknownSource|UnknownSources|CoinCopilotSettingsDefaultsAndIndependentFallbacks|AdminOCRE|GetLatestProviderStatus"`
  passed.
- The immutable Windows guard binary built from
  `0910bc7b86e6d5a567c62f8b25000c64fca493a2` passed the guard-only rollback
  harness with SHA-256
  `c0de8ac10ba956373057d570b16acfb46f24aa0440c15719daeda93c003e4338`.
- Tamper test temporarily recognized `copilot_draft` in the compatibility
  binary. The handler, repository, creation, and pipeline guards failed as
  required. The tamper was reverted.
- No `copilot_draft` source constant, `source_draft_id` column, handoff table,
  callback, migration, or Feature 362 row was added.

## Hosted artifact evidence

- Workflow run:
  [Feature 362 Compatibility 35393314716](https://github.com/briandenicola/Aurearia/actions/runs/35393314716)
  passed, including immutable build, guard-only harness, and artifact upload.
- Guard artifact: `feature362-guard-35393314716-1` (artifact id
  `10567285694`, 30,009,704 bytes).
- Guard artifact digest:
  `sha256:9e7a9993eb5b8a5440006fa105b1b17810fbe43b09b29febe2c963a248c7141e`.
- Guard artifact commit:
  `0910bc7b86e6d5a567c62f8b25000c64fca493a2`.

## External/manual Release A checkpoint

Status: **COMPLETE - SCHEMA WORK APPROVED**

Deployment date: 2026-09-18

- The owner explicitly approved deploying Release A to the disposable beta
  environment, then corrected the deployment target from Azure to the existing
  local server. No Azure deployment or resource change occurred.
- Environment: `https://coins-beta.denicolafamily.com`.
- Deployed app/agent commit:
  `1fd371a0adc42fdffd4e35136f37519f688fd9a0`.
- App OCI index digest:
  `sha256:6e280855644e55a79f2b574c890b69865e504d8e6d5686d7f7c43393eba253b0`.
- Running app platform digest:
  `sha256:f7d62405e6caedf5569521e7e5db95717d8be12431e6f349dccf10a9a54784eb`.
- `/health` returned HTTP 200 with `{"status":"ok"}` after deployment and
  again after the worker-adoption restart.
- A transactionally consistent pre-fixture SQLite backup was retained at
  `/home/brian/f014-release-a-20260918T210809Z.db`, size 1,712,128 bytes,
  SHA-256
  `8da4f9dd373888e7889698628de1e959a88f36f969846c53311cb5b78be3dbdf`.
- `DeepIdentificationEnabled` was `true`.
  `CoinCopilotAttributionEnabled` had zero database overrides and therefore
  resolved to its built-in default `false`.
- Raw fixture: owner-scoped job `13`, source `copilot_draft`, artifact `26`.
  Its canonical row/artifact/file preservation digest was
  `013b2fde46471e3b88eeec565c8bf2966db98ff6d94c2c9730f6f92924127964`.
- Guarded HTTP results:
  - list: HTTP 200, fixture absent;
  - get/status, stream, retry, cancel, proposal edit, and apply: HTTP 404;
  - no response disclosed source, report, proposal, notes, or artifact data.
- Worker-adoption verification restarted only the app container with Deep
  Analysis enabled. After startup and an eight-second adoption window, the
  fixture remained queued with attempt count 0, last sequence 0, zero events,
  zero provider runs, no matching logs, and the exact same preservation
  digest.
- Existing supported-source job `12` (`intake`, `completed`) remained readable
  through the deployed API with HTTP 200.
- Cleanup deleted exactly synthetic job `13`, artifact `26`, and its fixture
  file. All three were confirmed absent afterward. The backup remains retained.

No Feature 362 schema, `copilot_draft` model constant, callback, handoff row, or
source-draft binding had been added at the checkpoint. After reviewing the
completed deployment evidence, the owner explicitly directed: "Take F014 all
the way to completion." This separately authorizes the additive Feature 362
schema and row-producing work after the mandatory Phase 1-2 red tests.

## Local full mixed-binary rollback matrix

Date: 2026-09-18

- Command: `scripts/compat/feature362-rollback.ps1` with an immutable guard
  binary built from `0910bc7b86e6d5a567c62f8b25000c64fca493a2` and the
  current Feature 362 worktree binary.
- Guard SHA-256:
  `977b17b8a339631e8def8793a457211221cc9bb9bb73644173c6be9f4499abb4`.
- Feature SHA-256:
  `7bd01327f62404a08bb55c7657d68b8764926dd4f164edac7df363cdce4a4344`.
- Guard source-vocabulary tests passed in handlers, repository, and services.
- The feature binary, guard binary, and re-upgraded feature binary each
  returned HTTP 200 from `/health` against the same copied SQLite database.
- The copied database retained its supported `intake` row, `copilot_draft`
  row, source-draft binding, handoff row, default-off attribution setting,
  report payload, and artifact bytes across rollback and re-upgrade.
- The guard produced zero events and zero provider runs for the unknown
  `copilot_draft` row; its status remained queued with attempt count and
  sequence both zero.
- Final result: `feature362_compatibility=full-pass`.
- `.github/workflows/feature362-compatibility.yml` now runs this full matrix
  after the immutable guard-only prerequisite on pushes and pull requests to
  `beta` and `main`.
- The expanded fixture now includes completed saved-coin and `copilot_draft`
  handoffs plus a separately queued draft handoff. The authenticated lifecycle
  reads both completed targets and cancels the queued job before rollback.
- The guard still reads the supported `intake` job, returns HTTP 404 for the
  `copilot_draft` job and its apply endpoint, and leaves the canonical
  row/artifact digest unchanged.
- After re-upgrade, the feature binary restores draft and cancelled-job status
  and applies the saved-coin denomination proposal through the real
  owner-authenticated endpoint. The final database verifies the durable
  cancelled state, applied-job linkage, and destination value.
- Expanded local matrix result: `feature362_compatibility=full-pass`; guard
  SHA-256
  `85e39f703e582111646709161e4c9a2b5380cc26e2e057c2e430a30bc4826c40`,
  feature SHA-256
  `32520edb84125bb87ee11e91aa86c8f42bce19680c2f7f57ef9afb14db87d8b4`.

## Feature 362 local quality gates

Date: 2026-09-18

- Targeted Go Feature 362 contract, owner/auth, snapshot, idempotency, status,
  projection, apply, cancellation, and finish-existing suites passed:
  `go test ./services ./handlers ./integration -run "Feature362|DeepAnalysisHandoff|CoinCopilot" -count=1`.
- Architecture and route-contract gates passed:
  `go test -run TestArchitecture ./...` and
  `go test -run "TestRegisteredAPIRoutesAreDocumentedInOpenAPI|TestOpenAPI" ./...`.
- Existing Deep Analysis, Fast Identify/provider, legacy fallback, collection
  tools, specialist tools, manual-data, and additive-reference regressions
  passed through the complete Python, web, and Go suites below.
- OpenAPI was regenerated with `swag init -g main.go -o ./docs
  --parseDependency --parseInternal`; the four generated contract files now
  include optional `activeDraftId`, and OpenAPI route tests pass.
- Python gate passed: editable dev install, `uv sync --extra dev`,
  compileall, Ruff, and `uv run pytest tests -q` (`621 passed`; eight existing
  dependency/serialization deprecation warnings).
- Web gate passed: `npm ci`, ESLint with zero warnings, Vue TypeScript build,
  Vitest (`1,581 passed`, one skipped), and production Vite/PWA build.
- Go build, vet, architecture tests, and complete `go test ./...` passed.
- The cancellation settlement race passed 10 repeated 40-iteration runs
  after closing the cancel-after-result-before-settlement window.
- Local `CGO_ENABLED=1 go test -race ./...` was blocked before compilation
  because this Windows host has no `gcc`; Linux hosted evidence was therefore
  used for that gate.
- Hosted [Quality Gate run 35414768621](https://github.com/briandenicola/Aurearia/actions/runs/35414768621)
  passed all four jobs, including Linux `Go API (race detector)`, Go API,
  Python Agent, and Vue Web. This supplies the required race evidence and
  closes T078.

## Feature 362 local security gates

Date: 2026-09-18

- Gitleaks scanned the Feature 362 checkpoint diff: no leaks found.
- `govulncheck ./...`: zero called vulnerabilities. One required-module
  vulnerability is not reachable from imported code.
- `npm audit --audit-level=high`: zero vulnerabilities.
- `uv run pip-audit`: no known vulnerabilities; the local project package is
  correctly skipped because it is not a PyPI dependency.
- The agent runtime-image no-pip and `/health` smoke could not run locally
  because Docker is not installed on this Windows host. The repository's
  hosted `Agent image - no pip in runtime` workflow check is required before
  T079 can close.
- `.github/workflows/security-scan.yml` now includes app and agent OCI image
  builds with BuildKit SBOM/provenance, Trivy High/Critical enforcement,
  exported SPDX SBOM artifacts, and push provenance attestations.
- Initial OCI-layout scans correctly found patchable High/Critical runtime OS
  vulnerabilities. The app and agent runtime stages now install available
  Alpine/Debian security updates before running as their existing non-root
  users.
- Hosted [Security Scan run 35415353770](https://github.com/briandenicola/Aurearia/actions/runs/35415353770)
  passed all seven jobs: Gitleaks, Govulncheck, npm audit, pip-audit, agent
  no-pip/health, and app/agent container security. Both container jobs passed
  Trivy with zero fixable High/Critical findings, exported SPDX SBOMs, and
  completed provenance attestations. This closes T079.

## Feature 362 browser and mobile evidence

Date: 2026-09-18

- Added deterministic Playwright coverage in
  `src/web/e2e/workflows/coin-copilot-deep-analysis.spec.ts`.
- At a 390 by 844 viewport, the real chat drawer consumed a streamed typed
  Deep Analysis handoff, rendered a conflict, provider coverage, limitations,
  and deterministic omitted-item disclosure, and exposed no Apply action.
- The fixed `Open Deep Analysis` link measured at least 44 px high, the result
  card remained horizontally contained, and activation navigated only to the
  exact same-job `/deep-analysis/17` route.
- The new workflow passed together with all five existing deterministic Deep
  Analysis Playwright workflows, which cover intake/saved-coin admission,
  streamed progress, cancellation, partial proposal review, explicit draft
  apply, and explicit saved-coin apply.
- The expanded browser workflow streams 11 closed handoff results in one
  durable run and visibly verifies collection, wishlist, and draft targets;
  accepted/active/reused/retry/cancelled lifecycle states; missing-image,
  anonymous-ineligible, target-unavailable, and disabled fallbacks; and zero
  coin/draft/proposal/apply mutation requests from conversation. The original
  partial fixture separately verifies conflicts, provider coverage,
  limitations, and deterministic omission disclosure.
- A dedicated wishlist browser review accepts exactly `mint`, `notes`, and
  `catalogReferences`, rejects `denomination`, verifies the actual
  PATCH-proposal decisions followed by the existing POST-apply contract, and
  confirms manual notes, purchase price, storage, and wishlist status are not
  changed or bypassed by a direct coin update. The real SQLite integration
  matrix remains the authoritative persistence proof for collection,
  wishlist, and draft all-or-nothing apply, cancel races, and finish-existing.
- Mobile background/restore is exercised in Chromium by persisting a Coin
  Copilot cursor at sequence 4 and a separate Deep Analysis cursor at sequence
  42, dispatching hidden/visible lifecycle events, reopening the drawer/page,
  and observing independent `?since=4` and `?since=42` requests.
- The mobile workflow activates the review link with keyboard focus plus Enter
  and verifies the 44 px target and horizontal containment at 390 by 844. It
  uses the production dark-theme/token classes and the existing Deep page,
  with no duplicate editor in chat.
- The expanded local browser gate passes nine workflows: the three Coin
  Copilot/mobile cases plus six direct Deep Analysis admission, cancellation,
  draft/coin/wishlist review, and explicit-apply cases. The hosted workflow is
  required to rerun this exact suite before final merge readiness.

## Feature 362 hosted release gates and PR

Date: 2026-09-18

- [Quality Gate 35415527189](https://github.com/briandenicola/Aurearia/actions/runs/35415527189):
  Go API, Linux race detector, Python Agent, and Vue Web passed.
- [Feature 362 Compatibility 35415527170](https://github.com/briandenicola/Aurearia/actions/runs/35415527170):
  immutable mixed-binary guard and hosted Chromium acceptance passed.
- [Security Scan 35415527194](https://github.com/briandenicola/Aurearia/actions/runs/35415527194):
  all seven jobs passed, including both container scans, SPDX SBOM exports,
  provenance attestations, and agent runtime health.
- [CodeQL 35415527069](https://github.com/briandenicola/Aurearia/actions/runs/35415527069)
  passed.
- Release PR [#725](https://github.com/briandenicola/Aurearia/pull/725)
  targets `main` from `beta`, follows the repository template, cites ADR 0017,
  Principles II/III/IV/V/VIII/IX and Constitution §§17/21/22, enumerates the
  affected workflows and rollback/security evidence, and remains unmerged.

## Post-Major-Work QC Audit — Feature 362 / F014

**Audited by:** GitHub Copilot CLI with focused code-review verification
**Date:** 2026-09-18
**Changeset:** `d8d17be15e314306dfb620237b1b06fb3619ced6..090fc1f2`
**Spec / design:** `specs/362-coin-copilot-attribution/spec.md`, plan/tasks,
ADR 0017

### Scope Summary

The audit covered 106 F014 files across Go persistence/services/handlers,
Python harness contracts, Vue chat/review surfaces, schema migration, OpenAPI,
Docker/CI, compatibility scripts, tests, and documentation. It traversed
owner/auth boundaries, cancellation/idempotency, manual-data preservation,
bounded projection, feature-disable behavior, rollback, mobile/PWA behavior,
and supply-chain/deployment evidence.

### Artifact Checklist

- [x] Exact F014 diff bounded and all changed domains inventoried
- [x] Production, test, migration, workflow, generated contract, and
      documentation changes reviewed
- [x] Constitution, spec, plan, ADR 0017, CI workflows, and PR DoD reviewed
- [x] All eight audit domains traversed
- [x] Hosted Quality Gate, race, compatibility/browser, Security Scan, and
      CodeQL rerun after implementation changes

### Blockers

| ID | Domain | File : Line | Finding | Disposition |
|----|--------|-------------|---------|-------------|
| B1 | Test fidelity / §17 | `src/web/e2e/workflows/coin-copilot-deep-analysis.spec.ts` | Initial T080 evidence used component/integration tests for cases the task required at browser level. | **Resolved:** added the 11-outcome target/lifecycle/eligibility/fallback browser matrix with explicit mutation interception, plus wishlist scalar/notes/reference decision coverage in `deep-analysis.spec.ts`. |
| B2 | Mobile/PWA / §17 | `src/web/e2e/workflows/coin-copilot-deep-analysis.spec.ts` | Initial T081 browser evidence did not background/restore both SSE cursors. | **Resolved:** added a 390 by 844 Chromium scenario that persists separate cursor values, dispatches hidden/visible lifecycle events, restores both surfaces, and verifies independent `?since=4` and `?since=42` requests. |

### Follow-Ups

No non-blocking finding remains.

### Positive Observations

- Go retains fixed callback authority, owner scoping, durable idempotency,
  cancellation linearization, and transactional admission/apply.
- Collection, wishlist, and draft destinations retain closed allowlists,
  current-state revalidation, additive references, keyed note append, and
  all-or-nothing writes.
- Python remains stateless and receives no generic network, database,
  filesystem, shell, provider, or write capability.
- Vue exposes no conversational proposal mutation and validates the exact
  same-job review URL before navigation.
- The immutable guard matrix now proves cancel, fail-closed unknown-source
  read/apply, byte preservation, and re-upgrade review/apply restoration.
- All workflow actions remain SHA pinned; runtime images remain non-root and
  pass Trivy, SBOM, provenance, and health gates.

### Confidence Notes

Generated OpenAPI was validated by route-contract tests rather than manually
reviewing every generated line. Final local affected gates passed: nine
Playwright workflows, ESLint for both browser files, the expanded rollback
matrix, and the integration package. Post-fix hosted runs all passed:
[Quality Gate 35416506039](https://github.com/briandenicola/Aurearia/actions/runs/35416506039),
[Feature 362 Compatibility 35416506141](https://github.com/briandenicola/Aurearia/actions/runs/35416506141),
[Security Scan 35416506117](https://github.com/briandenicola/Aurearia/actions/runs/35416506117),
and [CodeQL 35416505650](https://github.com/briandenicola/Aurearia/actions/runs/35416505650).
The PR-triggered Gitleaks job additionally exposed an older F013 privacy-policy
prose false positive; `.gitleaks.toml` now has a narrow path-and-text allowlist,
and local Gitleaks 8.24.3 reports no leaks across the full 69-commit PR range.

**Final disposition:** PASS. Both audit blockers are resolved; no production,
security, contract, data-loss, rollback, or quality-gate blocker remains.
