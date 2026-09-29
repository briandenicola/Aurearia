---
updated_at: 2026-09-29
focus_area: Open issue batch on beta (issues #766-#784)
owner: Copilot CLI implementation owner; repository owner accepts results
work_branch: beta
baseline_commit: be5dd3e4
work_artifact: docs/audits/2026-09-29-open-issue-plans.md
tasks_artifact: docs/audits/2026-09-29-open-issue-plans.md
---

# Current Work

The owner is working through the open issues on `beta` before any merge to
`main`. The triage, grouping and fix plans are in
[the open-issue plans](../../docs/audits/2026-09-29-open-issue-plans.md), with a
status table at the top.

Done and pushed: typography fixes, group B (#770, #772, #769), group A (#768).
Group C (#776, #775, #777) is implemented and verified in the commit that
updates this file: `task check:web` passed; Go and agent code are unchanged.

## Next Action

Pick the next group with the owner: F (#780, #781, #783), E (#774, #771, #779),
D (#766, needs its own spec first) or #784. Pending owner approval:
`task setup:agent` (ruff 0.16.8 lock mismatch) and the local Go toolchain
update needed for `task check:go`. No deployment or release is authorized.