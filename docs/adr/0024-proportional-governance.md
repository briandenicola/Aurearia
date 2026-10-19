# ADR 0024: Proportional Governance

Date: 2026-10-19
Status: Accepted
Accepted: 2026-10-19, owner-approved PR #828 into beta.
Acceptance merge: `92259b31beb3c7ebe5db54bfcccb3331b98e57db`.
Lifecycle: acceptance metadata only; the merged decision body below is preserved.

## Context

The owner explicitly requested local adoption preparation of six principles:
durable concise policy, proportional lanes, bounded end-to-end work orders,
nonduplicated verification, a six-question PR and coordinated active consumers.
This work starts on clean `beta` at
`283d0c2ebac0f066f23650c5445fefb62c401ab3`. Constitution 4.0.0 and Accepted
ADR 0019 govern preparation. PR #827 is open with exact-head CI reported green;
its pending software audit and historical blocks are not waived.

Actual conflicts are mandatory small-change review, blanket high-risk triggers,
full repeated local completion, multi-ledger handoff/decision administration,
the universal eighteen-item PR checklist, and broad high-risk/release audits.
The owner initially authorized locked local authoring, then explicitly approved
focused governance checks, one ten-minute independent review budget including
repair follow-ups, and commit/publication on a dedicated amendment branch with a
PR into beta. Local application builds, installations, merge and deployment
remain unauthorized.

## Decision

Prepare constitution **5.0.0**, a MAJOR bump under current §22 because operational
authority, review, recordkeeping and audit requirements change incompatibly.
Preserve identifiers I–IX, substantive engineering/privacy/data safeguards,
historical records and original reviewer clearance rights.

- One issue/PR contains approved scope, criteria, evidence, review and outstanding
  work. Before one exists, the approved bounded work order is the source.
  Specs are optional planning aids; the pointer only links authoritative work.
- Routine work needs appropriate regression and CI, not mandatory independent
  review. Features need one scoped independent review; high-risk boundary changes
  add applicable security/compatibility/recovery proof. Risk overrides size, but
  touching existing contracts/queries does not alone create a new boundary.
- Explicit bounded work orders can authorize named implementation/test/review/
  repair and beta publication actions once. No implicit machine or release
  permission; conditional merge authority must identify an eligible exact
  candidate and require all gates. Deployment remains separate. Use one overall
  time/credit budget with checkpoints and stop conditions.
- Reuse unchanged valid evidence, focused local checks and applicable
  exact-candidate CI; review repair deltas unless broader evidence is invalidated.
  Preserve validate-once/identical-tree provenance and immutable promotion
  controls without promoting Proposed ADR 0022 or claiming live settings changed.
- Broad audits apply to major architecture or owner-designated major releases,
  not every beta-to-main PR. Preserve existing required audits/blocks.
- Six PR questions retain candidate/reviewer identity, exceptions, owner
  authority and acceptance/release distinctions without a universal checklist.

On acceptance this supersedes only conflicting operational requirements in ADRs
0001 (universal ADR/document-set triggers), 0019 (review/ledger/audit/completion
ceremonies), and active consumer instructions. Accepted bodies stay immutable;
their architecture, evidence preservation, independence and safety constraints
remain binding. No historical approval or audit is reclassified.

## Synchronization and evidence

The constitution Sync Impact Report and local diff identify actual changed
consumers: universal/scoped delivery instructions; active agents/prompts;
SpecKit templates; optional Squad routing/ceremonies and role charters;
contributor/testing/acceptance guidance; PR template; current-work pointer;
ADR index; governance checker and negative fixtures.
The active decisions view has a pending-only notice, not an acceptance
announcement; existing decisions and historical blocks are preserved.
The canonical `.github/skills/aurearia-software-qc-audit/SKILL.md` trigger is
aligned without changing the personal post-major-delivery skill or installing it.
Unused recipes, role histories, historical logs and accepted ADR bodies are not
cleanup scope. Taskfile/CI, reviewer wrapper and candidate/publish controls remain
unchanged: they already enforce the retained quality/capability/provenance rules.
The pointer checker is structural only, not proof of approval; it must accept
repository issue/PR canonical sources without inventing a second tasks ledger.
Its simplified pointer mode remains dormant unless the constitution records
both version 5.0.0 and Amendment Status Accepted. Negative fixtures cover
premature activation, invalid repository URLs/paths and malformed optional
legacy aliases. The prepared pointer retains the existing tasks alias for
4.0.0 compatibility; it is a link to the same source, not a second ledger.

