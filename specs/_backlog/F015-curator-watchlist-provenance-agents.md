---
id: F015
title: "Add collector context, curator guidance, and wishlist capture"
status: promoted
priority: P1
effort: XL
value: 5
risk: 4
owner: unassigned
created: 2026-06-09
updated: 2026-09-18
---

# F015 — Add collector context, curator guidance, and wishlist capture

**Promoted to**: [`specs/363-collector-curator-watchlist-provenance/`](../363-collector-curator-watchlist-provenance/)

The active Feature 363 specification is now the source of truth. This card is
retained only as the historical backlog pointer required by the SpecKit
workflow.

## History

- 2026-06-09: created (status: backlog).
- 2026-09-18: promoted to Feature 363 on the existing `beta` branch. Product
  decisions settled collector-profile storage, wishlist/goals/Coin-Copilot
  inputs, tiered needs-review risk language, and explicit confirmed wishlist
  actions.
- 2026-09-19: reduced to lightweight private collector context, read-only
  curator guidance, and restoration of the existing dealer-result **Add to
  Wishlist** action. Added native **Add to Wishlist by URL** intake based on
  the existing n8n workflow: normalize and validate one listing URL, prevent
  owner-scoped duplicates, retrieve one bounded page, extract reviewable coin
  fields without guessing, preserve listing status and provenance, optionally
  attach one safe image, and create through the canonical wishlist flow only
  after explicit owner confirmation. This does not add an n8n dependency,
  touch auction tracking, or revive staged wishlist-action/audit tables.
  Cleaned HTML evidence will reuse Deep Analysis' evidence-to-structured-coin
  projection instead of introducing a second extraction subsystem.
