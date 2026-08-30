---
id: F033
title: "Give the auction price alert the digest's lot block"
status: backlog
priority: P2
effort: XS
value: 3
risk: 1
owner: unassigned
created: 2026-08-29
updated: 2026-08-29
---

# F033 — Give the auction price alert the digest's lot block

## Summary

The watch-bid digest was reshaped into a scannable lot block with a linked lot
number (F032); the auction price alert was left in the old shape. It leads with
a full scraped catalog title, buries the lot number inside the sale line, and
reads `Current bid: current high bid 475.00 USD` — a stutter from prefixing a
value that already names itself. It also states the target and the bid without
connecting them, leaving the user to subtract the number the alert exists to
report. Worse in-app: the notification card renders the message in a plain
element, so the line breaks it is composed with collapse and the whole thing
arrives as one paragraph. "Done" is: a price alert that reads like a digest
entry on both surfaces, and says how far past the target the bid actually went.

## Acceptance criteria

- [x] The alert leads with the lot's shortened title and its lot number, then
      the sale on its own line, then the target and bid lines.
- [x] The bid line states the gap to the target — `475.00 USD (275.00 over
      target)`, `180.00 USD (20.00 under target)`, `200.00 USD (at target)` —
      and reads correctly for an alert watching for a rise or for a fall.
- [x] The stuttering `Current bid: current high bid …` is gone.
- [x] The Pushover push is rich HTML with the lot number linked to the lot on
      the auction site; scraped titles and sale names are escaped, and a lot URL
      that is not an absolute http(s) URL is neither linked nor offered as the
      push's action URL.
- [x] The in-app notification carries the same lines without markup, and the
      notification card preserves the line breaks instead of collapsing them.
- [x] A lot with no current bid still notifies, showing the bid as unavailable.

## Constitution alignment

- Principle I (Clear Layered Architecture) — composition stays in
  `services/notification_service.go`; the lot-rendering helpers it shares with
  the digest live with the other lot helpers in `services/auction_alert_service.go`.
  No repository, handler or model change.
- Principle IV (Simple Complete Changes) — one notification reshaped, plus the
  one-class fix that makes every notification card honour the newlines its
  message already contains.
- Principle V (Security, Auth, and Privacy by Default) — moving the push to HTML
  brings F031's escaping rule with it, and the lot URL is scheme-validated
  before it is rendered as a link or handed to Pushover.
- §17 Quality Gate, §21 Definition of Done.

## Open questions

- [x] Keep the full catalog title, since a single-lot alert has room for it? —
      No. The alert is read on a lock screen, and the identifying clause plus
      the linked lot number is what makes it scannable; the full description is
      one tap away through the alert's own link. This matches the digest, so the
      two notifications describe a lot the same way.
- [x] Compare the bid against the previous bid, as the digest does? — No. The
      digest's baseline is "what the last digest told you", which a price alert
      firing between digests would misreport. The target is the comparison this
      notification is about.
- [ ] The bid reminder (`NotifyAuctionBidReminder`) and the ending-soon
      notification share the old shape and the same `Current bid: current high
      bid …` stutter. Reshaping them is the obvious follow-up, deliberately left
      out of this card because only the price alert was requested.

## Notes

Reported from a live alert: `Vespasian. AD 69-79. AR Denarius (16.5mm, 3.22 g,
6h). Rome mint. Struck July-December AD 71. Near VF. / Classical Numismatic
Group - Electronic Auction 616 (Lot 684) / Target: 200.00 USD / Current bid:
current high bid 475.00 USD`, which now reads:

```text
<b>Vespasian</b> (<a href="…">Lot 684</a>)
Classical Numismatic Group - Electronic Auction 616
- Target: 200.00 USD
- Current high bid: 475.00 USD (275.00 over target)
```

`auctionLotShortTitle` and the headline helpers moved from the digest scheduler
to `auction_alert_service.go`, where the other shared lot-rendering helpers
live, and gained a plain-text variant (`auctionLotHeadline`) for the in-app
body. `formatAuctionTargetBid` derives "over"/"under" from the numbers, so the
alert's direction does not have to be plumbed into the notification service.
The push URL now goes through `auctionLotProviderURL`: Pushover rejects a
malformed `url` outright, which would have cost the whole notification.

The in-app fix is one Tailwind class (`whitespace-pre-line`) on the notification
card's message element. It applies to every notification type, all of which
compose multi-line bodies server-side.

## History

- 2026-08-29: created (status: backlog) — requested feature; implemented and
  unit-tested (target-gap wording in both directions and at the target, missing
  bid, link rendering, escaping, unusable-URL rejection, both surfaces' bodies).
  Status left at `backlog` pending Lead triage per `_backlog/README.md` —
  implementation does not self-advance status.
