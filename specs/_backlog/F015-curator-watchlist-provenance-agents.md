---
id: F015
title: "Add curator, watchlist, and provenance agents"
status: backlog
priority: P1
effort: XL
value: 5
risk: 4
owner: unassigned
created: 2026-06-09
updated: 2026-06-09
---

# F015 — Add curator, watchlist, and provenance agents

## Summary

Level up existing portfolio review, gap analysis, availability checking, and
similar-lot capabilities into collector-facing workflows: a curator agent for
gaps/themes/next buys, a watchlist agent for matching lots to goals, and a
provenance/risk agent for missing provenance, suspicious listings, duplicate
images, and documentation gaps.

## Acceptance criteria

- [ ] Curator agent can explain collection strengths, gaps, themes, and suggested
      next acquisitions using owned collection data.
- [ ] Watchlist agent can evaluate candidate lots against user goals, budget,
      duplicates, and existing collection coverage.
- [ ] Provenance/risk agent flags missing provenance, suspicious claims,
      duplicate images, broken source links, and coins needing better
      documentation.
- [ ] Recommendations include citations/evidence, confidence, and a clear
      "why this matters" explanation.
- [ ] User preferences such as budget, favorite periods, disliked categories,
      preferred dealers, and collecting goals are configurable and respected.
- [ ] No agent writes changes without confirmation and journal/audit metadata.

## Constitution alignment

- Principle II (Service Boundary Separation) — agent service performs inference;
  Go API owns data and writes.
- Principle III (Strict Types and Explicit Contracts) — recommendations and risk
  findings use schemas.
- Principle IV (Simple Complete Changes) — ship one agent slice at a time.
- Principle V (Security, Auth, and Privacy by Default) — personal collection
  preferences and private coins remain user-scoped.
- §17 Quality Gate, §21 Definition of Done.

## Open questions

- [ ] Should user collecting preferences live in admin settings, account
      settings, or a dedicated collector profile model?
- [ ] Should watchlist inputs come from wishlist coins, manual goals, auction
      URLs, saved searches, or all of them?
- [ ] What confidence/evidence threshold is required before surfacing a
      provenance risk warning?

## Notes

This card should follow F012 read tools and benefit from F014 attribution
evidence. It should improve and unify existing collection-analysis teams rather
than duplicate them. The first slice should be read-only recommendations.

## History

- 2026-06-09: created (status: backlog).
