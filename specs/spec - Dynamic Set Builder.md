# Feature Specification: Dynamic Set Builder (Agentic Tracker Set Creation)

**Feature Branch**: `011-dynamic-set-builder`
**Created**: 2026-07-26
**Status**: Draft
**Input**: User description: "From a natural language prompt (e.g., 'all American wheat pennies'), a multi-agent workflow (group chat / magentic orchestration pattern) determines user intent, researches and enumerates the complete roster of coins required for the set, and produces a plan and suggestion. After human-in-the-loop approval — delivered via the in-app notifications feature — the system creates all set components so a link appears under Sets and every goal coin is ready for display in the set's museum tray with silhouette placeholders."

**Depends on**: `010-tracker-sets` (the Tracker Set primitive is the creation target).

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Prompt to proposal (Priority: P1)

A collector types "all American wheat pennies" into the Dynamic Set Builder. A multi-agent workflow streams its progress (intent analysis, roster research, collection matching, validation) in real time, in the same streaming style as existing agent teams. It produces a proposal: suggested set name, slug, scope interpretation, the full slot roster with labels and match criteria, grouping (e.g., by decade), and a preview of how many slots the user's current collection already fills.

**Why this priority**: The proposal is the product; without a trustworthy, reviewable roster the rest of the flow has nothing to approve.

**Independent Test**: Submit a prompt with a well-known finite answer (e.g., "US state quarters"), and verify the proposal contains a complete, correctly labeled roster with valid criteria and an accurate pre-match count — without any set being created.

**Acceptance Scenarios**:

1. **Given** the builder input, **When** the user submits "all American wheat pennies", **Then** the workflow streams status events and produces a persisted proposal containing name, roster, grouping, and pre-match count, and no set is created.
2. **Given** a prompt with multiple reasonable scopes (date set vs. date+mint vs. varieties), **When** the workflow completes, **Then** the proposal presents the scope options with slot counts and a recommended default, and the user can select a scope before approving.
3. **Given** an ambiguous or unanswerable prompt (e.g., "cool coins"), **When** the workflow runs, **Then** it returns a clarification request instead of a fabricated roster.
4. **Given** the workflow exceeds its turn or budget limit, **Then** it terminates gracefully with a partial-result or failure status visible to the user; no proposal is silently dropped.

---

### User Story 2 - Notification-driven review and approval (Priority: P1)

When a proposal is ready, the collector receives an in-app notification: "Your set 'Lincoln Wheat Cents' is ready for review." The notification links to a review screen showing the full roster (scrollable checklist of slots), scope choice, matched-coin preview, and actions: Approve, Edit then approve, Reject, or Regenerate with feedback. Nothing is created until Approve.

**Why this priority**: Human-in-the-loop approval is an explicit product requirement; the agent must never create sets unilaterally.

**Independent Test**: With a seeded pending proposal, verify the notification appears in the inbox, the review screen renders the roster, and each action transitions the proposal to the correct terminal state.

**Acceptance Scenarios**:

1. **Given** a completed proposal, **When** it is persisted, **Then** an in-app notification is created for the owner linking to the review screen.
2. **Given** the review screen, **When** the user edits the proposal (rename, remove a slot, change grouping) and approves, **Then** creation uses the edited version.
3. **Given** the review screen, **When** the user rejects, **Then** the proposal is marked rejected, nothing is created, and the transcript remains viewable.
4. **Given** a pending proposal older than its expiry window, **When** the user opens it, **Then** it is marked expired and offers one-click regeneration.
5. **Given** a pending proposal, **When** the same user re-submits an identical prompt, **Then** the system surfaces the existing pending proposal instead of running a duplicate workflow.

---

### User Story 3 - Approval creates everything, deterministically (Priority: P1)

On Approve, the backend — not the agent — creates the Tracker Set transactionally: the set record, all slots with criteria and silhouette placeholder styles, the Sets-menu link, and an initial matching pass against the collection. A confirmation notification reports the result: "Lincoln Wheat Cents created — 12 of 145 filled." The user opens the Sets menu, taps the new link, and sees the full museum tray with 12 coins and 133 silhouettes.

