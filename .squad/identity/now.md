---
updated_at: 2026-09-29
focus_area: Open issue batch on beta (issues #766-#784)
owner: Copilot CLI implementation owner; repository owner accepts results
work_branch: beta
baseline_commit: d763826f
work_artifact: docs/audits/2026-09-29-open-issue-plans.md
tasks_artifact: docs/audits/2026-09-29-open-issue-plans.md
---

# Current Work

The owner is working through open issues on `beta` before any merge to `main`.
Plans and a status table are in
[the open-issue plans](../../docs/audits/2026-09-29-open-issue-plans.md).
Groups A, B, C, E and F are done: #779 grounds Quick Identify prices in current
dealer listings and #771 records the per-site adapter decisions, both reviewed
PASS and committed as `de0dd508`. #784 slice 1 is done on this tree: every
toggle now routes through `BaseToggle`, every admin run-status pill through
`BaseStatusBadge`, and `docs/design-system.md` is the canonical written
standard, with ratchet guards in `design-tokens.test.ts`. See
[the slice 1 log](../log/20260929T220000Z-784-slice1-ui-primitives.md).

#784 slice 1 is reviewed **PASS**, bound to `af2df567`, after four rounds: the
first returned INCOMPLETE (evidence gaps), the second and third FAIL — catching
a knob-geometry regression the reviewer had itself mistakenly requested, then
five defects in the completion record. All are repaired and cleared, with no
open findings and no revision-owner restriction.

## Next Action

Owner decision. Confirm the eight declared visual deltas in the slice 1 log —
screenshots were waived, so none has been visually confirmed by anyone — and
note two carried items: `COLOR_BUDGET = 139` rests on the author's count alone,
and the light-theme status-contrast gap recorded in `docs/design-system.md` §4
should be carried into slice 2 scope. Then agree whether #784 slice 2
(page-by-page typography and tables, which needs owner screenshots) or #766
comes next. #766 still needs its own spec.
`npm ci` and `npm run build` in `src/web` were owner-authorized and pass. No
deployment or release is authorized.
