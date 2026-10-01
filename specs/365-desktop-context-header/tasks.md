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

- [ ] T008 Capture the representative slice at normal and narrow desktop widths;
  confirm title spacing, truncation, icon grouping, divider placement, and global
  action priority with the owner.
- [ ] T009 Inventory the remaining local page headers by static title, dynamic
  title, explanatory content, and action count.
- [ ] T010 Migrate static-title pages without actions.
- [ ] T011 Migrate static-title pages with page-owned actions.
- [ ] T012 Migrate dynamic-title/detail pages with truncation and back actions.
- [ ] T013 Add a More actions menu only if the migrated narrow-desktop inventory
  cannot preserve usable targets without overlap.

## Phase 3: Completion

- [ ] T014 Add or update focused regression coverage for each migrated page
  family and PWA sibling behavior.
- [ ] T015 Run `task check:web` and affected desktop/mobile browser workflows on
  the exact candidate tree.
- [ ] T016 Obtain independent review and resolve all blocking findings.
- [ ] T017 Record final evidence and owner acceptance before merge or deployment.
