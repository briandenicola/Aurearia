# Agentic Delivery Improvement Plan

**Date:** 2026-09-21
**Status:** D05/P2 accepted via PR #736; P3 implemented with local evidence, independent review and hosted CI pending; P4-P6 not started
**Sponsor and approval owner:** Repository owner
**Scope:** AI-assisted development workflow, not application runtime behavior
**Authority:** Constitution Principles IV, VII, VIII, IX; sections 0, 17-22
**Related backlog:** F016, `specs/_backlog/F016-agentic-engineering-quality-cockpit.md`

## 1. Outcome

Make ordinary changes easier to deliver correctly, with less repeated context
loading, fewer contradictory instructions, and evidence-backed completion.

The intended operating model is:

- One authoritative project policy, with task-specific instructions loaded on demand.
- One implementation owner per change and an independent reviewer when required.
- Explicit approved scope, acceptance criteria, and stopping conditions.
- Executable validation shared between local development and CI.
- Distinct implemented, verified, accepted, and released states.
- Human approval for scope expansion, governance amendments, and release decisions.

This document is an action plan, not a new source of governing policy. Conflicting
current rules remain in force until changed through the constitution's amendment
process. Agreement with this plan does not retroactively clear reviewer blocks.

## 2. Boundaries and approvals

### Included

Constitution alignment; ADR lifecycle; active instructions; SpecKit templates and
integration; Squad routing, handoff, and memory; validation commands; process
checks; GitHub release controls; a five-change pilot.

### Excluded

- Application features, AI inference pipelines, database migrations, or UI changes.
- A new orchestration framework, autonomous work factory, or engineering dashboard.
- Broad cleanup of every historical spec, log, or decision.
- Rewriting published Git history or accepted ADR bodies.
- Automatically enabling Ralph, deployments, or unattended issue pickup.
- Promoting or implementing the full F016 cockpit backlog.

### Approval boundaries

| Action | Required approval |
|--------|-------------------|
| Record this plan | Requested by the owner; no implementation is implied |
| Change constitution, locked charters, or archive semantics | Explicit amendment approval through section 22; include the proposed ADR and synchronized consumers |
| Install or upgrade tools; enable hooks; run local builds or containers | Confirm the bounded operation with the owner before execution |
| Change GitHub protections, workflow activation, or release authorization | Present the exact setting diff and obtain owner approval |
| Merge or publish a release | Explicit owner release authorization for the candidate commit |

Ordinary implementation work continues on `beta`. Do not silently introduce
feature branches as a prerequisite. The governance amendment must still follow
the currently required section 22 PR process; agree its route before editing
locked artifacts. This plan does not open that PR or authorize a merge.

**Preparation approval (2026-09-21):** The owner selected: "Approve P1 local
edits on beta; include amendment in a separately approved beta-to-main PR".
This authorizes P1 preparation, not builds, tool upgrades, live GitHub changes,
commits, pushes, PR creation, acceptance, or P2 archival.

**Updated authorization (2026-09-21):** After approving the draft, the owner
authorized independent review, governance-only verification, and a temporary
branch/PR into `beta` containing only the amendment. This supersedes the earlier
beta-to-main amendment route and authorizes the necessary commit/push/PR work.
Merge, installs, deployments, application changes, and release remain prohibited
without further approval. An isolated worktree preserves concurrent unrelated
application edits in the original `beta` worktree.

**Acceptance and closeout (2026-09-21):** The owner approved and merged PR #734
into `beta` at `ca85f7830349a4472cb5409c082b9485e7e34c44`, then authorized a
metadata-only closeout PR to synchronize acceptance notices and the section 22
announcement. Constitution 4.0.0 is the accepted policy. Earlier preparation
entries below are historical snapshots, not current activation blockers.
This authorization does not extend to merging the closeout PR, installations,
deployments, P2 archival, or clearing unresolved historical review restrictions.

## 3. Baseline and observed problems

These observations are a dated baseline, not permanent assertions about the repo.
Recheck them when implementation begins.

