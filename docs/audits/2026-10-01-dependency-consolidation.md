# Post-Major-Work QC Audit - Dependabot Consolidation

**Audited by:** GitHub Copilot CLI  
**Date:** 2026-10-01  
**Changeset:** `origin/main...3768657e`, tree `0032a0f8bcd60d4bb25352ad3baf864a1d08ad55`  
**Authority:** Owner request to consolidate Dependabot PRs #792-#799 onto `beta`;
Constitution 4.0.0 Principles IV and IX, sections 17, 20, and 21; ADR 0019  
**Prior decisions:** `.squad/decisions.md`

## Scope Summary

The release candidate contains the exact combined dependency updates from
Dependabot PRs #792-#799, the dependency-integrity assertions required by those
new locks, and beta's existing test-only Coin Copilot race stabilization commit
`f1a8952c`. The audit covered the web and Python supply chain, lock
determinism, the Go test seam, all configured local completion gates, and the
release boundary. No production API, UI, schema, auth, configuration, workflow,
container, or runtime behavior changed.

## Artifact Checklist

- [x] Diff read (all five changed paths across two commits)
- [x] Owner authorization and applicable active decisions read
- [x] No new ADR required; no material architecture choice introduced
- [x] CI workflows and pull-request Definition of Done read
- [x] Constitution hierarchy, section 17, and section 21 reviewed
- [x] All eight audit domains traversed; UI/API/migration domains are N/A because
      no production source or contract changed

## Blockers

| ID | Domain | File : Line | Finding | Required Remediation |
|----|--------|-------------|---------|----------------------|
| B1 | Release governance | `.squad/decisions.md` (R-RELEASE) | The owner-appointed read-only successor returned INCOMPLETE for candidate `04c3b123` because organization content exclusion prevented it from reading the constitution and required `.squad` evidence. It explicitly left R-RELEASE blocking for opening the beta-to-main pull request. | Provide an owner-approved independent review mechanism that can access the required governance evidence without bypassing content exclusion, then obtain an exact-candidate PASS/clearance. |

## Follow-Ups

None.

## Positive Observations

- `src/web/package.json:32-53` records only the five requested web upgrades, and
  `src/web/package-lock.json` is the exact union of the corresponding Dependabot
  lock deltas rather than a broad resolver refresh.
- `src/agent/uv.lock:800-913,1785-1786` contains only the requested
  `langchain-anthropic`, `langgraph`, and Ruff upgrades plus the required
  `langchain-core` transitive update.
- `src/web/e2e/exploration/__tests__/dependency-integrity.test.ts:78-81` keeps
  the approved Python model/tool lock resolutions explicit. A deliberate Ruff
  downgrade tamper failed with `Received: "0.16.9"` and
  `Expected: "0.16.8"` before the assertion was restored.
- `src/api/integration/coin_copilot_seam_test.go:617,726-738` changes only test
  synchronization: it waits for worker token revocation before backdating the
  heartbeat, preventing a late usage write from invalidating the stale-run
  simulation.

## Validation Evidence

All commands passed on application candidate `3768657e`:

- `task check:web` - zero-warning lint, strict type check, 1,764 tests
  (1 skipped), and production build
- `task check:agent` - locked/offline environment check, Ruff, and 774 tests
- `task check:go` - build, vet, formatting, and full Go package tests
- `task test-race` - all Go race-detector packages
- `task check:delivery` - 91 Node delivery tests, SpecKit negative controls, and
  governance check with zero errors
- `task check:openapi` - regenerated snapshots with no tracked drift
- `npm audit --audit-level=high` - zero vulnerabilities
- `UV_PYTHON_DOWNLOADS=never uv run --no-sync pip-audit` - no known
  vulnerabilities; the local project package is correctly not auditable on PyPI
- `git diff --check` - clean

The local Node runtime is 22.12.0, below the repository minimum 22.22.2, although
the full web gate passed. The pull request's Node 24.15.0 job is therefore the
authoritative supported-runtime result. Hosted CodeQL, Gitleaks, container
scans, and release guards also remain pending until the pull request runs.

## Independent Review

The owner approved one bounded `task review:read-only` lease against candidate
`04c3b123`. The reviewer returned **INCOMPLETE**: organization content exclusion
blocked its required constitution and `.squad` reads, so it could not verify the
canonical completion record, R-RELEASE clearance authority, exact candidate
diff, or the applicability delta from application commit `3768657e`. It did not
clear R-RELEASE.

## Verdict and Confidence

**INCOMPLETE; merge and release remain blocked.** The
application candidate has no observed software blocker and every locally
available completion/release check passed, but the binding independent release
review is incomplete. The owner subsequently directed that PR #800 be opened
for hosted checks and review; that direction does not clear R-RELEASE. Do not
merge, publish, deploy, release, or close #792-#799 as superseded until an
authorized independent reviewer returns exact-candidate clearance.
