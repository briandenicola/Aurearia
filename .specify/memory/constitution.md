<!--
Sync Impact Report — ADR 0024 (PROPOSED)
Version preparation: 4.0.0 → 5.0.0, MAJOR: incompatible operational restructuring.
Effective baseline remains 4.0.0 / Accepted ADR 0019, PR #734.
Baseline text: git show 283d0c2ebac0f066f23650c5445fefb62c401ab3:.specify/memory/constitution.md
Principles I–IX retain identifiers and substantive safeguards; VIII narrows ADR
ceremony to material architecture; IX permits focused local + exact-candidate CI.
Sections 0,17–21 condensed; technology/tutorial/command duplication removed.
Sections 22–23 retained, with preparation/activation distinction explicit.
Consumers: .github/copilot-instructions.md, pull_request_template.md,
instructions/delivery.instructions.md, agents/*.agent.md, prompts/*.prompt.md;
.github/skills/aurearia-software-qc-audit/SKILL.md;
.specify/templates/*.md; .squad/{routing,ceremonies}.md,
.squad/agents/*/charter.md, .squad/identity/now.md;
.squad/decisions.md (pending notice only, no acceptance announcement);
CONTRIBUTING.md, docs/testing.md, docs/adr/README.md,
docs/agentic-acceptance-controls.md; scripts/delivery/check-governance.mjs
and governance.test.mjs. See ADR 0024 for bounded disposition.
Delivery controls passed; isolated branch/review/publication authorized.
Independent pre-push review passed; pending PR publication, exact-head CI,
approval/merge, announcement and lifecycle transition.
Exceptions: no historical block cleared; #827 audit remains required.
No live settings/workflows, installations, release or deployment changed.
-->
# Ancient Coins Constitution

**Project**: Ancient Coins (self-hosted personal collection PWA)
**Version**: 5.0.0 (prepared; not effective)
**Amendment Status**: Proposed — [ADR 0024](../../docs/adr/0024-proportional-governance.md)
**Ratified**: 2026-04-28
**Last Amended**: 2026-09-21
**Amendment Prepared**: 2026-10-19