| ID | Evidence from the audit | Delivery risk |
|----|-------------------------|---------------|
| B1 | Constitution section 0 claims highest authority; `.github/agents/squad.agent.md` says its own instructions win every conflict | Different entry points apply different policy |
| B2 | Four implementation/review charters and the audit prompt reference superseded principle numbers | Reviews enforce obsolete requirements |
| B3 | Active decisions: 12,210 lines, about 92,000 words; four agent histories: 149-212 KB | Important current decisions are buried in historical output |
| B4 | Task template makes tests optional; documented gates omit frontend lint that CI runs | A faithfully followed plan can still fail the actual gate |
| B5 | Features 359 and 362 have completed task lists but pre-implementation statuses; `identity/now.md` contains an outdated F015 pause | Agents restart completed work or stop on obsolete blockers |
| B6 | ADR 0005 remains Proposed; ADR index disagrees with Accepted records 0013 and 0017 | Approval state is not trustworthy |
| B7 | SpecKit's installed branch validation rejects `beta` without an explicit feature override | Workflow depends on undocumented session setup |
| B8 | Squad allows changed files to substitute for a missing completion response; handoff is fire-and-forget | Partial work can be reported as complete |
| B9 | Main has seven required status contexts, no approving-review requirement, and admin bypass; beta has no required contexts | Written approval rules exceed enforced controls |
| B10 | Ralph heartbeat is manually disabled while repository documentation describes ongoing monitoring | Automation appears active when it is not |

Existing strengths to preserve: architecture enforcement, exact-path regression
requirements, independent review, bounded feature scope, and Docker publishing
from the exact SHA of a successful Quality Gate.

## 4. Work packages and dependencies

Each package is one reviewable outcome, not a mandate to create a separate PR.
Use sequential execution by default. Do not start downstream implementation while
the governing decision is unresolved.

| Package | Outcome | Depends on |
|---------|---------|------------|
| P0 | Evidence baseline and approved amendment proposal | Owner authorizes implementation planning follow-through |
| P1 | One consistent policy and proportional delivery lanes | P0 and amendment approval |
| P2 | Small active context and trustworthy current-work pointers | P1 |
| P3 | Shared executable validation and a small governance check | P1; use P2's approved state format |
| P4 | Native Copilot integration and bounded optional Squad usage | P1-P3 |
| P5 | Evidence-bound acceptance and release controls | P3-P4 and settings approval |
| P6 | Five-change pilot and adoption decision | P2-P5 |

### P0. Establish the baseline and propose the amendment

**Primary surfaces:** constitution, ADR index, instruction files, charters,
`identity/now.md`, selected active specs, GitHub protection/workflow settings.

- [x] **D01 - Capture the implementation baseline.** Record the starting commit,
  worktree state, active instructions, relevant tool versions, and GitHub settings.
  Keep exports in approved local artifacts or workflow artifacts, not a new
  permanent transcript. Identify actual open review blocks and their owners.
- [x] **D02 - Propose one governance amendment.** Allocate the next available ADR
  number at implementation time. Cover precedence, mutable current-state views,
  proportional validation, author revision after rejection, and evidence-based
  completion. Propose the appropriate version bump; operational restructuring
  likely warrants a major bump under section 22.

**Acceptance:** The owner can approve or reject a concrete set of changes without
approving a tool upgrade or application change. Every existing reviewer block has
an explicit disposition or remains open. Nothing is inferred as approved.

**Stop point:** Do not modify locked artifacts until the amendment route and
authority are approved.

**P0 evidence (2026-09-21):** Baseline captured at
`cacc177c2ecd85d62899507413184cfc395a7f6a` on `beta`. See
[ADR 0019](adr/0019-evidence-based-agentic-delivery.md) for the refreshed baseline,
carried-forward review restrictions, concrete amendment decisions, and proposed
4.0.0 constitutional version. Diagnostic settings/tool observations and file
hashes are in the initiating session's `files/delivery-baseline.json`.
Historical review records have not been globally adjudicated; no block is
cleared by this work. The installed SpecKit CLI is already 1.0.8, while the
checked-in integration remains 0.5.1.dev0; P4 must evaluate repository artifact
refresh separately from binary upgrades. No locked policy, tool installation,
build, deployment, or live GitHub setting was changed during P0.

