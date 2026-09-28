# Typography compliance

**Date:** 2026-09-28
**Branch:** `beta`
**Baseline:** `15ff45ffb6c91d34b09c6360c514a0b0b1813ea2`
**Pre-receipt web diff:** `89946e64af23b3b3c2d7b78d26e26237737119e4`

## Scope and outcome

- Replaced remote-only Google font loading with pinned local Inter and Cinzel
  packages so installed and offline PWA rendering uses the approved families.
- Added shared sans/display family tokens and migrated every direct font-family
  declaration and arbitrary Cinzel utility to those tokens.
- Restored all Admin Schedules section and history headings to the documented
  h3 hierarchy: Cinzel, `1.2rem`, weight 500, heading color.
- Added enforcement for remote font imports, raw font-family declarations,
  arbitrary Tailwind font families, required family tokens, and Schedule h3
  hierarchy.

## Evidence

- Focused enforcement: 13 tests passed.
- Mutation proof: restoring the old Schedule heading and raw Cinzel utility
  caused exactly two enforcement failures; reverting the mutation restored all
  13 tests.
- `task check:web`: passed on the `beta` candidate, including lint, strict type
  checking, full tests, and production build. The built CSS contains bundled
  `@font-face` rules.
- `git diff --check`: passed before persistence records were added.

## Remaining evidence

There is no authenticated Admin Schedules browser specification in the current
Playwright suite. Physical installed-PWA visual inspection remains an owner
acceptance check; it is not claimed by the automated evidence above.
