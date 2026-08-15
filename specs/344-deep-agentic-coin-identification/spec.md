# Feature Specification: Deep Agentic Coin Identification

**Feature Branch**: `344-deep-agentic-coin-identification`
**Created**: 2026-08-15
**Status**: Draft
**Input**: User description: "Deep Agentic Coin Identification: an explicit optional
deep-analysis path alongside the existing fast Identify Coin flow, using a
persisted resumable background job that fans out to multiple numismatic
authority providers (Nomisma, Numista, OCRE, NGC, RPC Online) with bounded
parallelism, contradiction/provenance evaluation, and a cited synthesized
report plus editable, confirm-gated draft. Supports ephemeral hint images,
cancel/retry, SSE reconnect/replay, and provider routing/override."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Start Deep Analysis from new Identify Coin intake (Priority: P1)

A collector uploads obverse and reverse photos of an unidentified coin (the
existing fast Identify Coin flow) and, instead of accepting the quick result,
explicitly opts into a slower, more thorough Deep Analysis to get a richer,
multi-source attribution before deciding whether to add the coin.

**Why this priority**: This is the primary new entry point and the smallest
slice that proves the deep pipeline works end-to-end without touching the
existing fast path at all.

**Independent Test**: From new-coin intake, upload required obverse and
reverse images, optionally add text notes and separate hint/reference images,
choose "Deep Analysis" instead of (or in addition to) the fast lookup, and
verify a background job starts, the fast lookup result (if requested) is
unaffected, and the user can navigate away and later find the job and its
result.

**Acceptance Scenarios**:

1. **Given** a user on new-coin intake, **When** they upload obverse and
   reverse images and start the fast Identify Coin lookup, **Then** the
   existing synchronous result returns exactly as it does today, with no
   Deep Analysis job created and no added latency.
2. **Given** a user on new-coin intake, **When** they explicitly start Deep
   Analysis with required obverse and reverse images, **Then** the system
   accepts the request, creates a persisted background job, and returns
   immediately with a job identifier the UI can use to track progress.
3. **Given** a user has provided optional text notes and/or optional hint
   images, **When** Deep Analysis runs, **Then** those notes and hint images
   are used only as analysis context and are never treated as coin-face
   evidence.
4. **Given** a user attempts to start Deep Analysis without both an obverse
   and a reverse image supplied (directly or via an existing saved coin),
   **Then** the system rejects the request with a clear, specific validation
   message and creates no job.

---

### User Story 2 - Start Deep Analysis from an existing saved coin (Priority: P1)

A collector viewing an already-saved coin wants a deeper, multi-source
re-identification using the coin's existing photos, without re-uploading
anything, and without the analysis silently changing their saved record.

**Why this priority**: Re-analysis of owned coins is a first-class use case
distinct from intake and must be independently valuable and independently
testable; it also proves the "review, don't auto-write" contract that makes
the whole feature safe to ship.

**Independent Test**: From a saved coin detail view that already has obverse
and reverse images, start Deep Analysis with no new uploads, optionally add
text notes or hint images, and verify the job runs against the coin's
existing images and the coin record is completely unchanged until the user
explicitly reviews and applies proposed changes.

**Acceptance Scenarios**:

1. **Given** a saved coin with existing obverse and reverse images, **When**
   its owner starts Deep Analysis without uploading new photos, **Then** the
   job uses the coin's existing saved images to satisfy the required
   obverse/reverse inputs.
2. **Given** a saved coin missing one of obverse or reverse images, **When**
   its owner starts Deep Analysis, **Then** the system requires the missing
   image to be supplied before the job can start and does not silently
   substitute a hint image or absent image for a required coin-face image.
3. **Given** a saved coin owned by another user, **When** any other user or
   an unauthenticated caller attempts to start or view Deep Analysis for that
   coin, **Then** the system rejects the request as not authorized and no job
   or data is created or exposed.
4. **Given** a completed Deep Analysis on a saved coin proposes changed
   field values, **When** the owner reviews the draft, **Then** the coin's
   currently saved fields remain exactly as they were until the owner
   explicitly confirms an update through the existing coin-edit write path.

---

### User Story 3 - Observe progress, reconnect, cancel, and retry (Priority: P1)