### P1. Reconcile policy and define proportional delivery lanes

**Primary surfaces:** `.specify/memory/constitution.md`,
`.github/copilot-instructions.md`, `.github/agents/`, `.github/prompts/`,
`.squad/agents/*/charter.md`, `.squad/routing.md`, `.squad/ceremonies.md`,
`CONTRIBUTING.md`, `docs/testing.md`, ADR index, SpecKit templates.

- [x] **D03 - Reconcile authority and references (local draft prepared).** Make Squad explicitly
  subordinate to repository policy. Explain where accepted ADRs fit and how an
  approved amendment changes existing policy. Distinguish owner-authorized scope
  changes from ordinary implementation judgment. Replace obsolete references in
  active guidance; preserve historical quotations and logs.
- [x] **D04 - Align planning and delivery guidance (local draft prepared).** Remove optional-test defaults
  that conflict with project policy. Remove obsolete Phase 3 placeholders and
  divergent gate recipes. Define the three lanes below, including proportionate
  docs-only validation and platform-specific gate responsibilities.
- [x] **D05 - Reconcile ADR and spec lifecycle metadata.** Check approval evidence
  before updating status; never accept an ADR simply because code exists. Align
  the ADR index with source headers. Reconcile currently relevant feature statuses
  and record supersession links without rewriting accepted historical bodies.

| Lane | Minimum preparation | Required completion evidence |
|------|---------------------|------------------------------|
| Bug or small behavior-preserving change | Reproduction, expected behavior, bounded cause, sibling paths, non-goals | Original symptom checked; targeted regression or explicit manual exception; applicable shared gates; review |
| Feature | Approved outcome/non-goals, acceptance criteria, short plan, bounded tasks | Usable slice demonstrated against criteria; applicable gates; review; status reconciliation |
| High-risk change | Feature/bug preparation plus ADR when required, compatibility and rollback plan | Risk-specific evidence, independent review, explicit owner acceptance/release approval |

High-risk examples include auth, data semantics, migrations, service boundaries,
external providers, and changes to the delivery controls themselves. A small diff
does not automatically make a change low risk.

**Acceptance:** Active instructions agree on authority, tests, branching, and
completion. Historical artifacts are not falsely flagged as current policy.
The Sync Impact Report lists every affected consumer and unresolved exception.

**P1 preparation evidence (2026-09-21):** Constitution 4.0.0 and active consumers
are prepared locally under ADR 0019, still Proposed. Its consumer inventory and
D05 table record exact scope and unresolved lifecycle evidence. D03/D04 checkmarks
mean authored draft, not verified runtime behavior, amendment acceptance, or
release readiness. D05 is partial: ADR index/header mismatches are corrected;
uncertain approvals and stale feature states are not invented or retroactively
rewritten.

Source-level walkthrough covered authority conflicts, beta feature selection,
required tests, generated-instruction overwrite, empty agent results, existing
review lockouts, missing execution evidence, proportional docs validation, and
CI-versus-release authority. The amended prompts explicitly reject the unsafe
paths. This is a document review, not a runtime/fixture pass.

`git diff --check` passes after normalizing the affected LF files. Baseline
SHA-256 checks confirm the decisions ledger/archive, current pointer, and all six
agent histories are unchanged. Application code, scripts, workflows, and feature
specifications are unchanged. No builds, installs, commits, pushes, or GitHub
mutations were performed.

**Initial draft gaps:** independent amendment review, prompt/behavior evidence in an
authorized environment, D05 approval/clearance reconciliation, and separate
publication/acceptance approval. P2 archival remains blocked until section 22
acceptance; P3-P6 are not started. Warning-budget exceptions and uninstalled
audit skills remain explicitly tracked in ADR 0019.

**Authorized governance verification (2026-09-21):** In the isolated amendment
worktree, actual prerequisite execution with an explicit
`SPECIFY_FEATURE=359-coin-copilot-harness` resolved the exact existing spec/tasks.
Full prerequisites rejected absent/invalid selections through missing-directory
checks. A numbered new-feature dry-run returned expected paths without creating
files or switching branches. Whitespace/structure/scope checks passed; scripts,
workflows, application code, historical state, and feature specs remain unchanged.

