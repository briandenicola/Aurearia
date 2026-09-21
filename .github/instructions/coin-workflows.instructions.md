---
applyTo: "src/api/**,src/web/**"
---

# coin-workflows guidance

Apply the [constitution](../../.specify/memory/constitution.md) and selected approved work. These scoped instructions implement, not replace, that authority.

### Notable Endpoints & Features

- **AI Provider Status:** `GET /ai-status` returns `{ provider, available, model, message }`. Frontend uses this provider-agnostic check before AI analysis instead of legacy `/ollama-status`.
- **Random Gallery Sort:** Collection list accepts `?sort=random&seed=N` where `N` is an integer (validated via `strconv.Atoi`). Order is `((id * seed) + seed) % 2147483647` for SQL-safe deterministic shuffle. Frontend persists the seed in `sessionStorage` under `coins:randomSeed` for stable pagination within a session.
- **Coin of the Day:** Daily scheduler picks one coin per enrolled user, sends in-app notification + Pushover. Clicking the notification opens `FeaturedCoinModal` (not a route).
  - Admin settings: `CoinOfDayEnabled`, `CoinOfDayStartTime` (24h `HH:MM`)
  - Per-user opt-in field: `User.CoinOfDayEnabled` (default `true`) — surfaced as toggle in Settings → Account
  - Endpoints:
    - `GET /featured-coins/latest` — most recent for the current user
    - `GET /featured-coins/:id` — fetch one (user-scoped); preloads `Coin.Images`
    - `POST /admin/coin-of-day/run` — admin manual trigger; returns `{ picked, skipped, errors }`
  - Notification type: `coin_of_day`; `referenceId` is the `FeaturedCoin.ID` (NOT a coin id).
  - Selection algorithm (`PickNextCoinID`): cycles through every owned, non-wishlist, non-sold coin via LEFT JOIN on `featured_coins` (`ORDER BY (last_shown IS NULL) DESC, last_shown ASC, c.id ASC`); each coin appears once before any repeats.
  - Dual idempotency: in-memory `map[userID]string` + DB check `HasBeenFeaturedToday` — safe across process restarts on the same day.
  - Summary is cached at pick time (`buildCoinSummary` fallback chain: `AIAnalysis` → `Obverse + Reverse` → structured fields → bare name) so the modal renders cached prose without an extra AI call.

## Completion

Select applicable Go/web gates and test the complete affected workflow and sibling paths. See [testing](../../docs/testing.md#6-running-tests-locally-vs-ci).
Setup/installations require separate authorization; missing execution is incomplete.