While Deep Analysis runs, a collector wants to watch progress in real time,
safely navigate away or lose connectivity without losing the job, cancel a
run they no longer want, and retry a failed or unsatisfying run without
starting completely from scratch.

**Why this priority**: Because the job is a persisted background process
that can outlive the request/connection that started it, resumable
observability and explicit run control are foundational to trusting the
feature; without this, users cannot safely use a multi-minute analysis.

**Independent Test**: Start a Deep Analysis job, observe streamed progress
events, disconnect and reconnect the client mid-run and verify events resume
without duplication or gaps, cancel a running job and verify it stops
promptly and cleanly, and retry a completed or failed job and verify a new
attempt is created and linked to the original.

**Acceptance Scenarios**:

1. **Given** a running Deep Analysis job, **When** the client subscribes to
   its event stream, **Then** it receives ordered progress and
   partial-provider-result events as they occur.
2. **Given** a client disconnects mid-run and reconnects (same session, later
   time, or different device for the same owner), **When** it resubscribes
   using the last event it saw, **Then** it receives exactly the events it
   missed, in order, with no duplicates and no gaps, followed by any new
   events as they occur.
3. **Given** a running job, **When** the owner requests cancellation,
   **Then** the job stops issuing new provider work promptly, is marked
   cancelled, and any in-flight provider calls are abandoned or safely
   discarded rather than being reflected in a final report.
4. **Given** a cancellation request arrives at nearly the same moment the job
   reaches natural completion, **When** the system resolves the race,
   **Then** the job settles into exactly one terminal state (cancelled or
   completed) — never both, and never left ambiguous — and the UI is told
   which one occurred.
5. **Given** a completed, failed, or cancelled job, **When** the owner
   requests a retry, **Then** the system starts a new attempt that is linked
   to the original job (retry lineage), reusing the same inputs unless the
   owner supplies new ones, without deleting or overwriting the prior
   attempt's history.
6. **Given** a user submits a second Deep Analysis start request for the
   same coin/inputs while an equivalent job is already active, **When** the
   system processes it, **Then** it does not create a duplicate concurrent
   job; it returns the existing in-flight job instead.
7. **Given** the backing service process restarts while jobs are queued or
   running, **When** it comes back up, **Then** in-flight jobs are either
   safely resumed or transitioned to a clearly reported failed/stale state —
   never left silently stuck as "running" forever.
8. **Given** a job has been sitting completed, failed, or cancelled beyond
   its retention window, **When** a client tries to reconnect to or replay
   its event stream, **Then** the system reports that the job's live stream
   is no longer available while still allowing access to its final stored
   result (report/draft) for as long as that result itself is retained.

---

### User Story 4 - Review synthesized report and confirm-gated draft (Priority: P1)

Once Deep Analysis finishes (fully or partially), a collector wants a single
readable report with citations, plus a structured, editable draft of
proposed coin fields they can review, adjust, and explicitly accept — never
one that silently changes their data.

**Why this priority**: The synthesized, citable output is the actual value
delivered by the feature; without a trustworthy, editable, confirm-gated
result the background job is pointless.

**Independent Test**: Run Deep Analysis to completion (including a run with
partial provider failures) and verify the final result contains a narrative
report, per-field proposed values with confidence and citations, explicit
disagreements/unresolved questions, and provider coverage status; verify
edits to the draft are possible and that nothing is written to a coin without
an explicit confirm action.

**Acceptance Scenarios**:

1. **Given** a completed Deep Analysis job, **When** the owner opens the
   result, **Then** they see a narrative report, a structured set of
   proposed coin fields each carrying a confidence indicator and at least
   one source citation, and a list of any disagreements between sources or
   unresolved questions.
2. **Given** a completed result with proposed fields, **When** the owner
   edits a proposed value before accepting, **Then** their edit is preserved
   and clearly distinguished from the original AI-proposed value.
3. **Given** a reviewed and edited draft, **When** the owner explicitly
   confirms it, **Then** for new intake the confirmation flows through the
   existing Quick Capture/AI-intake promotion path to create the coin, and
   for a saved coin the confirmation flows through the existing coin-edit
   write path to update it — in neither case does Deep Analysis itself write
   directly to a coin record.
