# 1. Record Architecture Decisions

Date: 2026-05-28
Status: Accepted

## Context

Constitution v2.0.0 §22 (Amendment Process) now mandates an ADR-first workflow
for material design choices. Before today this project had no formal ADR
practice — design rationale lived in commit messages, inline code comments,
PR threads, and oral tradition. As the codebase grew across three services
(Go API, Vue PWA, Python agent) and the constitution itself reached
sixteen principles, the absence of a durable decision log became a real
risk: principles were being interpreted from memory, and new contributors
(human or AI) could not reconstruct *why* a given boundary existed.

Constitution §19 (Documentation Requirements) explicitly lists `docs/adr/`
as a required artifact. This ADR formalises the practice.

## Decision

We will record significant architecture decisions as Markdown files in
`docs/adr/` using the Michael Nygard template (Context / Decision /
Consequences / Related).

- **File naming:** `NNNN-kebab-slug.md`, monotonically increasing.
  Numbers are immutable once assigned.
- **Status values:** `Proposed` | `Accepted` | `Deprecated` |
  `Superseded by NNNN`. Status transitions are recorded by editing the
  header — the body of an Accepted ADR is otherwise immutable.
- **An ADR is required for:**
  - Constitution principle additions, removals, or material changes
  - New third-party services or runtime dependencies
  - Authentication, authorization, or security posture changes
  - Multi-service contract changes (Go ↔ Python, Go ↔ Vue)
  - Data-model migrations that change semantics, not just shape
  - UI / UX framework or design-system changes
- **PR integration:** ADRs are cited by spec, plan, and tasks documents,
  and referenced in PR descriptions per Constitution §17 (Quality Gate).
- **Lifecycle:** new ADRs open as `Proposed`. PR review promotes to
  `Accepted` on merge. Later ADRs may `Supersede` earlier ones — the
  superseded ADR's header is amended to point forward; its body remains.

## Consequences

+ Architectural rationale becomes durable and reviewable.
+ New contributors (human and AI agents) can read the project's history
  without spelunking commit logs.
+ Constitution amendments now have a clear audit trail via paired ADRs.
+ The ADR index doubles as a roadmap of where the system has been.
− ADR maintenance adds overhead — mitigated by keeping each ADR short
  (~1–2 pages, Nygard-style, not RFC-style).
− Decisions made before this practice (the v1.0 surface) require
  retroactive documentation. ADRs 0002–0004 cover the highest-value
  pre-existing decisions; further backfill is deferred unless a
  specific need arises.

## Related

- Constitution §22 (Amendment Process)
- Constitution §19 (Documentation Requirements)
- Constitution §0 (Hierarchy of Authority)
- `docs/adr/README.md` (index and process)
