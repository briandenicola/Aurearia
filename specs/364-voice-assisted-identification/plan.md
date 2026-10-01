# Implementation Plan: Voice-Assisted Coin Identification Notes

**Work ID / spec directory**: `364-voice-assisted-identification`
**Working branch**: `beta`
**Date**: 2026-10-01 | **Spec**: `specs/364-voice-assisted-identification/spec.md`
**Lane**: High risk
**Owner**: One implementation owner | **Reviewer**: Owner-appointed independent reviewer

Subordinate to Constitution 4.0.0, ADR 0019, Feature 348, Feature 344 as
amended by Feature 351, and accepted ADR 0018.

## Summary and Scope

Add browser-first, push-to-talk dictation to the existing optional
identification notes fields used by Quick Identify and Deep Analysis. The
collector reviews and edits final recognized text before explicitly starting
analysis. Typed notes remain fully functional and authoritative.

The plan satisfies FR-001 through FR-024 and SC-001 through SC-008. It stops
before implementation if the browser-provider privacy/compatibility ADR is not
accepted. It does not add backend transcription, audio persistence, structured
field extraction, spoken responses, or voice input outside identification.

## Technical Context

- **Frontend**: Vue 3, TypeScript, Vite/PWA under `src/web`.
- **Existing notes contracts**:
  - `src/web/src/components/coin-lookup/CoinLookupCaptureWizard.vue`
  - `src/web/src/components/coin-lookup/PwaCaptureShell.vue`
  - `src/web/src/pages/CoinLookupPage.vue`
  - `src/web/src/components/deep-identification/DeepAnalysisStartPanel.vue`
- **Existing tests**:
  - `src/web/src/components/coin-lookup/__tests__/CoinLookupCaptureWizard.test.ts`
  - `src/web/src/pages/__tests__/CoinLookupPage.test.ts`
  - `src/web/src/components/deep-identification/__tests__/DeepAnalysisStartPanel.test.ts`
  - Add focused composable/component tests beside the new implementation.
- **Browser seam**: Use the browser speech-recognition API when available,
  including its prefixed form where required. Define repository-owned minimal
  TypeScript interfaces instead of adding an ambient package or casting through
  `any`.
- **State**: Component-scoped only. No module singleton, Pinia store, session
  storage, database state, or persisted audio.
- **Data flow**: Final transcript callback -> existing notes `v-model`/emit ->
  existing Quick Identify or Deep Analysis submission. No API contract changes.
- **Constraints**:
  - English `en-US`.
  - Existing 2,000-character notes bound.
  - User-initiated microphone start.
  - No auto-submit.
  - Browser/platform processing disclosure.
  - Existing image, camera, PWA, and optional-notes behavior preserved.

## Constitution Check

- **Principles**:
  - I/II/III: No service or data-boundary change; Go/Python remain untouched.
  - IV: One shared lifecycle and control, integrated into existing notes fields.
  - V: Clear unsupported/error behavior; typed fallback never hidden.
  - VI: No new dependency or implicit environment setup.
  - VII: Existing mobile/PWA interaction remains primary.
  - VIII: Microphone state and disclosure are accessible.
  - IX: No new supply-chain entry; future transcription provider is deferred.
- **Operational sections**: §17 completion evidence, §18 bounded high-risk lane,
  §20 software QC audit because microphone/privacy/PWA behavior changes, §21
  exact-tree DoD, §22 ADR for the material browser-provider/privacy decision.
- **Authorizing artifact**: Feature 364 spec with owner-selected MVP choices.
- **Affected workflows**:
  - Desktop Identify Coin Notes -> Quick Identify.
  - Immersive PWA Identify Coin Notes -> Quick Identify.
  - Identify Coin Notes -> captured-evidence Deep Analysis.
  - Saved-coin Deep Analysis notes -> job start.
- **Sibling paths**:
  - Typed-only notes.
  - Empty optional notes.
  - Add Coin/intake purpose, which must not gain voice controls.
  - Camera lifecycle and image-role mapping.
  - Price-estimate toggle.
  - Deep Analysis reused-evidence mode, which must not duplicate notes controls.
