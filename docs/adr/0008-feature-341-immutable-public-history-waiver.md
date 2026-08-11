# ADR 0008: Feature 341 Immutable Public-History Waiver

Date: 2026-08-11
Status: Accepted
Approved: 2026-08-11 — Brian explicitly selected “Approve ADR 0008 (Recommended)”

## Context

Feature 341's technical and release-evidence gates pass, but four commits are
already published on the public `beta` branch with commit-hygiene deviations.
Rewriting that shared history would require an amend, rebase, reset, or
force-push and would invalidate existing commit and image references.

Constitution §0 places the Constitution above the task list, project decisions
ledger, and agent judgment. Brian's earlier inbox directive selected preserving
public history, but it cannot itself waive Principle VII, §17, or §21 item 17.
The Constitution header requires deviations to have an explicit ADR-backed
waiver under §22. This ADR supplies that auditable proposal without changing
the Constitution or relaxing its rules prospectively.

The commit messages and trailers were verified with `git show`,
`%(trailers:...)`, and `git interpret-trailers --parse`.

| Commit | Exact subject | Trailer state | Exact deviation proposed for waiver |
| --- | --- | --- | --- |
| `31cb6033875bcb6da0db82e9fc59a1278a56b0f6` | `scribe: orchestration logs and decisions for Numista 341 planning (2026-08-11T13:06Z–13:07Z)` | Required Copilot co-author is present and parseable | `scribe:` is not an allowed Principle VII prefix |
| `a8f59b3bf7e2479e1083ee21f0737369c89c3a91` | `merge: integrate Numista lookup improvements (Feature 341)` | Required Copilot co-author is present and parseable | `merge:` is not an allowed Principle VII prefix |
| `8e77500f05dde63ed7335fa12ba14614fe6e2ba2` | `merge: reconcile remote beta (14 commits)` | No trailer is present; no required Copilot-trailer waiver is asserted for this reconciliation merge | `merge:` is not an allowed Principle VII prefix |
| `460dbfcd0ba4bd36d39d150945d9c39546551be3` | `docs: Scribe orchestration/session logs and merged decisions from 2026-08-11` | The message contains `- Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>` as a list item; Git parses no trailer | The required AI co-author trailer is not parseable |

`Copilot-Session` metadata is outside the Constitution and is not part of this
waiver.

## Decision

Brian explicitly approved this ADR on 2026-08-11 by selecting
“Approve ADR 0008 (Recommended).” Feature 341 therefore receives a one-time
waiver for only the four deviations enumerated above:

1. Preserve the already-published `beta` history exactly as it is. Do not
   amend, rebase, reset, rewrite, or force-push these commits.
2. Treat the enumerated deviations as disclosed exceptions when evaluating
   Feature 341 against Principle VII, §17, and §21 item 17.
3. Apply no prospective relaxation. Every later AI-assisted commit must use
   one of `feat:`, `fix:`, `docs:`, `refactor:`, or `chore:` and must contain
   the parseable required Copilot co-author trailer.
4. Enforce the prospective rules during PR and release review. A future
   violation requires correction before publication; this ADR cannot be cited
   to excuse it.
5. Disclose this ADR, its status, all four SHAs, and the exception matrix in
   the Feature 341 PR. The PR must cite Constitution §0, Principle VII, §17,
   §21, and §22 and distinguish historical exceptions from compliant new
   release-evidence commits.

This approval establishes the §22 waiver authority needed to evaluate the
enumerated immutable exceptions. It does not itself clear Maximus's reviewer
block; final clearance still requires Maximus to explicitly re-review and
clear that block.

## Alternatives Considered

- **Rewrite public `beta` history:** rejected because it changes published
  commit identities, breaks audit references, and requires prohibited history
  manipulation.
- **Rely on the inbox directive or PR disclosure alone:** rejected under §0;
  lower-authority evidence cannot override constitutional requirements.
- **Amend Principle VII for all commits or exempt merge commits generally:**
  rejected as disproportionate and prospective. The release-integrity rule
  remains useful and unchanged.
- **Leave Feature 341 permanently blocked:** available if the ADR is not
  approved, but rejected as the proposed outcome because all technical gates
  pass and the deviations can be completely bounded and audited.

## Consequences

### Positive

- Public commit and image references remain stable.
- The release record states the deviations exactly rather than claiming full
  historical compliance.
- §22 provides a durable, reviewable authority trail above the inbox decision.
- Future commit hygiene remains fully enforced.

### Negative and risks

- The four immutable commits remain mechanically noncompliant in history.
- Reviewers must consult this ADR when evaluating Feature 341's historical
  release evidence.
- Approval creates a risk of being cited as precedent; the scope and expiry
  controls below explicitly prohibit that use.

## Audit and Mitigations

- The immutable matrix records full SHAs, exact subjects, and parsed trailer
  state.
- Feature 341 quickstart and T086 evidence link this accepted ADR and preserve
  the distinction between completed waiver evidence and pending final reviewer
  clearance.
- The future PR must reproduce or link the matrix and state that no history
  rewrite occurred.
- PR/release review must verify all commits created after these four use an
  allowed prefix and, when AI-assisted, the parseable required trailer.
- This waiver changes no product behavior, tests, branch protection, or
  release tooling.

## Expiry and Non-Precedent

This waiver is exhausted by the single Feature 341 `beta`-to-`main` release
review and applies only to the exact SHAs above. It does not apply to amended
commit objects, replacement SHAs, other branches, other features, or any
future commit. It is not precedent for accepting another published-history
exception; any later case requires its own §22 review and ADR.

## Rollback

There is no rollback for immutable published history. Before approval, the
fallback would have been to leave T086 and the release blocked without
changing history. Now that the ADR is accepted, withdrawing the waiver can
stop or revert the release through a new governance decision, but it cannot
alter the four historical commit objects.

## Related

- [Constitution §0, Principle VII, §17, §21, and §22](../../.specify/memory/constitution.md)
- [Feature 341 quickstart evidence](../../specs/341-improve-numista-lookup/quickstart.md#release-record-corrections-and-immutable-exceptions)
- [Feature 341 tasks](../../specs/341-improve-numista-lookup/tasks.md)
- [ADR 0001: Record Architecture Decisions](0001-record-architecture-decisions.md)
- `.squad/decisions/inbox/copilot-directive-20260811T164500-0500.md`
- `.squad/decisions/inbox/cincinnatus-adr-0008-accepted.md`
- `.squad/decisions/inbox/maximus-feature-341-final-clearance-block.md`
