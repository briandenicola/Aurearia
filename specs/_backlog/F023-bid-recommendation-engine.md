---
id: F023
title: "Suggest a maximum bid based on prior wins/losses and market data"
status: backlog
priority: P2
effort: L
value: 3
risk: 3
owner: unassigned
created: 2026-07-19
updated: 2026-07-19
---

# F023 — Suggest a maximum bid based on prior wins/losses and market data

## Summary

From issue #482 (comment, 2026-07-17): "Add a recommendation engine to suggest a
maximum bid amount with a good chance of winning based on prior wins/losses and
maybe online search of similar coins." This is a feature request layered on top
of — not part of — the auction bug-fix work tracked directly on #482; it depends
on auction status data (won/lost/winningBid) actually being trustworthy, which is
the subject of the CNG rebuild (see commit `0943a56` on
`claude/auction-functionality-review-17b5cj`) and the NumisBids parity work
(F022). A recommendation trained on wrong won/lost/winningBid data would just be
confidently wrong.

## Acceptance criteria

- [ ] Given a tracked auction lot (watching or bidding), the app can suggest a
      maximum bid with some stated confidence/rationale, not just a bare number.
- [ ] The suggestion is grounded in the user's own prior `AuctionLot` history
      (won lots' `winningBid` vs. `estimate`, lost lots' losing `maxBid` vs. the
      winning amount) for comparable coins (same category/denomination/era, or
      closest available match).
- [ ] Optionally incorporates external market data — the existing Team 9 (price
      trends, `src/agent/app/teams/price_trends.py`) already searches auction
      results and analyzes price direction for a described coin type; this
      should reuse that pipeline rather than duplicating web-search logic.
- [ ] Recommendation is surfaced in the auction lot UI (detail modal or card) as
      an optional aid, not an autofilled/auto-submitted bid — the user places
      bids on the provider's own site, this app only tracks and suggests.
- [ ] Works with too little history (new user, no won/lost lots yet) by falling
      back to estimate/market-data-only reasoning, and says so rather than
      presenting a number with false confidence.

## Constitution alignment

- Principle II (Service Boundary Separation) — recommendation logic that uses
  LLM reasoning or web search belongs in the Python agent service (likely a new
  team, or an extension of Team 9), not the Go API; the Go API only supplies the
  user's own auction/collection history data per-request, since the agent is
  stateless and has no direct DB access.
- Principle III (Strict Types and Explicit Contracts) — new agent request/response
  schemas must use Pydantic models (`app/models/`); any new Go endpoint needs
  Swagger annotations.
- Principle VI (Consistent User Experience) — must read as a suggestion/aid
  consistent with how the rest of the app presents AI-assisted analysis (e.g.
  coin analysis, portfolio review), not a new interaction pattern.

## Open questions

- [ ] Is this a new agent team, or an extension of Team 9 (price trends) /
      Team 5 (auction search)? Leans toward extending existing teams given
      overlap in "search auction results for similar coins."
- [ ] How much prior history is "enough" to base a recommendation on, versus
      falling back to market-data-only?
- [ ] Should the recommendation consider the user's own stated budget/collecting
      priorities anywhere, or purely historical + market signal?
- [ ] Does this need a new endpoint, or can it ride on the existing agent chat
      surface (ask the agent about a specific tracked lot)?

## Notes

Blocked in practice (not formally) on won/lost/winningBid data being correct —
training or reasoning over data produced by the pre-fix CNG scraper (which
silently showed the wrong current bid and never auto-resolved won/lost) would
have produced a recommendation engine confidently wrong in ways impossible to
notice from the UI. Recommend sequencing this after F022 lands, or at minimum
scoping it CNG-only until NumisBids reaches the same verified state.

## History

- 2026-07-19: created (status: backlog) — split out from issue #482's comment
  thread; kept separate from the bug-fix work since it's a net-new feature, not
  a fix to reported functionality.