These checks do **not** prove the existing branch-validation helper rejects
invalid branches: its success-stream diagnostics can obscure its Boolean result.
`-PathsOnly` is therefore documented as resolution, never validation. Plan and
clarify prompts explicitly compare paths against the selected existing directory
and required spec. Add a regression and repair the script in the authorized P3
work, not by expanding this amendment.

Independent review found an application UI-policy row accidentally included in
the instruction diff. A separate revision owner restored the baseline row under
the existing strict lockout; the original reviewer must explicitly clear that
block. Reviewer disposition and the exact candidate identity belong in the PR.
The roster edit was excluded to avoid triggering automatic label synchronization
on the temporary-branch push. No application behavior or deployment is authorized.
Bounded script checks and semantic review do not replace the later P3/P4
automated fixtures and end-to-end prompt evaluation.

### P2. Reduce active memory and make work selection explicit

**Primary surfaces:** `.squad/decisions.md`, `.squad/decisions-archive.md`,
`.squad/agents/*/history.md`, `.squad/identity/now.md`, session prompts.

- [x] **D06 - Curate active decisions and histories.** Inventory entries before
  moving anything. Retain current, cross-cutting decisions with stable identifiers,
  scope, status, and source links. Move historical reports to the approved archive
  surface without changing their meaning. Preserve unresolved rejections and
  acceptance evidence. Have the reviewer verify an inventory/checksum record of
  preserved original material.
- [x] **D07 - Establish one current-work pointer.** Reuse `identity/now.md` rather
  than adding a competing session-state file. Keep the issue/spec, task pointer,
  owner, blockers, evidence links, and next action. Detailed task state remains in
  the task list; the pointer must not duplicate every task or test result.

Proposed operating budgets, to be confirmed in P1:

- Repository-wide automatic instructions: at most 150 lines.
- Active decisions: at most 20 KiB; each current agent history: at most 12 KiB.
- Current-work pointer: at most 100 lines.
- Initial required policy/context reading: target at most 6,000 words, excluding
  the selected feature artifacts and source evidence.

Budgets are maintenance signals, not permission to truncate binding decisions.
Exceeding a budget prompts consolidation or an explicit exception.

Resolve the active feature from an explicit request or validated current-work
pointer. On `beta`, never guess from the highest-numbered spec. Supply the feature
identifier on every relevant invocation; do not depend on an environment variable
surviving between fresh shell processes. Missing or conflicting selection must
stop with a clear diagnostic.

**Acceptance:** A fresh session can identify current work, authority, unresolved
blocks, and the next action without scanning archives. Completed work does not
appear paused or ready to start. Archival preserves original evidence.

**D05/P2 execution evidence (2026-09-21):** Owner authorized the isolated
`docs/delivery-lifecycle-context` branch and a PR into beta, with no merge,
installation, deployment or later-phase work. D05/D06/D07 checkmarks mean their artifacts are authored and independently
verified, not owner acceptance of this PR.

**Independent review: PASS**, candidate tree
`7c0427831ef7c3c508e38de05a78681b75cf9f91`, against baseline
`e6ab8313346b971dce4b288804036222f7d4c95d`. The independent reviewer recomputed
all original bytes/sections, checked 38 local links against the Git tree,
fetched PR merge/file evidence, and confirmed preservation of reviewer terms
and immutable bodies. No historical application block was cleared.
Final handoff: [.squad/log/2026-09-21-delivery-context-review.md](../.squad/log/2026-09-21-delivery-context-review.md).

The [active decisions](../.squad/decisions.md) contain lifecycle dispositions,
stable decision IDs and unresolved reviewer records. Actual owner-merged PRs
#248, #626 and #732 include the previously Proposed ADR files; their headers and
index are reconciled. ADR bodies and landed spec requirement bodies are unchanged.
Feature 357's T093 acceptance claim is unchecked against its recorded REJECT.
Features 359/362/363 no longer appear ready to start or paused after T011.
The still-open combined F014/F015 audit is not silently certified by #732's merge.

