# Project Context

- **Owner:** Brian
- **Project:** Ancient Coins frontend — Vue 3 / TypeScript / Pinia / Vite PWA
- **Architecture:** All API calls through `src/web/src/api/client.ts`; UI follows design tokens from `variables.css` and global classes from `main.css`.

## Core Context

- Aurelia owns frontend implementation and UX polish. Durable frontend rules: `<script setup lang="ts">`, strict nullable handling with `?.`/`??`, no emojis, lucide icons, dark theme, PWA/mobile support, and design-token-only CSS when tokens exist.
- Established patterns: accessible modals follow `FeaturedCoinModal` structure; composables expose cleanup functions and pages call them on unmount; timer/resource cleanup is mandatory; auth store syncs through client refresh callback.
- Feature #219 coin detail patterns: overview uses dual-side media, metadata table rows, settings-style section links, and section pages. Detail sections use `<h3>` headings with section spacing; tags are a full section and use `.chip` sizing for interactive pills.
- Camera/intake patterns from #216: 3-column capture-slot grid, active tile state via tokenized gold glow, circular focus-guide overlay, and AddCoinPage camera controls aligned under slots with Camera/Images lucide icons.

## Recent Updates

- **2026-06-01:** Tags UI refinements completed for #219: Tags promoted to full section after Details, before Catalog References; `.chip` sizing used; type-check/build clean.
- **2026-06-01:** AddCoinPage camera actions changed to a 3-column grid matching capture slots; shutter centered under REVERSE and photo library button aligned under CARD; Upload icon replaced by Images.
- **2026-06-01:** Purchase metadata moved into the Details table as full-width rows. `CoinDetailMetadataRow` gained `fullWidth?: boolean` and later `url?: string | null`; purchase location renders as store-only with optional sanitized `SafeExternalLink`.
- **2026-06-01:** Store prefix label added for purchase location: `Store: ` is rendered only for `row.key === 'purchaseLocation'`, styled muted/italic; only the store name is clickable when a URL is present.