- **Validation tiers**:
  - Focused composable and component tests during implementation.
  - `task check:web` for completion.
  - Targeted browser/mobile workflow with mocked recognition on supported CI;
    real microphone behavior requires an authorized manual supported-browser
    check because CI cannot grant trustworthy microphone/provider access.
  - Exact-candidate hosted checks and software QC audit before release.
- **Review boundary**: An independent reviewer must evaluate the exact candidate,
  privacy disclosure, unsupported fallback, lifecycle cleanup, and no-auto-submit
  behavior. Owner acceptance and release remain separate.

### Complexity Tracking

| Required exception | Why necessary / simpler option rejected | Amendment and approval evidence |
|--------------------|------------------------------------------|---------------------------------|
| New ADR before implementation | Browser speech recognition may use platform-managed remote processing; this privacy and compatibility decision is material even without backend code. | Owner selected the phased browser-first direction; ADR acceptance remains pending. |
| Manual supported-browser microphone check | Automated tests can mock recognition events but cannot prove real browser permission/provider behavior. | Must be explicitly authorized and recorded; unavailable evidence remains pending. |

## Existing Structure and Reuse

- Keep `CoinLookupPage.vue` as the owner of `captureNotes`; do not introduce a
  second transcript state.
- Reuse the notes `update:notes` contract in both
  `CoinLookupCaptureWizard.vue` and `PwaCaptureShell.vue`.
- Reuse `DeepAnalysisStartPanel.vue`'s existing `notes` ref for saved-coin mode.
- Do not render another control when `DeepAnalysisStartPanel.vue` uses
  `reuseCapturedEvidence`; those notes were already reviewed in Identify Coin.
- Follow the user-initiated media and cleanup patterns already used by camera
  components, but do not share camera streams or request `getUserMedia` directly
  in the browser-recognition MVP.
- Add:
  - `src/web/src/composables/useVoiceDictation.ts`
  - `src/web/src/composables/__tests__/useVoiceDictation.test.ts`
  - `src/web/src/components/voice/VoiceDictationButton.vue`
  - `src/web/src/components/voice/__tests__/VoiceDictationButton.test.ts`
- The composable owns the typed browser adapter, lifecycle, normalized errors,
  final-result de-duplication, and abort-on-unmount behavior.
- The component owns accessible Start/Stop presentation, privacy/help text, and
  bounded status/error display. It emits final text; parent notes remain the
  source of truth.

## Implementation Slices

### Slice 0 - Decision and contract lock

- Propose and obtain acceptance for an ADR covering:
  - browser/platform audio processing and disclosure;
  - no Aurearia audio persistence;
  - English-only MVP;
  - unsupported-browser typed fallback;
  - future self-hosted transcription as separate scope;
  - why no backend/API contract is added now.
- Confirm the existing 2,000-character notes contract remains unchanged.

### Slice 1 - Shared recognition lifecycle

- Add focused tests first for unsupported detection, direct start/stop, final
  transcript extraction, interim exclusion, normalized errors, duplicate/late
  callback suppression, disabled-state abort, and unmount cleanup.
- Implement a minimal typed recognition adapter and component-scoped composable.
- Add a reusable accessible dictation button with idle/listening/stopping/error
  states and explicit privacy/help text.
- Stop after the shared unit/component tests pass; do not integrate workflow
  surfaces before the lifecycle guard is stable.

### Slice 2 - Quick Identify desktop and immersive PWA

- Integrate the shared button beside the existing notes fields in
  `CoinLookupCaptureWizard.vue` and `PwaCaptureShell.vue`.
- Use one shared helper to append final transcript text to the current notes:
  preserve existing text, add readable spacing, cap at 2,000 characters, and
  expose truncation feedback.
- Keep the control absent for `purpose="intake"`.
- Extend component/page tests to prove transcript propagation, editing,
  explicit-only submission, price-toggle preservation, and cleanup during
  navigation/responsive replacement.

