---
id: F032
title: "Show how each watched lot's bid moved since the last digest"
status: backlog
priority: P2
effort: S
value: 4
risk: 2
owner: unassigned
created: 2026-08-29
updated: 2026-08-29
---

# F032 — Show how each watched lot's bid moved since the last digest

## Summary

The Auction Watch Bid Digest lists every active watched lot with its current
high bid, but nothing else. A bid of "80.00 USD" is only meaningful against
what it was yesterday, so reading the digest means remembering — or scrolling
back to — the previous push and diffing it by eye. On a watchlist of a dozen
lots the one that actually moved is invisible. "Done" is: each lot in the
digest says how its bid compares with the bid the previous digest reported for
that same lot, and the digest is laid out so that comparison is the thing the
eye lands on.

## Acceptance criteria

- [x] Each lot's bid line says how the bid compares with the one the previous
      digest reported for that lot: up from, down from, or no change.
- [x] A lot appearing in a digest for the first time shows its bid with no
      comparison, rather than a misleading "no change".
- [x] The comparison baseline moves only when a digest is actually delivered —
      a failed push, or a lot trimmed off the end of a length-limited message,
      leaves the baseline alone so the change surfaces in the next digest
      instead of being lost.
- [x] A lot whose current bid the provider stops reporting keeps its previous
      baseline rather than having it erased.
- [x] Each lot renders as a scannable block: title with lot number, the sale,
      then the bid line — instead of one long title line and one long
      metadata-plus-bid line.
- [x] Provider catalog titles are shortened to their identifying clause so a
      digest of many lots stays readable and more lots fit inside Pushover's
      message limit.
- [x] The digest still trims itself with a summary line rather than exceeding
      Pushover's message limit.

## Constitution alignment

- Principle I (Clear Layered Architecture) — the baseline is persisted by a new
  `AuctionLotRepository` method, written only by the scheduler that delivers the
  digest; message composition stays in `services/`; no handler change.
- Principle IV (Simple Complete Changes) — one added column
  (`auction_lots.last_digest_bid`), one write path, no new table, no new
  scheduler, no API surface.
- §17 Quality Gate — unit tests cover every comparison branch, the
  first-digest case, delivery-gated persistence, trimming, and the
  updated_at guarantee.

## Open questions

- [x] Compare against the previous *digest*, or the previous value *sync*
      observed? — The previous digest. The digest's job is "what changed since I
      last told you"; a sync-observed baseline would report the last increment
      only (78 → 80) and hide the rest of the day's movement (75 → 80), and on a
      quiet sync would claim "no change" for a lot that has moved since the user
      last saw it.
- [x] Where does the baseline live? — `auction_lots.last_digest_bid`, written
      with `UpdateColumn` so digest bookkeeping never bumps `updated_at`, which
      the UI reads as "when this lot last changed". It is excluded from the JSON
      contract (`json:"-"`): it is bookkeeping, not lot data the UI renders, and
      keeping it out avoids a swagger regeneration for a field nothing reads.
- [x] Should the currency be repeated inside the comparison? — No. The
      comparison is always in the lot's own currency, so
      "80.00 USD (up from 75.00)" reads cleanly and keeps the line short.

## Notes

Requested with a worked example: the digest line

```text
PAMPHYLIA, Aspendos. Circa 380/75-330/25 BC. AR Stater (20mm, 10.85 g, 2h). VF.
Classical Numismatic Group - Electronic Auction 616 (Lot 337): current high bid 80.00 USD
```

should read

```text
PAMPHYLIA, Aspendos (Lot 337)
Classical Numismatic Group - Electronic Auction 616
- Current high bid: 80.00 USD (up from 75.00)
```

Implemented in `AuctionWatchBidDigestScheduler`. `buildAuctionWatchBidDigestMessage`
now returns the number of lots the body actually named (via a new
`buildBatchedLotMessageWithIncluded`, the trimming builder shared with the
newly-tracked-lots push), and `notifyUser` snapshots exactly those lots through
`AuctionLotRepository.SaveWatchBidDigestBids` after a successful send. Titles are
shortened by `auctionLotShortTitle`, which keeps the leading clause of a scraped
catalog description ("PAMPHYLIA, Aspendos. Circa 380/75-330/25 BC. AR Stater…" →
"PAMPHYLIA, Aspendos") and caps titles that have no sentence break. The lot
number moved onto the title line, so `auctionLotLabel` was split: the new
`auctionLotSaleLabel` is the "house - sale" half without it, and `auctionLotLabel`
is unchanged for the single-lot alerts that still use it.

Side effect of the shorter blocks: roughly half again as many lots now fit inside
Pushover's 1024-character limit before the digest starts trimming.

## History

- 2026-08-29: created (status: backlog) — requested feature; implemented and
  unit-tested (comparison branches, first digest, title shortening, delivery- and
  trim-gated persistence, updated_at guarantee). Status left at `backlog` pending
  Lead triage per `_backlog/README.md` — implementation does not self-advance
  status.
