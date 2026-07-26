# Agentic Set Builder Correction Plan

**Date:** 2026-07-26  
**Branch:** beta  
**Status:** Draft for review  
**Source specs:** `spec - Dynamic Set Builder.md`, `spec - Generalize the Roman Emperor Tracker.md`  

## Goal

Replace the current deterministic Agentic set creation with the spec-required workflow:

Prompt -> Python multi-agent workflow -> persisted proposal -> notification -> human review -> approve/edit/reject/regenerate -> transactional set creation.

No set should be created before human approval.

## Phase 0 - Stop the Misleading Behavior

1. Disable the deterministic Agentic generator.
2. Change Agentic prompt submission so it starts or requests a proposal workflow, not a set.
3. If the Python agent service or AI provider is unavailable, fail visibly with an actionable error.
4. Remove or hide UI copy implying the current deterministic output is agentic.
5. Preserve safe read behavior for Agentic rows already created during beta testing.

**Acceptance:** Submitting an Agentic prompt no longer creates a set immediately.

## Phase 1 - Backend Data Model

Add persistent proposal and run records owned by the Go API.

### SetBuilderRun

- User ID
- Prompt
- Status
- Started/completed timestamps
- Provider/model metadata
- Transcript or summary
- Error/termination reason
- Budget and turn counters, when available

### SetProposal

- User ID
- Builder run ID
- Original prompt
- Status: `pending`, `approved`, `rejected`, `expired`, `creation_failed`
- Proposed name, slug/scope, description/color
- Selected scope and alternate scope options
- Roster payload
- Pre-match summary
- Idempotency key
- Expiration timestamp
- Approved/rejected timestamps

### ProposalSlot

- Label
- Criteria JSON
- Group
- Sort order
- Verification status: `verified`, `unverified`
- Source note or validation note

**Acceptance:** Proposals persist without creating sets or set targets.

## Phase 2 - Python Agent Workflow

Add a new Python agent team endpoint for set building.

### Workflow roles

1. **Intent Analyst**
   - Extracts numismatic subject, scope, ambiguity, and boundedness.
   - Rejects non-numismatic or unbounded prompts unless scope options can be proposed.
2. **Roster Researcher**
   - Enumerates candidate slots using available tools/provider search when needed.
   - Produces structured roster entries only.
3. **Collection Matcher**
   - Uses collection context passed from Go.
   - Estimates filled count and likely matches.
4. **Validator/Critic**
   - Checks roster completeness and flags uncertainty.
   - Marks unverifiable slots as `unverified`, not silently trusted.
5. **Orchestrator**
   - Maintains the task ledger.
   - Enforces max turns, budget, and wall-clock limits.
   - Produces structured final proposal JSON.

**Acceptance:** Backend logs and provider logs show real Python/AI calls for Agentic set creation.

## Phase 3 - Backend Agent Proxy and Proposal Service

Add Go service flow:

1. `POST /set-builder/runs`
   - Auth required.
   - Validates prompt.
   - Dedupes identical pending prompts.
   - Calls Python agent service.
   - Streams or records progress.
   - Persists `SetBuilderRun` and `SetProposal`.
   - Creates proposal-ready notification.
2. `GET /set-builder/proposals`
   - Lists the current user's proposals.
3. `GET /set-builder/proposals/:id`
   - Returns review payload, roster, transcript summary, and status.
4. `PUT /set-builder/proposals/:id`
   - Edits proposed metadata, slots, or scope while pending.
5. `POST /set-builder/proposals/:id/approve`
   - Transactionally creates the set and targets.
   - Idempotent.
   - Creates confirmation notification.
6. `POST /set-builder/proposals/:id/reject`
7. `POST /set-builder/proposals/:id/regenerate`
   - Runs workflow again with feedback.

**Acceptance:** Approval is the only code path that creates the actual set.

## Phase 4 - Human Review UI

Replace Agentic set creation behavior in the frontend.

1. Set creation wizard
   - Agentic prompt submission starts a builder run/proposal request.
   - Success message says the proposal is being prepared.
   - No immediate set detail navigation.
2. Notifications
   - Proposal-ready notification deep-links to review screen.
3. Proposal review screen
   - Shows proposed set name, description, and scope interpretation.
   - Shows alternate scopes when present.
   - Shows full roster grouped by decade/category/etc.
   - Shows filled/total preview.
   - Shows verification badges.
   - Shows transcript or summary.
   - Supports Approve, Edit then approve, Reject, and Regenerate with feedback.
4. Approval result
   - Routes to created set detail page.
   - Shows confirmation notification.

**Acceptance:** The user reviews and approves before a set appears under Sets.

## Phase 5 - Roman-Emperor-Style Matching Semantics

Agentic sets should behave like the Roman Emperor Tracker. Coins are not manually added to the set. The approved roster defines slots, and the system automatically matches owned collection coins to those slots from the user's collection.