The [preservation inventory](../.squad/artifacts/context-curation-2026-09-21.json)
indexes 2,795 original sections covering 2,040,296 canonical Git-blob bytes,
including the untouched existing archive and two unchanged small histories.
Snapshots precede curation; byte equality, full/section SHA-256, contiguous
coverage and line mappings pass. Twenty-eight negative controls reject changed
bytes, truncated originals, wrong offsets and missing inventory sections.
Current decision/history/now.md budgets and newly authored local links pass.

**Explicit maintenance exception:** In the reviewed candidate, a conservative full initial load of automatic
instructions + constitution + active decisions + now.md is 9,169 words, above the
6,000-word warning. Automatic instructions remain 324 lines versus the 150-line
warning. P4's approved on-demand instruction extraction remains the next place
to address those budgets; this batch does not truncate binding policy to pass.
Roles and selected historical evidence remain on-demand.

**Preservation whitespace exception:** Archive snapshots deliberately retain
original whitespace. The decision snapshot includes an existing whitespace-only
line (original decisions.md:9733, appended archive:22763). Byte preservation is
the archive check; ordinary diff-whitespace checks apply to newly authored current
views and lifecycle metadata. This is not permission to normalize old evidence.
No application build/test result or release acceptance is claimed by this batch.

### P3. Make validation executable and detect governance drift

**Primary surfaces:** `Taskfile.yml`, package scripts, `.github/workflows/ci.yml`,
`.github/workflows/security-scan.yml`, contributor/testing instructions.
**Proposed new surface:** minimal process-check scripts and their fixture tests
under `scripts/delivery/`; do not add a separate service or framework.

- [ ] **D08 - Extract shared validation entry points.** Reuse existing commands
  through Taskfile targets; make local instructions and CI call the same targets.
  Cover Go build/vet/tests, frontend lint/type-check/tests/build, Python locked
  environment lint/tests, and OpenAPI consistency. Preserve CI race and security
  coverage. Keep dependency installation separate from validation.
- [ ] **D09 - Add a deterministic governance check.** Start with active-file
  references, supported principle identifiers, ADR header/index agreement,
  valid current-work targets, native skill metadata, and context-size warnings.
  Define the active surface explicitly so archived history is not policed as
  current instructions. Emit file/line diagnostics and nonzero exit on errors.
- [ ] **D10 - Prove the checks detect violations.** Add fixture tests containing
  obsolete principle references, invalid ADR status links, a missing spec,
  malformed state, and invalid skill metadata. Deliberately break each blocking
  guard, prove the expected failure, and restore it. Verify documented validation
  does not omit a CI command such as zero-warning frontend lint.

Validation tiers must be explicit:

- **Fast feedback:** targeted tests during implementation.
- **Completion:** the applicable shared recipes for the change and its sibling
  workflows; no unapproved skips.
- **CI/release:** runner-only checks, full release checks, and required scans.

Use Windows-compatible command invocation locally and verify Linux CI behavior.
Do not silently skip a missing linter or treat an unavailable tool as a pass.
OpenAPI generation may write files: label it accordingly, and fail if generated
tracked artifacts remain inconsistent.

Keep semantic judgment out of simplistic lint rules. A governance check cannot
prove that prose is correct, a scope expansion is acceptable, or an AI review is
independent merely because a Markdown field says so.

**Acceptance:** One documented route reproduces the applicable CI validation.
The checker runs offline, makes no repository changes, and has no dependency on
an LLM. Intentional fixture failures demonstrate each blocking rule. Start size
and historical-lifecycle findings as warnings; do not blanket-waive new defects.

**P3 execution (2026-09-21):** Owner requested "go start P3" after accepting
#736. Work uses `docs/delivery-validation` from
`af2ac2488cf38cd4ce82a33cdd7de986d3fe8045`, preserving the original dirty beta
worktree. No local installation, merge or deployment is authorized or performed.
P4-P6 and live required-context changes remain outside this batch.

