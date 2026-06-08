# Coin Lookup

> Capture a coin or slab photo at a show, extract lookup details, verify NGC Ancients certs, and save the result to your wish list or collection.

## Overview

Coin Lookup is designed for in-person acquisition workflows. When you see a coin at a show, open **Lookup Coin**, take or upload photos, review the extracted details, and save the result without leaving the show-floor workflow.

## Entry Points

- **Main menu** — Open **Lookup Coin** from the primary app navigation
- **Wish List** — Open lookup from the Wish List page when evaluating a potential purchase

## Lookup Flow

1. Capture one or more photos with the device camera, or upload existing images
2. The Go API sends the images to the configured vision provider through the agent proxy
3. The response extracts visible slab/label text, candidate coin fields, and NGC certification data when present
4. The results page shows extracted details, verification links, and possible catalog matches
5. Save the result to the Wish List or Collection

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

## Saving Results

Coin Lookup supports:

- **Add to Wishlist** — Creates a wishlist coin from the lookup
- **Add to Collection** — Creates a collection coin from the lookup
- **Captured Images** — Uploads captured photos after the coin is created
- **Structured References** — Adds generated NGC or Numista references after the coin is created

The save flow creates the coin first, then attaches images and references. This keeps the normal coin-create payload valid and avoids coupling lookup-only data to the core coin API.

## Configuration

### Required

- An AI vision provider configured in **Admin → AI Configuration**

### Optional

- **Numista API Key** in **Admin → System** for fallback catalog matches
- Admin-configured Category and Era values in **Admin → Coin Properties**

## Related Features

- [Wish List](wish-list.md) — Save lookups as potential acquisitions
- [Numista Catalog Lookup](numista-integration.md) — Catalog reference integration
- [Admin Settings](admin-settings.md) — AI provider, Numista API key, and coin property configuration
- [Camera Capture](camera-capture.md) — Device camera support in PWA mode
