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
| #784 | Slice 1 (shared toggle + status-badge primitives, design-system doc) reviewed PASS at `af2df567`. Slice 2 (typography scale + colour utilities, 170 arbitrary sizes and 154 raw `var()` classes to zero) reviewed PASS at `f265ff87`; owner review of the running app still outstanding. Slice 3 (casing + shared `.data-table` recipe) reviewed BLOCK at `8bdbbcb7` — five tables inlined the header recipe per-`th` and escaped both the codemod and the guard — repaired and reviewed PASS at `8dd0dbb7`. Slice 4 (colour tokens, overlays, and the light-theme WCAG AA failure on every status colour) implemented, awaiting review: literals 139 to 61, budget now per-file. The last 61 literals need an owner palette decision and the `text-xs` fold is deferred (see the slice 1 and slice 2 logs in `.squad/log/`) | slice 2 commit |

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

## G. Open owner decisions from #784 (blocking "accepted", not "implemented")

These are recorded here rather than only in `.squad/`, because agent-authored
handoff logs cannot supply owner authorization.

**Owner decision, 2026-09-30:** the owner accepted #784 and reported the visual
pass complete ("accepted. visual passed."). That resolves item 2 and the
cumulative palette shift in item 1b as shipped. Items 1 (the 65-literal
palette choice), 3 and 4 were not addressed by that decision and remain open
as follow-up work, not as blockers to #784.

1. **The remaining 65 template colour literals need a palette decision.** They
   are not oversights. The app carries near-duplicate values with no single
   winner: greens `#2ecc71` / `#27ae60` / `#229954` / `#4ade80`; reds
   `#e74c3c` / `#f87171` / `#ef4444` / `#c0392b`; ambers `#f39c12` / `#f59e0b`
   / `#e67e22` / `#f97316`; greys `#6b7280` / `#7f8c8d` / `#999999`.
   `AuctionLotDetailModal.vue` holds 16 in a Tailwind-ish palette distinct from
   the rest of the app; `CollectionHealthScorecard.vue` holds 16, mostly A-F
   grade-ramp colours that are deliberately *not* status tokens. Picking which
   green wins is a design call that cannot be made from the source.

1b. **The cumulative palette shift needs an explicit owner record.** Across
   three review rounds the status colours moved repeatedly, each move driven by
   a contrast guard that kept widening. This is the whole change, `8dd0dbb7`
   (before slice 4) to `1600e4f2`, in one place — no user has seen any of it.

   `:root`, which every theme except `light` inherits:

   | Token | Before | After | Seen in |
   |---|---|---|---|
   | `--color-negative` | `#e74c3c` | `#fb8476` | all 6 dark themes |
   | `--status-info-fg` | `#3498db` | `#54b1f0` | all 6 dark themes |
   | `--status-neutral-fg` | `#95a5a6` | `#a3b1b2` | all 6 dark themes |
   | `--confidence-high` | `#69b77f` | `#7abf8d` | all 6 dark themes |
   | `--confidence-low` | `#e08d8d` | `#e19090` | all 6 dark themes |

   The `light` theme previously had **no** status foreground overrides at all —
   it inherited the dark palette, which is why its badges were failing so badly.
   All eight are new:

   | Token | Inherited before | Light override now |
   |---|---|---|
   | `--color-positive` | `#2ecc71` | `#146e44` |
   | `--color-negative` | `#e74c3c` | `#a63125` |
   | `--text-warning` | `#f5c36a` | `#835c00` |
   | `--status-info-fg` | `#3498db` | `#1d6292` |
   | `--status-neutral-fg` | `#95a5a6` | `#546364` |
   | `--confidence-high` | `#69b77f` | `#296b44` |
   | `--confidence-medium` | `#f0c261` | `#835c00` |
   | `--confidence-low` | `#e08d8d` | `#a32316` |

   Part of the drift comes from requiring AA on `--bg-input` and
   `--bg-secondary` **composited**, surfaces where no badge is *demonstrated* to
   render. Requiring all five surfaces was the conservative choice — proving
   which surfaces host a badge is exactly the reasoning that failed four times —
   but it does buy accessibility with saturation. An owner may prefer a narrower
   surface set and brighter colours. That is a design call, not an engineering
   one.

2. **The visual pass is outstanding across all four #784 slices.** Screenshots
   were waived each time. Start in the **light theme**: every status colour in
   it changed, and the greens darkened noticeably. Then check a dark theme's
   badges — the error red lightened in all five.

3. **`text-xs` is deferred, not dropped.** It duplicates `text-sm` at
   `0.75rem` across 35 call sites but carries a different Tailwind line-height
   (1.3333 vs 1.4286), so folding them is a ~7% height change on those
   elements, not a rename. It needs its own slice and its own visual check.

4. **Three guard gaps stay open.** The `<style scoped>` hex guard is still a
   single net total (190), and a colour literal moved into `<script>` is
   counted by no guard at all.

   The concrete instance of that second gap is
   `components/AuctionLotCard.vue:161-165`: `biddingIndicator.badgeCls` builds
   `'bg-[#4ade80] text-[#052e13]'` / `'bg-[#f87171] text-[#450a0a]'` in
   `<script>` and applies it at line 25 via `:class="biddingIndicator.badgeCls"`.
   That "Winning"/"Outbid" pill escapes the contrast guard, the non-token
   pairing check **and** the template literal budget, all three of which read
   templates only. It measures **8.57:1**, so there is no accessibility failure
   today — but it is four hardcoded colours outside every ratchet.

   A related blind spot in the contrast guard: it matches `bg-status-*-bg`
   fills but not `bg-status-*-tint`, so the tint pairings in
   `AdminAISection.vue:52, 97, 105` are not derived. Measured at 4.73 and 4.86
   on light `--bg-secondary`, so again passing, but the fill set is listed
   rather than derived — the same shape as the defect that took four rounds to
   fix. Closing any of these is a separate change.