**Bounded portability approval:** Hosted Windows checkout exposed 33 historical
log filenames containing colons. Sparse exclusions did not avoid Git's NTFS
protection check. The owner explicitly approved a filename-only fix, preserving
contents and recording old/new paths. The
[path map](../.squad/artifacts/windows-log-path-mapping-2026-09-21.json)
records each unchanged Git blob identity. Both hosted jobs use full checkout with
NTFS protection enabled; local Git configuration and historical bodies remain
unchanged. This is not approval for broader archive cleanup.

- D08 shared `check:*` targets are wired into Quality Gate; setup is separate,
  npm lint is unconditional, Python uses locked/offline/no-sync validation,
  and the race/security/browser/compatibility surfaces remain present.
- D09 is a Node-built-in-only offline checker with explicit active scope,
  diagnostics, native metadata checks, current-work targets and warning budgets.
  Historical bodies/archives and optional/runtime examples are not policed.
- D10 includes invalid fixtures, diagnostic-ablation controls, real Task
  failure/missing-linter execution and the SpecKit stdout/Boolean regression.
  The original SpecKit bug is restored only in a disposable fixture to prove
  detection; real feature artifacts are never mutated.
- Local Windows delivery fixtures, actual `task check:go` (build/vet/full tests),
  and actual `task check:openapi` pass using existing tools/cached dependencies.
  Full web/Python application execution and Linux behavior await the hosted jobs;
  fake command fixtures are not presented as application test results.
- Shared OpenAPI generation exposed existing version drift. The `main.go`
  annotation and four generated snapshots are synchronized from 4.0.0 to the
  canonical VERSION 4.3.0; these five single-value changes do not change endpoints.
  Swagger's displayed banner still says v1.16.4 for module v1.16.6, so the tool
  pin is checked against Go build information rather than that stale banner.
- Independent review and matching hosted evidence are required before marking
  D08-D10 complete. Owner merge/release approval remains separate.

### P4. Align Copilot tooling and make Squad optional

**Primary surfaces:** `.github/copilot-instructions.md`, `.github/agents/`,
`.github/prompts/`, `.specify/`, `.squad/routing.md`, `.squad/ceremonies.md`.
**Proposed native surfaces:** `.github/instructions/` and `.github/skills/`.

- [ ] **D11 - Use native instruction and skill discovery.** Keep universal rules
  short. Put Go, Vue, Python, and workflow-specific guidance in path-scoped
  instructions. Migrate relevant reusable skills to valid native skill packages;
  resolve duplicate skills and leave links rather than duplicate authoritative
  copies. Confirm discovery in a fresh Copilot session.
- [ ] **D12 - Evaluate a pinned SpecKit upgrade.** Compare the current installed
  and checked-in versions with an explicitly selected release. Review generated
  file changes before adoption. Evaluate the bug workflow and convergence review,
  retaining mandatory test policy and direct-`beta` compatibility through supported
  project customization. Do not force-reinitialize over customized files.
- [ ] **D13 - Bound collaboration and handoff.** Default to one implementation
  owner. Use an independent read-only reviewer with restricted tools. Use Squad
  only for work that benefits from separate contexts and disjoint ownership.
  Remove automatic success inference from changed files and wait for final
  handoff persistence before reporting completion.

Every delegated assignment must include:

| Field | Required content |
|-------|------------------|
| Objective | One bounded outcome and its acceptance criteria |
| Authority | Relevant approved scope and policy references |
| Ownership | Allowed write paths and shared files that must not be edited |
| Non-goals | Explicit exclusions and prohibited scope expansion |
| Evidence | Required reproduction, checks, and result format |
| Lease | A task-specific checkpoint and maximum effort approved before launch |
| Stop condition | Completion, blocker, exhausted lease, or newly discovered scope |

A checkpoint must report evidence and remaining work, not merely continued
activity. No nested delegation without approval. Stop or re-scope unproductive
work instead of extending leases automatically.

After the P1 amendment, the proposed rejection model is: a reviewer block remains
binding, but the original author may normally repair it. Independent replacement
is an explicit reviewer escalation, not the default for every correction. Only
the responsible reviewer or documented escalation authority clears the block.

