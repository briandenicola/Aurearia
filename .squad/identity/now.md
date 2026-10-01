---
updated_at: 2026-10-01
focus_area: Feature 365 desktop context header
owner: Copilot CLI implementation owner; repository owner accepts results
work_branch: 365-desktop-context-header
baseline_commit: 63247d1a
work_artifact: specs/365-desktop-context-header/spec.md
tasks_artifact: specs/365-desktop-context-header/tasks.md
---

# Current Work

Feature 365 moves desktop page titles and page-owned actions into the shared
application bar as `Aurearia | Page Title`, while preserving existing local
headers in mobile/PWA layouts. Pages retain ownership of callbacks, routing,
loading state, and modal state through lifecycle-bound deferred Teleports.

The first usable slice implements the shared shell and migrates Wishlist,
Followers, Identify Coin, and Stats. It is synchronized with `beta` through the
voice permission and CI repairs at `63247d1a`.

## Next Action

Inventory and migrate all remaining applicable desktop page headers by static,
dynamic, and action-bearing page families, then complete visual and automated
verification before merging back to `beta`.
