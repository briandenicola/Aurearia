# Session Log: Format Currency Refactor (2026-04-28T18:16)

## Brief

Aurelia completed frontend refactor: consolidated 6 local `formatCurrency()` implementations into centralized `src/web/src/utils/formatters.ts`. Enhanced signature with optional currency parameter. All callers updated. Type check clean. Committed.

## Files Modified
- Centralized: `utils/formatters.ts`
- Updated: `CoinCard.vue`, `coins.ts` (store), `StatsPage.vue`, `AdminPage.vue`, `CoinDetailPage.vue`, `FollowersPage.vue`

## Result
✅ No regressions. Ready for merge.
