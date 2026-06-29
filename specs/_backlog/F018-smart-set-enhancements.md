---
id: F018
title: "Improve smart set rule reuse and discovery"
status: triaged
priority: P2
effort: M
value: 4
risk: 2
owner: Maximus
created: 2026-06-29
updated: 2026-06-29
---

# F018 — Improve smart set rule reuse and discovery

**GitHub issue**: #356

## Summary

Build on the existing Smart Sets implementation from `specs\main\spec.md` US4 / FR-008 / FR-009. Done means collectors can reuse saved criteria, start from suggested smart set templates, and understand/edit complex rules more easily without creating a duplicate set system.

## Acceptance criteria

- [ ] User can save a smart set's criteria as a reusable template and apply it when creating or editing another smart set.
- [ ] System offers suggested smart sets based on common criteria such as material, era/date range, category, mint, value range, wishlist, and sold status.
- [ ] Smart criteria UX makes AND/OR grouping, invalid fields, and preview counts understandable before save.
- [ ] Existing smart set derived membership behavior remains source-of-truth and does not add manual membership rows.

## Plan outline

1. Treat this strictly as an enhancement to shipped/promoted smart sets, not a new feature duplicate.
2. Add criteria-template storage and service APIs only if current set criteria JSON cannot support reuse cleanly.
3. Improve the existing rule builder with template apply/save, suggested starter rules, validation messaging, and preview summaries.
4. Add regression tests proving FR-008/FR-009 behavior remains intact.

## Task breakdown

- Backend: extend `src\api\models\set.go`, `src\api\repository\set_repository.go`, and `src\api\services\set_criteria.go` / `set_service.go` for saved criteria templates, user scoping, and suggested templates.
- Frontend: enhance `src\web\src\components\sets\SetSmartRuleBuilder.vue`, `SetCreationWizard.vue`, set API calls in `src\web\src\api\client.ts`, and smart criteria types in `src\web\src\types\index.ts`.
- Testing: add Go tests for criteria template CRUD/apply and safe query translation; add frontend tests for validation, preview, and template application.
- Documentation: update set workflow documentation to distinguish open, defined, goal, smart, and template-backed smart sets.

## Constitution alignment

- §0 Hierarchy of Authority — `specs\main\spec.md` remains higher authority for existing smart set behavior.
- Principle I (Clear Layered Architecture) — criteria template persistence must stay in repositories and business rules in services.
- Principle III (Strict Types and Explicit Contracts) — criteria/template payloads need explicit Go/TypeScript types and Swagger annotations.
- Principle IV (Simple Complete Changes) — improve reuse and UX without replacing the working smart set foundation.
- §17 Quality Gate, §21 Definition of Done.

## Open questions

- [ ] Are suggested smart sets fixed built-ins, learned from the user's collection, or both?
- [ ] Should templates be user-private only, or can future shared/community templates be planned in the data model?
- [ ] Which criteria operators are most confusing today and need first-pass UX focus?

## Notes

Scope decision: no duplicate "new smart sets" card. Current `specs\main\spec.md`, `plan.md`, and `tasks.md` already cover smart sets; this card is for follow-up criteria reuse, suggestions, and UX.

## History

- 2026-06-29: created (status: triaged).