Do not hardcode a stale model catalog or silently retry through expensive models.
Use configured Copilot model preferences and explicit task budgets. Keep Scribe
restricted to named artifacts; prohibit broad staging of unrelated work.

**Acceptance:** A routine fix completes without mandatory team fan-out. The
reviewer cannot edit the implementation. Missing agent results remain incomplete.
Fresh-session discovery loads the right scoped rules and skills. Upgrades
preserve the approved policy and can be reverted independently.

### P5. Tie acceptance and release to evidence

**Primary surfaces:** PR template, handoff/checkpoint prompts, GitHub protections,
publishing workflows, and optional repository hooks.

- [ ] **D14 - Define an evidence-backed completion record.** Reuse task/PR/handoff
  surfaces, linking raw output rather than copying it into every history. Record
  acceptance criteria, affected workflows, verification results, reviewer verdict,
  unresolved exceptions, and the reviewed commit. For dirty-tree validation,
  record the tested tree identity; changed implementation invalidates old proof.
- [ ] **D15 - Align live controls with intended release behavior.** Present a
  before/after settings proposal to the owner. Preserve direct beta validation
  pushes. Require an explicit accepted release candidate and owner authorization
  before main/release actions. Decide which additional CI contexts are mandatory
  and how admin bypass is treated. Document Ralph as intentionally disabled unless
  the owner explicitly approves enabling it.

| State | Meaning |
|-------|---------|
| Implemented | The scoped change exists; verification may still be outstanding |
| Verified | Applicable checks and original acceptance path have evidence |
| Accepted | Required review blocks are cleared and owner approval exists where required |
| Released | The authorized candidate was promoted/published; deployment is separately recorded if applicable |

Do not require an unavailable second human approver in a solo-maintainer workflow.
Use an owner-approval mechanism appropriate to this repository, and ensure an
agent-authored checkbox cannot impersonate that authorization.

Native hooks are optional defense in depth. Pilot read-only diagnostics before
blocking tool use. They must not auto-install dependencies, execute deployments,
or run broad builds at session start. Use permission restrictions and GitHub
controls as backstops; a shell-command regex is not a reliable security boundary.

Preserve publishing from the successfully checked SHA. Do not weaken existing
checks or equate successful CI with acceptance of the user's actual workflow.

**Acceptance:** A stale review or open block prevents an accepted/release-ready
claim. Missing evidence is explicitly incomplete. The owner can distinguish
validation pushes from approved releases. Any settings change has recorded
approval and a precise restoration procedure.

### P6. Pilot five changes and retain only useful process

- [ ] **D16 - Run a measured pilot.** Select five ordinary changes, preferably
  including a backend fix, frontend fix, cross-service change, and small feature.
  Compare with five recent comparable changes where reliable evidence exists.
  Do not fabricate missing baseline effort or cost data.
- [ ] **D17 - Make the adoption decision.** Review outcomes with the owner.
  Retain useful checks and remove ceremony that adds effort without preventing
  defects. Record any further work as bounded backlog items. Do not automatically
  expand into the full F016 cockpit.

| Measure | Pilot target |
|---------|--------------|
| Wrong or ambiguous active-work selection | Zero |
| Known conflicting active instructions | Zero |
| First complete CI gate attempt | At least 4 of 5 pass; failures classified |
| Acceptance evidence and exact reviewed commit | Present for all 5 |
| Unapproved scope expansion or release | Zero |
| User-reported recurrence of the same symptom | Zero during a seven-day observation period after each change |
| Review revision cycles | Record per change; investigate repeated cycles without imposing a safety-reducing cap |
| Agent effort | Record elapsed active effort and available usage/cost; compare medians where the baseline is valid |
| Context budgets | Met or explicitly excepted without dropping current decisions |

Five changes are a directional pilot, not statistical proof. A missed target
requires diagnosis, not hiding a failure or weakening a gate. Mark unavailable
measurements as unavailable.

**Acceptance:** The owner has evidence to adopt, adjust, or roll back the process.
All pilot changes have an explicit disposition, including the observation period.

## 5. Verification scenarios for the delivery system

Use synthetic fixtures and disposable approved test contexts, not intentionally
corrupted production policy.