Owner-authorized `task check:delivery` **passed**: all 134 Node tests,
PowerShell SpecKit selection/stream regression and negative control, and offline
governance (zero errors; four visible Proposed-ADR warnings). This includes the
native reviewer restrictions and retained candidate/publication negative tests.
Removing the explicit Accepted-status requirement made the premature-activation
regression fail at its expected assertion; the guard was restored.
Independent pre-push review passed; CI and fresh-client discovery remain pending.

Initial independent review of tree `871fee43f24c1484de8702efe5e41309a62c6a21`
identified one documentation blocker: the detailed pointer/coverage reference in
`docs/testing.md` still described only 4.0.0 rules. Author corrected both modes
and version-qualified the section references; executable behavior is unchanged.
Original reviewer session `687658a8-e570-45de-b31b-a55fc9103de0` must clear the
delta within the remaining approved review budget before publication.

The same reviewer explicitly cleared B1 and returned PASS on revised staged tree
`ac6c3968fb116f3d15c80151b6ebaedd8825db2d`, parent `283d0c2e`, using approximately
five and a half of the ten approved active review minutes. No blockers remain
within the scoped review. Subsequent changes are factual clearance receipts only;
reviewed operational policy, checker and fixtures are unchanged. This is not
acceptance, activation or merge authority.

Existing verification commands from repository root:

```powershell
node --test scripts/delivery/governance.test.mjs scripts/delivery/native-integration.test.mjs
node scripts/delivery/check-governance.mjs
task check:delivery
git --no-pager diff --check
```

The aggregate includes the first two, existing delivery/candidate/publish/SpecKit
negative fixtures and governance; it was run once without application builds.
Fresh-client instruction discovery is separate runtime evidence, not established
by static fixtures. Supply exact dirty-tree diff/manifest to the read-only
reviewer; never substitute this implementation owner's inspection for review.

## Acceptance route and rollback

The owner authorized isolation on `chore/proportional-governance` from beta,
focused checks, one ten-minute review budget including repair follow-ups,
commit/push and an amendment PR into beta. This leaves #827 unchanged.
Do not infer permission to merge either PR or perform a release/deployment.
The PR must link this ADR and contain the coordinated amendment. Only required
review, explicit owner approval and authorized merge activate 5.0.0; then record
merge evidence, announce in `.squad/decisions.md`, transition ADR/index and update
last-amended metadata. Set constitution Amendment Status to Accepted and remove
its prepared-version qualifier only in that authorized lifecycle action, together
with the prospective consumer banners. Ratification remains 2026-04-28.

Before activation, abandon only this scoped diff if authorized; 4.0.0 remains
effective. After activation, rollback is an explicitly approved coordinated
revert/amendment of this candidate's consumers/checks to the recorded baseline,
not a rewrite of historical evidence or Accepted ADR bodies. Preserve any review
blocks and release restrictions throughout.

## Consequences

Checkpoint: authored on the authorized isolated branch; delivery controls and
independent pre-push review passed, including the deliberate activation-guard
negative proof. Publication and exact-head CI remain pending. No application
build, setup, live setting, merge or deployment was performed.

Less duplicate administration and repeated proof; stronger need for concise,
candidate-bound canonical evidence. No safety gate, reviewer restriction,
live setting or release permission is removed implicitly. Until accepted,
prospective text is preparation, not operating authority.
