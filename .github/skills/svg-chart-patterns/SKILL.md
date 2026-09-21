---
name: svg-chart-patterns
description: "Build or change token-themed Vue SVG charts with consistent scales and responsive tests."
---

# Existing Vue SVG chart patterns

Inspect the nearest chart under `src/web/src/components/stats/`; prefer reuse
over a new chart dependency. Follow the web-scoped design tokens and typography.

- For line charts, define a consistent viewBox and data domain, sparse labels,
  a clear endpoint/value callout and non-scaling strokes where appropriate.
  `preserveAspectRatio="none"` can distort shapes and text; rem units do not
  guarantee undistorted SVG labels. Test narrow/wide layouts and use a separate
  HTML label layer when readability requires it.
- For flow/Sankey charts, use one shared count-to-height scale across columns,
  deterministic node ordering and per-source/target offsets for stacked bands.
  Account for node gaps/minimum sizes so small categories do not overflow.
  Preserve source/target totals in tests; handle zero totals without division.
- Reuse category/material/era tokens and accessible positive/negative states.
  Do not create a private chart palette or new arbitrary typography.
- Handle empty, loading, failure and paginated data explicitly. Distinguish an
  unavailable request from a valid empty chart.
- Guard nullable/indexed data rather than using unexplained non-null assertions.

Use mocked paginated API responses and focused component assertions for loading,
empty/error states, totals and geometry. Test real mobile/theme rendering where
relevant, then run authorized `task check:web` and affected browser checks.
