# Open issue triage and fix plans (#766-#783), baseline main 57f00398

## Status (updated 2026-09-29, branch beta)
| Group | Status | Commit |
|---|---|---|
| A. #768 | Done | be5dd3e4 |
| B. #770, #772, #769 | Done | a363e4aa |
| C. #776, #775, #777 | Done | commit that adds this file |
| D. #766 | Not started; needs its own spec | - |
| E. #774 | Done (see `.squad/log/2026-09-29-listing-relevance-774.md`) | commit that adds that log |
| E. #779, #771 | Done (see `.squad/log/20260929T213823Z-comparables-price-range-and-dealer-adapters.md`) | commit that adds that log |
| F. #783 | Done (context budget, see `.squad/log/2026-09-29-context-budget-783.md`) | commit that adds that log |
| F. #780 | Done (see `.squad/log/2026-09-29-gofmt-780.md`) | commit that adds that log |
| F. #781 | Done (see `.squad/log/2026-09-29-create-agent-781.md`) | commit that adds that log |
| #784 | Slice 1 (shared toggle + status-badge primitives, design-system doc) reviewed PASS at `af2df567`. Slice 2 (typography scale + colour utilities, 170 arbitrary sizes and 154 raw `var()` classes to zero) implemented; awaiting review and owner review of the running app. Casing, table density and the remaining 139 colour literals are still open (see the slice 1 and slice 2 logs in `.squad/log/`) | slice 2 commit |

Owner-approved local setup on 2026-09-29 (this machine): Go 1.27.1,
`task setup:go`, `task setup:openapi`, Python 3.12 via uv and
`task setup:agent`. Owner acceptance of group F is separate from its
reviews. No deployment or release is authorized.

Note on #775: the shipped fix differs from the plan below. The web now uses a
label map (`src/web/src/utils/copilotToolLabels.ts`) with a drift test against
the agent's `_TOOL_LABELS` and Go `CoinCopilotAllowedTools`, so the frame
contract did not change.

## Priority order
| P | Group | Issues | Why |
|---|---|---|---|
| P1 | A. Scheduler timing | #768 | User-visible bug: notices at wrong time; 11 schedulers |
| P1 | B. Sold/stale listings | #770, #769, #772 | Wrong data shown as "available"; vCoins WAF block |
| P2 | C. Copilot UX + admin gaps | #776, #775, #777 | Small, cheap, remove confusion |
| P2 | D. Copilot parity | #766 | Regression vs Coin Agent; largest change, high-risk (cross-service contract) |
| P3 | E. Search quality/coverage | #774, #771, #779 | Enhancements; depend on B |
| P3 | F. Hygiene | #780, #781, #783 | No user impact; #781 is a time bomb on LangGraph 2.0 |

## A. #768 Scheduler start-time + time zone (Go + web, high-risk: settings contract)
- Cause verified: 11 schedulers in src/api/services sleep once via time.After(wait); set_snapshot/collection_health/reminder/wishlist/bid_digest roll anchors with Add(24h) (DST drift).
- Plan: extract Coin of the Day's #762 logic into a shared helper in scheduler_contract.go
  (bounded 1-min tick, recompute next run, calendar-date anchor in a zone). Add one
  app-wide ScheduleTimezone setting (IANA, empty = server time); CoinOfDayTimezone falls back to it.
  Migrate schedulers one by one; show "Runs daily at HH:MM <zone>" in Admin > Schedules.
- Tests: helper unit tests (change applies mid-wait, DST spring/fall, invalid zone rejected), per-scheduler smoke; web component test for zone text.
- Gates: task check:go, task check:web. Non-goal: per-job zones.

## B. Sold/stale listings (agent)
1. #770 (small, first): drop bare "purchase" in search.py:497 AND buy_indicators at :225; add reserved/on hold/no longer for sale/this item has sold; weak signal -> unknown. Extend existing test.
2. #772: per-process dealer cool-down on 202+x-amzn-waf-action / 429 (honour Retry-After), short TTL cache per (dealer, keywords) in RegisteredDealerHttp; warning text + log lines. No challenge workarounds.
3. #769: route create_coin_search_team through run_market_search provider logic, map SpecialistResult -> CoinSuggestion, same budget rule; update parity test + guide.
- Gates: task check:agent (ruff + pytest). Non-goal: new dealers (#771).

## C. Copilot UX/admin (web, small)
- #776: AttributionEnabled toggle in AdminSystemSection Coin Copilot block + dependency hint; component test; guide admin section.
- #775: send `label` from _TOOL_LABELS on tool_started/tool_completed frames; CopilotRunProgress uses it (fallback to existing title-case only if absent); agent + web tests; collapse guide table. Touches frame contract -> Go passthrough check.
- #777: HelpSection Coin Copilot entry (no show-search claim until #766); update HelpSection.test.ts.
- Gates: task check:web, check:agent (for #775), check:go if frames pass through Go validation.

## D. #766 Copilot coin show search (Go + agent + web, high-risk)
- Needs its own spec/plan per constitution lane (new tool, new evidence kind, new request fields incl. ZIP = PII boundary).
- Order: agent specialist wrapping coin_shows team -> Go allowed tools + evidence validation + execute request (coin_shows_prompt, zip) -> web evidence card -> prompt -> docs. Then update #777 Help text.
- Gates: all three check targets + parity test. Recommend independent review.

## E. Enhancements (after B)
- #774 relevance filter (fail-open structured call over titles), #771 per-site decision + adapters (ToS research first), #779 comparables for Quick Identify (reuse dealer search; bounded, fail-open).

## F. Hygiene
- #780: gofmt -w the 17 files (formatting-only commit) + add gofmt -l to task check:go.
- #781: migrate provider.py:99 and collection_chat.py:19 to langchain.agents.create_agent; may need dependency lock change (needs approval).
- #783: rewrite now.md to current focus; archive superseded decisions; make budget warning list per-file counts (+ governance.test.mjs). Budget change itself = owner decision.
