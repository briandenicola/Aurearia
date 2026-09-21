# Maximus: Current Working Context

Curated 2026-09-21 under Constitution section 18.3; not a new review or approval.
Read [charter](charter.md) when acting as Maximus,
[current work](../../identity/now.md), and [active decisions](../../decisions.md).

## Durable architecture and delivery notes

- Use the accepted authority hierarchy and smallest complete lane. Governance
  4.0.0 supersedes historical compulsory fan-out for new work; it does not
  retroactively remove older author restrictions.
- Keep design authorization, implementation, independent verification, owner
  acceptance and release evidence distinct. A checked QA task beside a REJECT
  is a reconciliation problem, not clearance.
- Shared primitives need contracts proven through their real consumers.
  Linux-only and on-device conditions must retain their exact evidence boundary.
- Worker-scoped CSP isolation preserves app security while allowing the
  background-removal model to execute; ADR 0014 and the explicit Brutus
  re-review supersede the earlier app-wide relaxation suggestion.

## Review restrictions

R357-ARCH requires Brutus's frontend clearance before Maximus re-review. Marcus
was assigned independent revision; neither assignment nor a later merge clears
the original rejection. Aurelia/Livia exclusions remain.

R-SWIPE explicitly lifted round-one lockout but retained the specified Linux
runner condition and owner device acceptance. Do not turn that into a blanket
lockout or a blanket PASS. R337 (Wishlist Search Alerts, issue #357) is a
DIFFERENT scope: locate the original reviewer approval behind Scribe's report.

## Preserved evidence

[Full original history](history-archive.md#curation-2026-09-21-maximus), canonical
baseline `e6ab8313346b971dce4b288804036222f7d4c95d`.
[Inventory](../../artifacts/context-curation-2026-09-21.json): record `maximus`;
original line N is archive line N+11. Final Quick Access review is at original
lines 951 onward. All prior reports and restrictions remain unchanged.
