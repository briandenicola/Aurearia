---
updated_at: 2026-10-01
focus_area: Feature 364 voice-assisted identification planning
owner: Copilot CLI implementation owner; repository owner accepts results
work_branch: 364-voice-assisted-identification
baseline_commit: 8f11df00
work_artifact: specs/364-voice-assisted-identification/spec.md
tasks_artifact: specs/364-voice-assisted-identification/tasks.md
---

# Current Work

Feature 364 plans push-to-talk browser dictation into the existing optional
notes fields for Quick Identify and Deep Analysis. The transcript remains
editable and analysis starts only from the existing explicit action. Typed
notes remain the fallback; the MVP adds no backend transcription or audio
persistence.

The [spec](../../specs/364-voice-assisted-identification/spec.md),
[plan](../../specs/364-voice-assisted-identification/plan.md), and
[tasks](../../specs/364-voice-assisted-identification/tasks.md) are drafted.
[ADR 0020](../../docs/adr/0020-browser-speech-identification-notes.md) is
Accepted through owner-merged PR #801.

## Next Action

Freeze the verified implementation candidate, run the required software QC
audit and independent read-only review, then reconcile the remaining manual
browser evidence and owner acceptance boundary.
