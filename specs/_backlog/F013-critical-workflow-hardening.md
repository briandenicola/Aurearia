---
id: F013
title: "Harden critical collection workflows"
status: promoted
priority: P0
effort: L
value: 5
risk: 4
owner: Maximus
created: 2026-06-09
updated: 2026-06-09
promoted_to: specs/220-critical-workflow-hardening/
---

# F013 — Harden critical collection workflows

## Summary

Make the core collection workflows boringly reliable before expanding agentic
write surfaces. The first target is coin create/edit/update because those flows
touch identity fields, storage locations, tags, sets, references, images,
legacy/custom values, and value history.

**Promotion:** Active SpecKit feature lives at
`specs/220-critical-workflow-hardening/`.

## Acceptance criteria

- [ ] Coin create/update uses explicit typed request contracts instead of
      binding broad model payloads for mutation.
- [ ] Regression tests cover editing one property, storage location, sets, tags,
      references, images, legacy/custom era values, and value-history side
      effects.
- [ ] A golden test collection fixture includes Roman, Greek, Byzantine,
      wishlist, sold, private, tagged, set-member, storage-location, image-heavy,
      and legacy/custom-era coins.
- [ ] Browser workflow tests cover login, add coin, edit one field, edit storage
      location, edit tags/sets, upload/delete images, search/filter, and mobile
      viewport edit.
- [ ] Documentation identifies the critical workflow suite and the command to
      run it locally and in CI.

## Constitution alignment

- Principle I (Clear Layered Architecture) — typed request handling must preserve
  handler/service/repository boundaries.
- Principle III (Strict Types and Explicit Contracts) — update contracts and API
  payloads are explicit.
- Principle IV (Simple Complete Changes) — fixes cover real workflows and
  directly related sibling paths.
- Principle IX (Automated Enforcement Over Manual Memory) — workflow regressions
  become repeatable tests.
- §17 Quality Gate, §21 Definition of Done.

## Open questions

- [ ] Should browser workflow tests use Playwright as deterministic tests before
      F011 adds AI-driven exploration?
- [ ] Should the golden fixture live in Go test helpers, frontend test fixtures,
      or a shared seed endpoint used only in tests?
- [ ] Which workflows should block every PR versus run nightly?

## Notes

This is the foundation for agentic writes. Agents should not be allowed to
commit collection changes until the ordinary human update paths are typed,
tested, and stable.

## History

- 2026-06-09: created (status: triaged).
- 2026-06-09: promoted to active SpecKit feature
  `specs/220-critical-workflow-hardening/` (owner: Maximus).
