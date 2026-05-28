---
title: Coin Analysis via Vision Model
id: F004
status: promoted
priority: P2
effort: M
value: 5
risk: 4
owner: Maximus
created: 2026-05-28
updated: 2026-05-28
---

## Summary

Image-based numismatic analysis using vision models. User uploads obverse/reverse photos, Go API proxies to Python agent (`coin_analysis.py` team), vision model analyzes surfaces, result cached as markdown AIAnalysis record. Frontend renders cached analysis with optional accept flow.

## Acceptance Criteria

- When user clicks "Analyze with AI" on coin detail, obverse and reverse images are sent to agent service
- When agent receives images, it dispatches to vision model with separate prompts (obverse/reverse context)
- When vision analysis completes, markdown-formatted result is cached in `Coin.AIAnalysis` table
- When user accepts analysis, cache persists; when rejected, cache is cleared
- When coin lacks images, graceful error returned to frontend

## Constitution Alignment

**Principle IX (AI Provider Configuration):** Analysis respects user's selected provider (Anthropic Claude vision or Ollama llava).
**Principle XVIII (Agent Operating Rules):** Vision model agent outputs are cached to prevent re-inference on re-render.
**Principle XI (Security):** Image uploads validated by MIME type; paths sanitized to `uploads/coins/`.

## Implementation Notes

- Vision analysis endpoint: `POST /coins/:id/analyze` (expects obverse, reverse image files)
- Cache table: `AIAnalysis` model (coinID, provider, analysisText, createdAt)
- Agent team: `app/coin_analysis.py` uses vision model with system prompts for obverse/reverse
- OCR extraction for store cards available via separate `POST /ocr` endpoint
- Image extraction fallback: og:image scraping for wishlist coins

## Open Questions

None — feature shipped.

## Notes

Retroactive card created 2026-05-28 for governance traceability under Constitution §0 (Hierarchy) Phase 2. Supports Anthropic vision_claude_3_5_sonnet and Ollama llava; timeout configurable in admin settings.
