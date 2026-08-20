# Collection Time Machine

> Scrub back through time and see your collection exactly as it stood on any past date.

## Overview

Every other view in the app shows the collection as it is **now**. The Time
Machine (`/stats/time-machine`) shows it as it **was**: what you owned on a
given date, what it was worth then, how it was distributed across categories,
materials, and eras, and what your largest holdings were.

It is built entirely from data the app already records — no new tracking, and
nothing to backfill. Drag the slider, jump with the preset buttons ("1 year
ago", "The beginning"), or type an exact date.

## How a date is reconstructed

**Which coins you owned on date D:**

- The coin is not a wish list entry, and
- it has a purchase date on or before D (the purchase date itself counts — scrub
  to the day you bought a coin and you will see it), and
- it was either never sold, or sold *after* D.

**What each coin was worth on date D:** the most recent valuation recorded at or
before D, taken from the per-coin valuation history the valuation scheduler
writes. When no valuation had been recorded yet, the purchase price is used
instead.

**Collection health score:** the most recent daily health snapshot at or before
D. Dates before health snapshots began simply show no score rather than a
misleading one.

## Honesty about the numbers

Historical reconstruction is only as good as the history that was recorded, so
the page is explicit about its own limits rather than presenting every figure
with equal confidence:

- **Value basis.** Below the summary cards, the page states how many coins used
  a real recorded valuation versus a purchase-price fallback — for example,
  *"4 of 5 coins had no recorded valuation by this date; those use purchase
  price."* Early dates, before the valuation scheduler had ever run, say so
  outright. Individual holdings valued from purchase price are marked with `*`.
- **Undated coins.** Coins with no purchase date cannot be placed anywhere on a
  timeline. They are excluded from every figure, and the page reports how many
  there are so the totals are understood as partial rather than wrong.
- **Wish list history is not reconstructed.** The app records only a coin's
  *current* wish list flag, not when it changed. A coin now in the collection
  that was on your wish list on date D is judged by its purchase date, which is
  the correct answer for ownership.

## What it is useful for

- **Seeing progress.** Compare "the beginning" with today and watch the mint
  map, category mix, and value fill in.
- **Answering "when did I…".** Scrub to find the date a category first appeared
  or the collection crossed a value threshold.
- **A year in review.** The "Added That Year" figure counts acquisitions in the
  twelve months before the selected date — the raw material for a shareable
  year-end summary or a [Showcase](collection-showcase.md).

## API

| Endpoint | Purpose |
|---|---|
| `GET /api/stats/time-machine?date=YYYY-MM-DD` | The collection as of that date |
| `GET /api/stats/time-machine/bounds` | Earliest acquisition on record through today; `hasData: false` when nothing is dated |

Both require authentication and are scoped to the calling user. A future date
returns `400`.

## Notes and limits

- Valuation history only exists from the point the valuation scheduler first
  ran, so the further back you scrub the more the figures lean on purchase
  price. This is disclosed on the page rather than hidden.
- Sold coins leave the timeline on their sold date; realized gains are not part
  of this view. See [Sold Coins](sold-coins.md).
- Dates are handled in UTC and a date is treated as end-of-day.

## Related

- [Collection Statistics](statistics.md)
- [Collection Health Scorecard](../../specs/208-collection-health-scorecard)
- [Price Trends](price-trends.md)
