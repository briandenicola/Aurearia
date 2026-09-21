# P5 Acceptance Controls - Repository Draft Handoff

Date: 2026-09-21
Authority: Constitution Principles IV/VII/IX, sections 17-18/21; ADR 0019
Work: [D14/D15 plan](../../docs/agentic-delivery-improvement-plan.md)
Branch: `docs/delivery-acceptance-controls`
Base: `7706bb633146fa733e0ed2494d7f2f871416e999`
Reviewed/tested source tree: `8e4187d920164f6cfc3bce72f7560bc82938b1a2`

This is a preparation receipt, not a competing current-status ledger. After
publication, the P5 PR's Completion evidence section holds the final commit/tree,
hosted results and owner decisions. The plan records criteria and this checkpoint.

## Delta

D14 reuses the PR/approved-work record for criterion/workflow/result/evidence/
candidate identity, independent verdict, block disposition and actual owner
decision. Handoff/checkpoint/reviewer/coordinator guidance links it rather than
duplicating raw logs. Stale reviews and missing evidence cannot imply acceptance.

D15 has a [before/after and restoration proposal](../../docs/agentic-acceptance-controls.md),
main-only protected approval prerequisite, read-only evidence validator and
per-publisher retry checks. Main publishing is serialized; both image checkouts
retain the checked SHA. Beta workflow bytes and live settings are unchanged.
The owner approved twenty main PR requirements but nineteen publishing checks:
the CodeQL aggregate exists on PR heads, not pushed merge commits. All four
CodeQL analyses remain mandatory on the published candidate.

P4's current metadata now records owner-merged #740 and green PR/post-merge
evidence. Its original logs remain unchanged; no application block was cleared.

## Verification and independent review

`task check:delivery` passed on the source tree above: 83 Node tests, PowerShell
selection/stream regression and negative control, zero governance errors.
Fixtures exercise candidate/owner/check/API failure paths, malformed evidence,
pagination bounds, source-guard bypass and removal of either publisher's
dependency/recheck. `git diff --cached --check` passed.

Native reviewer `4f05f96c-2545-45b0-978c-0c046934c60e` returned PASS with no
findings on the same tree. It used 11 of 16 authorized reads, 105 supervised
seconds out of ten minutes; runtime schemas remained read-only. Matching
instruction bodies were explicitly loaded, not assumed automatically injected.
The reviewer distinguished supplied Git/API/test evidence from direct source
inspection. No second agent, shell, SQL, skill, write or network tool was used.

Five synthetic cases returned the expected outcomes: stale proof -> implemented;
green tests with an open block -> verified but not accepted; agent checkbox
without owner decision -> verified; real beta-only acceptance -> accepted into
beta, not release-ready; missing required runner/browser evidence -> implemented.
These synthetic decisions are not real owner authorization.

Raw source evidence is retained in the initiating session's `files` directory:

| Artifact | SHA-256 |
|---|---|
| `p5-reviewed-delivery.log` | `a6a8e8fb70a54311ec04423b33f864845b0abd9f98532d7901e06dceca34dae5` |
| `p5-review-verdict.txt` | `d739351bbabec72248366f98719841820ffc19fe0dffc34f09bfde34262ef667` |

Do not publish raw native-client debug logs containing personal/global context.
The PR provides the scoped verdict/evidence and hosted run links.

## Applicability and remaining work

The reviewer expressly allowed factual-only result/handoff updates with a
scoped delta and final delivery gate. This receipt and current metadata do not
change reviewed executable behavior. Record the final tree and rerun result
in the PR; any executable delta requires applicable re-review.

Existing-category warnings: sparse historical links and 7,372 initial-context
words. Preserve the P4 maintenance exception for owner review, not silent
truncation. Local Node is 24.14.0 versus hosted 24.15.0; hosted results remain
pending at this receipt. No application build, installation or container run
was used as a substitute for relevant delivery behavior.

Owner acceptance, live main protection/environment changes, readback, default-
branch activation and the first real approval/API path remain pending.
D15 is not fully activated. The repository draft does not authorize main merge,
publication, local-server deployment, P6 or clearance of historical application
reviews. Keep Ralph disabled and preserve the original dirty beta worktree.
