# Cassius: Current Working Context

Curated 2026-09-21 under Constitution section 18.3; not a new review or approval.
Read [charter](charter.md) when acting as Cassius,
[current work](../../identity/now.md), and [active decisions](../../decisions.md).
Use the selected spec/tasks, not historical implementation counts, for new work.

## Durable implementation notes

- Preserve Go's layered boundaries and owner scopes. Coordinate atomic
  cross-repository work through the existing repository transaction boundary;
  do not import GORM into ordinary services to bypass architecture enforcement.
- Lifecycle cleanup must cover sibling mutations and provider sync, not just a
  new endpoint. Preserve Quick Access's authoritative store and transactional
  compatibility mirror for pinned Sets. Source: original history lines 387-395.
- SQLite failure triggers are useful for proving a target mutation rolls back
  when related cleanup fails. Query-count evidence must observe both GORM Query
  and Row callbacks when the path uses Find and Scan.
- Keep background-removal worker CSP separate from app CSP; scoped worker
  permission is not authorization for app-wide unsafe-eval.
- Contract fixtures are not acceptance by themselves. Derived values, provenance,
  bounded payloads and adversarial cases require meaningful validation.

## Review restrictions

R361 in active decisions preserves Brutus's rejection of foundational specialist
contracts. Cassius must not revise that rejected batch; Livia was assigned, but
assignment is not clearance. R352/R353 retain independent-revision terms until
the original reviewer clearance is established. This history clears no block.

## Preserved evidence

[Full original history](history-archive.md#curation-2026-09-21-cassius), canonical
baseline `e6ab8313346b971dce4b288804036222f7d4c95d`.
[Inventory](../../artifacts/context-curation-2026-09-21.json): record `cassius`;
original line N is archive line N+11. Retrieve other implementation details by
indexed heading rather than loading the whole historical transcript.
