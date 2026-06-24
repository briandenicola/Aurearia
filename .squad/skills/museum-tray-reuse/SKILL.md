---
name: "museum-tray-reuse"
description: "Reuse the shared museum tray renderer across authenticated and public coin presentations."
domain: "frontend-display"
confidence: "high"
source: "earned"
---

## Context

Use this when a feature wants the visual museum tray presentation for coins, including public/read-only views.

## Pattern

- Reuse `components/tray/MuseumTray.vue`, `MuseumTrayWell.vue`, `TrayControls.vue`, and `utils/trayLayout.ts` rather than creating a separate tray/card renderer.
- Keep authenticated collection behavior as the default: `interactive` defaults to `true`, and wells use `AuthenticatedImage`/private media handling unless a resolver is passed.
- For public surfaces, pass `:interactive="false"` and an `imageSrcResolver` that maps raw file paths to the feature's public media route, e.g. `publicShowcaseMediaUrl(slug, filePath)`.
- Map feature-specific coin contracts into `TrayCoin` with `name ?? 'Untitled'`, `diameterMm ?? null`, and `images ?? []`; missing diameters safely use the tray default size.

## Anti-Patterns

- Do not duplicate the felt tray CSS or well sizing logic in a feature page.
- Do not route public/private upload paths through the wrong media helper.
- Do not make public showcase wells keyboard-focusable unless they actually navigate somewhere.

## Testing

- Prefer focused component/page regressions over snapshots: assert `.museum-tray`/`.tray-well` render, legacy grid/card selectors are absent, and the resolved `<img>` `src` uses the intended public or private media route.
- For paged public trays, use more than 12 coins to prove `TrayControls` advances drawers without depending on visual layout measurements.
