---
updated_at: 2026-08-21T08:16:05Z
focus_area: Feature 356 — Valuation Journal Defragmentation & Tag Suggestion Restoration (implementation + review complete; all reviewer blocks cleared; awaiting merge)
active_issues: []
handoff_commit: pending (product code changes not yet committed per Scribe orchestration protocol)
---

# What We're Focused On

**Feature 356 — Valuation Journal Defragmentation & Tag Suggestion Restoration (COMPLETE, Ready to Merge)**

## Status Summary

Feature 356 implementation COMPLETE. Core backend + frontend DONE. All reviewer blocks cleared (Maximus CLEAR on B1–B4). Product changes remain uncommitted pending final orchestration. **Journal bloat eliminated**: scheduled AI value estimates removed from activity journal; value history is now the single source of truth. **Tag suggestions restored**: ruler-weight gate fixed; medium-confidence thematic tags now surface with balanced scoring. **Silent-failure bug fixed**: error states now visually distinct from empty results with explicit Retry.

## Key Deliverables

### Value Trend Table (D5 — Aurelia Frontend)
- Bounded scrolling table on CoinDetailValuationPage.vue: Date | Value | Change | Source columns
- Newest-first display; sticky headers; mobile-responsive
- Design tokens enforced; no hardcoded colors/radii/spacing
- 22 new tests; npm run type-check and npm run build pass

### Valuation History & Cleanup (D1–D4 — Cassius + Marcellus Backend)
- D1: Scheduled valuation journal writes removed; history is single record
- D2: CoinValueHistory.Source column added with confidence-based legacy backfill
- D3: On-demand estimates routed to history at apply time (source='ai_estimate')
- D4: One-time idempotent cleanup; legacy "Scheduled AI Value Estimate: $%" rows deleted on boot
- Marcellus: Fixed B1 (backfill WHERE clause correction) and B2 (error gating); added 4 migration-order regression tests
- All 11 Go packages pass; go build, go vet clean

### Tag Scoring Rebalance (Item 2 — Cassius)
- Ruler weight reduced from 0.45 to 0.30; weights flattened for balanced scoring
- Medium confidence floor (>= 0.45) now reachable by thematic tags
- Category/Material="Other" noise filtering added
- Brutus quantitative validation: 162 anonymized pairs; 47% medium-tier reachability; 12-suggestion cap not flooded

### Error Surface Fix (CoinTagsSection — Aurelia Frontend)
- Silent-failure bug fixed: error states visually distinct from empty results
- Four-branch template: loading | error+retry | empty | items
- Error cleared on retry success

## Review Cycle Summary

1. **Design Review** (Maximus, sync): D1–D5 design + Item 2 rebalance strategy; 7 risks identified
2. **Implementation** (Cassius/Aurelia/Brutus, background): All deliverables shipped
3. **Post-Implementation Review** (Maximus, sync): **BLOCK issued** on B1–B4
4. **Escalation**: Spawned **Marcellus** (Data Migration Engineer) for B1–B3; reassigned B4 to Brutus
5. **Revision** (Marcellus/Brutus, background): All findings resolved
6. **Revision Review** (Maximus, sync): **CLEAR / APPROVE** on B1–B4

## Non-Blocking Items (Deferred Post-Merge)

1. NB1: Stale comment in CoinDetailActionsPage.vue (Aurelia)
2. NB2: Empty Confidence on ai_estimate rows (document deviation)
3. NB3: Hardcoded "Other" literals → use constants
4. NB4: confidenceMeetsMinimum unknown-tier fallthrough
5. NB5: text-xs off type scale
6. NB6: Test gate branch unreachable (migrate test)
7. NB7: No drift guard for source backfill

## Product Code Status

✅ Implementation complete — All 11 Go packages pass; npm run type-check and npm run build pass
✅ Reviewer gates cleared — Maximus CLEAR on B1–B4; Brutus APPROVE on B4
✅ Test coverage verified — 4 migration tests + 22 Vue tests
⏸️ Uncommitted — Product code changes staged but not yet committed per Scribe protocol

## Previous Focus: Feature 355 Wishlist Purchase Reminders

Feature 355 complete and merged to beta. Implementation: timezone hotfix (Alpine zoneinfo portability), UX polish (reminder detail-row). All gates cleared; beta release pending operational validation (T034, T035).