4. **Given** a completed result the owner has not yet confirmed, **When**
   they close the review screen and return later, **Then** the report and
   draft are still available unchanged, with no auto-created or auto-updated
   coin.
5. **Given** a job that completed with one or more providers unavailable or
   failed, **When** the owner views the result, **Then** the report and
   draft are still produced from whatever evidence succeeded, clearly marked
   as a partial-success result, rather than being withheld or presented as if
   nothing were missing.

---

### User Story 5 - Provider routing, override, and partial-success transparency (Priority: P2)

A collector wants the system to intelligently choose which numismatic
authorities are worth querying for their coin, while retaining the option to
adjust that choice before the run starts, and wants clear visibility into
which providers actually contributed, were skipped, or failed.

**Why this priority**: Bounded, relevant provider selection keeps runs fast
and cost-controlled, and transparency about provider participation is
required for the result to be trustworthy — but the feature is still usable
end-to-end (P1 stories) without user-facing override, so this is P2.

**Independent Test**: Start Deep Analysis and verify the system proposes a
relevant provider set based on an initial quick lookup and image evidence,
allow the owner to adjust that set before starting, run with one or more
providers deliberately unavailable/not-automated, and verify the final
report accurately reflects which providers ran, contributed, were skipped, or
failed, with no fabricated results standing in for unavailable ones.

**Acceptance Scenarios**:

1. **Given** initial quick-lookup context and image evidence for a coin,
   **When** the router proposes a provider set, **Then** the proposal is
   based on that evidence (e.g., era/region/authority signals) rather than
   always querying every provider.
2. **Given** a proposed provider set, **When** the owner reviews it before
   starting, **Then** they may deselect or add eligible providers, and the
   job runs only against the confirmed set.
3. **Given** a provider is not automatable in the current phase (e.g., gated
   behind manual reference or pending legal/API validation), **When** the
   router or owner selects it, **Then** the report clearly labels that
   provider's contribution as manual-reference/not-automated rather than
   showing it as queried-and-no-match or omitting it silently.
4. **Given** one provider fails or times out while others succeed, **When**
   the job completes, **Then** the report is produced from the successful
   providers, and the failed provider is listed with its status, never
   silently treated as "no match" or blended into the confidence of
   successful providers.
5. **Given** two providers return conflicting claims about the same field
   (e.g., mint or date range), **When** the report is synthesized, **Then**
   both claims and their sources are shown as a disagreement rather than one
   silently overriding the other or being dropped.

---

### User Story 6 - Ephemeral hint-image privacy and cleanup (Priority: P2)

A collector attaches reference or note photos (e.g., a photo of a dealer tag,
an auction listing, or a handwritten note) purely to help the analysis
understand context, and expects those images to never become part of their
permanent coin record and to be deleted once analysis is no longer using
them.

**Why this priority**: This is a privacy and data-minimization guarantee
that must hold even though the rest of the feature (P1 stories) can be
demonstrated without ever exercising it, so it is independently prioritized
and independently testable.

**Independent Test**: Start Deep Analysis with one or more hint/reference
images clearly separated from the required coin-face images, let the job
reach each terminal state (completed, cancelled, failed) in separate runs,
and verify in each case that the hint images are used only during the run
and are deleted afterward, never appearing among the coin's stored images or
in the final report as an attachment.

**Acceptance Scenarios**:

1. **Given** a Deep Analysis request with hint/reference images distinct
   from the required obverse/reverse images, **When** the request is
   submitted, **Then** the system stores the hint images only for the
   duration of the analysis and never associates them with the coin's
   permanent image set.
2. **Given** a job with hint images reaches completion, **When** cleanup
   runs, **Then** the hint images are deleted and are not retrievable
   through the job's stored result or any coin image endpoint.
3. **Given** a job with hint images is cancelled or fails, **When** it
   reaches that terminal state, **Then** the hint images are deleted exactly
   as they would be on successful completion — cleanup is not conditional on
   success.
4. **Given** a completed report that referenced information visible in a
   hint image, **When** the owner reads the report, **Then** the report may
   describe what was learned from that context but does not embed or
   link to the hint image file itself.

---

### Edge Cases

- What happens if a user submits obverse/reverse images that fail existing
  image-upload validation (disallowed type, bad magic bytes, over size
  limit)? → Rejected before any job is created, using the same validation
  behavior as existing image uploads; no partial job is created.