| Scenario | Expected behavior |
|----------|-------------------|
| A task is started on beta without an explicit or unambiguous active feature | Stop and request the target; never choose the newest spec |
| An active charter references an obsolete principle | Governance check identifies the file and reference |
| A historical log quotes that same obsolete principle | Preserve it; do not fail active-policy validation |
| An ADR index disagrees with its header | Fail with both paths; do not automatically accept either state |
| An agent returns no result but has changed files | Report incomplete and inspect work; never infer success |
| A required frontend lint command fails with warnings | Completion fails under the same policy as CI |
| A required tool is missing | Report the missing prerequisite; request approved setup |
| The implementation changes after review | Old acceptance evidence is stale until applicable re-verification |
| A reviewer block is unresolved | No accepted or release-ready claim |
| Scribe or another actor leaves unrelated staged files | Commit only the approved path set; preserve unrelated changes |
| A proposed tool upgrade overwrites local policy | Reject the upgrade diff until customization is preserved |
| A session cannot run a runner-only check | Record pending CI evidence, not a local pass |

## 6. Rollout, rollback, and maintenance

Deliver packages as small, independently reviewable changes. Do not bundle tool
upgrades, GitHub settings changes, and archival into an unreviewable migration.

- **Governance:** Roll back or supersede through the amendment process; never erase
  the original approval record.
- **Memory:** Preserve original entry content and a migration inventory. Restore
  from that record if the curated view loses a current decision.
- **Validation:** Keep behavior equivalent while extracting shared commands.
  Revert a defective wrapper rather than bypassing its underlying checks.
- **Tooling:** Pin the adopted version and retain a reviewed file diff. Upgrade
  CLI binaries and generated project artifacts deliberately, not implicitly.
- **GitHub settings:** Capture the original values and exact approved update.
  Restore only the settings changed by this work, with owner authorization.

At completion, use existing handoff surfaces to record the outcome and next
action. Commit only approved work and follow repository commit conventions.
Confirm authorization before pushing changes that trigger remote automation.
No application deployment or automatic main merge is part of this plan.

Ongoing maintenance should be lightweight: run deterministic checks in CI, update
the active-work pointer at handoff, reconcile ADR status on acceptance, and review
context-size warnings when they occur. Do not introduce another mandatory weekly
multi-agent ceremony.

## 7. Sources and implementation references

Repository evidence:

- `.specify/memory/constitution.md`
- `.specify/scripts/powershell/common.ps1`
- `.specify/templates/tasks-template.md`
- `.github/agents/squad.agent.md`
- `.github/prompts/load-context.prompt.md`, `checkpoint.prompt.md`, `handoff.prompt.md`
- `.squad/agents/`, `.squad/decisions.md`, `.squad/identity/now.md`
- `docs/adr/README.md`, ADRs 0005, 0013, and 0017
- `docs/testing.md`, `Taskfile.yml`, `.github/workflows/`
- `specs/359-coin-copilot-harness/`, `specs/362-coin-copilot-attribution/`,
  `specs/363-collector-curator-watchlist-provenance/`

Upstream references verified during the 2026-09-21 audit; recheck release support
before adopting:

- [Copilot scoped instructions](https://docs.github.com/en/copilot/how-tos/copilot-cli/customize-copilot/add-custom-instructions)
- [Copilot agent skills](https://docs.github.com/en/copilot/concepts/agents/about-agent-skills)
- [Custom agent configuration](https://docs.github.com/en/copilot/reference/custom-agents-configuration)
- [Copilot hooks](https://docs.github.com/en/copilot/reference/hooks-reference)
- [SpecKit v1.0.8 release](https://github.com/github/spec-kit/releases/tag/v1.0.8)
- [SpecKit bug workflow](https://github.github.io/spec-kit/guides/bugfix.html)
- [SpecKit implementation and convergence](https://github.github.io/spec-kit/reference/agentic-sdd.html)
- [SpecKit upgrade guidance](https://github.github.io/spec-kit/upgrade.html)
- [Squad project and upgrade guidance](https://github.com/bradygaster/squad)
