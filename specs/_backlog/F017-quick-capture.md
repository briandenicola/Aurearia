---
id: F017
title: "Add quick capture for new coins"
status: promoted
priority: P1
effort: M
value: 5
risk: 3
owner: Maximus
created: 2026-06-29
updated: 2026-06-29
---

# F017 — Add quick capture for new coins

**Promoted to**: specs\336-quick-capture\

## Summary

Create a fast, mobile-first intake flow for recording a coin at the moment of acquisition or handling. Done means a collector can capture photos, minimal identity fields, purchase context, and notes into a draft or coin record without walking through the full edit form, then finish enrichment later.

## Acceptance criteria

- [ ] User can open Quick Capture from mobile/PWA navigation and create a saved draft with obverse/reverse photos, title, date range or era, source, price, and notes before promoting it to a normal coin.
- [ ] Captured items are user-scoped, resumable, and clearly marked as incomplete until promoted to a normal coin record.
- [ ] Existing collection counts, wishlist/sold flags, image handling, and edit workflows are not regressed.
- [ ] The flow works on narrow mobile viewports and uses existing design tokens, buttons, chips, and image upload patterns.

## Plan outline

1. Define the product boundary: quick manual intake first; AI enrichment can be a follow-up unless existing intake draft services make reuse trivial.
2. Add or reuse a backend draft/intake model with user ownership, image references, lifecycle status, and promote-to-coin service logic.
3. Add a compact capture UI that favors camera/file input, sparse fields, save-and-continue-later, and promote/edit actions.
4. Validate with targeted backend service tests plus frontend component/workflow coverage for create, resume, promote, and mobile layout.

## Task breakdown

- Backend: add or extend draft models in `src\api\models\`, repository methods in `src\api\repository\`, service orchestration in `src\api\services\`, handlers in `src\api\handlers\`, and DI/routes in `src\api\main.go`.
- Frontend: add Quick Capture route/page under `src\web\src\pages\`, focused components under `src\web\src\components\capture\`, API methods in `src\web\src\api\client.ts`, and types in `src\web\src\types\index.ts`.
- Testing: cover user scoping, draft promotion, image attachment behavior, collection count contracts, and mobile/PWA workflow with existing Go/Vitest/Playwright patterns.
- Documentation: update user-facing workflow docs and API reference only after the promoted spec fixes the API shape.

## Constitution alignment

- §0 Hierarchy of Authority — backlog card stays lightweight until promoted to a full SpecKit spec.
- Principle I (Clear Layered Architecture) — capture persistence and promotion must stay Handler → Service → Repository → Database.
- Principle V (Security, Auth, and Privacy by Default) — uploads and draft records are authenticated, user-scoped, and validated.
- Principle VI (Consistent User Experience) — mobile/PWA capture must reuse existing design tokens and upload patterns.
- §17 Quality Gate, §21 Definition of Done.

## Open questions

- [x] Quick Capture creates separate draft/intake records first, then promotes them to normal `Coin` records after explicit confirmation.
- [x] AI enrichment is deferred from v1 unless existing draft services make reuse trivial; v1 is deterministic manual capture.
- [x] Drafts do not appear in the main collection until promoted; promotion uses the existing normal coin minimum field rules.

## Notes

This should be treated as a workflow accelerator, not a replacement for the full coin form. Favor reuse of existing upload and coin edit infrastructure over a parallel intake system.

## History

- 2026-06-29: promoted to `specs\336-quick-capture\`.
- 2026-06-29: created (status: triaged).