- What happens if a hint image is submitted but mislabeled as a coin-face
  image, or vice versa? → The system validates declared image role against
  the request structure (obverse/reverse/hint are distinct, required fields
  for obverse/reverse), and rejects a request that fails to supply both
  required coin-face roles even if the same number of files were uploaded
  under the wrong role.
- What happens if all selected providers are unavailable or not-automated?
  → The job still completes (not fails) using image-evidence and quick-lookup
  context alone, and the report clearly states that no external provider
  evidence was available, rather than fabricating a result.
- What happens if the owner deletes the saved coin while a Deep Analysis job
  for it is still running? → The job continues to completion or failure
  using the inputs it already captured, but its resulting draft can no
  longer be applied to a coin that no longer exists; the UI communicates
  this rather than silently discarding the report.
- What happens if a retried job's inputs (e.g., a saved coin's images) have
  changed since the original attempt? → The retry uses the current inputs at
  the time it starts, and the report/draft states which inputs (images,
  notes) were used for that specific attempt, preserving traceability across
  retry lineage.
- What happens if a client reconnects with an event ID/sequence number the
  server has already pruned as part of bounded retention? → The server
  reports that full replay is not possible and returns the current known
  state (latest status and any still-retained events) instead of an error
  with no information.
- What happens to a job accepted while Deep Analysis is disabled (feature
  gate later turned off mid-run)? → Already-running jobs are allowed to
  reach a terminal state; the feature gate only blocks new job starts.
- What happens if the fast Identify Coin path is called while a Deep
  Analysis job is in progress for the same session? → The fast path is
  fully independent, synchronous, and unaffected; it does not wait for,
  merge with, or get blocked by any Deep Analysis job.

## Requirements *(mandatory)*

### Functional Requirements

**Fast-path preservation**

- **FR-001**: System MUST leave the existing fast Identify Coin lookup
  (synchronous, image-in/result-out) completely unchanged in behavior,
  latency characteristics, and API contract; it remains the default action
  and MUST NOT be replaced, gated, or made to depend on Deep Analysis in any
  way.

**Starting Deep Analysis**

- **FR-002**: System MUST offer Deep Analysis as an explicit, separate,
  opt-in action from both new-coin intake and an existing saved coin's
  detail view.
- **FR-003**: System MUST require both an obverse and a reverse coin-face
  image to start Deep Analysis, accepting either newly uploaded images or,
  for a saved coin, its existing stored obverse/reverse images; it MUST
  reject a start request missing either required image with a specific
  validation message.
- **FR-004**: System MUST accept optional free-text notes and optional
  hint/reference images as separate, clearly distinguished inputs from the
  required coin-face images, and MUST NOT treat hint images as coin-face
  evidence.
- **FR-005**: System MUST validate uploaded images (obverse, reverse, and
  hint) against the same file-type allowlist, MIME/magic-byte, and body-size
  limits already enforced for other coin image uploads.
- **FR-006**: System MUST restrict starting or viewing Deep Analysis for a
  saved coin to that coin's owner; all Deep Analysis jobs, events, and
  stored images/results MUST be scoped to their owning user.
- **FR-007**: System MUST detect and reuse an existing in-flight job instead
  of creating a duplicate when an equivalent Deep Analysis start request is
  submitted again while one is already active, so duplicate submissions
  (e.g., accidental double-click, retried network request) do not create
  parallel redundant jobs.
- **FR-008**: System MUST allow Deep Analysis to be enabled/disabled
  independently of the fast Identify Coin path and independently of saved
  coin create/read/update/delete, such that disabling it never blocks or
  degrades those other flows.

**Background job execution and persistence**

- **FR-009**: System MUST run Deep Analysis as a persisted background job
  that continues to execute and record progress even if the initiating
  client disconnects, closes, or navigates away.
- **FR-010**: System MUST record job and event state (status transitions,
  progress, partial/provider results, terminal outcome) in durable storage
  owned by the system of record, independent of any single request/response
  cycle or streaming connection.
- **FR-011**: System MUST support the following job states at minimum:
  accepted/queued, running, partial, completed, failed, and cancelled, with
  well-defined, one-directional transitions into a single terminal state.
