# Numista Catalog Lookup

> Find and explicitly save Numista entries as structured catalog references.

## Overview

Numista lookup is part of catalog-reference management rather than a
standalone destination. Saved coins use **Catalog References** as the canonical
surface. Identify Coin and Quick Capture retain lookup where it provides
context while reviewing a draft attribution.

Results are decision support only. Aurearia saves a Numista reference only
after the collector selects a candidate and confirms the relevant save action;
it never attaches every result or infers a selection from result order.

## Setup

1. Obtain a Numista API key from [numista.com/api](https://en.numista.com/api/).
2. Paste the key in **Admin → System → Numista API Key**.
3. Save the setting.

The configured allowance is shared by collectors using the Aurearia instance.
Fresh equivalent searches may reuse cached results.

## Saved Coins: Catalog References

1. Open a saved coin.
2. Find **Catalog References** on the coin detail page.
3. Select **Search Numista** beside **Add Reference**.
4. Review or edit the proposed query, then submit it explicitly.
5. Review the candidates and select one.
6. Confirm **Add selected reference**.

The lookup expands inline without replacing manual reference management. After
the selected reference is persisted and the refreshed reference list contains
it, the lookup collapses and the new Numista reference appears in the list.

The existing **Actions** page remains compatible during this placement
transition, but it does not contain the full lookup panel. It provides only a
compact link back to the saved coin's Catalog References section.

## Identify Coin and Quick Capture

1. Open **Identify Coin** and capture or upload photos.
2. Select **Analyze Photos**.
3. Review the extracted coin details.
4. If no usable NGC certification is found, review or edit the contextual
   Numista query and submit it when ready.
5. If a usable NGC result is found, NGC remains primary. Select
   **Also search Numista** to reveal the editable lookup. Revealing it does not
   call Numista; a request occurs only after explicit search submission.
6. Optionally select one Numista candidate and choose **Save as Draft**.

The selected reference is retained with the Quick Capture draft. The draft
list shows `Numista #<identifier>` when a selection exists and no Numista chip
when it does not. Promoting the draft to the collection or wish list copies
exactly the retained selection; saving or promoting without a selection adds
no Numista reference.

## Navigation and Compatibility

- Aurearia adds no standalone Numista route, sidebar item, or top-level menu
  item.
- Existing coin detail, Actions, Identify Coin, and Quick Capture routes remain
  available.
- Manual structured-reference add, edit, delete, validation, and
  deduplication remain authoritative.
- The legacy Numista search endpoint remains available for compatibility.

## API Endpoints

```text
POST /api/numista/lookup             # Typed explicit lookup
GET  /api/numista/search?q=...       # Deprecated compatibility adapter
POST /api/coins/lookup               # Analyze photos; does not eagerly search Numista
```

## Related Features

- [Coin Details](coin-details.md) — Structured catalog references
- [Coin Lookup](coin-lookup.md) — Photo analysis and NGC-first review
- [Quick Capture](../quick-capture.md) — Draft retention and promotion
- [Admin Settings](admin-settings.md) — Configure the Numista API key

See also: [Numista.com](https://en.numista.com/)
