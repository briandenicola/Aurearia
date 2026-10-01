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

Feature 364 is merged into `beta`. Manual Edge testing found that microphone
permission could not be granted. The root cause was Aurearia's own
`Permissions-Policy: microphone=()` response header, which disables microphone
access before browser permission UI can succeed.

The repair now requests permission from the explicit dictation click and changes
the application and documented nginx policy to `microphone=(self)`. This permits
only Aurearia's HTTPS origin, preserves the browser-neutral implementation, and
retains complete typed fallback when recognition is unsupported.

## Next Action

Run the required Go, web, and delivery checks; commit and push the policy repair
to `beta`; update the deployed nginx header; then record Chrome, Edge, and
Firefox-build behavior for the exact candidate.
