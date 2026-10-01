# Feature Specification: Voice-Assisted Coin Identification Notes

**Work ID / spec directory**: `364-voice-assisted-identification`
**Working branch**: `beta`
**Lane**: High risk (microphone permission, browser-provider audio processing, and mobile/PWA behavior)
**Created**: 2026-10-01  
**Status**: Draft; scope decisions are owner-approved, implementation is not yet authorized  
**Input**: User wants voice input to quickly enter evidence for coin identification. Owner-selected MVP: Quick Identify and Deep Analysis, browser-first transcription with a future self-hosted seam, editable notes with explicit submission, English only.

This specification is subordinate to Constitution 4.0.0, ADR 0019, Feature 348,
Feature 344 as amended by Feature 351, and accepted ADR 0018.

## Scope and Non-Goals

- **Approved outcome**: A collector can dictate identification observations into
  the existing optional notes fields for Quick Identify and Deep Analysis, review
  and edit the transcript, and explicitly start analysis using the same notes
  contracts already used by typed input.
- **Non-goals**:
  - Continuous listening, wake words, or a full duplex voice conversation.
  - Spoken Copilot responses or text-to-speech.
  - Automatic analysis submission when dictation ends.
  - AI extraction of structured coin fields directly from speech.
  - Audio recording persistence, audio uploads, or a new backend transcription
    endpoint in the MVP.
  - Voice entry for Add Coin, collection chat, saved notes, or unrelated forms.
  - Non-English recognition selection in the MVP.
  - Implementing the future self-hosted transcription fallback.
- **Stop conditions**:
  - Browser speech recognition cannot be introduced without a clear disclosure
    that audio processing is controlled by the browser/platform provider.
  - Any implementation starts microphone access without a direct user action.
  - Dictation requires changing the 2,000-character notes contract, Quick
    Identify request shape, or Deep Analysis request shape.
  - The implementation would auto-submit, persist audio, or silently replace
    existing typed notes.
  - An accepted ADR is required but not approved before implementation.
- **Authorizing evidence / ADRs**:
  - User input: "I like the idea around quickly entering data for coin identitifcations."
  - Owner choices on 2026-10-01: Quick Identify and Deep Analysis; phased
    browser-first architecture; editable notes with explicit submission; English.
  - ADR 0018 preserves optional collector notes as evidence for Deep Analysis.
  - A new ADR must document the browser speech-recognition privacy, compatibility,
    fallback, and no-audio-persistence decision before implementation.
- **Lifecycle evidence**: Planning only. No implementation, verification,
  acceptance, release, or deployment is authorized by this draft.

## User Scenarios & Testing

### User Story 1 - Dictate Quick Identify observations (Priority: P1)

A collector photographing a coin can dictate visible legends, measurements,
symbols, mint marks, provenance clues, or uncertainty into the existing
Identification Notes field without putting down the coin to type.

**Why this priority**: This is the highest-friction mobile capture moment and the
smallest independently useful voice-assisted workflow.

**Independent Test**: In both desktop and immersive PWA Identify Coin flows,
navigate to Notes, start dictation with a user click, provide a final transcript,
edit it, and run Quick Identify. Verify the existing notes payload receives the
edited text and analysis is never started by dictation alone.

**Acceptance Scenarios**:

1. **Given** speech recognition is supported and Identification Notes is empty,
   **When** the collector starts dictation and the browser returns final text,
   **Then** the transcript appears in the notes field and remains editable.
2. **Given** notes already contain typed text, **When** dictation returns final
   text, **Then** the transcript is appended with readable spacing and the typed
   text is preserved.
3. **Given** dictation has produced text, **When** recognition ends, **Then**
   Quick Identify does not start until the collector explicitly chooses Analyze
   Photos.
4. **Given** the collector edits or removes dictated text, **When** Quick Identify
   starts, **Then** only the final reviewed notes value is submitted.

---

### User Story 2 - Dictate Deep Analysis context (Priority: P1)

A collector starting Deep Analysis can dictate optional contextual evidence into
the existing notes field, including from new Identify Coin intake and from a
saved coin's Deep Analysis start panel.

**Why this priority**: ADR 0018 explicitly uses collector notes during separate
obverse/reverse analysis and hypothesis construction, making high-quality
dictated observations directly valuable.

