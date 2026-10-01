---
description: "Voice-assisted identification notes with bounded browser dictation and explicit submission"
---

# Tasks: Voice-Assisted Coin Identification Notes

**Work ID / spec directory**: `364-voice-assisted-identification`
**Working branch**: `beta`
**Input**: `specs/364-voice-assisted-identification/spec.md` and `plan.md`
**Lane**: High risk
**Implementation owner**: One implementation owner
**Non-goals / stop conditions**: No continuous conversation, spoken responses,
auto-submit, structured field extraction, audio persistence/upload, backend or
Python transcription, Add Coin voice input, non-English selection, or
self-hosted fallback implementation. Stop if the required ADR is not accepted.

This task set is subordinate to Constitution 4.0.0 and ADR 0019. No checkbox
alone proves implementation, verification, acceptance, or release.

## Phase 1: Scope, Authority, and Existing Foundations

- [x] T001 Confirm the owner-approved Feature 364 criteria and non-goals against
  Feature 348, Feature 344/351, ADR 0018, current active decisions, and any open
  review restrictions. Record the implementation candidate base and verify that
  `beta` is the authorized work branch.
- [x] T002 Draft `docs/adr/0020-browser-speech-identification-notes.md` covering
  browser/platform audio processing, disclosure, no Aurearia audio persistence,
  English-only MVP, typed fallback, unsupported browsers, and the separately
  scoped future self-hosted path. Obtain required owner acceptance before T004.
- [x] T003 Reconfirm the actual notes consumers and sibling workflows in
  `CoinLookupCaptureWizard.vue`, `PwaCaptureShell.vue`, `CoinLookupPage.vue`, and
  `DeepAnalysisStartPanel.vue`; verify no API, Go, Python, schema, settings, or
  dependency change is needed. Stop and re-scope if that assumption is false.

## Phase 2: Shared Voice Recognition Lifecycle (US3, US4)

**Goal**: Provide one component-scoped, typed browser recognition lifecycle
that is safe to reuse without owning or submitting notes.

**Independent test**: Mount the composable/control with a fake recognition
adapter and prove supported/unsupported detection, explicit Start/Stop, final
text, errors, disabled-state abort, and unmount cleanup.

### Verification

- [x] T004 [US3] Add failing focused tests in
  `src/web/src/composables/__tests__/useVoiceDictation.test.ts` for unavailable
  APIs, `en-US`, user-triggered start, final-only text, multiple final segments,
  duplicate/late callback suppression, Stop, abort, permission/no-speech/network
  errors, disabled-state cleanup, and unmount cleanup.
- [x] T005 [US4] Add failing component tests in
  `src/web/src/components/voice/__tests__/VoiceDictationButton.test.ts` for
  accessible Start/Stop names, visible/listed recording state, keyboard use,
  disabled/loading behavior, privacy disclosure, and bounded error output.

### Implementation

- [x] T006 [US3] Implement
  `src/web/src/composables/useVoiceDictation.ts` with repository-owned minimal
  TypeScript interfaces for standard/prefixed browser recognition, injected
  factory support for tests, component-scoped state, normalized errors, and
  abort-on-disable/unmount. Do not use `any`, global singleton state, storage, or
  direct `getUserMedia`.
- [x] T007 [US4] Implement
  `src/web/src/components/voice/VoiceDictationButton.vue` using existing button,
  icon, status, typography, and color patterns. Emit final text only; never own
  notes or invoke analysis submission.
- [x] T008 [US3] Run the focused composable/component tests and tamper the
  abort-on-unmount guard to prove it fails, then restore it. Record the exact
  candidate/tree and result before workflow integration.

## Phase 3: Quick Identify Desktop and PWA (US1)

**Goal**: Dictate editable identification notes in both existing Identify Coin
notes surfaces while preserving explicit submission and the 2,000-character
contract.

**Independent test**: In desktop and immersive PWA component tests, dictate
final text into empty and prefilled notes, edit it, and verify analysis starts
only from the existing Analyze/Deep action.

### Verification

- [x] T009 [US1] Extend
  `src/web/src/components/coin-lookup/__tests__/CoinLookupCaptureWizard.test.ts`
  with failing cases for empty/prefilled append, readable spacing,
  2,000-character truncation feedback, unsupported fallback, submitting state,
  `purpose="intake"` exclusion, and no auto-analysis.
- [x] T010 [US1] Extend the existing `PwaCaptureShell` tests with the same voice
  notes contract at mobile layout, including active-listening cleanup when the
  shell unmounts or leaves the Notes step.
- [x] T011 [US1] Extend
  `src/web/src/pages/__tests__/CoinLookupPage.test.ts` to prove the final edited
  transcript reaches the existing Quick Identify notes request and the existing
  captured-evidence Deep Analysis handoff without a second value or submit path.

### Implementation

- [x] T012 [US1] Add one shared, pure append helper under the voice composable or
  a focused utility: preserve existing notes, normalize only join spacing,
  append final transcript text, cap at 2,000 characters, and return whether
  truncation occurred. Do not alter user text beyond the bounded join.
- [x] T013 [US1] Integrate `VoiceDictationButton` beside the existing notes field
  in `CoinLookupCaptureWizard.vue`, emitting only through the existing
  `update:notes` contract and disabling/aborting with submission or step exit.
- [x] T014 [US1] Integrate the same control in `PwaCaptureShell.vue` without
  changing immersive camera ownership, image roles, step navigation, price
  toggle, or `purpose="intake"` behavior.
