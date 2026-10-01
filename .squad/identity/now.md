---
updated_at: 2026-10-01
focus_area: Feature 364 microphone permission repair
owner: Copilot CLI implementation owner; repository owner accepts results
work_branch: beta
baseline_commit: 09f39e22
work_artifact: specs/364-voice-assisted-identification/spec.md
tasks_artifact: specs/364-voice-assisted-identification/tasks.md
---

# Current Work

Feature 364 is merged into `beta`. Manual Edge testing found that relying on
`SpeechRecognition.start()` did not reliably display the microphone permission
prompt and returned `not-allowed`.

The repair requests microphone permission from the same explicit dictation
click, immediately releases the temporary audio track, and starts browser speech
recognition only after access is granted. The implementation remains
browser-neutral and retains complete typed fallback when recognition is
unsupported.

## Next Action

Commit and push the verified repair to `beta`, deploy it to the beta environment,
then record Chrome, Edge, and Firefox-build behavior for the exact candidate.
