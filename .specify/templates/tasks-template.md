---
description: "Bounded feature tasks with required verification and independent acceptance"
---

# Tasks: [FEATURE NAME]

**Work ID / spec directory**: `[###-feature-name]`
**Working branch**: `beta` unless explicitly approved otherwise
**Input**: [approved spec/plan paths]
**Lane**: [feature / high risk; small repairs may use a bounded issue instead]
**Implementation owner**: [owner]
**Non-goals / stop conditions**: [scope and approval boundaries]

This template is subordinate to constitution 4.0.0 / ADR 0019, accepted via
PR #734. Generate only tasks needed for the
approved outcome; do not copy illustrative setup or technology scaffolding.

**Tests are required where applicable**, not optional because the user omitted
the word "tests." Include exact-path regression, failure, contract, and sibling
workflow evidence. Record an explicit reason/approval for any manual exception.
Missing tools or permission are incomplete, not a passing task.

## Format: `[ID] [P?] [Story] Description`

- IDs are sequential; include verified paths and observable completion evidence.
- `[P]` means independent/non-conflicting work is possible, not mandatory fan-out.
- `[Story]` maps to acceptance criteria. Default to one implementation owner.
- Discover actual repository paths; do not invent `backend/` or `frontend/`.
- Keep implementation, verification, review, and release state distinct.

## Phase 1: Scope and Existing Foundations

- [ ] T001 Confirm approved criteria, non-goals, existing helpers, sibling paths,
  relevant ADRs, and unresolved review restrictions in [verified artifacts].
- [ ] T002 Record only genuine prerequisites; identify required authorization
  for setup, migrations, external resources, or delivery-control changes.

## Phase 2: User Story 1 - [Title] (Priority: P1)

**Goal**: [usable outcome and criterion IDs]
**Independent test**: [exact user path, expected output, failure path]

### Verification

- [ ] T003 [US1] Add regression/contract tests in [existing test location];
  prove a behavior guard fails when its protected behavior is broken.

### Implementation

- [ ] T004 [US1] Implement the smallest complete change in [verified paths],
  reusing [existing helpers] and preserving [sibling workflows].
- [ ] T005 [US1] Run relevant checks against the changed tree and demonstrate
  [acceptance criteria]; record actual results and unavailable evidence.

Repeat the story phase only for approved additional stories. Do not add generic
performance, refactoring, or security-hardening tasks outside the scope.

## Final Phase: Applicable Gates and Acceptance

- [ ] T006 Update directly affected docs/contracts and run applicable completion
  gates from `docs/testing.md` §6; preserve CI-only release evidence requirements.
- [ ] T007 Obtain independent review of the exact commit/tree and explicit
  clearance of any block from its reviewer; record remaining exceptions.
- [ ] T008 Reconcile task/lifecycle state and persist an authorized handoff.
  Publication and release remain separate owner-authorized actions.

## Dependencies and Execution

[List real dependencies, owner, checkpoint/effort lease, and stop conditions.
Do not initialize frameworks already present or serialize independent work
without reason. Do not parallelize writes to shared files.]

## Evidence

| Criterion / task | Evidence and command/result | Commit/tree | State / reviewer |
|------------------|-----------------------------|-------------|------------------|
| [ID] | [exact evidence or pending reason] | [identity] | [implemented/verified/accepted] |

No checkbox alone proves acceptance. Do not deploy, commit, push, install,
or create a branch as an implicit consequence of completing a phase.
