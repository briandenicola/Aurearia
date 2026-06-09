# Architecture Decision Records

This directory holds Architecture Decision Records (ADRs) for the
Ancient Coins project, using the [Michael Nygard format][nygard]
(Context / Decision / Consequences / Related).

ADRs capture *why* a material design choice was made — the forces
in play at the time, the option chosen, and the trade-offs accepted.
They complement (do not replace) the project constitution at
`.specify/memory/constitution.md`.

## File Naming

`NNNN-kebab-slug.md`, where `NNNN` is a zero-padded, monotonically
increasing integer. **Numbers are immutable once assigned** — even
for deprecated or superseded ADRs.

## Status Lifecycle

```
Proposed ──► Accepted ──► Deprecated
                  │
                  └─────► Superseded by NNNN
```

- **Proposed** — opened in a PR, under review.
- **Accepted** — merged. Body becomes immutable; only the header
  may be amended (status, supersession link).
- **Deprecated** — no longer applies, but no replacement exists.
- **Superseded by NNNN** — replaced by a later ADR; the superseding
  ADR's `Related` section points back.

## Process

Per Constitution §22 (Amendment Process):

1. Open a PR adding `docs/adr/NNNN-slug.md` with status `Proposed`.
2. Reviewer(s) discuss on the PR thread.
3. Merge promotes status to `Accepted`.
4. If the ADR amends a constitution principle, the constitution's
   revision-history row is updated in the same PR.

## When to Open an ADR

Required for: principle additions or changes, new third-party
services, auth or security changes, multi-service contract changes,
data-model semantic migrations, UI/UX framework changes. See
ADR 0001 for the full list.

## Index

| #    | Title                                                              | Date       | Status   |
|------|--------------------------------------------------------------------|------------|----------|
| 0001 | [Record Architecture Decisions](0001-record-architecture-decisions.md) | 2026-05-28 | Accepted |
| 0002 | [Three-Service Architecture](0002-three-service-architecture.md)   | 2026-05-28 | Accepted |
| 0003 | [JWT Auth with Refresh and WebAuthn](0003-jwt-with-refresh-and-webauthn.md) | 2026-05-28 | Accepted |
| 0004 | [Design Token System](0004-design-token-system.md)                 | 2026-05-28 | Accepted |
| 0005 | [Streamline Constitution Principles](0005-streamline-constitution-principles.md) | 2026-06-09 | Proposed |

[nygard]: https://cognitect.com/blog/2011/11/15/documenting-architecture-decisions
