# Tasks: Desktop Context Header

**Work ID / spec directory**: `365-desktop-context-header`
**Working branch**: `365-desktop-context-header`
**Input**: `spec.md` and `plan.md`
**Lane**: Feature

## Phase 1: Shared Shell and Representative Slice

- [x] T001 Add desktop-only title and page-action hosts to `App.vue`, preserving
  existing global and collection actions.
- [x] T002 Add `DesktopPageContext.vue` with deferred, lifecycle-bound Teleports
  and no module-level page state.
- [x] T003 Extend `AppIconButton` to support router destinations while retaining
  button behavior, accessibility, disabled state, and 44-pixel sizing.
- [x] T004 Add navigation tests proving title/action rendering, placement before
  global actions, and cleanup when navigating away.
- [x] T005 Migrate Wishlist, Followers, Identify Coin, and Stats while preserving
  their existing PWA/mobile headers and page-owned behavior.
- [x] T006 Run focused tests and `task check:web`.
- [x] T007 Tamper the deferred-Teleport guard, confirm the integration test
  fails, restore it, and confirm the test passes.

## Phase 2: Visual Review and Broad Migration

- [x] T008 Capture the representative slice at normal and narrow desktop widths;
  confirm title spacing, truncation, icon grouping, divider placement, and global
  action priority with the owner.
- [x] T009 Inventory the remaining local page headers by static title, dynamic
  title, explanatory content, and action count.
- [x] T010 Migrate static-title pages without actions.
- [x] T011 Migrate static-title pages with page-owned actions.
- [x] T012 Migrate dynamic-title/detail pages with truncation and back actions.
- [x] T013 Add a More actions menu only if the migrated narrow-desktop inventory
  cannot preserve usable targets without overlap.

## Phase 3: Completion

- [x] T014 Add or update focused regression coverage for each migrated page
  family and PWA sibling behavior.
- [x] T015 Run `task check:web` and affected desktop/mobile browser workflows on
  the exact candidate tree.
- [x] T016 Obtain independent review and resolve all blocking findings.
- [x] T017 Record final evidence and owner acceptance before merge or deployment.

## Candidate Evidence

- `task check:web` — passed on the uncommitted candidate tree: lint, strict
  `vue-tsc --build`, build, and 1,799 Vitest tests (1,798 passed, 1 skipped).
- `task check:delivery` — passed on the same candidate tree: 91 delivery tests
  plus governance validation.
- Focused contextual-header tests — 41 passed across the shared component,
  Wishlist, Identify Coin, and Stats.
- Owner visually accepted the preserved 768px, 769px, installed-PWA, and crowded
  Wishlist screenshots on 2026-10-01 and authorized commit/push of the feature
  branch. Merge into `beta` remains separately gated.
- `npx playwright test e2e/workflows/desktop-context-header.spec.ts
  e2e/workflows/add-coin-capture.spec.ts` — 6 passed, covering the 768/769px
  boundary, installed-PWA local headers at desktop width, normal desktop Add
  Coin behavior, and the crowded four-action Wishlist header.
- Independent read-only re-review — PASS for candidate patch SHA-256
  `a29dc341be1cca6b434eea1def84cd4ce1067805023484a0cd776619238eaef4`;
  the prior Add Coin disabled-state block is cleared.
