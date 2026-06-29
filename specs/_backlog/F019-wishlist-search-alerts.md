---
id: F019
title: "Add wishlist search alerts for acquisition ideas"
status: triaged
priority: P1
effort: L
value: 5
risk: 4
owner: Maximus
created: 2026-06-29
updated: 2026-06-29
---

# F019 — Add wishlist search alerts for acquisition ideas

**GitHub issue**: #357

## Summary

Add saved search alerts that recommend listings or ideas a collector may want to add to their wishlist. This is distinct from existing wishlist availability checks: the system searches for candidate acquisitions based on criteria, verifies source data, and lets the user convert promising results into wishlist items.

## Acceptance criteria

- [ ] User can define alert criteria such as ruler/issuer, type, date range, mint, material, grade/condition, price range, dealer/source, and cadence.
- [ ] Scheduled or manual runs return candidate listings with source URL, observed price, title, reason for match, and last-seen timestamp.
- [ ] User can dismiss a candidate, save it as a wishlist item, or adjust criteria to reduce noise.
- [ ] Existing wishlist availability checks continue to verify already-saved wishlist URLs and are not conflated with discovery alerts.
- [ ] Agent/search results preserve provenance and do not invent dealer details or availability claims.

## Plan outline

1. Define alert criteria and result lifecycle separately from wishlist availability history.
2. Decide whether v1 uses deterministic dealer/source searches, the Python search agent, or both behind a Go-owned service boundary.
3. Add alert CRUD, run history, candidate review, and convert-to-wishlist flows.
4. Validate source provenance, rate limits, duplicate suppression, and notification behavior before enabling scheduled runs.

## Task breakdown

- Backend: add alert/result models in `src\api\models\`, repositories in `src\api\repository\`, orchestration in `src\api\services\`, handlers in `src\api\handlers\`, scheduler wiring in `src\api\main.go`, and settings if admin cadence is configurable.
- Agent/search: if search uses Python, add bounded request/response schemas in `src\agent\app\models\` and a provenance-preserving pipeline under `src\agent\app\` while keeping database access in Go only.
- Frontend: add alert management and candidate review pages/components under `src\web\src\pages\` and `src\web\src\components\wishlist\`, API client methods, types, and notification affordances.
- Testing: cover user scoping, criteria validation, duplicate suppression, conversion to wishlist, scheduler behavior, and agent provenance with Go/Python/frontend tests as applicable.
- Documentation: update wishlist docs to separate discovery alerts from availability checks.

## Constitution alignment

- §0 Hierarchy of Authority — this card must promote to a full spec before implementation because it spans Go, Vue, scheduler, and possibly agent service boundaries.
- Principle II (Service Boundary Separation) — Python may search/analyze, but Go owns persistence, scheduling, auth, and wishlist mutations.
- Principle V (Security, Auth, and Privacy by Default) — criteria, source URLs, outbound fetching, and user-specific candidates require strict validation and scoping.
- Principle VIII (Documented Decisions) — source/search provider and scheduler choices may require ADR or decision capture.
- §17 Quality Gate, §21 Definition of Done.

## Open questions

- [ ] Which sources are allowed in v1: specific dealers, SearXNG/web search, user-provided URLs, or all of these?
- [ ] Should alerts notify immediately, digest daily/weekly, or only show in-app?
- [ ] What duplicate detection key is sufficient across dealer title, URL, image, and price changes?

## Notes

This complements F002 but must not modify the meaning of availability checks. F002 answers "is my saved wishlist target still available?" This card answers "what should I consider adding to my wishlist?"

## History

- 2026-06-29: created (status: triaged).
