# Research: Refine Coin Details Page for PWA and Desktop (#219)

## Decision 1: Use an overview + section-subpage information architecture

- **Decision**: Keep `/coin/:id` as the streamlined overview and move dense sections (Journal, Notes, Actions, AI Analysis) into dedicated subpages linked from overview.
- **Rationale**: This directly satisfies the "streamline, consistent, elegant" goal by reducing cognitive load on the primary page while preserving section depth.
- **Alternatives considered**:
  - Keep all sections on one page with collapsible panels (rejected: still visually dense and inconsistent with requested settings-style navigation).
  - Modal overlays for each section (rejected: weaker deep-link support and higher mobile complexity).

## Decision 2: Represent metadata as standardized rows (table/list-row pattern)

- **Decision**: Replace current boxed info cards with one row-based metadata surface (label left, value right; consistent separators and spacing).
- **Rationale**: Row-based presentation aligns with the requested visual references and improves scanability/consistency across desktop and PWA.
- **Alternatives considered**:
  - Keep existing `CoinInfoGrid` card style and only retheme (rejected: does not address core structure request).
  - Free-form paragraph metadata blocks (rejected: lower readability and weaker consistency).

## Decision 3: Add coin detail child routes for section pages

- **Decision**: Introduce authenticated child routes (or route siblings) for:
  - `/coin/:id/journal`
  - `/coin/:id/notes`
  - `/coin/:id/actions`
  - `/coin/:id/analysis`
- **Rationale**: Route-backed sections provide URL shareability/deep-linking and mirror the settings-style "row to page" interaction requested.
- **Alternatives considered**:
  - Query-param tabs on one route (rejected: less explicit IA and weaker standalone section UX).
  - In-page anchors (rejected: does not reduce overview density).

## Decision 4: Enforce strict PWA/desktop behavior parity with existing mode rules

- **Decision**: Preserve current mode contract: no sticky behavior in PWA/mobile; desktop may keep sticky affordances where appropriate.
- **Rationale**: Constitution Principle XIII explicitly distinguishes PWA and desktop interaction rules and prevents regressions in mobile UX.
- **Alternatives considered**:
  - Single responsive behavior for both modes (rejected: conflicts with established mode-specific UX policy).
  - Sticky controls in PWA (rejected: explicit rule violation).

## Decision 5: Reuse existing section components where possible

- **Decision**: Reuse/adapt `CoinActivityJournal`, `CoinActionsPanel`, and `CoinAIAnalysis` within dedicated pages instead of building new feature logic.
- **Rationale**: Limits risk, preserves behavior, and focuses this feature on layout/navigation refinement rather than business logic changes.
- **Alternatives considered**:
  - Rebuild each section component from scratch (rejected: high regression risk and unnecessary scope increase).
  - Keep section logic in overview and proxy with iframes/teleports (rejected: complexity without UX gain).