- **FR-012**: System MUST recover from a service process restart by either
  safely resuming in-flight jobs or transitioning them to a clearly reported
  failed/stale state; no job may remain indefinitely reported as "running"
  after the process that was running it is gone.
- **FR-013**: System MUST bound the total number of provider calls and the
  degree of parallel fan-out per job so that a run cannot grow unbounded in
  cost or duration regardless of how many providers are eligible.
- **FR-014**: System MUST target completing a Deep Analysis run (from
  accepted to a terminal state) within 5 minutes under normal conditions,
  and MUST enforce an upper bound so a run cannot hang indefinitely; a run
  that cannot finish within its bound MUST resolve to a terminal
  partial/failed state rather than continue silently.

**Streaming, reconnect, and control**

- **FR-015**: System MUST stream job progress and partial/provider results
  to the UI as they occur, using an event sequence that a client can resume
  from a last-seen position after a disconnect.
- **FR-016**: System MUST assign each job event a monotonically increasing
  sequence (or equivalent ordering identifier) so a reconnecting client can
  request "everything after event N" and receive exactly the missing events,
  in order, with no duplicates and no gaps.
- **FR-017**: System MUST retain job events for a bounded retention window
  sufficient to support reconnect/replay of an active or recently finished
  job, and MUST clearly report to a client when requested events have aged
  out of retention rather than silently returning nothing.
- **FR-018**: System MUST provide an explicit Cancel control for a job's
  owner that, when invoked on a running job, stops new provider work
  promptly and marks the job cancelled.
- **FR-019**: System MUST resolve a cancellation that races with natural
  completion into exactly one terminal state (never both) and MUST report
  that resolved state consistently to every client observing the job.
- **FR-020**: System MUST provide an explicit Retry control for a job that
  has reached completed, failed, or cancelled, which starts a new job
  attempt linked to the original (retry lineage) without deleting or
  overwriting the original job's stored history.
- **FR-021**: System MUST perform terminal cleanup (including ephemeral
  hint-image deletion, per FR-030) for every job reaching completed, failed,
  or cancelled, regardless of which terminal state was reached.

**Provider routing and evidence handling**

- **FR-022**: System MUST use an initial quick-lookup pass plus image
  evidence to automatically propose a relevant, bounded set of providers for
  a given job, rather than always querying every configured provider.
- **FR-023**: System MUST allow the owner to review and adjust (add or
  remove eligible providers from) the proposed provider set before the job
  starts running provider work.
- **FR-024**: System MUST treat each provider integration as non-substitutable
  for another: provider workers return typed claims/evidence/citations,
  status, and confidence rather than free-form prose, and no provider's
  result MAY silently override image evidence or another provider's claim
  without both being preserved and surfaced.
- **FR-025**: System MUST distinguish, per provider, between: contributed
  successfully, ran but found no match, failed/errored, timed out, and
  not-automated/manual-reference — and MUST NOT collapse any of these into
  a generic "no result" or fabricate a result to fill a gap.
- **FR-026**: System MUST allow the job to reach a terminal
  completed/partial state using only the providers that did contribute when
  one or more selected providers fail, time out, or are unavailable, and
  MUST clearly mark such an outcome as partial success.
- **FR-027**: System MUST evaluate contradictions and provenance across
  contributing sources (including image evidence) before synthesis, and
  MUST surface unresolved disagreements between sources rather than silently
  resolving them by precedence or discarding conflicting claims.

**Report, draft, and confirm-gated persistence**

- **FR-028**: System MUST produce, for every terminal completed/partial job,
  a single synthesized narrative report with source citations, a structured
  set of proposed coin field values, a confidence indicator per proposed
  field, and an explicit list of disagreements and/or unresolved questions.
- **FR-029**: System MUST report, alongside the synthesized result, which
  providers were selected, which contributed, which failed/were unavailable,
  and which were not-automated, so provider participation is transparent to
  the owner.
- **FR-030**: System MUST treat all hint/reference images as ephemeral:
  retained only for the duration of an active analysis, deleted upon that
  job reaching any terminal state (completed, cancelled, or failed), and
  never attached to, or retrievable as part of, the coin's permanent image
  set or the final stored report.
- **FR-031**: System MUST NOT automatically create or update a coin record
  from a Deep Analysis result; the proposed draft MUST require explicit
  owner review and confirmation before any coin data changes.
