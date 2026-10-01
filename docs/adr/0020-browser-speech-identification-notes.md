# ADR 0020: Browser Speech Recognition for Identification Notes

Date: 2026-10-01
Status: Proposed

## Context

Feature 364 adds push-to-talk dictation to the existing optional notes fields
used by Quick Identify and Deep Analysis. The user selected a phased approach:
ship a small browser-first MVP and preserve a seam for a separately scoped
self-hosted transcription fallback.

Browser speech recognition is not a uniform local capability. Availability,
permission behavior, and audio processing vary by browser and platform, and a
browser vendor may process microphone audio through a managed remote service.
Aurearia cannot truthfully promise offline or on-device transcription when it
uses this API.

The alternative production-grade design would record bounded audio in the web
client, upload it through an authenticated Go endpoint, and forward it to an
approved stateless speech-to-text runtime. That approach would add an audio
contract, upload limits and formats, provider/runtime selection, resource
requirements, privacy and retention controls, cancellation, security review,
and cross-service tests. It is disproportionate for validating whether
collectors find voice-assisted identification useful.

The existing workflows already accept bounded optional collector notes:

- Feature 348 sends up to 2,000 characters of untrusted notes to Quick Identify
  and retains them in the saved draft.
- ADR 0018 supplies optional collector notes to separate obverse and reverse
  analysis and the bounded Deep Analysis hypothesis stage.

Voice therefore does not need a new data model or analysis contract. It can be
an additional input method for the existing editable notes string.

## Decision

Feature 364 will use browser speech recognition for an English-only MVP with the
following boundaries:

1. Recognition starts only from a direct collector action on a microphone
   control in Quick Identify or Deep Analysis notes entry.
2. Only final recognition results are appended to the existing notes value.
   Interim results are never committed as submitted notes.
3. Dictated text remains editable and never automatically starts Quick Identify,
   Deep Analysis, or Coin Copilot.
4. Typed notes remain the authoritative fallback. Unsupported browsers do not
   show a broken microphone action, and recognition errors never block typing or
   image-based identification.
5. Aurearia does not record, upload, persist, replay, or log raw microphone audio
   in this MVP.
6. The UI discloses that the browser/platform provider may process microphone
   audio and that Aurearia stores only the reviewed transcript when the
   collector submits it as ordinary identification notes.
7. Recognition uses English (`en-US`). Selectable languages and numismatic
   language tuning are deferred.
8. Recognition lifecycle state is component-scoped. Active recognition is
   aborted on Stop, disable/submission, workflow replacement, and unmount.
9. The implementation uses a small repository-owned typed adapter around the
   standard or prefixed browser constructor. It adds no third-party dependency,
   global singleton, Pinia store, storage entry, backend endpoint, or Python
   service behavior.
10. A future self-hosted speech-to-text fallback is a separate feature requiring
    its own provider/runtime decision, privacy review, contracts, resource
    sizing, limits, and verification. The MVP seam must not pretend that fallback
    exists.

## Consequences

- The MVP is small and reuses the exact notes contracts already covered by Quick
  Identify and Deep Analysis.
- Browser compatibility is intentionally uneven. Voice is an enhancement, never
  a prerequisite for identification.
- Some browsers may send audio to a platform-managed service outside Aurearia's
  deployment. The product must disclose this before use and must not describe
  the MVP as self-hosted speech recognition.
- Automated tests can deterministically validate lifecycle and transcript
  behavior with an injected adapter, but they cannot prove a real browser
  provider's permission or network behavior. One explicitly authorized manual
  supported-browser check remains required.
- No migration or persisted-audio rollback is needed. Removing the control
  restores the unchanged typed notes workflow; submitted transcripts remain
  ordinary collector notes.
- If browser support or collector value is inadequate, the feature can be
  removed without changing Go, Python, database, or API contracts.

## Related

- Feature 364: `specs/364-voice-assisted-identification/`
- Feature 348: `specs/348-identify-coin-wizard/spec.md`
- Feature 344: `specs/344-deep-agentic-coin-identification/spec.md`
- Feature 351: `specs/351-vision-first-deep-identification/spec.md`
- ADR 0018: `docs/adr/0018-role-specific-deep-analysis.md`
- ADR 0019: `docs/adr/0019-evidence-based-agentic-delivery.md`
