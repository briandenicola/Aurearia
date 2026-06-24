---
name: "contextual-share-cards"
description: "Extend existing coin share cards with feature-specific context without duplicating renderers."
domain: "frontend-sharing"
confidence: "high"
source: "earned"
---

## Context
Use this when a feature needs a share/download card for a coin plus extra contextual text, such as Coin of the Day summaries.

## Patterns
- Reuse `useCoinShareCard()` and `utils/coinShareCard.ts` rather than creating another canvas/share implementation.
- Keep the base `shareCoinCard(coin)` call behavior unchanged.
- Add optional context (`heading`, `summary`) so feature pages can include cached or feature-specific prose in the generated card and native share text.
- Keep private/value fields out unless the caller explicitly passes context intended for sharing.
- Regression-test the entry component by mocking the API load and share composable together, then assert the loaded `coin` remains the first argument and the feature context is the second argument.

## Examples
- `FeaturedCoinModal.vue` calls `shareCoinCard(coin, { context: { heading: 'Coin of the Day', summary } })`.
- Coin detail pages continue calling `shareCoinCard(coin)` with no context.
- `FeaturedCoinModal.test.ts` stubs `Teleport` and mocks the composable busy ref to prove Share, contextual arguments, and `Sharing...` disabled state without snapshotting the modal.

## Anti-Patterns
- Do not duplicate canvas rendering logic for each share entry point.
- Do not change the default coin detail card layout when adding feature context.
