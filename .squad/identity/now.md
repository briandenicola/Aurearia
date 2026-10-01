---
updated_at: 2026-10-01
focus_area: Feature 365 desktop context header
owner: Copilot CLI implementation owner; repository owner accepts results
work_branch: 365-desktop-context-header
baseline_commit: 09f39e22
work_artifact: specs/365-desktop-context-header/spec.md
tasks_artifact: specs/365-desktop-context-header/tasks.md
---

# Current Work

Feature 365 moves desktop page titles and page-owned actions into the shared
application bar as `Aurearia | Page Title`, while preserving existing local
headers in mobile/PWA layouts. Pages retain ownership of callbacks, routing,
loading state, and modal state through lifecycle-bound deferred Teleports.

The first usable slice implements the shared shell and migrates Wishlist,
Followers, Identify Coin, and Stats. Focused tests and `task check:web` pass on
the dirty tree based on `09f39e22`; the deferred-Teleport guard was tamper-tested.

## Next Action

Capture and review the representative slice at normal and narrow desktop widths,
then migrate the remaining standalone page headers by page family.
