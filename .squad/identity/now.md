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

#784 slice 1 is reviewed **PASS**, bound to `af2df567`, after four rounds. #784
slice 2 is implemented on top of it: the typography scale gained five steps so
every size in the app is a named token, all 170 arbitrary `text-[Nrem]` uses
and all 154 raw `text-[var(--…)]` classes are gone, and two zero-tolerance
guards hold the line. See
[the slice 2 log](../log/20260929T234500Z-784-slice2-typography-colour.md).

## Next Action

Independent review of slice 2, then owner review of the running app. The slice
2 log lists six individually noticeable size changes and one line-height
change; screenshots were waived, so none is visually confirmed. Three carried
items: `COLOR_BUDGET = 139` still rests on the author's count alone, the
light-theme status-contrast gap in `docs/design-system.md` section 4 is still
open, and `text-xs` remains a Tailwind built-in duplicating `text-sm` at
`0.75rem` across 35 call sites. Then agree whether #784 continues into
casing and table density, or #766 comes next. #766 still needs its own spec.
`npm ci` and `npm run build` in `src/web` were owner-authorized and pass. No
deployment or release is authorized.
