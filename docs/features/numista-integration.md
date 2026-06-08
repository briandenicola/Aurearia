# Numista Catalog Lookup

> Search and link your coins to the Numista catalog for structured reference data.

## Overview

Numista Catalog Lookup integrates with the Numista coin database to help you find matching coins and add structured catalog references to your collection. Coin Lookup also uses Numista as a fallback enrichment source when a photo does not contain an NGC Ancients certification number.

## Features

- **Direct Search** — Search Numista from any coin detail page
- **Automatic Matching** — Uses coin name, denomination, and ruler as search terms
- **Browse Results** — View thumbnails, title, issuer, year range
- **Link to Catalog** — Direct link to full Numista entry
- **Add References** — Import catalog ID as structured reference
- **Lookup Fallback** — Coin Lookup can show Numista matches when no NGC cert is detected

## Setup

1. Get free Numista API key: [numista.com/api](https://en.numista.com/api/) (2,000 requests/month)
2. Paste API key in **Admin → System → Numista API Key**
3. Save and you're ready to go

## How to Use

1. Open any coin detail page
2. Scroll to **Structured Catalog References**
3. Click **Search Numista**
4. Results display with:
   - Coin image thumbnail
   - Title
   - Issuer/ruler
   - Year range
   - Numista ID
5. Click result to view full entry on Numista
6. Click **Add Reference** to import as a structured reference

### From Coin Lookup

1. Open **Lookup Coin** from the main menu or Wish List page
2. Capture or upload coin photos
3. If no NGC cert is detected, the backend searches Numista with extracted coin fields
4. Possible matches appear with thumbnails and links to Numista entries
5. Saving the lookup adds selected/generated Numista references after the coin is created

## API Endpoints

```
GET    /api/numista/search           # Search Numista by query
```

## Pricing & Limits

- **Free Tier** — 2,000 requests/month
- **Paid Tier** — Higher limits available
- **Shared Quota** — Monthly quota shared across all users on your instance
- **Caching** — Results are cached to minimize API calls

## Related Features

- [Coin Details](coin-details.md) — Structured references
- [Coin Lookup](coin-lookup.md) — Photo-based lookup with Numista fallback matches
- [AI Coin Analysis](ai-analysis.md) — Candidate references from search
- [Admin Settings](admin-settings.md) — Configure API key

See also: [Numista.com](https://en.numista.com/)
