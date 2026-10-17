# Share card: coin title before ruler

**Authority:** Owner requested the coin title at the top with ruler beneath,
because ruler is not always known. Owner approved focused share-card tests;
no build, container or deployment authorization was requested or assumed.
**Lane:** Small presentation repair; Principles III-IV, sections 17 and 21.
**Base:** beta `8fd3cb698ef3c34eb297d42912d3d30ddd8101ca`.

The shared renderer now uses the saved coin name as the ticket heading and
places the saved ruler/date underneath. Unknown ruler is omitted; known date
is retained. Missing name remains `Untitled Coin`. Image selection, privacy,
context, native sharing and PNG download paths are unchanged.

Updated metadata and canvas ordering regressions, the existing browser
assertion, and the web README. Author reviewed the precise four-file delta.
No independent agent was used for this small follow-up.

## Exact source and evidence

| Path | Git blob |
|---|---|
| `src/web/src/utils/coinShareCard.ts` | `19ac8ae6fbf598e701a6960e401cf118d1b2a2dd` |
| `src/web/src/utils/__tests__/coinShareCard.test.ts` | `08a2d0da51657d1e859a6578f7a120fb343de1d6` |
| `src/web/e2e/workflows/coin-share-card.spec.ts` | `82401322c04742db896611f0c34d65bfd2d7713a` |
| `src/web/README.md` | `763ff05d6dab5c6750772675a806993db368e723` |

From `src\web`, `node .\node_modules\vitest\vitest.mjs run coinShareCard.test.ts
useCoinShareCard.test.ts FeaturedCoinModal.test.ts` passed all 28 tests across
three files. This covers title/ruler canvas order, blank/unknown ruler, date
preservation, privacy, PNG rendering, composable sharing/download and Coin of
the Day entry context. `git diff --check` passed.

## State and next action

Implemented locally and focused tests passed; not committed, pushed or released.
Full web completion gate and real-browser rerun were not authorized/executed for
this follow-up and remain pending, not passed. Preserve the separately staged,
reviewed delivery changes. Resume the delivery cutover with explicit scope and
exact-candidate applicability; do not silently combine pending evidence.

## Owner-authorized publication update

Owner requested push/PR after green gates and selected **Both changes together**.
`node .\node_modules\@playwright\test\cli.js test coin-share-card --workers=1
--project=chromium` then passed two real PNG-download cases (1440px and 390px),
using the existing local Vite/browser environment without a build/install.
The four source blobs above are unchanged. Full web completion is pending the
combined beta CI commit; the PR will be its canonical completion record.
