---
title: Dealer / Source Database
id: F010
status: promoted
priority: P3
effort: M
value: 3
risk: 2
owner: unassigned
created: 2026-05-28
updated: 2026-05-28
---

## Summary

Replace the free-text dealer/source field on coin records with a normalized, searchable `Dealer` entity. Users can record auction houses, online marketplaces, and individual sellers once and reference them across many coin purchases, enabling provenance grouping, dealer-level spend analytics, and consistent attribution.

## Acceptance Criteria

- New `Dealer` entity with: name (required), type (auction_house / dealer / marketplace / private_sale / other), website (optional URL), location (optional), notes (optional), user_id (per-user scoped)
- Coin purchase metadata gains a nullable `DealerID` foreign key alongside (or replacing) the existing free-text source field
- Coin add/edit form provides a dealer typeahead with "Create new dealer" inline action
- Migration backfills: distinct free-text source values per user become draft `Dealer` rows (status `unverified`); coins link to them; user can later merge duplicates
- Dealer detail page lists all coins purchased from that dealer with totals (count, sum spent, sum current value)
- Dealer can be deleted only if no coins reference it; otherwise UI offers re-assignment
- Search/filter on the coins list supports "by dealer"

## Constitution Alignment

**Principle II (Layered Architecture):** New `dealers` repository + service + handler; standard four-layer pattern.
**Principle VI (Data Integrity & Immutability):** Foreign key with `ON DELETE RESTRICT`; soft-merge tooling for duplicates.
**Principle VII (Schema-Driven Contracts):** New Dealer DTO; coin DTO extended with `dealer: { id, name }` projection.
**Principle X (Architecture Enforcement):** New repo/service must pass `architecture_test.go` rules.

## Implementation Notes

- New model: `Dealer { ID, UserID, Name (indexed), Type (enum), Website, Location, Notes, CreatedAt, UpdatedAt }`; unique index on `(UserID, Name)` (case-insensitive) to prevent duplicates per user
- Migration is two-phase:
  1. Add `DealerID` column (nullable) and create dealer rows from distinct existing source strings per user
  2. Keep original free-text field for one release as `legacy_source` (read-only fallback); remove in a later release
- Endpoints:
  - `GET /dealers` (typeahead + list, supports `?q=`)
  - `POST /dealers`
  - `GET /dealers/:id` (includes coin rollup)
  - `PUT /dealers/:id`
  - `DELETE /dealers/:id` (404 if referenced; surfaced via service-level check)
  - `POST /dealers/:id/merge` (target_id) — re-points all coins, soft-deletes source
- Frontend: shared Dealer picker component; admin tooling on portfolio page for "Dealers by spend" view
- AI agent: coin search agent can optionally surface dealer recommendations if a coin's dealer field links to a known auction house — defer to F003 follow-up; not in scope here

## Open Questions

- Should dealers be **per-user** (current proposal) or **global, with per-user references** (allows aggregate trust signals but raises moderation questions)?
- Should dealer URLs be validated for liveness at create time (similar to wishlist availability checks)?
- Backfill heuristic: case-insensitive exact match only, or fuzzy (Levenshtein) with a confirmation step?

## Dependencies

- None blocking. Coordinate sequencing with F002 (wishlist availability) if dealer URL liveness checks share infrastructure.

## References

- PRD §8 Q6 (Dealer/source tracking — resolved Yes, 2026-05-28)
- Constitution Principle X (Architecture Enforcement) — new repo + service must pass tests