- **FR-032**: System MUST allow the owner to edit proposed draft field
  values prior to confirming, and MUST preserve the distinction between the
  original AI-proposed value and any owner edit.
- **FR-033**: System MUST route confirmed changes through existing,
  Go-owned write paths only: new-intake confirmation reuses the existing
  Quick Capture/AI-intake promotion path to create a coin, and saved-coin
  confirmation reuses the existing coin-edit write path to update a coin;
  the Deep Analysis pipeline itself MUST NOT perform direct coin writes.
- **FR-034**: System MUST retain a completed/failed/cancelled job's stored
  report and draft for the owner to review after the job's live event
  stream is no longer replayable, for at least as long as event retention
  allows reconnect, and MUST NOT delete a job's final result at the same
  time as its ephemeral hint images.

**Architecture boundaries and data protection**

- **FR-035**: System MUST keep all Deep Analysis job/event persistence,
  image storage, authorization, and confirmed-write logic in the Go API
  layer; the Python analysis pipeline MUST remain stateless with no direct
  database access, receiving all needed context per invocation.
- **FR-036**: System MUST NOT write user or provider note/query content
  (free-text notes, hint-image derived context, provider query terms) into
  application logs.
- **FR-037**: System MUST scope every job, event, image, and result strictly
  to its owning user; no cross-user access to another user's Deep Analysis
  data is permitted through any code path.

### Key Entities

- **Deep Identification Job**: A persisted, owner-scoped unit of work
  representing one Deep Analysis attempt. Tracks its optional linkage to a
  saved coin (absent for jobs started at new-coin intake before a coin
  exists), its declared inputs (obverse/reverse image references, optional
  notes, optional hint-image references), selected/confirmed provider set,
  current status (accepted/queued, running, partial, completed, failed,
  cancelled), retry lineage (link to the job it was retried from, if any),
  and timestamps for creation, state changes, and terminal completion.
- **Deep Identification Event**: An append-only, monotonically sequenced
  record belonging to a job, representing a progress update, a
  partial/provider result, a status transition, or the terminal outcome.
  Supports resumable replay from a last-seen sequence and is subject to
  bounded retention.
- **Provider Result (Claim)**: A typed, per-provider contribution to a job,
  carrying its source provider, a status (contributed, no-match,
  failed/errored, timed out, not-automated/manual-reference), one or more
  claims about coin attributes with citations, and a confidence indicator.
  Never a free-form prose blob.
- **Synthesized Report**: The single, terminal, per-job output combining
  narrative explanation, structured proposed coin fields (each with
  confidence and citations), and an explicit list of disagreements/unresolved
  questions, plus overall provider coverage/status.
- **Proposed Draft**: The editable, structured set of proposed coin field
  values derived from the Synthesized Report, distinguishing original
  AI-proposed values from any owner edits, and requiring explicit owner
  confirmation before being applied through an existing coin-write path.
- **Hint/Reference Image**: An ephemeral, job-scoped image (not a coin-face
  image) supplying analysis context such as tags, notes, or listings; exists
  only for the lifetime of its job's active analysis and is deleted upon
  that job's terminal state, never becoming part of a coin's permanent image
  set.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A user can start Deep Analysis from either new-coin intake or
  an existing saved coin and receive a job identifier immediately (well
  under the time it takes the analysis itself to run), confirming the
  request is accepted as a background job rather than blocking the UI.
- **SC-002**: 95% of Deep Analysis runs under normal conditions reach a
  terminal state (completed, partial, failed, or cancelled) within 5
  minutes of being accepted.
- **SC-003**: 100% of client reconnects to an active or recently finished
  job, made from the last event the client saw, receive every missed event
  exactly once, in order, with zero duplicated or dropped events, as long as
  those events are still within the retention window.
- **SC-004**: 100% of jobs reaching any terminal state (completed, failed,
  or cancelled) have their ephemeral hint images deleted and unretrievable
  afterward, verified across success, failure, and cancellation paths alike.
- **SC-005**: Zero coin records are created or modified by Deep Analysis
  without a recorded, explicit owner confirmation action — matching the
  same "no silent write" guarantee already proven for Nomisma linking and
  AI intake.
