---
id: F021
title: "Re-verify NumisBids scraper against real account data"
status: backlog
priority: P1
effort: M
value: 4
risk: 3
owner: unassigned
created: 2026-07-19
updated: 2026-07-19
---

# F021 — Re-verify NumisBids scraper against real account data

## Summary

The CNG side of the auction-tracking feature (issue #482) was just audited against
**live, authenticated ground-truth data** from a real CNG account, and the scraper
was found to be reading JSON field names (`bid_amount`, `autobid`) that no longer
exist in CNG's current API response — the real current-bid and max-bid data live in
different, nested fields (`highest_live_bid.amount`, `absentee_bid.max_bid`) that
aren't in our `cngLot` struct at all. This was a silent, 100%-reproducible bug that
unit tests (built from assumed/fixture HTML) never caught, because the fixtures
encode the same wrong assumption the production code makes.

`numisbids_service.go` has the same structural exposure: it extracts current-bid
data via regex against HTML text (`Current\s+bid:\s*...`) captured during the
original integration work, and nothing has re-verified that markup against a real,
live NumisBids account with actual bid activity since. This sandbox's egress to
`www.numisbids.com` is blocked by a Cloudflare bot challenge (confirmed via
`cf-mitigated: challenge` response header), independent of the environment's
network allowlist, so this verification could not be completed as part of the CNG
audit and needs to happen on its own track.

**Goal**: do for NumisBids what was just done for CNG — fetch real authenticated
account pages (watchlist + at least one lot with active bid history), diff the
real markup/data against what `numisbids_service.go` assumes, and fix any drift
found, the same way the CNG scraper is being fixed.

## Acceptance criteria

- [ ] Real NumisBids watchlist page and at least one actively-bid-on lot page
      (HTML) captured from a live account and preserved as a test fixture.
- [ ] Every regex/extraction in `numisbids_service.go` (`currentBidRe`,
      `saleNameRe`, `saleDateRe`, `lotNumberRe`, `estimateRe`, `currencyValRe`,
      image extraction) re-verified field-by-field against the real fixture, not
      just against whatever fixture currently exists in
      `numisbids_service_test.go`.
- [ ] Confirm whether NumisBids exposes a "your max bid" / "you are winning"
      concept analogous to CNG's `absentee_bid.max_bid` /
      `highest_live_bid.registration_id`, and whether it's currently captured.
- [ ] Confirm whether NumisBids lots in a multi-lot sale have per-lot close times
      distinct from the sale-wide close time (the CNG equivalent of this —
      `lot.extended_end_time` vs `auction.effective_end_time` — was found silently
      wrong by ~2 hours in this audit).
- [ ] Any drift found is fixed with a regression test built from the real fixture,
      not a hand-authored one that could encode the same wrong assumption again.

## Constitution alignment

- Principle I (Clear Layered Architecture) — fixes stay inside
  `services/numisbids_service.go`; no handler/repository changes expected unless
  the CNG-side model changes (see the active CNG audit) require a shared field.
- Principle III (Strict Types and Explicit Contracts) — replace assumed markup
  shapes with ones verified against a real response.
- Principle IX (Automated Enforcement Over Manual Memory) — regression tests must
  be derived from a real fixture so this can't silently drift again unnoticed.
- §20 Audit & Continuous Improvement — this card exists because a spot-check of
  the sibling provider (CNG) found unverified assumptions had gone stale; the same
  check hasn't been run on NumisBids yet.

## Open questions

- [ ] Who can pull authenticated NumisBids HTML from a real account (view-source
      or saved response) for a lot with real bid activity, since this sandbox
      cannot reach numisbids.com directly (Cloudflare challenge, not a policy
      block)?
- [ ] Has NumisBids changed its markup/API at all since the original integration
      (unknown — no changelog/version tracking exists for third-party site
      structure)?
- [ ] Should this be promoted alongside the CNG fixes (`specs/NNN-...`) once
      ground truth is available, or handled as a standalone follow-up PR?

## Notes

Prior art: the CNG findings that prompted this card are in the
`claude/auction-functionality-review-17b5cj` session covering issue #482. Key
CNG findings for reference when doing the equivalent NumisBids pass:

1. Scraper read dead JSON field names (`bid_amount`, `autobid`) instead of the
   real nested fields (`highest_live_bid.amount`, `absentee_bid.max_bid`).
2. An extra per-lot HTTP scrape was solving a problem the watchlist list page
   had already solved — the list page carried full bid detail all along.
3. Lot-specific close time (`extended_end_time`) was being ignored in favor of
   the auction-wide close time (`auction.effective_end_time`), off by ~2 hours
   in the observed case — directly breaks bid-reminder timing.

Do not assume NumisBids has the same three problems — it's a different site with
a different (HTML/regex, not JSON) scraping approach — but do assume nothing
about it is verified until re-checked the same way.

## History

- 2026-07-19: created (status: backlog) — split out from the CNG-focused
  first-principles audit of issue #482 so CNG fixes aren't blocked on NumisBids
  account access.
