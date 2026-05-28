---
title: Coin of the Day
id: F001
status: promoted
priority: P1
effort: M
value: 4
risk: 2
owner: Cassius
created: 2026-05-28
updated: 2026-05-28
---

## Summary

Daily curated coin selection feature. System selects one coin per enrolled user daily via cyclic scheduler, sends in-app notification + Pushover integration, opens modal on notification click. Idempotent across process restarts.

## Acceptance Criteria

- When admin or daily scheduler runs, exactly one eligible coin (owned, non-wishlist, non-sold) per user is marked featured
- When notification is delivered and clicked, modal opens showing coin summary (cached at pick time)
- When scheduler runs on same calendar day twice, no duplicate coins selected
- When user opts out via Settings toggle, no notifications sent
- When coin has no AI analysis, fallback chain: Obverse + Reverse → structured fields → coin name

## Constitution Alignment

**Principle VI (Notifications):** In-app notification system with unread badges and Pushover integration.
**Principle IV (Database schema):** `FeaturedCoin` model tracks selection history and idempotency per day.

## Implementation Notes

- Selection algorithm in `PickNextCoinID()`: cycles via LEFT JOIN on `featured_coins`, order by (last_shown IS NULL DESC, last_shown ASC, id ASC)
- Dual idempotency: in-memory map[userID]string + DB check `HasBeenFeaturedToday()`
- Summary cached at pick time; modal renders without extra AI call
- User opt-in field: `User.CoinOfDayEnabled` (default true)
- Admin config: `CoinOfDayEnabled`, `CoinOfDayStartTime` (HH:MM 24h)

## Open Questions

None — feature shipped.

## Notes

Retroactive card created 2026-05-28 for governance traceability under Constitution §0 (Hierarchy) Phase 2. All endpoints exist: `GET /featured-coins/latest`, `GET /featured-coins/:id`, `POST /admin/coin-of-day/run`.
