# Session Log — Coin Detail Purchase Row Refinement

**Timestamp:** 20260601T184742Z  
**Agent:** Aurelia (Frontend Dev, background)

## Summary

Refined the Coin Detail full-width purchase row display:
- Removed redundant date and "Purchased from" prefix (date already has its own row)
- Shows store name only (`coin.purchaseLocation`)
- Renders as clickable `SafeExternalLink` when `coin.referenceUrl` present (sanitized)
- Plain text otherwise

## Changes

| File | Change |
|---|---|
| `src/web/src/composables/useCoinDetailMetadataRows.ts` | Store-only row with sanitized URL field |
| `src/web/src/components/coin/CoinDetailMetadataTable.vue` | Conditional SafeExternalLink rendering |
| `src/web/src/types/index.ts` | Added optional `url?: string \| null` to CoinDetailMetadataRow |

**Validation:** type-check ✅, lint ✅

---

Merge decision inbox into decisions.md. Code review pending.