- [x] T015 [US1] Run the focused Quick Identify/PWA/page tests. Tamper the
  no-auto-submit guard so recognition completion invokes analysis, prove the
  test fails, then restore it.

## Phase 4: Deep Analysis Notes (US2)

**Goal**: Reuse dictated Identify Coin notes for intake and support dictation in
the saved-coin Deep Analysis notes field without changing job contracts.

**Independent test**: Verify captured evidence reuses reviewed notes without a
duplicate control, while saved-coin mode dictates and submits the existing
optional `notes` value.

### Verification

- [x] T016 [US2] Extend
  `src/web/src/components/deep-identification/__tests__/DeepAnalysisStartPanel.test.ts`
  with failing cases for saved-coin dictated notes, typed-plus-dictated append,
  empty optional notes, unsupported fallback, submitting cleanup, and absence of
  the control when `reuseCapturedEvidence` is true.
- [x] T017 [US2] Confirm existing Coin Lookup tests continue to prove that
  Identify Coin notes, including dictated text, are reused by Deep Analysis and
  that no duplicate notes editor/control appears in reused-evidence mode.

### Implementation

- [x] T018 [US2] Integrate `VoiceDictationButton` into
  `DeepAnalysisStartPanel.vue` only when its notes textarea is rendered. Preserve
  saved-coin image reuse, hint images, provider overrides, validation, and the
  existing `CreateDeepIdentificationJobInput.notes` contract.
- [x] T019 [US2] Run the focused Deep Analysis tests and verify the no-notes path
  still submits normally with valid obverse/reverse evidence.

## Phase 5: Compatibility, Documentation, and Acceptance

- [x] T020 [P] Add a deterministic browser workflow using the existing
  Playwright infrastructure and a mocked recognition constructor at a mobile
  viewport. Prove Start, final transcript, edit, explicit Analyze, Stop/error,
  and navigation cleanup without requiring a real microphone in CI.
- [x] T021 Update directly affected help/design documentation with the
  push-to-talk behavior, typed fallback, English-only limit, browser support
  boundary, privacy disclosure, and the fact that Aurearia stores transcript
  notes but not audio. Do not advertise the future self-hosted fallback as
  implemented.
- [x] T022 Run `task check:web` and the verified targeted browser selector on the
  exact candidate. Record supported-runtime hosted checks as pending until CI;
  missing tools or authorization remain incomplete.
- [ ] T023 With explicit owner authorization, manually verify one supported
  desktop/mobile PWA browser: permission grant, denial, natural silence end,
  Stop, navigation cleanup, editable transcript, Quick Identify submission, and
  Deep Analysis submission. Record browser/version and exact candidate.
- [ ] T024 Run the required software QC audit for microphone/privacy/PWA scope,
  covering browser-provider disclosure, no raw audio persistence/logging,
  unsupported fallback, state isolation, accessibility, and rollback.
- [ ] T025 Obtain independent read-only review of the exact commit/tree with the
  accepted ADR and all evidence. Only that reviewer or an owner-appointed
  independent successor may clear a block.
- [ ] T026 Reconcile the Feature 364 lifecycle state, task evidence, ADR index,
  directly affected docs, `.squad/log/`, and `.squad/identity/now.md`. Do not
  merge, publish, deploy, or release without separate owner authorization.

## Dependencies and Execution

- T001-T003 are prerequisites for implementation.
- T002 (accepted ADR) blocks T004 and all later implementation tasks.
- T004-T008 must complete before integrating any workflow surface.
- T009-T015 and T016-T019 both depend on T008. They touch shared voice files and
  should remain under one implementation owner; do not parallelize writes to
  the shared composable/component.
- T020 depends on Quick Identify and Deep Analysis integration.
- T021 may proceed after UI behavior is stable.
- T022 depends on all implementation and automated tests.
- T023 requires explicit machine/browser authorization and can remain pending
  while CI runs.
- T024 depends on the complete candidate and available evidence.
- T025 depends on the exact audited candidate; implementation changes after
  review require applicability analysis or re-review.
- T026 depends on the reviewer verdict and owner disposition.
- No dependency installation, branch creation, commit, push, PR, deployment, or
  release is authorized merely by this task list.

## Evidence

| Criterion / task | Evidence and command/result | Commit/tree | State / reviewer |
|------------------|-----------------------------|-------------|------------------|
| T001-T003 | PR #801 merged as `150eb23d`; ADR 0020 accepted and scope authorized | `150eb23d` | Complete |
| T004-T008 | Focused lifecycle/control suite passed; removing unmount abort failed 1 expected test, then restoration passed | Implementation candidate | Complete |
| T009-T015 | Desktop/PWA/page focused suites passed; adding transcript-triggered Analyze failed the explicit-submit guards, then restoration passed | Implementation candidate | Complete |
| T016-T019 | Deep Analysis focused suite passed, including dictated, empty-notes, disabled, unsupported, and reused-evidence paths | Implementation candidate | Complete |
| T020-T022 | Mocked mobile Playwright workflow passed; `task check:web` passed lint, strict type-check, full tests, and production build | Implementation candidate | Complete |
| T023 | Real supported-browser microphone permission/provider behavior | Pending | Requires explicit manual environment evidence |
| T024 | Pending software QC audit | Pending | Planned |
| T025 | Pending independent exact-candidate verdict | Pending | Planned |
| T026 | Pending owner acceptance and handoff | Pending | Planned |
