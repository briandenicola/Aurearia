# Collector-ticket share cards

**Date:** 2026-10-16
**Owner:** Copilot CLI implementation owner
**Branch:** beta
**Baseline:** `616557a27b7c5137f46c3a93cdef08b34a796a13`
**Authority:** Owner-approved Option A mockup and explicit instruction to implement,
push to beta, then open a PR only after all gates are green.
**Lane:** Small frontend presentation enhancement; Principles III-V and sections
17-21. No API, migration, auth, AI/provider, dependency or deployment changes.

## Scope and acceptance

- One PNG: obverse/reverse together above a cream, serif collector's ticket.
- Attribution comes from stored ruler/name, dates, mint, issue, dimensions,
  inscriptions/descriptions and references; no inferred identification.
- Missing faces remain correctly labelled, including reverse-only coins.
- Long text wraps and expands the ticket rather than losing attribution.
- Coin detail and Coin of the Day retain their existing shared entry points,
  native file sharing, optional context and PNG-download fallback.
- Private notes, prices, purchase/owner/storage information and AI analysis
  remain excluded. No Harlan Berk branding or copied proprietary artwork.

## Evidence and review disposition

The canonical bounded audit and exact application-blob manifest are in
[`docs/audits/2026-10-16.md`](../../docs/audits/2026-10-16.md).
Targeted tests, `task check:web` and both desktop/mobile browser cases passed.
The image-containment regression was tamper-proven before final validation.

The owner authorized local checks but explicitly declined an independent
reviewer. Independent acceptance is therefore not claimed. The approved
mockup is design approval, not approval of a merge or deployment.

## Next action

Commit only the share-card implementation, tests, README and these evidence
artifacts. Push beta, wait for exact-head Quality Gate, Security Scan,
compatibility and CodeQL results, and open the template-based beta-to-main PR
only when every required check is green. Record exact CI/commit/tree evidence
in that PR's canonical completion record. Do not merge or deploy.
