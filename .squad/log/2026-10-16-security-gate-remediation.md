# Share-card security-gate remediation

**Date:** 2026-10-16
**Owner:** Copilot CLI implementation owner
**Branch/baseline:** beta, `4e3ae234804eac93512fde07b2d7a0f33e0d846d`
**Authority:** Owner explicitly authorized fixing security-gate blockers, beta
push, then a PR only after all gates are green. Separate local setup approval
covered npm locked restore and a session-local, checksum-verified Go 1.26.9 SDK.
**Lane:** Narrow dependency/toolchain security maintenance; Principles IV-V,
sections 17-21. No application contract, workflow policy, migration or deployment
changes. No Python installation, uv upgrade or container execution authorized.

The canonical [audit addendum](../../docs/audits/2026-10-16.md) records the
initial failures, exact dependency blobs, patched versions, local evidence and
environment limitations. The original share implementation remains unchanged.
Unrelated Python wheel metadata was preserved rather than broadening the lock
refresh. Independent review was declined and is not claimed.

Patched local web, Go package tests, reachable Go vulnerability scan, npm audit,
desktop/mobile downloads and delivery validation passed. The Windows CRLF
formatting interruption is recorded without changing source or the checker.
Supported Node/Python versions, canonical Go formatting and container/race
results still require exact-head CI; old-candidate results are not reused.

## Next action

Stage only the authorized patch and receipts, commit and push beta.
Require every exact-head security, quality, compatibility,
exploration and publication result before opening the template-based PR.
Missing/deleted checks are not passing evidence. Bind final CI and candidate
commit/tree in that PR; do not merge, release or deploy.
