# Active Squad Decisions

Curated 2026-09-29 for issue #783 under Constitution 4.0.0 section 18.3. This is a
current view, not a constitution or task ledger. Start with
[current work](identity/now.md). Authority: Constitution > PRD > accepted ADRs >
active spec > plan > tasks > backlog > active decisions > agent judgment.

The full prior text, including the D05 lifecycle reconciliation, unresolved
review blocks and located clearance pairs, is preserved byte-for-byte in the
[2026-09-29 snapshot](decisions-archive.md#curation-2026-09-29-decisions); older
material is in the [2026-09-21 snapshot](decisions-archive.md#curation-2026-09-21-decisions)
with inventories for [2026-09-29](artifacts/context-curation-2026-09-29.json) and [2026-09-21](artifacts/context-curation-2026-09-21.json). Archiving never
revokes a decision or clears a block.

## Current decisions

| ID | Status | Decision |
|---|---|---|
| GOV-001 | Accepted | Constitution 4.0.0 and [ADR 0019](../docs/adr/0019-evidence-based-agentic-delivery.md) are in effect (#734, #735). |
| GOV-002 | Active | One implementation owner, smallest sufficient lane, bounded delegation, independent review where required. Evidence distinguishes implemented, verified, accepted and released. Sections 17-22 govern NEW reviews; historical restrictions keep their terms. |
| GOV-003 | Active | Warning budgets: decisions <=20 KiB, role history <=12 KiB, now.md <=100 lines, startup context <=6000 words. Budgets never authorize deletion; preserve before curating. Current work lives only in now.md. |
| GOV-004 | Accepted (#737) | Taskfile `check:*` targets are shared with CI; setup is separate; missing tools or lint fail. Offline checks validate structural references, not approvals. Small work may cite an issue instead of a spec. See [testing](../docs/testing.md#6-running-tests-locally-vs-ci). |
| GOV-005 | Verified (#739, #740) | Scoped instructions and native skills replace eager guidance. Required review uses `task review:read-only`; installed-client limitations require explicit tool exclusions and matching-file loads. SpecKit upgrade is deferred, not installed. #739's merge does not retroactively certify its failing gate. See [P4 record](decisions/inbox/copilot-p4-native-integration.md). |
| GOV-006 | Merged (#741); runtime approval proof pending | Acceptance uses one criterion/evidence/candidate record. Main protections follow the [control guide](../docs/agentic-acceptance-controls.md) and [P5 record](decisions/inbox/copilot-p5-acceptance-controls.md); owner kept environment admin bypass while retaining actual owner approval and all exact-candidate checks ([receipt](../docs/audits/2026-09-21.md)). Beta publishing and Ralph are unchanged. Stale proof, open review blocks or missing owner approval cannot support acceptance. No promotion or deployment is inferred. |
| GOV-007 | Owner exception (2026-10-01) | R-RELEASE is excepted only for beta receipt candidate `105240c3` and its beta-to-main PR. The missing Feature 362/363 combined audit remains unperformed; the exception is not audit completion, merge permission, deployment approval, or a waiver for later releases. See [exception record](decisions/inbox/release-exception-2026-10-01.md). |
| ENG-001 | Active | Go owns auth, durable state, data access and typed contracts; Python stays stateless (Principles I-III). A framework or role memory is not permission to cross a service/data boundary. |
| ENG-002 | Active | Prove the exact changed workflow, sibling paths and negative cases per section 17. Do not substitute local Windows evidence for a required Linux/browser gate. Checked tasks and old transcripts are not current verification. |
| UI-001 | Active | Reuse local components, tokens and navigation. Narrow tables scroll horizontally; swipe consumers share one primitive ([tables][tables], [swipe][swipe]). |
| SEC-001 | Accepted | Eval-requiring background-removal code stays in its same-origin worker; do not relax app-wide CSP; the earlier unsafe-eval suggestion is not authority ([ADR 0014](../docs/adr/0014-background-removal-worker-csp-isolation.md), [clearance][worker-clear]). |
| AI-001 | Accepted | Coin Copilot: Go owns run state; Python runs bounded allowed tools. The MVP excluded writes and dollar-cost enforcement; new capabilities need their own spec. Keep legacy fallback, the existing drawer entry, confirmation boundaries and iteration/tool/time/concurrency/token/payload limits ([scope][copilot], [ADR 0016](../docs/adr/0016-go-owned-durable-coin-copilot-state.md)). |
| AI-002 | Accepted | Deep Analysis reuses separate collection-grade face analysis and optional notes before provider verification, keeping evidence, disagreements, confidence and narrative; Quick Lookup stays the fast combined-image NGC-first path. Supersedes only ADR 0012's single-vision-call constraint ([ADR 0018](../docs/adr/0018-role-specific-deep-analysis.md)). |
| AI-003 | Owner-approved | Reduced F015 ([Feature 363](../specs/363-collector-curator-watchlist-provenance/spec.md)): private collector context, read-only curator guidance, existing UI-owned Add to Wishlist, native listing-URL intake. No auction rewrite, wishlist-action platform, action/audit tables, watchlist ranking or provenance-risk workflow. The archived T011 pause is superseded; do not restart completed work. |

## Open review blocks

Unresolved records (details and located clearance pairs in the 2026-09-29
snapshot): R352, R353, R-SWIPE, R225, R320, R337. R-RELEASE has a scoped owner
exception for candidate `105240c3`; its underlying Feature 362/363 audit remains
unperformed. Only the original
reviewer, or an owner-appointed independent successor after re-review, can clear
one, unless that reviewer set an automatic evidence condition (for example
R-SWIPE B1). Assignment or reassignment is not clearance. Historical author and
reviser restrictions keep their terms. Check the snapshot before touching those
areas; do not reopen a located clearance from an old REJECT alone.

[tables]: https://github.com/briandenicola/Aurearia/blob/e6ab8313346b971dce4b288804036222f7d4c95d/.squad/decisions.md#L72-L119
[copilot]: https://github.com/briandenicola/Aurearia/blob/e6ab8313346b971dce4b288804036222f7d4c95d/.squad/decisions.md#L11289-L11343
[worker-clear]: https://github.com/briandenicola/Aurearia/blob/e6ab8313346b971dce4b288804036222f7d4c95d/.squad/decisions.md#L11463-L11569
[swipe]: https://github.com/briandenicola/Aurearia/blob/e6ab8313346b971dce4b288804036222f7d4c95d/.squad/decisions.md#L10912-L11035
