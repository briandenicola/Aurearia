---
name: vue-router-parent-child-navigation
description: "Preserve predictable save, cancel, hub, and deep-link navigation in Vue Router."
---

# Navigation with an explicit return contract

Read existing route definitions and navigation helpers before changing behavior.
Preserve the distinction between editing a parent resource and browsing sibling
detail sections.

- Parent-to-edit navigation normally pushes an entry. When the predecessor is
  the intended app-owned parent, save/cancel can go back without adding another
  detail entry. Back moves the history position; it does not erase the forward
  entry. Test the actual browser behavior instead of relying on a stack slogan.
- `replace` replaces the current entry; it does not itself append one. Replacing
  an edit entry with detail can still leave two adjacent detail entries when
  detail was already the predecessor.
- Do not infer an app-owned predecessor from `window.history.length`. For direct
  entry or an external predecessor, use the existing safe in-app return contract
  or obtain a decision before introducing a new fallback.
- Hubs with sibling subpages may need an explicit "Back to Gallery" destination.
  Preserve local overview/sibling-navigation conventions rather than applying
  `back()` indiscriminately. After deletion, navigate to a surviving resource.
- Auth redirects commonly replace the transient login entry. Do not reuse that
  behavior blindly for ordinary edit/save flows.

Test save, cancel, repeated hub visits, browser back/forward, deep links and
mobile gestures. Apply constitution Principles III and VI and authorized
`task check:web`; warnings are not waived in zero-warning lint.
