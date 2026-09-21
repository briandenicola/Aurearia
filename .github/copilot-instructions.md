# Aurearia: Universal Copilot Instructions

Aurearia is a self-hosted coin-collection PWA: Go API (`src/api`), Vue/TypeScript
SPA (`src/web`), and stateless Python agent service (`src/agent`).

## Authority and work selection

Read the [constitution](../.specify/memory/constitution.md), relevant
[active decisions](../.squad/decisions.md), and the explicitly selected
issue/spec/process plan before editing. Constitution 4.0.0 / ADR 0019 governs.
Authority is Constitution > PRD > applicable accepted ADRs > active spec > plan >
tasks > backlog > active decisions > agent judgment. Frameworks cannot override it.

Use [current work](../.squad/identity/now.md) as a pointer, not a competing ledger.
Never infer work from the newest spec or branch name. Ordinary work stays on
`beta`; branches/worktrees need applicable owner authorization. Load a role's
charter/history only when acting in that role. Preserve unrelated dirty files.

Choose the smallest complete lane under section 18.1. Delivery controls, auth,
migrations, data semantics and service boundaries require high-risk review.
Keep scope/non-goals, acceptance criteria and sibling workflows explicit.
Do not change locked policy, accepted ADR/spec bodies or archived evidence
without the required authority. Stop on conflicting authority or scope expansion.

## On-demand guidance

Path-scoped rules live in [instructions](instructions). Load only matching
`applyTo` guidance; if the client provides only its path/glob index, read the
matching file explicitly. Do not eagerly load every language/design recipe.
Reusable workflows live in [native skills](skills/README.md).
Legacy `.squad/skills` entries are compatibility links, not separate authority.
If native discovery is unavailable, read the relevant canonical file explicitly
and report that fallback; do not claim the client discovered it.

Preserve Go -> Python service boundaries, layered data access, typed contracts,
ownership/auth/privacy rules and PWA behavior from the constitution.
No emojis in UI text, prompts or AI responses. Use local design patterns/tokens.
Verify paths, symbols and actual source before relying on a recipe.

## Execution and review

Default to one implementation owner. Squad is optional for genuinely separable
work, not routine fan-out. Every delegation needs objective, authority, allowed
paths, non-goals, evidence, an approved effort/checkpoint lease and stop condition.
No nested delegation or silent expensive-model fallback. Use configured models.

Required review must be independent. The
[read-only reviewer](agents/aurearia-reviewer.agent.md) cannot implement fixes.
Use `task review:read-only`; plain profile selection alone does not exclude the
installed client's session SQL and skill-loading tools.
For new reviews, the author may repair unless an independent reviser is required;
only the blocking reviewer or an owner-appointed independent successor can clear
the block. Preserve historical author restrictions. Assignment, changed files,
empty responses and green CI are not review clearance or completion evidence.

## Verification and persistence

Use [testing section 6](../docs/testing.md#6-running-tests-locally-vs-ci) and
applicable shared `task check:*` targets. Fast feedback is not completion.
Missing tools or unauthorized checks remain incomplete. Setup/installations,
builds, containers and publishing require their applicable owner authorization.
Never install implicitly, skip missing lint, use `--quiet` to hide warnings,
or replace strict `vue-tsc --build` with `--noEmit`. OpenAPI generation writes files.
Runner-only race/security/browser/compatibility checks remain additional.

Bind results and reviewer verdicts to the exact tested commit/tree. Distinguish
implemented, verified, accepted and released. Main/release/deployment requires
separate explicit owner approval; a beta push is not release authorization.

The implementation owner persists completed/incomplete work, evidence, decisions,
blocks and next action in `.squad/log/` and the current-work pointer, then verifies
the write completed. Scribe is optional and may write only named artifacts.
Record cross-cutting decisions in `.squad/decisions/inbox/` and reconcile active
decisions without erasing history. Never use `SESSION-NOTES.md` or `.copilot-state.md`.
Do not automatically stage, stash, commit, push or deploy as a handoff side effect.

When authorized to commit, stage only approved paths, use Conventional Commits,
cite applicable Principles/sections, and include:

```text
Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>
```

Use the [PR checklist](pull_request_template.md) for sections 17/21.
Software QC and post-major-release delivery audits are separate; invoke the
appropriate audit explicitly under section 20. A skill is not a scheduled trigger.
