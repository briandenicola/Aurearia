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
slice 2 is reviewed **PASS**, bound to `f265ff87`: the typography scale gained
five steps so every size in the app is a named token, all 170 arbitrary
`text-[Nrem]` uses and all 154 raw `text-[var(--…)]` classes are gone, and two
zero-tolerance guards hold the line. The review blocked first on an
under-declared line-height set — 37 elements, 8 of them `form-input` controls
that change height — and cleared once the record was corrected. See
[the slice 2 log](../log/20260929T234500Z-784-slice2-typography-colour.md).

## Next Action

Owner review of the running app for slice 2. Check the auction lot edit form
first — eight `form-input` controls change height there. Screenshots were
waived, so nothing is visually confirmed.

Slice 3 (casing rules and a shared data-table header/cell recipe) is in
progress on this tree. Three carried items remain for slice 4:
`COLOR_BUDGET = 139` still rests on the author's count alone, the light-theme
status-contrast gap in `docs/design-system.md` section 4 is still open, and
`text-xs` remains a Tailwind built-in duplicating `text-sm` at `0.75rem`
across 35 call sites. Then #766, which still needs its own spec.
No deployment or release is authorized.