**Independent Test**: Dictate and edit notes in the Identify Coin wizard before
opening Deep Analysis, and separately dictate notes in the saved-coin Deep
Analysis start panel. Verify the existing `notes` field is submitted unchanged
and the optional no-notes path still works.

**Acceptance Scenarios**:

1. **Given** a collector dictates notes in Identify Coin, **When** they choose
   Deep Analysis, **Then** the existing captured-evidence handoff reuses the
   reviewed notes without rendering a second dictation control.
2. **Given** a saved coin's Deep Analysis panel renders its own notes field,
   **When** the collector dictates and edits context, **Then** the submitted job
   uses the reviewed text through the existing request contract.
3. **Given** the collector provides no typed or dictated notes, **When** Deep
   Analysis starts with valid images, **Then** the job starts normally with
   empty optional notes.

---

### User Story 3 - Recover safely from unsupported or failed dictation (Priority: P1)

A collector can always continue by typing when speech recognition is unavailable,
permission is denied, no speech is detected, or recognition fails.

**Why this priority**: Voice is an enhancement to an established critical
workflow and must never block identification.

**Independent Test**: Exercise unsupported, permission-denied, aborted,
no-speech, and recognition-error cases. Verify the notes field, image capture,
Quick Identify, and Deep Analysis remain usable and no stale listening state
survives navigation.

**Acceptance Scenarios**:

1. **Given** the browser does not expose the supported recognition API, **When**
   Notes renders, **Then** typed notes remain fully usable and the UI does not
   present a broken microphone action.
2. **Given** microphone permission is denied or recognition fails, **When** the
   error is reported, **Then** the collector sees a concise actionable message,
   existing notes are preserved, and typing remains available.
3. **Given** the collector leaves the component while recognition is active,
   **When** it unmounts, **Then** recognition is aborted and no later callback
   mutates the new screen or shared state.
4. **Given** analysis is already submitting, **When** Notes renders, **Then**
   dictation cannot be started or left active.

---

### User Story 4 - Understand privacy and recording state (Priority: P2)

A collector can tell when the microphone is active and understands that the
browser/platform may process speech outside Aurearia before choosing to dictate.

**Why this priority**: Microphone access and browser-provider processing require
clear consent and trustworthy state, even though typed input remains available.

**Independent Test**: Inspect the notes controls with keyboard and screen-reader
semantics, start and stop dictation, and verify the visible/listed state and
privacy disclosure are available without relying on color alone.

**Acceptance Scenarios**:

1. **Given** dictation is available, **When** the microphone action is shown,
   **Then** its accessible name explains whether it will start or stop listening.
2. **Given** recognition is active, **When** the collector views the control,
   **Then** listening state is visible and announced without using color as the
   only indicator.
3. **Given** the collector has not started dictation, **When** they inspect the
   control's help text, **Then** Aurearia discloses that recognition may be
   processed by the browser/platform provider and that Aurearia does not retain
   audio in this MVP.

### Edge Cases

- Recognition returns multiple final result segments in one session.
- Interim results arrive before a final result; interim text must not be
  committed as submitted notes.
- A duplicate final callback is delivered after stop or unmount.
- Existing notes end in punctuation or whitespace when dictated text is appended.
- Appending the transcript would exceed the existing 2,000-character limit.
- Recognition ends naturally after silence before the collector presses Stop.
- The user presses Start repeatedly or presses Stop before recognition starts.
- Permission is granted once but later revoked by the browser.
- The component becomes disabled while recognition is active.
- Desktop and immersive PWA notes surfaces render at the same time during a
  responsive transition; only the user-activated instance may listen.

## Requirements

### Functional Requirements

- **FR-001**: The MVP MUST add voice dictation only to Quick Identify and Deep
  Analysis notes entry surfaces.
- **FR-002**: Dictation MUST populate the existing optional notes value and MUST
  NOT create a second voice-only data field.
- **FR-003**: Recognition MUST start only from a direct user action on the
  microphone control.
- **FR-004**: Recognition language MUST be fixed to English (`en-US`) in the MVP.
- **FR-005**: Final recognized text MUST remain editable before submission.
- **FR-006**: Dictation completion MUST NOT automatically start Quick Identify,
  Deep Analysis, or any Coin Copilot run.
- **FR-007**: Dictated text MUST append to, not replace, existing notes unless
  the collector edits the combined value.