Corrected requirements:

1. No manual add/remove membership for Agentic sets.
2. Approved proposal creates the roster/slots only.
3. Collection coins match into slots automatically based on slot criteria.
4. Matching recalculates when coins are created, edited, deleted, or restored.
5. Wishlist and sold coins do not count as filled slots by default.
6. If multiple owned coins match one slot, use deterministic selection.
7. If one owned coin matches multiple slots, assign it to only one slot in roster order.
8. The tray renders every slot:
   - matched slot = owned coin image
   - unmatched slot = placeholder/silhouette
9. Manual pin/override is a separate future feature unless explicitly added.

**Acceptance:** Collection changes update Agentic progress without manual add/remove.

## Phase 6 - Tests and Quality Gates

### Backend tests

1. Agentic prompt creates proposal, not set.
2. Python agent unavailable returns visible failure and creates no set.
3. Completed workflow creates pending proposal and notification.
4. Proposal review fetch is user-scoped.
5. Approval creates set and slots transactionally.
6. Approval is idempotent.
7. Reject creates no set.
8. Expired proposal cannot be approved without regeneration.
9. Unverified slots remain visible.
10. Legacy deterministic-created Agentic rows remain readable.

### Frontend tests

1. Agentic wizard submits builder run request.
2. Wizard does not navigate to created set immediately.
3. Notification link opens proposal review.
4. Review screen renders roster, scope, transcript, and verification.
5. Approve calls approval endpoint and then routes to set.
6. Reject does not create set.
7. Regenerate submits feedback.
8. Agentic set tray renders filled and placeholder slots.

### Integration/manual checks

1. AI provider logs show calls for prompt submission.
2. Python service logs show workflow execution.
3. In-app notification appears after proposal completion.
4. Set appears only after approval.

## Task Breakdown

### Backend

| ID | Task |
|---|---|
| B1 | Add `SetBuilderRun`, `SetProposal`, and `ProposalSlot` models and migrations |
| B2 | Add repositories for runs/proposals/slots with user-scoped access |
| B3 | Add proposal lifecycle service: create, list, get, edit, approve, reject, expire |
| B4 | Add transactional approval path that creates set and targets only after approval |
| B5 | Add idempotency handling for identical pending prompts and repeated approval |
| B6 | Add notification creation for proposal-ready, approval success, approval failure |
| B7 | Replace current Agentic create-set path with proposal-run creation |
| B8 | Add Python agent proxy client for set-builder workflow |
| B9 | Add API handlers/routes/OpenAPI docs |
| B10 | Add backend tests for lifecycle, approval, auth scoping, and failure states |
| B11 | Implement Agentic slot matching using the Roman Emperor Tracker pattern |
| B12 | Recalculate matching on coin create/update/delete and on set view |
| B13 | Enforce one coin per slot and one slot per coin deterministically |

### Python Agent

| ID | Task |
|---|---|
| P1 | Define Pydantic request/response schemas for set-builder workflow |
| P2 | Build orchestrator state graph/group-chat workflow |
| P3 | Implement intent analyst node |
| P4 | Implement roster researcher node |
| P5 | Implement collection matcher node |
| P6 | Implement validator/critic node |
| P7 | Add execution limits and structured failure output |
| P8 | Add streaming/progress events |
| P9 | Add Python tests for benchmark prompts and ambiguous prompts |

### Frontend

| ID | Task |
|---|---|
| F1 | Add API client methods/types for builder runs and proposals |
| F2 | Change Agentic wizard submit to start builder run/proposal |
| F3 | Add proposal review route/page |
| F4 | Add roster review UI with grouping, verification badges, and transcript summary |
| F5 | Add edit/remove slot interactions before approval |
| F6 | Add approve/reject/regenerate actions |
| F7 | Wire notification deep links to proposal review |
| F8 | Render matched coins and unmatched placeholders in the Agentic tray |
| F9 | Add frontend tests for wizard, review screen, actions, and tray slots |

### Cleanup and Migration

| ID | Task |
|---|---|
| C1 | Remove deterministic prompt parser or isolate it only as test fixture/mock data |
| C2 | Decide how to handle Agentic sets already created during beta testing |
| C3 | Update specs to rename Tracker wording to Agentic only where superseded |
| C4 | Regenerate OpenAPI docs |
| C5 | Run quality gate and push directly to beta during testing-cycle fixes |

## Suggested Execution Order

1. C1 + B7 first, to stop misleading behavior.
2. B1-B6 data/lifecycle foundation.
3. P1-P7 Python workflow.
4. B8-B10 Go integration and tests.
5. F1-F7 proposal UX.
6. F8 + B11-B13 slot/tray completion.
7. C2-C5 cleanup, docs, and validation.
