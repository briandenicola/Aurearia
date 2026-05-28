---
title: Public Per-Coin Share Links
id: F008
status: promoted
priority: P2
effort: M
value: 3
risk: 2
owner: unassigned
created: 2026-05-28
updated: 2026-05-28
---

## Summary

Allow a logged-in collector to generate an unauthenticated, read-only share link for a single coin (mirroring the existing showcase pattern at coin granularity). The recipient sees a stripped-down public coin page — images, attribution, descriptive metadata — without account access, edit controls, valuation fields, or owner-private notes. Links can be revoked.

## Acceptance Criteria

- When the owner clicks "Share" on a coin detail page, the app issues an opaque, non-guessable share token and surfaces a copyable URL
- When an unauthenticated visitor opens the share URL, they see a public, read-only render of the coin (images + public-safe fields only)
- When the owner revokes a share link, subsequent requests return 404 (not 410, to avoid confirming token format)
- When a coin is deleted or marked sold, all active share links are auto-revoked
- When a coin is marked private/wishlist, share links cannot be created
- Public-safe field set is explicit and documented (no purchase price, no private notes, no dealer info)
- Rate-limited per IP to deter scraping

## Constitution Alignment

**Principle XI (Security Hardening):** Tokens are opaque, ≥128 bits of entropy, single-purpose; field allowlist prevents data leakage.
**Principle XII (Authentication & Token Policy):** Share tokens are distinct from JWT/refresh tokens; no session implied; revocable.
**Principle II (Layered Architecture):** New `share_links` repository + service; thin handler at `/p/coin/:token`.
**Principle VII (Schema-Driven Contracts):** Public response DTO is explicit, separate from authenticated coin DTO.

## Implementation Notes

- New model: `CoinShareLink { ID, CoinID, Token (unique, indexed), CreatedAt, RevokedAt }`
- Endpoints:
  - `POST /coins/:id/share` (auth required) — issues token
  - `DELETE /coins/:id/share/:token` (auth required) — revokes
  - `GET /p/coin/:token` (public) — read-only render
- Public DTO whitelist: `images, name, attribution, mint, denomination, era, category, obverse, reverse, inscriptions, references` — explicitly exclude `purchasePrice, currentValue, dealer, privateNotes, location, condition_notes_private`
- Frontend: new public route `/p/coin/:token`, distinct layout (no app chrome), Open Graph meta for link previews
- Cascade: deleting a coin or flipping to `sold`/`private` must revoke all share links in the same transaction

## Open Questions

- Should share links carry an optional expiry (e.g., 30 days, 1 year, never)?
- Should we render Open Graph preview cards with the obverse image by default?
- Do we want a per-user "share-link audit" page (active tokens, last access timestamp)?

## Dependencies

- None blocking. Aligned with existing showcase patterns (`src/api/handlers/showcase_handler.go`); reuse field-allowlist pattern.

## References

- PRD §8 Q1 (Public link sharing — resolved Yes, 2026-05-28)
- Existing showcases as prior art for unauthenticated read access
