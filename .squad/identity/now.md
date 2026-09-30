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

Slice 3 (casing rules and a shared `.data-table` header/cell recipe) was
reviewed **BLOCK** at `8bdbbcb7` and has been repaired; it is awaiting
re-review. The block: five tables spelled the header recipe out on each `th`
rather than as `[&_th]:` variants, so the codemod and the guard both missed
them and the docs' claim that every table carries `data-table` was false at
the commit that made the claim. All five are now migrated, the guard rejects
both spellings, and the docs state what it does not catch. The largest single
visible change in the slice is `AdminSystemSection.vue`'s provider table,
whose headers were never on the standard and now become uppercase. `.badge`
(12 sites) was also converged onto `BaseBadge`'s recipe. See
[the slice 3 log](../log/20260930T020000Z-784-slice3-casing-tables.md);
its label counts are recorded as approximate and do not reproduce exactly.
Slice 3 was cleared **PASS** at `8dd0dbb7`.

Slice 4 (colour tokens, overlays, light-theme contrast) is implemented and
awaiting review. The one real defect it fixes: `[data-theme="light"]` overrode
no status foreground, so every status colour in the light theme failed WCAG AA
on a white card — `--color-positive` measured 2.10:1. Eight overrides now clear
4.5:1 against both `--bg-card` and `--bg-primary`, held by a guard that
computes the ratio rather than asserting it. Template colour literals went
139 to 61, and `COLOR_BUDGET` is now a per-file map so a swap cannot net to
zero. See [the slice 4 log](../log/20260930T040000Z-784-slice4-colour-contrast.md).

Two items are deliberately **not** done. The `text-xs` fold (35 sites) is
deferred because `text-xs` and `text-sm` share `0.75rem` but not their
line-heights, so it is a 7% height change, not a rename. The remaining 61
literals are an owner decision: the app carries four near-duplicate greens and
five reds with no single winner, and picking one is a design call.

**The owner's visual pass is now overdue across all four slices**, and the
light theme should be first — its status colours all move in slice 4 and
nobody has looked at any of it. Then #766, which still needs its own spec.
No deployment or release is authorized.
