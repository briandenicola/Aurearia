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

The shared shell now covers the authenticated page inventory across static,
action-bearing, supporting-copy, and dynamic-title page families. Page-owned
actions remain lifecycle-bound, and ordinary narrow browsers plus installed
PWAs retain local headers. The specialized owned-coin detail shell remains
unchanged because its action bar and content title are integral to that layout.

The candidate is synchronized with `beta` through `63247d1a`. `task check:web`,
the affected contextual-header Playwright workflows, and `task check:delivery`
pass on the current uncommitted tree. Independent re-review passed for repaired
candidate patch SHA-256
`a29dc341be1cca6b434eea1def84cd4ce1067805023484a0cd776619238eaef4`.
The repository owner accepted the visual evidence on 2026-10-01 and authorized
commit/push of the feature branch.

## Next Action

Commit and push the feature branch, then request separate approval before
merging it into `beta`.
