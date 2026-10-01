# Coin Lookup

> Capture a coin or slab photo at a show, extract lookup details, verify NGC Ancients certs, and save the result to your wish list or collection.

## Overview

Coin Lookup is designed for in-person acquisition workflows. When you see a coin at a show, open **Lookup Coin**, take or upload photos, review the extracted details, and save the result without leaving the show-floor workflow.

## Entry Points

- **Main menu** — Open **Lookup Coin** from the primary app navigation
- **Wish List** — Open lookup from the Wish List page when evaluating a potential purchase

## Lookup Flow

1. Capture one or more photos with the device camera, or upload existing images
2. Optionally type or dictate editable identification notes
3. The Go API sends the images and reviewed notes to the configured vision provider through the agent proxy
4. The response extracts visible slab/label text, candidate coin fields, and NGC certification data when present
5. The results page shows extracted details, verification links, and possible catalog matches
6. Save the result to the Wish List or Collection

## Voice Notes

On supported browsers, the optional notes step includes push-to-talk English
dictation. Start dictation, review or edit the resulting text, and then use the
existing **Analyze Photos** or **Deep Analysis** action. Recognition never
submits analysis automatically.

Speech recognition may send audio to the browser or device provider. Aurearia
does not upload or retain raw audio; only the reviewed transcript is handled as
ordinary notes after explicit submission. Browser support varies, and typing
remains the complete fallback when the microphone control is unavailable or
permission is denied. The first dictation action requests microphone permission
through the browser and immediately releases the temporary permission-check
audio track before recognition starts. If access is blocked, allow the
microphone in the browser's site permissions and try again. Notes retain the
existing 2,000-character limit.

## NGC Ancients Verification

When the image contains an NGC Ancients certification number, Coin Lookup:

- Normalizes compact and hyphenated cert formats
- Displays the normalized cert on the results page
- Generates an official NGC Ancients verification URL:

```text
https://www.ngccoin.com/certlookup/{compactCert}/NGCAncients/
```

For example, `2412821-034` becomes:

```text
https://www.ngccoin.com/certlookup/2412821034/NGCAncients/
```

NGC does not currently expose a public developer API for cert lookup, so the app links to the official NGC verification page rather than scraping NGC data.

## Numista Fallback

When no NGC cert is detected, Coin Lookup uses extracted fields such as ruler, denomination, and era to search Numista if a Numista API key is configured.

Possible Numista matches show:

- Title
- Issuer
- Year range
- Thumbnail when available
- Link to the Numista catalog entry

NGC-slab lookups return as soon as the cert is extracted; they do not wait for Numista enrichment.

## Optional Price Range

**Include an estimated price range** is off by default. When it is on, Coin Lookup
first runs one bounded dealer search on the proposed attribution (ruler,
denomination and category, or the proposed name when those are thin) against the
configured dealer search sources. Dealers with a direct site adapter return
current stock only.

- When priced, available listings come back in the requested currency, the
  results page shows **Current Dealer Listings** with the range, the listing
  links and the dealer names. These are asking prices, not completed sales.
- Only listings whose dealer page was actually read are used. A listing that was
  never fetched is dropped rather than shown as current, because it may already
  have sold. Sold, unpriced, link-less and non-USD listings are dropped too.
- At most five listings are shown, always including the cheapest and the dearest,
  so both ends of the stated range have a link you can open.
- When the search finds nothing, times out or fails, the page falls back to the
  labelled **Estimated Price Range** from the vision model's general knowledge.
  That fallback says a dealer listing was not available to price against; it does
  not claim the dealers were searched and had nothing.
- Whichever range is shown is carried into the draft notes when you save.

The comparables search is capped at one extra call per opt-in lookup with a short
timeout, and it never blocks the rest of the lookup result.

## Saving Results

Coin Lookup supports:

- **Add to Wishlist** — Creates a wishlist coin from the lookup
- **Add to Collection** — Creates a collection coin from the lookup
- **Captured Images** — Uploads captured photos after the coin is created
- **Structured References** — Adds generated NGC or Numista references after the coin is created

The save flow creates the coin first, then attaches images and references. This keeps the normal coin-create payload valid and avoids coupling lookup-only data to the core coin API.

## Deep Analysis

When the feature is enabled by an administrator, Coin Lookup also offers
**Deep Analysis** as an optional path. It does not replace or delay the normal
quick lookup. Deep Analysis requires obverse and reverse images and may also use
collector notes or temporary hint images. It runs in the background, streams
replayable progress, and returns a cited report with an editable proposal.
Nothing is saved to a coin or draft until the collector explicitly accepts and
applies selected fields.

See [Deep Analysis](deep-analysis.md) for provider, attribution, privacy, and
configuration details.

## Configuration

### Required

- An AI vision provider configured in **Admin → AI Configuration**

### Optional

- **Numista API Key** in **Admin → System** for fallback catalog matches
- **Deep Analysis** and **OCRE** toggles in **Admin → System** for the optional
  background workflow and Roman Imperial authority evidence
- Admin-configured Category and Era values in **Admin → Coin Properties**

## Related Features

- [Wish List](wish-list.md) — Save lookups as potential acquisitions
- [Numista Catalog Lookup](numista-integration.md) — Catalog reference integration
- [Admin Settings](admin-settings.md) — AI provider, Numista API key, and coin property configuration
- [Deep Analysis](deep-analysis.md) — Background provider routing and cited proposals
- [Camera Capture](camera-capture.md) — Device camera support in PWA mode