### Slice 3 - Deep Analysis

- Add the shared control to the notes field rendered by
  `DeepAnalysisStartPanel.vue` when `reuseCapturedEvidence` is false.
- Preserve saved-coin and new-upload modes.
- Keep the control absent in reused-evidence mode because Identify Coin already
  owns and reviewed the notes.
- Extend tests for dictated notes payload, no-notes submission, existing image
  modes, provider override, and no duplicate control.

### Slice 4 - Compatibility, accessibility, and release evidence

- Add a targeted browser workflow using a deterministic recognition mock at a
  mobile viewport. Prove notes remain editable and no submit occurs when
  recognition ends.
- Verify keyboard operation, accessible names/status, permission/error
  messaging, and privacy disclosure.
- Run the complete web gate and relevant software QC audit.
- With explicit authorization, perform one real supported-browser/PWA manual
  check covering permission grant, denial, natural end, Stop, navigation
  cleanup, and Quick/Deep submission.

## Verification and Recovery

### Focused evidence

- `useVoiceDictation.test.ts`
  - API unavailable.
  - `en-US`, non-continuous, interim enabled only for display/filtering.
  - Start/Stop/abort lifecycle.
  - Final-only transcript emission.
  - Duplicate and post-abort events ignored.
  - Permission/no-speech/network/generic error mapping.
  - Disabled and unmount cleanup.
- `VoiceDictationButton.test.ts`
  - Accessible Start/Stop names and announced status.
  - Privacy disclosure.
  - Disabled/loading state.
  - Bounded error and truncation messaging.
- Existing Quick Identify and PWA component/page tests
  - Append semantics and 2,000-character handling.
  - No auto-analysis.
  - Typed-only and no-notes regression.
  - `purpose="intake"` unchanged.
- Existing Deep Analysis tests
  - Saved-coin dictated notes.
  - Captured-evidence reuse without duplicate UI.
  - Empty notes and existing provider/image behavior.

### Completion and CI

- `task check:web`
- Targeted browser test selector added under the existing web browser suite.
- `task test-critical-workflows` only if the selected workflow is included in
  its verified selectors; otherwise run the exact Playwright selector directly
  and record why.
- Hosted Vue, security, and container checks remain required for the exact
  candidate.
- Tamper-test at least:
  - remove abort-on-unmount and prove the lifecycle guard fails;
  - enable auto-submit or mutate the submit path from recognition completion and
    prove the workflow guard fails.

### Recovery

- Browser support and recognition failures degrade to the unchanged typed notes
  path.
- The feature writes no audio or new persisted state, so rollback is removal of
  the dictation UI/composable with no data migration.
- Dictated text already submitted is ordinary collector notes and remains valid
  if the feature is later disabled or removed.

## Supporting Artifacts

- **ADR**: Required before implementation for browser-provider privacy and
  compatibility. Proposed path:
  `docs/adr/0020-browser-speech-identification-notes.md`.
- **Research document**: N/A for initial planning; the ADR must cite verified
  browser behavior and privacy limitations. Create separate research only if
  browser compatibility cannot be bounded in the ADR.
- **Data model**: N/A; no persisted entity or schema change.
- **Contracts**: N/A; existing notes strings remain unchanged.
- **Quickstart**: N/A; user guidance belongs in the control disclosure and
  directly affected help/design documentation.

## Lifecycle Evidence

- **Specified/planned**: This spec, plan, and task set.
- **Implemented**: Shared browser recognition lifecycle, reusable control,
  desktop/PWA Quick Identify integration, and saved-coin Deep Analysis
  integration are complete on the feature branch.
- **Verified**: Focused Vitest, deterministic mobile Playwright, full
  `task check:web`, and both required tamper guards pass on the implementation
  candidate. Real-microphone verification remains pending.
- **Accepted**: ADR 0020 and implementation scope were owner-accepted through
  PR #801. Final candidate acceptance remains pending manual evidence and
  independent review.
- **Released**: Not authorized.