- **FR-008**: Appended text MUST preserve readable spacing and MUST respect the
  existing 2,000-character limit.
- **FR-009**: If text is truncated to the existing limit, the UI MUST report that
  the transcript was shortened and MUST preserve the beginning of the combined
  reviewed notes value.
- **FR-010**: Interim recognition results MUST NOT be committed to the notes
  value; only final results may be appended.
- **FR-011**: The UI MUST expose idle, listening, stopping, and error states with
  accessible text and without relying on color alone.
- **FR-012**: The collector MUST be able to stop an active recognition session.
- **FR-013**: Active recognition MUST be aborted when its component unmounts,
  when the form becomes disabled/submitting, or when a sibling workflow replaces
  the notes surface.
- **FR-014**: Unsupported browsers MUST retain the complete typed-notes workflow
  and MUST NOT show an action that cannot work.
- **FR-015**: Permission denial, no-speech, aborted, network, and generic
  recognition errors MUST preserve existing notes and present bounded,
  user-legible feedback.
- **FR-016**: The UI MUST disclose that browser/platform speech services may
  process microphone audio and that Aurearia does not persist audio in the MVP.
- **FR-017**: The MVP MUST NOT add audio persistence, audio uploads, a database
  entity, an API route, Python-agent transcription, or a new dependency.
- **FR-018**: The shared implementation MUST support both desktop and immersive
  PWA Identify Coin notes surfaces without duplicating recognition lifecycle
  logic.
- **FR-019**: The saved-coin Deep Analysis notes field MUST reuse the same shared
  dictation control and lifecycle.
- **FR-020**: The captured-evidence Deep Analysis handoff MUST reuse notes
  dictated in Identify Coin and MUST NOT render a duplicate microphone control.
- **FR-021**: Existing typed notes, no-notes, image-role, price-estimate, Quick
  Identify, and Deep Analysis behavior MUST remain unchanged when dictation is
  unused.
- **FR-022**: The browser adapter MUST be isolated behind a small typed seam so a
  future self-hosted transcription implementation can replace recognition
  without changing the notes consumers.
- **FR-023**: Recognition state MUST remain component-scoped and MUST NOT leak
  across navigation, modal close/reopen, or concurrent component instances.
- **FR-024**: Production code and user-facing messages MUST not log or expose raw
  audio, recognition event objects, or unintended transcript content.

### Key Entities

No persisted entities are added.

- **Voice recognition session**: Ephemeral component-scoped state containing
  availability, listening/stopping status, final text callbacks, and bounded
  user-legible errors.
- **Identification notes**: The existing bounded string used by Quick Identify
  and Deep Analysis. Dictation is only another input method for this value.

## Success Criteria

### Measurable Outcomes

- **SC-001**: A collector can start dictation from the Quick Identify Notes step,
  receive final text, edit it, and explicitly submit analysis without typing.
- **SC-002**: The same reviewed dictated notes reach Quick Identify and the Deep
  Analysis captured-evidence handoff through their existing contracts.
- **SC-003**: Saved-coin Deep Analysis accepts dictated notes with no API or
  backend changes.
- **SC-004**: Unsupported and denied-permission paths leave all existing typed
  identification workflows usable.
- **SC-005**: Automated tests cover availability, user-initiated start, final
  transcript append, interim-result exclusion, 2,000-character handling, stop,
  errors, disabled state, and unmount cleanup.
- **SC-006**: Automated component tests cover desktop Identify Coin, immersive
  PWA Identify Coin, captured-evidence Deep Analysis, and saved-coin Deep
  Analysis.
- **SC-007**: A deliberate lifecycle or no-auto-submit guard break causes a
  focused test to fail before the implementation is accepted.
- **SC-008**: `task check:web` passes, and supported-runtime/mobile browser
  evidence is recorded for the exact candidate.

## Assumptions

- Typed notes remain the authoritative fallback and source of truth.
- The browser may implement speech recognition through a platform-managed
  remote service; Aurearia cannot promise on-device processing for this MVP.
- HTTPS or another browser-recognized secure context is available where
  microphone recognition is supported.
- English recognition is sufficient for the MVP even when dictated numismatic
  legends contain Latin, Greek, abbreviations, or proper names; collectors can
  correct transcription before submission.
- The future self-hosted fallback will require separate scope, provider/runtime
  selection, resource sizing, privacy review, and likely Go-to-Python contract
  work.
