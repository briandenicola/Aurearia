# Session Log: Comparables Price Range (#779) and Dealer Adapter Decisions (#771)
**Timestamp:** 2026-09-29T21:38:23Z
**Branch:** beta (owner-authorized direct work)

## Agent Outcome
Implementation owner — closed out #779 and #771 together in one changeset.

### #779 — Ground Quick Identify's price range in real listings
Quick Identify's opt-in price range now runs one bounded dealer search through the
canonical `run_market_search` workflow and reports current dealer listings. The
vision model's estimate stays as a clearly labelled fallback.

- New stateless agent endpoint `POST /api/search/comparables` wrapping
  `run_market_search` (no persistence, fails open to an empty partial result).
- Go `CoinLookupService` calls it only when the price opt-in is set, with a 25s
  timeout, and falls back to the model estimate on any failure.
- Web shows a **Current Dealer Listings** card with the range, linked listings
  and dealer names; the range is carried into the draft notes.

### #771 — Direct adapters for the remaining dealer sources
Written decision per site, based on live probes, rather than new adapters:

| Site | Evidence | Decision |
| --- | --- | --- |
| forumancientcoins.com | HTTP 202 + `x-amzn-waf-action: challenge` on every request including `robots.txt` | No adapter. Stays on host-restricted web search; we never work around a challenge. |
| catawiki.com | Akamai HTTP 403 on `robots.txt` and search, even with a browser UA; ToS bars automated querying | No adapter. Stays on host-restricted web search. |
| biddr.com | `robots.txt` allows all (Crawl-delay 3); `/search?s=` is server-rendered; results are timed auction lots with bids | Moved from the default dealer sources to the default auction sources. |

Only default values changed; installations that set their own source lists in
Admin are untouched.

## Review
Independent read-only review (aurearia-reviewer) returned INCOMPLETE on the
first #779 candidate. Addressed in this changeset:

1. **Blocking** — unverified listings (`verification_state != "verified"`, i.e.
   the dealer page was never fetched and the coin may already be sold) were
   shown under "Current Dealer Listings". Now dropped.
2. **Blocking** — the fallback copy claimed "No current dealer listings were
   found" even when the search never ran. Reworded to state only that no dealer
   listing was available to price against.
3. Range ends are now both linked: the shown set keeps the cheapest listings
   plus the dearest one.
5. Dead `currency` field removed from the comparables contract on both sides.
6. The comparables card uses `SafeExternalLink` like the other external links.

Deferred with rationale: (4) agent-side cancellation plumb-through for the 25s
budget and (7) `sanitizeAgentErrorBodyForLog` in `SearchComparables` — both are
pre-existing patterns shared with `DiscoverAlertCandidates`, so they belong in a
separate change that fixes every call site.

## Verification
- `task check:go` ✅ (build, vet, gofmt, full test suite)
- `task check:agent` ✅ (ruff + 766 tests)
- `src/web`: `npm run lint` ✅ (`--max-warnings 0`), `npm run type-check` ✅
  (`vue-tsc --build`), `npm run test` ✅ (1736 passed, 1 skipped)
- `task openapi` regenerated; artifacts committed here so `task check:openapi`
  is clean afterwards.
- Tamper tests (each broke a test, then reverted):
  - removed the `verification_state == "verified"` filter →
    `TestSummarizeComparablesDropsUnusableListings` FAIL
  - reverted range-end retention to a plain truncate →
    `TestSummarizeComparablesKeepsBothEndsOfTheRangeLinked` FAIL
  - put `biddr.com` back in the dealer defaults →
    `TestSearchSourceDefaultsAreSeparated` FAIL

## Known Issue (pre-existing, not from this change)
`npm run build` in `src/web` fails with
`Rolldown failed to resolve import "@fontsource/inter/300.css"`.
`@fontsource/inter` and `@fontsource/cinzel` are declared in `package.json` but
`node_modules/@fontsource` is absent. Needs an owner-authorized `npm ci`.

## Files Changed
- src/agent/app/models/requests.py, responses.py
- src/agent/app/routes.py
- src/agent/app/teams/coin_search.py
- src/agent/app/tools/dealer_sites.py (written per-site decision in the docstring)
- src/agent/tests/test_comparables_search.py (NEW), test_dealer_sites.py
- src/api/services/agent_proxy.go, coin_lookup_service.go, settings_service.go
- src/api/services/coin_lookup_service_test.go, settings_service_test.go
- src/api/handlers/coin_lookup.go, swagger_types.go
- src/api/docs/{docs.go,swagger.json,swagger.yaml}, docs/openapi.json
- src/web/src/types/coin.ts, src/web/src/pages/CoinLookupPage.vue
- src/web/src/pages/__tests__/CoinLookupPage.test.ts
- docs/features/coin-lookup.md, ai-search-agent.md, how-coin-copilot-works.md

## Next Action
Re-review #779 against the committed SHA, then discuss #766 and #784 with the
owner (the owner may want #784 first). No release or deployment is authorized.