**Why this priority**: This is the promised end state ("link appears under Sets, all goal coins ready in the tray with placeholders") and must be atomic — no half-created sets.

**Independent Test**: Approve a seeded proposal and verify in one transaction-equivalent outcome: set exists, slot count matches proposal, nav link renders, tray shows correct filled/placeholder split, and confirmation notification exists. Force a mid-creation failure and verify zero artifacts remain.

**Acceptance Scenarios**:

1. **Given** an approved proposal with 145 slots, **When** creation runs, **Then** a Tracker Set with 145 slots exists, appears in the Sets nav, the tray renders all slots, and the confirmation notification states the filled count.
2. **Given** a failure during creation (validation, storage), **When** the transaction aborts, **Then** no set, slots, or nav entry exist, the proposal returns to pending with an error note, and the user is notified of the failure.
3. **Given** an approved proposal, **When** the approval action is retried (double-tap, network retry), **Then** exactly one set is created (idempotent approval).

---

### User Story 4 - Trust, transparency, and validation (Priority: P2)

The collector can inspect why the roster looks the way it does: the review screen exposes the workflow transcript/summary and marks any slot whose criteria could not be validated (e.g., a catalog reference the validator couldn't confirm) as "unverified" so the user can remove or keep it knowingly.

**Why this priority**: Roster hallucination is the main failure mode of this feature; surfacing uncertainty preserves trust without blocking usefulness.

**Independent Test**: Feed the validator a proposal containing one fabricated entry and verify it is flagged "unverified" on the review screen rather than silently included or dropped.

**Acceptance Scenarios**:

1. **Given** a proposal, **When** viewed on the review screen, **Then** an expandable transcript/summary of the agent workflow is available.
2. **Given** a slot the validator could not confirm against known references, **When** the roster renders for review, **Then** that slot carries an "unverified" badge and can be individually removed before approval.
3. **Given** slot criteria referencing an Era or Category value not present in the admin-configured option lists, **When** validation runs, **Then** the proposal maps to an existing value or flags the slot; it never invents new admin option values.

---

### Edge Cases

- Prompt describes an unbounded set ("all Roman coins"): workflow must respond with scope options or a clarification, never an arbitrary truncated roster presented as complete.
- Roster exceeds the Tracker Set slot ceiling: proposal offers a narrowed scope or split into multiple sets; approval of an over-limit roster is blocked.
- Proposed slug collides with an existing set: system generates a unique slug and shows it in the proposal.
- The agent service is unreachable or the AI provider is unconfigured: builder fails fast with an actionable message; no phantom pending proposals.
- User deletes coins between proposal and approval: filled-count preview may be stale; creation-time matching is authoritative and the confirmation reflects actual counts.
- Concurrent proposals: multiple pending proposals are allowed across different prompts; identical prompts dedupe (US2 scenario 5).
- Non-numismatic or abusive prompt: intent analysis declines and returns a polite unsupported-request result.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST accept a free-text prompt and run a multi-agent workflow using the group-chat/magentic orchestration pattern: an orchestrator maintaining a task ledger delegating to specialist roles — intent analysis, roster research/enumeration, collection matching, and validation/critique.
- **FR-002**: The workflow MUST stream progress status to the user in real time, consistent with existing agent teams' streaming behavior.
- **FR-003**: The workflow output MUST be a structured Set Proposal (data only): set metadata, scope interpretation(s), complete slot roster with labels/criteria/grouping/order, per-slot verification status, and a pre-match summary. The agent workflow MUST NOT create or modify any sets, slots, coins, or navigation directly.
- **FR-004**: The roster researcher MAY use external reference sources available to the agent service to enumerate rosters; every externally sourced slot MUST pass through validation before inclusion, and unconfirmed slots MUST be marked unverified (never silently dropped or silently trusted).
- **FR-005**: Proposals MUST be persisted with a lifecycle: pending → approved | rejected | expired, plus a failed-creation note state; each proposal stores the originating prompt, chosen scope, transcript reference, and an idempotency key.
- **FR-006**: Proposal readiness, creation success, and creation failure MUST each generate an in-app notification via the existing notifications feature; the readiness notification MUST deep-link to the review screen.
- **FR-007**: The review screen MUST support approve, edit-then-approve (rename, remove/edit slots, change grouping/scope), reject, and regenerate-with-feedback.
- **FR-008**: Approval MUST create the Tracker Set through the standard backend set services in a single transaction: set record, all slots (with v1 silhouette placeholder styles per spec 010), navigation registration, and an initial matching pass. Partial creation MUST be impossible.
- **FR-009**: Approval MUST be idempotent per proposal; a proposal can be approved at most once.
- **FR-010**: Proposals MUST expire after an admin-configurable window (default 7 days) and identical pending prompts per user MUST dedupe to the existing proposal.
- **FR-011**: The workflow MUST enforce configurable execution limits (maximum orchestrator turns, per-request token/cost budget, wall-clock timeout) and report limit-triggered termination in the proposal status.
- **FR-012**: All proposal and builder endpoints MUST require authentication and be scoped to the owning user; proposals are never visible cross-user.
- **FR-013**: Scope ambiguity MUST be resolved by presenting enumerated scope options with slot counts and a recommendation; the workflow MUST NOT pick a scope silently when materially different interpretations exist.
- **FR-014**: Slot criteria in proposals MUST conform to the Tracker Set criteria vocabulary and admin-configured Era/Category option lists (spec 010); non-conforming proposals fail validation with actionable errors.
- **FR-015**: The workflow transcript (or a faithful summary) MUST be retained with the proposal and viewable from the review screen for the lifetime of the proposal record.
- **FR-016**: Admins MUST be able to configure: proposal expiry window, workflow execution limits, and whether external reference lookups are enabled for roster research.

### Key Entities

- **SetProposal**: Persisted output of the workflow — owner, prompt, status, scope options and selected scope, proposed set metadata, roster payload, verification flags, pre-match summary, idempotency key, expiry, transcript reference, timestamps.
- **ProposalSlot**: One proposed roster entry — label, criteria, group, order, verification status (verified/unverified), source note. Becomes a TrackerSlot (spec 010) on approval.
- **BuilderRun**: The workflow execution record — status stream, turn/budget usage, termination reason; referenced by the proposal for transparency.
- **Notification (existing)**: Reused for proposal-ready, creation-success, and creation-failure events with deep links.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: For benchmark prompts with known finite rosters (wheat cents by date+mint, state quarters, Twelve Caesars), proposals achieve ≥95% roster completeness and ≤2% fabricated entries, with all fabrications flagged unverified.
- **SC-002**: Median prompt-to-notification time is under 3 minutes; the user sees streaming status within 5 seconds of submission.
- **SC-003**: 100% of set creations occur only after explicit user approval; zero sets created by agent action in testing.
- **SC-004**: Approval-to-usable outcome ("link in Sets menu + fully rendered tray with placeholders") completes in under 15 seconds for rosters up to the slot ceiling, and forced mid-creation failures leave zero partial artifacts across the test suite.
- **SC-005**: A first-time user can go from prompt to created set with no documentation, using only the notification and review screen (validated in a hallway/usability pass).

## Assumptions

- Spec `010-tracker-sets` is implemented: Tracker Set type, slots, silhouette placeholders, nav registration, and matching are available as the creation target.
- The existing stateless agent service and its per-request configuration/streaming model host the new workflow as an additional team; no persistent state lives in the agent service — all proposal persistence is owned by the backend.
- The existing in-app notifications feature supports deep links to app routes (or will be minimally extended to do so).
- The proposal-approval pattern follows the two-phase-commit approach already established for external tool writes.
- v1 placeholders are the bundled silhouettes from spec 010; per-slot reference imagery, pricing enrichment, and wish-list generation from missing slots are out of scope for v1 (natural v2 candidates).
- Roster research quality depends on the configured AI provider; benchmark success criteria (SC-001) are evaluated against the recommended provider configuration.
