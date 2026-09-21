# ADR 0019: Evidence-Based Agentic Delivery

Date: 2026-09-21
Status: Accepted
Accepted: 2026-09-21 via [PR #734](https://github.com/briandenicola/Aurearia/pull/734)
Merge commit: `ca85f7830349a4472cb5409c082b9485e7e34c44`

Lifecycle note: The body below is preserved as the approved proposal.
Its preparation-time pending statements are historical; this header records the
owner-approved acceptance. Deferred work and historical review blocks remain open.

## Context

The delivery audit identified conflicting authority, outdated active instructions,
oversized decision/history files, and completion claims that are not reliably
bound to evidence. The owner requested implementation of
`docs/agentic-delivery-improvement-plan.md`.

This proposal now has owner authorization for P1 local preparation only, not
permission to apply later packages. The accepted constitution remains authoritative until the amendment is
accepted through section 22. No existing rejection is cleared by this proposal.

### Refreshed baseline

Baseline commit: `cacc177c2ecd85d62899507413184cfc395a7f6a` on `beta`.
At capture, the worktree had no tracked changes; the delivery plan and `logs/`
were untracked. The unrelated logs were not inspected or modified.

- Constitution version: 3.1.0.
- SpecKit executable: 1.0.8; checked-in integration: 0.5.1.dev0.
- Default `squad` executable: 0.13.0; separate `squad.cmd`: 0.9.1;
  checked-in coordinator: 0.9.1. No installation was changed.
- Copilot CLI: 1.0.87-0. No repository-native scoped instructions, skills,
  hooks, or Copilot settings were found in the corresponding `.github/` paths.
- Active decisions: 685,739 bytes / 12,210 lines. Four agent histories range
  from 149,183 to 212,196 bytes.
- Main requires Go API, Vue Web, Python Agent, Gitleaks, Govulncheck, npm audit,
  and pip-audit, with strict status checks. It has no required approving review
  and does not enforce protections for administrators.
- Beta has no required status contexts; force pushes and deletions are permitted
  by its current protection settings. This records the baseline, not approval
  to use those capabilities.
- Ralph heartbeat is manually disabled. The other checked-in workflows are
  active. No open PRs or repository rulesets were returned at capture.

Selected tool/settings observations and pre-change file hashes are retained in
the initiating Copilot session's `files/delivery-baseline.json`. They contain no
credentials and are diagnostic evidence, not another policy surface.

### Review blocks carried forward

The following records were found without a matching explicit clearance in the
reviewed material. They are not assertions that the historical code defect is
still present; their unresolved review status must be reconciled separately.

| Artifact | Review owner | Evidence | Disposition |
|----------|--------------|----------|-------------|
| Feature 357 Quick Access frontend re-review | Brutus | `.squad/decisions.md:11801-11886` | Preserve REJECT and its explicit author restrictions; require Brutus clearance |
| Feature 357 final architecture audit | Maximus | `.squad/decisions.md:12036-12154` | Preserve REJECT; Brutus clearance precedes Maximus re-review |
| Other historical rejection/conditional-clearance records | Original named reviewer | Active and archived decision ledgers | Not globally adjudicated by P0; preserve restrictions until P2 maps each record to explicit clearance or an unresolved state |

Task checkboxes, later commits, merged releases, and an empty open-PR list are not
substitutes for reviewer clearance. No old block is waived or narrowed here.

## Decision

**Proposed constitutional version: 4.0.0.** The operational authority, archival,
review, and completion rules are materially restructured, requiring a major bump
under section 22. Preserve the nine core principles and engineering safeguards;
do not use this amendment to change application architecture or runtime behavior.

### 1. One project authority

The constitution remains the highest repository policy. An accepted ADR records
a binding design decision within its declared scope, subordinate to the
constitution and product requirements. A constitutional exception or amendment
requires section 22 approval; an ordinary ADR cannot silently override policy.

The proposed order is constitution, product requirements, applicable accepted
ADRs, active spec, active plan, active tasks, backlog, active decisions, and agent
judgment. Conflicts between peer ADRs require explicit supersession or owner
resolution, not a "newest file wins" assumption.

Framework instructions, charters, templates, skills, and session state implement
this policy; none may claim superior authority. Owner-approved product scope
changes must be reflected in the authorizing artifact before implementation.
They do not implicitly waive security, testing, or release controls.

### 2. Proportional work and explicit selection

Use a bug/small-change lane, a feature lane, and a high-risk lane as defined in
the implementation plan. Small repairs need not generate an entire feature
document set. Existing mandatory regression and contract coverage still applies.

Continue ordinary development on `beta`. Select work explicitly from the request
or a validated current-work pointer; never infer the active feature from the
highest-numbered spec. A bounded governance plan or issue may be the authorizing
artifact for process work that does not introduce an application feature.

If a new discovery exceeds approved scope, stop and request authorization.
Do not automatically stash, commit, or incorporate unrelated changes.

### 3. Current views are mutable; evidence is preserved

Authorize named maintainers or explicitly delegated agents to maintain current
views in `.squad/decisions.md`, `.squad/agents/*/history.md`, and
`.squad/identity/now.md`. These views may be curated, not silently rewritten to
alter an approval, decision, or unresolved restriction.

Before archival, preserve original material in append-only historical surfaces
with a source inventory and checksum evidence. Link current decisions to their
source and supersession history. Accepted ADR bodies and existing session and
orchestration logs remain immutable.

Use current decisions/history for concise actionable context, not transcripts.
Proposed warning budgets: automatic repository instructions 150 lines, active
decisions 20 KiB, each agent history 12 KiB, current-work pointer 100 lines, and
initial policy/context reading 6,000 words excluding selected feature artifacts.
An excess requires consolidation or a documented exception, never truncation of
binding constraints.

The current-work pointer identifies the source of task state and evidence; it
does not duplicate the complete task list or claim to be a higher authority.

### 4. Bounded ownership and independent review

Default to one implementation owner. Delegate only work that benefits from an
independent context, with an objective, write boundary, evidence requirement,
lease/checkpoint, and stop condition. No automatic speculative fan-out.

For new reviews after this amendment is accepted, a rejection normally permits
the original author to repair the work. The reviewer may explicitly require an
independent revision owner. The implementation author cannot approve their own
work or clear the reviewer's block.

Existing blocks and explicit author restrictions retain their original terms.
If a reviewer is unavailable, escalate to the owner for a documented assignment
of an independent successor reviewer; reassignment does not itself clear a block.

Scribe is an optional recording role, not a mandatory extra agent. A single
implementation owner may persist handoff evidence in the existing canonical
surfaces. Wait for required recording to finish before claiming completion.
Never infer success from changed files or a missing agent response.

### 5. Executable validation, without weaker gates

Make Taskfile recipes the shared executable entry points for local and CI
validation. Documentation references those recipes rather than maintaining
independent command lists. Until P3 implements them, use the existing verified
commands explicitly; do not cite a proposed command as available.

Distinguish fast targeted feedback, completion checks for affected layers and
contracts, and full CI/release checks. Frontend lint with zero warnings is an
explicit requirement. Preserve Go race, security, compatibility, and container
checks and identify which require the CI environment.

Pure documentation changes need document/link/consistency review and any existing
documentation checks, not application builds. Executable prompt, workflow,
validation, or policy changes are not automatically docs-only; require the
relevant fixtures and behavior checks as those tools are introduced.

Missing required tools or evidence means incomplete, not passed. Dependency
installation is a separate authorized setup operation. This policy does not
authorize local builds or other machine changes without required owner approval.

### 6. Evidence-backed acceptance and release

Distinguish implemented, verified, accepted, and released. Record criteria,
affected workflows, commands/results, exceptions, and reviewer disposition against
the tested commit or dirty-tree identity. Later implementation changes invalidate
applicable earlier verification and acceptance.

Green CI is necessary where required, but does not prove an untested user outcome.
A pushed beta commit is not automatically accepted or authorized for release.
Main/release actions require explicit owner authorization for the candidate.
Do not invent a second-human-review requirement for a solo-maintainer workflow.

Commit only approved paths; broad staging of `.squad/` or unrelated work is not a
handoff requirement. Publishing or settings changes remain separately authorized.

### 7. Native tooling and risk-based audit cadence

Use native scoped instructions and skills with one canonical copy. Keep Squad
optional and subordinate to policy. Do not hardcode model catalogs or silently
escalate model cost. Tool upgrades and hooks remain separate reviewed changes.

Run the software quality audit for major/high-risk changes and release readiness.
Run the separate agentic-delivery audit after each owner-designated major release,
including follow-through on prior actions. A skill file is not an automatic
trigger: wire explicit invocation/evidence into release closeout.

Replace mandatory weekly broad agent ceremonies with deterministic CI checks,
event-driven investigation, and the release audits above. Preserve per-release
SBOM/threat-model obligations and scheduled product/dependency/restore reviews.
An audit finding closes only when its corrective action has evidence.

## Activation and synchronization

Approval to prepare the amendment is not acceptance of the amendment.

On 2026-09-21 the owner selected: **"Approve P1 local edits on beta; include
amendment in a separately approved beta-to-main PR"**. This records preparation
authority for locked-file edits and the selected route. It does not authorize
builds, upgrades, live settings changes, commits, pushes, or PR creation.
The amendment remains **Proposed**.

The owner subsequently approved the draft and, on 2026-09-21, replaced the
publication route with: **"Proceed with independent review and governance-only
verification. Prepare a temporary branch and PR into beta containing only the
amendment. Do not merge, install tools, or deploy anything without my approval."**
This authorizes bounded review/checks and commit/push/PR preparation for this
amendment, not a merge, installation, deployment, or application release.
The governance-only PR targets `beta`; the previously selected beta-to-main
amendment route is superseded. Application-release approvals remain separate.

Do not publish, merge, or mark this ADR Accepted merely because local preparation
was approved. Follow section 22's version/header, Sync Impact Report, revision
history, PR, announcement, and acceptance requirements. P2 archival cannot
precede acceptance of the new archival authority.

Synchronize the following consumers in the amendment changeset:

- Constitution section 0, operational sections 17-23, and affected gate
  applicability wording; retain the substance of Principles I-IX.
- `.github/copilot-instructions.md`, active SpecKit agents/templates, and
  load-context/checkpoint/handoff/audit prompts.
- Squad coordinator authority/rejection/completion rules, routing, ceremonies,
  and owner-approved charter updates.
- `CONTRIBUTING.md`, `docs/testing.md`, PR template, and ADR lifecycle guidance.
- Only evidence-supported active ADR/spec status corrections; no blanket
  acceptance of Proposed records and no retroactive editing of landed specs.

Subsequent packages implement archival (P2), shared validation/checks (P3),
native integration (P4), and release enforcement (P5). Pending implementation
must be visible; this ADR does not assert those controls already exist.

### P1 prepared consumer inventory

| Surface | Prepared files |
|---------|----------------|
| Policy | `.specify/memory/constitution.md`, `.github/copilot-instructions.md` |
| Squad coordination | `.github/agents/squad.agent.md`, `.squad/routing.md`, `.squad/ceremonies.md` |
| Charters | `.squad/agents/{maximus,brutus,cassius,aurelia,scribe,ralph}/charter.md` |
| Session/audit entry points | `.github/prompts/{load-context,checkpoint,handoff,audit}.prompt.md` |
| SpecKit | `.github/agents/speckit.{specify,plan,tasks,implement,constitution,analyze,clarify,checklist,taskstoissues}.agent.md` |
| Templates | `.specify/templates/{spec,plan,tasks,agent-file}-template.md` |
| Contributor and lifecycle guidance | `CONTRIBUTING.md`, `docs/testing.md`, `.github/pull_request_template.md`, `docs/adr/README.md` |

The local coordinator and four pipeline prompts replace lengthy bundled recipes
with repository-specific instructions so conflicting directives are removed,
not merely hidden behind an override. This is not a Squad/SpecKit binary upgrade.
No scripts, workflows, native skills, hooks, model settings, or live protections
were changed. The later tool refresh and end-to-end agent behavior evaluation remain pending;
source consistency and bounded script checks are not universal enforcement.

The global instructions still exceed the proposed 150-line warning budget;
P4's scoped extraction is the explicit temporary exception. Active decisions and
histories retain the baseline sizes until accepted P2 archival. Existing wrapper
skills remain subordinate to policy and require P4 review/native migration;
the revised audit skills supplied in conversation are not installed here.

Governance verification exercised explicit feature selection, full-prerequisite
failure on missing directories, and nonmutating new-feature dry-run behavior.
The pre-existing branch-validation helper's success-stream diagnostics prevent
claiming its Boolean rejection is enforced; P3 must add regression coverage and
repair it. Until then, path-resolution consumers explicitly verify the selected
directory, returned paths, and required spec rather than trusting `-PathsOnly`.
The implementation plan records these bounded results and limitations.

The independent reviewer's application-policy scope finding was revised by a
separate owner under the existing lockout. The baseline UI row is preserved;
explicit reviewer clearance and the exact candidate identity are recorded in
the PR, not inferred from the repair. `.squad/team.md` is excluded to avoid the
unrelated roster-label workflow on the temporary-branch push.

### Lifecycle reconciliation (D05 partial)

| Record | Source evidence | Prepared disposition / remaining authority |
|--------|-----------------|--------------------------------------------|
| ADR 0013 | Source header lines 3-4 says Accepted | Index corrected to match; accepted body unchanged |
| ADR 0017 | Source header lines 3-4 says Accepted | Index corrected to match; accepted body unchanged |
| ADR 0005 | Header says Proposed; constitution section 23 records adoption of 3.0.0 | Preserve Proposed pending approval/merge evidence reconciliation; principle implementation is not acceptance evidence |
| ADRs 0011, 0016, 0018 | Source headers remain Proposed | No promotion from later code, task counts, or references in accepted ADRs |
| Feature 359 | `specs/359-coin-copilot-harness/spec.md:5` says Ready for Implementation; tasks have 70 checked items through `tasks.md:126` | Historical status is stale relative to task claims; do not rewrite landed spec or infer verified/accepted/released status |
| Feature 362 | `specs/362-coin-copilot-attribution/spec.md:5` says Ready for Planning; tasks have 84 checked items through `tasks.md:216` | Same restriction; reconcile candidate-specific gate/review/owner evidence before advancing lifecycle |
| Feature 363 / current pointer | `specs/363-collector-curator-watchlist-provenance/tasks.md:134-135`: T043 checked, T044 combined audit open; `.squad/identity/now.md` still pauses after T011 | Do not resume from the stale pointer or claim release readiness; P2 must derive the current view from reconciled evidence after archival authority is accepted |
| Feature 357 review blocks | Carried-forward records above | Unchanged; no clearance inferred or granted |

D05 remains open. The above are bounded metadata/evidence dispositions, not a
new authoritative feature-status ledger. No historical spec, decision, archive,
history, or current-work pointer was rewritten. A later authorized reconciliation
must link exact approval/clearance evidence or keep the state unresolved.

## Alternatives

- Add more instructions without consolidation: rejected because conflicts and
  duplicated maintenance would grow.
- Replace SpecKit and Squad with another orchestrator: rejected because no
  observed problem requires a new framework.
- Make every change a full multi-agent feature workflow: rejected as
  disproportionate for ordinary repairs.
- Clear historic blocks during cleanup: rejected because missing evidence is
  not approval.
- Upgrade every CLI immediately: rejected because SpecKit is already current
  in the observed installation and generated repository state is a separate issue.

## Consequences

Positive: clearer authority, smaller startup context, fewer unnecessary handoffs,
and completion that can be traced to actual verification and review.

Costs: a one-time synchronization and preservation pass, small deterministic
checks to maintain, and explicit distinction between proposed and active controls.

Risks: losing a binding historical decision while summarizing; overfitting
checks to Markdown formatting; mistaking an agent-authored approval field for
human authorization. Mitigate with preserved originals, tested diagnostics,
independent review, and authenticated approval evidence.

## Verification and rollback

Use the implementation plan's failure scenarios and P0-P6 acceptance criteria.
Review an exact-file diff; confirm no application behavior or live GitHub setting
changes. Do not require or perform a production deployment for this amendment.

If curation loses meaning, restore from the preserved originals. If an executable
wrapper is defective, revert that wrapper without bypassing its underlying gate.
Reverse or supersede governance changes through section 22; do not erase their
approval history. Any tool/settings rollback is separately authorized.

## Related

- `docs/agentic-delivery-improvement-plan.md`, P0-P6
- `.specify/memory/constitution.md`, Principles IV, VII-IX and sections 0, 17-23
- ADR 0001 (decision lifecycle), ADR 0005 (principle consolidation),
  ADR 0006 (workflow-contract regression gates)
