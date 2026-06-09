---
id: F014
title: "Build attribution and reference assistant"
status: backlog
priority: P1
effort: XL
value: 5
risk: 4
owner: unassigned
created: 2026-06-09
updated: 2026-06-09
---

# F014 — Build attribution and reference assistant

## Summary

Improve the existing coin intake and analysis foundation into an attribution and
reference assistant that drafts coin attribution from images, OCR text, existing
notes, and trusted sources. The assistant proposes ruler, mint, denomination,
date range, legends, catalog references, authority links, confidence, and
evidence. Users review before anything is saved.

## Acceptance criteria

- [ ] Assistant returns a structured attribution draft with field-level
      confidence and evidence.
- [ ] Suggested catalog references include source/citation and normalized
      reference fields compatible with existing `CoinReference` behavior.
- [ ] UI presents a review-and-apply flow where the user can accept individual
      fields, reject fields, or save the draft to notes.
- [ ] Agent never overwrites existing coin identity/reference fields without
      explicit user confirmation.
- [ ] Tests cover low-confidence results, conflicting sources, no-match results,
      and preserving existing user-entered data.

## Constitution alignment

- Principle II (Service Boundary Separation) — Python agent remains stateless;
  Go API owns persistence.
- Principle III (Strict Types and Explicit Contracts) — drafts use structured
  schemas.
- Principle IV (Simple Complete Changes) — build as reviewable slices, not a
  broad rewrite of coin editing.
- Principle V (Security, Auth, and Privacy by Default) — no cross-user data
  exposure; source URLs are validated.
- §17 Quality Gate, §21 Definition of Done.

## Open questions

- [ ] Which external reference sources are in v1: NGC, Numista, Wildwinds, RPC,
      OCRE/RIC, ACSearch, dealer pages, or a smaller starter set?
- [ ] Should confidence be numeric, low/medium/high, or both?
- [ ] Should attribution drafts be persisted as separate review records before
      being applied to a coin?

## Notes

This complements F012 and the existing `coin_intake` team. F012 gives agents
safe access to collection data; F014 adds deeper numismatic attribution skill
and evidence-backed reference drafting.

## History

- 2026-06-09: created (status: backlog).
