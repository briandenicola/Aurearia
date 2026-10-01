# Dependency consolidation handoff

## Outcome

Consolidated Dependabot PRs #792-#799 onto `beta` in commit `3768657e`.
The release diff also contains the existing test-only Coin Copilot stabilization
commit `f1a8952c`.

## Changed paths

- `src/agent/uv.lock`
- `src/api/integration/coin_copilot_seam_test.go`
- `src/web/package.json`
- `src/web/package-lock.json`
- `src/web/e2e/exploration/__tests__/dependency-integrity.test.ts`

## Evidence

- `task check:web` passed.
- `task check:agent` passed with 774 tests.
- `task check:go` passed.
- `task test-race` passed.
- `task check:delivery` passed with zero governance errors.
- `task check:openapi` passed with no drift.
- npm audit and pip-audit found no known vulnerabilities.
- The dependency guard's Ruff assertion was tamper-tested and failed when
  deliberately set to the prior version.
- QC audit: `docs/audits/2026-10-01-dependency-consolidation.md` - PASS for
  opening a beta-to-main pull request.

## Boundary and next action

Push `beta`, open the template-compliant beta-to-main pull request, wait for its
exact-candidate hosted checks, and close #792-#799 as superseded by that pull
request. Merge, release, publication, and deployment require separate owner
approval.