> **Activation boundary:** Constitution **4.0.0** / Accepted ADR 0019 remains
> effective, accepted through owner-approved PR #734 on 2026-09-21 at
> `ca85f7830349a4472cb5409c082b9485e7e34c44`. Its complete text is preserved at
> [the preparation baseline](https://github.com/briandenicola/Aurearia/blob/283d0c2ebac0f066f23650c5445fefb62c401ab3/.specify/memory/constitution.md).
> Everything below is prepared 5.0.0 text, activated only through §22.
> Synchronized consumers carry the same boundary; they do not waive current
> obligations, PR #827's pending audit, or historical reviewer restrictions.

## §0. Hierarchy of Authority

Constitution > PRD > applicable Accepted ADRs > owner-approved issue/PR scope
and criteria (or explicitly selected spec, plan, tasks) > backlog > active
decisions > agent judgment. Frameworks, templates, skills and charters implement
this order; none grants authority. Proposed ADRs are not policy. Peer ADR
conflicts require explicit supersession or owner resolution, not newest-file wins.
Stop on conflicts; synchronize affected consumers in the same amendment.

Select work explicitly; never infer it from branch or newest spec. Ordinary work
stays on `beta`; other branches/worktrees require applicable owner authorization.
Use one issue or PR for scope, criteria, evidence, review and outstanding work;
until one exists, the approved bounded work order is the source. Selected specs
remain binding where applicable, but no per-change spec/decision-inbox/handoff/
now/audit document set is required. The current-work pointer links sources only.
Preserve unrelated work and historical evidence.

## Core Principles

### I. Clear Layered Architecture

Go follows Handler → Service → Repository → Database. Handlers parse/delegate/
respond, with no business logic or raw SQL. HTTP-agnostic services own business
logic; repositories own GORM queries and reuse `repository/scopes.go`. Models
import only the standard library. Constructor injection is explicit; only
`main.go` may import `database`. Multi-step writes use transactions. Log
internal errors server-side; return generic errors to clients.

### II. Service Boundary Separation

Go owns auth, durable state, database access and SSE proxying, with zero LLM
logic. Python remains stateless with no direct database access; keys, prompts
and context arrive per request. Vue calls only Go `/api/*`, not Python. SSE
flows Python → Go → Vue. AI pipelines preserve tool-data provenance, Pydantic
worker schemas and supervisor iteration limits.

### III. Strict Types and Explicit Contracts

Keep Go compilation/vet, zero-warning frontend lint, strict `vue-tsc --build`
(not the weaker `--noEmit`), and locked-environment Ruff clean. Do not mask nullable
props; use typed optional/nullish handling. Python requests/responses use Pydantic.
Public Go handlers require Swagger annotations and synchronized generated Go
Swagger/OpenAPI snapshots. Vue API access uses `src/web/src/api/client.ts`.

### IV. Simple Complete Changes

Prefer direct typed code and existing patterns. Fix the real workflow and
directly related sibling paths, not only the first failure. Keep scope proportional
unless evidence proves a broader cause. Simplicity never waives architecture,
security, privacy, typing or data-contract safeguards.

### V. Security, Auth, and Privacy by Default

- Whitelist CORS origins; never allow all origins in production. Parameterize SQL
  or validate against allowlists. Validate upload allowlisted extensions
  (`.jpg`, `.jpeg`, `.png`, `.gif`, `.webp`) and magic-byte MIME, not headers alone.
- Rate-limit auth; cap multipart and JSON bodies (recommended 10 MB / 1 MB).
  Do not expose internal errors. Production containers run non-root.
- JWT access expiry is 15 minutes with `JWT_SECRET`; production should refuse an
  unset/low-entropy secret. Refresh tokens have 30-day rolling lifetime,
  `rt_` + 32 random hex bytes, SHA-256-only storage and one-time rotation/revocation.
  Client localStorage and concurrent 401 refresh/replay remain the accepted model.
- WebAuthn uses platform authenticators, configured production RP ID/origin and
  challenge TTL (recommended five minutes). API keys are hashed and revocable.
  First registered user becomes admin. New registration requires validated
  username/password/email; legacy missing-email handling retains `/auth/me`
  `emailMissing` and dismissible seven-day localStorage snooze.
- Accepted followers see only allowed gallery data, never private coins,
  price/value or AI analysis. `isPublic=false` permanently deletes followers.
  Preserve ownership checks and privacy/data-safety failure paths.

### VI. Consistent User Experience

Reuse design tokens (`variables.css`), global classes (`main.css`), local
components and navigation. Dark theme and `lucide-vue-next` are defaults.
No emojis in UI, prompts or AI responses. Desktop changes preserve mobile/PWA;
the offline app shell loads without connectivity, while API calls need network.

### VII. CI, Supply Chain, and Release Integrity

Use Conventional Commits (`feat`, `fix`, `docs`, `refactor`, `chore`); AI-assisted
commits include `Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>`.
Signed commits are not required. Pin Actions by SHA; production base-image
digests are recommended. Protect `main`. Taskfile/CI are executable quality
sources; `docs/testing.md` is the single command reference. Validate once on
beta, verify identical complete trees and trusted exact-candidate evidence on
PR/main, and promote receipt-bound immutable image digests. Changed content or
missing/unsuccessful proof fails closed. Never infer live activation from policy.

### VIII. Documented Decisions

Record material architectural decisions in ADRs: constitutional, service/security
boundary, significant provider/runtime dependency, semantic migration or UI
framework decisions. Editing an existing DTO or query is not automatically a new
architectural decision. Record lesser choices in canonical work. Keep applicable
Accepted ADR constraints; resolve conflicts explicitly. Never rewrite accepted
ADR bodies or historical logs.

### IX. Automated Enforcement Over Manual Memory

Prefer architecture/contract tests, schemas, lint and CI to repeated ceremonies.
Focused local feedback plus exact-candidate applicable CI can establish completion;
do not rerun unchanged valid proof. Applicable Go package tests, web tests and
locked Python tests remain required. Review addresses judgment and real workflow
coverage, not repeated checklist transcription.

## §17. Quality Gate

Choose fast focused feedback, then applicable completion and exact-candidate CI.
Keep actual regression, failure/retry/ownership/privacy and sibling-workflow
coverage; new service methods need unit coverage. Configurable UI values must
work on every submitted API path or be blocked with a clear message.
Preserve architecture/OpenAPI/service contracts, zero-warning lint, strict types,
Go race, browser/mobile, security findings, containers and compatibility/recovery
checks where applicable. Analysis-job success alone is not security clearance.
Executable policy/prompts/checks need meaningful behavior/negative fixtures.

No skipped failures, warning suppression, broad catch-success fallback or weakened
tests. Do not disable lint rules or use `any`, `@ts-ignore` or `nolint` without
inline justification; never use these to hide a failed gate.
Missing tools, unauthorized execution and runner-only checks stay pending,
not passed or N/A. N/A needs an applicability reason. Evidence identifies the
tested commit/tree; changed implementation invalidates affected proof, not every
unchanged result. Exact-candidate CI is necessary, not proof of an untested outcome.

## §18. AI Agent Operating Rules

### 18.1 Proportional lanes

| Lane | Scope and required evidence |
|------|-----------------------------|
| Routine | Text/layout, contained bugfix, behavior-preserving refactor: brief scope, appropriate regression and applicable CI; no mandatory independent review or broad audit. |
| Feature | New workflow within existing architecture: short criteria, workflow/error-path tests, ONE scoped independent review and applicable CI. |
| High risk | Changed auth/privacy boundaries, destructive migrations, service boundaries or delivery/release controls: feature requirements plus applicable security, compatibility and recovery proof; ADR only for material architecture. |

Risk overrides size. Touching an existing contract or many queries alone is not
a new high-risk boundary. Genuine privacy/data-safety changes remain high risk.
Review repairs only over the affected delta unless earlier evidence is invalidated.
Broad audits apply under §20, not automatically to every high-risk feature.

### 18.2 Authority and budget

An explicit bounded work order may authorize implementation, relevant tests,
review, in-scope repairs, beta commit/push and PR opening when green once for the
whole scope. It must name authorized actions; silence grants none. Ordinary
repairs within that scope/budget need no renewed permission. Merge may be
conditionally preauthorized for an exact eligible candidate with no unmet gate,
otherwise obtain explicit later approval. Deployment remains separate.
Install/setup, live settings and operational side effects need applicable
explicit authority; neither a work order nor this constitution implies them.

Default to one implementation owner; load scoped instructions/skills on demand.
Delegation needs objective, paths, evidence and stop conditions; no speculative
fan-out, unapproved nesting or silent expensive-model fallback. Use one approved
overall time/credit budget and meaningful checkpoints, not per-read permission.
Stop at exhaustion, scope expansion, conflicting authority or newly required
operational actions. Preserve unrelated changes; never stage/stash/publish as a
handoff side effect.

### 18.3 Review restrictions

Required review stays independent and read-only through `task review:read-only`
and its native tool exclusions; no shell/edit/delegation or capability workaround.
Authors may repair new rejections unless the reviewer requires an independent
reviser. Only the blocker or an owner-appointed independent successor after
re-review clears a block. Historical blocks, author restrictions and clearance
sequences retain their original terms. Reassignment, green CI and new policy
are not clearance.

### 18.4 Evidence and archival

Keep scope, criteria, result links, exact tested/reviewed identity, exceptions,
blocks and next action in canonical work. Current-work state only links there.
No mandatory duplicate inbox/handoff/agent-history ledger. Never invent facts,
silently expand scope, store secrets, or use empty responses as completion.
Current views may be curated only when authorized after append-only preservation
with source inventory/checksums; never archive away unresolved evidence.
Accepted ADR bodies and historical session/orchestration logs are immutable.
No `SESSION-NOTES.md` or `.copilot-state.md` alternative source.

### 18.5 Context and checkpoints

Load only relevant files/roles; verify paths and source before using recipes.
Checkpoint to canonical work within the approved budget, preserving pending work.
Context warning budgets remain: universal instructions 150 lines, active
decisions 20 KiB, per-role history 12 KiB, pointer 100 lines, initial policy
context 6,000 words. Consolidate or explain exceptions; never truncate safeguards.

## §19. Documentation Requirements

Keep affected instructions, contracts, architecture/security/contributor docs and
templates aligned with actual changes; do not eagerly rewrite unused recipes.
Use `docs/adr/README.md` for ADR lifecycle and `docs/testing.md` for commands.
Specs/plans/tasks are optional scoped planning aids, not a universal document set.
An index mirrors source status; code/checkboxes do not accept a Proposed ADR.

## §20. Audit & Continuous Improvement

Broad software audits are for major architectural changes or owner-designated
major releases, not every beta-to-main PR. High-risk work retains applicable
security/recovery proof without duplicate feature review/audit ceremonies.
Existing required audits/blocks remain required until explicitly resolved.
After owner-designated major releases, explicitly invoke the separate
agentic-delivery audit and track prior corrective actions; a skill is no trigger.
Keep per-release SBOM/threat-model review, quarterly PRD review and annual
dependency-major review/restore drill. Investigate concrete drift/recurring
failures, not weekly broad ceremonies. Audits are read-only absent write authority;
findings close with corrective evidence, never by retrospective or release alone.

## §21. Definition of Done

Use the issue/PR to answer: what changed, is scope complete, what could break,
what evidence passed, what review was required, what remains blocked/unauthorized.
Include criteria/sibling workflows, applicable Principles/gates, exact candidate
and review identities, evidence links, exceptions and actual owner decisions.
Do not duplicate an eighteen-item checklist.

**Implemented** means authored; **verified** means applicable proof exists;
**accepted** means required review blocks cleared and owner approval exists;
**released** means the authorized candidate was published/promoted. Deployment
is separate. Check secrets and commit hygiene before acceptance. CI, beta pushes
and agent-authored checkboxes cannot supply owner approval or release authority.

## §22. Amendment Process

1. Propose a bounded ADR with status **Proposed**, rationale, version impact,
   affected consumers, acceptance route and rollback.
2. Obtain explicit owner authority for locked local edits and the PR route.
   Preparation, review/checks, commit/push/PR creation, merge/acceptance,
   installation, live settings and release/deployment are distinct permissions.
3. Submit the constitution change PR with its linked ADR and synchronized active
   consumers. Prepare the version header and Sync Impact Report: MAJOR for removed/
   renumbered Principles, incompatible governance or operational restructuring;
   MINOR for added Principles/sections or material expansion; PATCH for clarification.
4. Update the Sync Impact Report and append a **proposed** revision row. Retain
   effective baseline and pending activation visibly. Do not advance ratification/
   last-amended dates before acceptance. Unexecuted automation/checks stay pending.
5. After required review, owner approval and authorized merge, announce in
   `.squad/decisions.md`, transition the ADR/index to Accepted, and record the
   effective version, last-amended date and merge evidence. Remove pending banners
   only as part of that lifecycle action. Existing restrictions are not self-cleared.

Deviations require an explicit documented waiver under this process, justified in
canonical work (and a selected plan's Complexity Tracking table when used).
Automation is preferred, never an excuse to silence a guard by changing authority.

## §23. Revision History

| Version | Date | Author | Summary | ADR |
|---------|------|--------|---------|-----|
| 1.0.0 | 2026-04-28 | Brian | Initial 10-principle constitution covering layered architecture, DI, service boundaries, typing, design tokens, AI isolation, schemas, commits, UI/UX, and architecture enforcement. | — |
| 1.1.0 | 2026-04-28 | Brian | Gap closure: added Principles XI–XVI (Security Hardening, Authentication & Token Policy, PWA/Mobile Rules, Social & Privacy, Supply Chain & CI, Account Lifecycle). | — |
| 2.0.0 | 2026-05-28 | Maximus (approved by Brian) | Added §0 Hierarchy of Authority, §17 Quality Gate, §18 AI Agent Operating Rules, §19 Documentation Requirements, §20 Audit & Continuous Improvement, §21 Definition of Done, §22 Amendment Process, §23 Revision History. All 16 Principles (I–XVI) preserved verbatim. | ADR 0001 (to be added in Phase 3) |
| 3.0.0 | 2026-06-09 | Brian | Consolidated 17 principles into 9 streamlined principles and made Simple Complete Changes Principle IV. | ADR 0005 |
| 3.1.0 | 2026-06-11 | Brian | Added workflow-contract, blast-radius, configurable-value, and exact regression coverage gates to reduce repeated user-flow regressions. | ADR 0006 |
| 4.0.0 | 2026-09-21 | Copilot; approved and merged by owner in PR #734 | Reconciles authority, proportional gates, current views, bounded agents, review restrictions, evidence-backed completion, and audit cadence. Accepted at ca85f7830349a4472cb5409c082b9485e7e34c44; historical review restrictions remain unchanged. | ADR 0019 |
| 5.0.0 (Proposed) | Pending acceptance | Copilot; local preparation authorized by owner 2026-10-19 | Condensed durable policy, proportional review, one canonical work record and bounded delivery authorization; not effective until §22 approval/merge. | ADR 0024 |

**Prepared version**: 5.0.0 | **Effective version**: 4.0.0 | **Ratified**: 2026-04-28 | **Last Amended**: 2026-09-21