- **SC-006**: 100% of synthesized reports include at least one source
  citation for every proposed field that carries provider-derived evidence,
  and never state a provider result that provider did not actually return.
- **SC-007**: 100% of completed/partial results clearly and correctly report
  provider participation (contributed, no-match, failed, timed out, or
  not-automated) with no provider silently omitted from that accounting.
- **SC-008**: The existing fast Identify Coin lookup shows no measurable
  regression in response time or behavior after this feature ships, verified
  by before/after comparison.
- **SC-009**: Users starting Deep Analysis from a saved coin can review and
  either accept or decline every proposed field-level change individually,
  with 100% of accepted changes traceable to the specific report that
  proposed them.

## Non-Goals (Out of Scope)

- Replacing, slowing down, or gating the existing fast Identify Coin lookup
  behind Deep Analysis in any way.
- Scraping or reverse-engineering undocumented NGC or RPC Online endpoints;
  any integration beyond documented/authorized access remains out of scope
  until a documented API or agreement exists.
- Bulk or background ingestion of any provider's full dataset (Nomisma,
  Numista, OCRE, RPC, or NGC); all provider access remains on-demand,
  per-job, and scoped to the coin being analyzed.
- Any automatic coin creation or update — every persisted change requires
  explicit owner confirmation through an existing write path.
- Unbounded provider fan-out or unbounded run duration; the pipeline always
  operates under a fixed provider/parallelism bound and a completion-time
  ceiling.
- Persisting hint/reference images beyond the lifetime of their job, or
  attaching them to a coin's permanent image set under any circumstance.
- Making the Python analysis pipeline stateful or granting it direct
  database access; all persistence, authorization, and confirmed writes stay
  in the Go API.
- Delivering production-ready OCRE and RPC Online automated integrations in
  this feature's initial phase (see Assumptions — these remain staged,
  gated integrations).

## Assumptions

- **Provider phase boundaries (Phase 1 / MVP honesty)**: At initial ship,
  Nomisma and Numista are expected to be fully automatable via their
  existing clients; NGC is expected to be limited to reusing OCR
  cert-number extraction plus a canonical link-out (no undocumented live
  API call); OCRE and RPC Online are expected to remain gated behind
  separate license/API validation and may be typed as
  not-automated/manual-reference for some or all of the initial phase. The
  overall target architecture includes all five as staged provider
  integrations, added as each is legally/technically validated, without
  requiring a respecification of this feature's core job/event/report
  contracts.
- **Feature gating**: Deep Analysis ships behind an admin-controlled feature
  flag for staged rollout, consistent with existing admin-controlled AI
  provider/configuration patterns; when disabled, only new job starts are
  blocked — the fast path and existing coin CRUD are unaffected, and
  already-running jobs are allowed to finish.
- **Retention window**: "Bounded event retention" and "bounded caching" are
  implemented as a fixed, short-to-moderate window (long enough to cover
  ordinary reconnect scenarios, such as a closed laptop lid or a lost mobile
  connection) rather than indefinite storage; the final stored report/draft
  itself is retained separately and for longer than the live replayable
  event stream, per FR-034.
- **Ownership and visibility**: Deep Analysis jobs, events, images, and
  results are visible only to the owning user, consistent with the
  project's single-primary-user-plus-invited-friends model and existing
  per-coin privacy rules; invited friends do not see another user's Deep
  Analysis activity even for coins they can view read-only.
- **Reuse of existing write paths**: "Existing Quick Capture/AI intake
  promotion path" and "existing coin-edit write path" refer to the
  already-shipped, Go-owned coin creation/update flows; this feature adds a
  new upstream proposal source but does not introduce a second way to write
  a coin.
- **Bounded parallelism value**: The specific numeric limits for maximum
  concurrent provider calls and maximum total providers per job are
  implementation-tunable operational parameters, not user-facing product
  commitments; the product commitment is that such a bound always exists
  and is enforced (FR-013).

## Clarifications

No open [NEEDS CLARIFICATION] markers remain. The major product decisions
for this feature (fast-path preservation, deep-analysis entry points and
inputs, hint-image ephemerality, background job persistence with
resumable streaming, and confirm-gated draft persistence through existing
write paths) were supplied as binding decisions ahead of specification and
are reflected directly in the Requirements and Assumptions above.
