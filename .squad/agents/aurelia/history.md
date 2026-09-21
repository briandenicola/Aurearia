# Aurelia: Current Working Context

Curated 2026-09-21 under Constitution section 18.3; not a new review or approval.
Read [charter](charter.md) only when acting as Aurelia, then
[current work](../../identity/now.md) and [active decisions](../../decisions.md).
Feature state belongs in the selected tasks, not this history.

## Durable implementation notes

- Reuse existing UI patterns and tokens; preserve both desktop and PWA branches.
  Check the current instructions instead of treating old screenshots or history
  as authority for a new UI pattern.
- Use real component identity for lucide icon assertions; an arbitrary SVG does
  not prove the requested icon. `size` can be a fallthrough attribute, not a prop.
  Source: original history lines 676-690.
- Shared Quick Access state needs deduplicated bootstrap, generation invalidation,
  no browser persistence, and reconciliation after the COMPLETE successful
  workflow, including image changes. The implementation notes are not acceptance:
  the frontend rejection chain remains unresolved.
- Background-removal model execution belongs in the isolated module worker.
  Earlier advice to relax app-wide CSP is superseded by ADR 0014 and Brutus's
  explicit worker re-review.

## Review restrictions

Active decisions R357-QA/R357-ARCH preserve the exclusion of Aurelia and Livia from
the next rejected Quick Access revision; Brutus must clear before Maximus re-review.
R225 retains the non-Aurelia revision requirement until clearance is established.
R-SWIPE is conditional, NOT a blanket current lockout; apply its exact terms.
No restriction is cleared by this summary.

## Preserved evidence

[Full original history](history-archive.md#curation-2026-09-21-aurelia), canonical
baseline `e6ab8313346b971dce4b288804036222f7d4c95d`.
[Inventory](../../artifacts/context-curation-2026-09-21.json): record `aurelia`;
original line N is archive line N+11. Search indexed headings for other domains.
The archive preserves all earlier decisions and review terms unchanged.
