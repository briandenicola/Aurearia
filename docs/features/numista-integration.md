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
Fresh equivalent searches and catalog details may be reused across saved-coin
and photo-assisted lookup. Cache content is disposable and never becomes a
collection record.

## Query and Ranking Contract

- Saved-coin queries are proposed from available name, ruler/issuer,
  denomination, mint, era/date text, material, and obverse/reverse
  inscriptions.
- Photo-assisted queries use the bounded evidence returned by image analysis.
  The collector can also enter a query when analysis produces no useful text.
- The query remains editable before the first search and every retry. Broad
  lookup preserves the submitted text, including intentional internal and
  surrounding whitespace, as `effectiveQuery`; the follow-up enrichment
  request trims surrounding whitespace so both enrichment and retry use one
  stable query identity.
- Ranking is application-owned and deterministic (`numista-v1`). Candidate
  cards explain meaningful matches, conflicts, and unavailable evidence.
  Scores support comparison; they are not an attribution or authenticity
  opinion.

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

Selecting a result changes only local review state. A `CoinReference` is
created only after **Add selected reference** succeeds. Aurearia reconstructs
the canonical URL as
`https://en.numista.com/catalogue/pieces<identifier>.html`; manual reference
validation and deduplication remain authoritative.

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

Draft updates have three explicit behaviors:

- omit selection fields to preserve the retained reference;
- provide a canonical selected ID/URL pair to replace it;
- send the explicit clear control to remove it.

Validation failures and unrelated edits do not silently replace or clear the
selection. Repeated promotion returns the already-created coin and does not
duplicate the reference.

## Lookup Outcomes

Expected provider/configuration outcomes return a typed state rather than
being collapsed into “no results”:

| Status | Meaning and next action |
|---|---|
| `success` | One or more usable candidates; review and select explicitly. |
| `empty` | Configured lookup completed with no usable candidates; revise the query. |
| `unconfigured` | Numista is not configured or rejected the credential; admins receive configuration guidance, other collectors are told to contact an administrator. |
| `quota-limited` | Numista returned 429; wait, using `retryAfterSeconds` only when the provider supplied a positive value. |
| `timeout` | The provider deadline elapsed; the query and selection remain available for retry. |
| `unavailable` | Network, provider, malformed-response, or lookup-capability failure; retry later. |

Malformed requests still use HTTP 400, authentication/authorization use
401/403, and unexpected application failures use a generic HTTP 500. Caller
cancellation ends the request and is not misreported as a provider status.

## Freshness and Coalescing

Broad search responses may include:

- `cache.hit=true` when a fresh persisted in-memory entry was reused;
- `cache.coalesced=true` when the request joined equivalent in-flight work;
- `createdAt`, `expiresAt`, and `ageSeconds` for the shared result.

`hit` and `coalesced` are mutually exclusive. Expired entries are removed and
never served as current. Successful empty searches are cacheable; provider
failures are not. Current configuration is checked before cache access, so
removing the API key cannot be concealed by old cached results.

## Progressive Detail Enrichment

Broad candidates render first. The browser then submits the complete broad
candidate set to `POST /api/numista/enrich`; the server reranks it, chooses the
leading bounded subset, and fetches at most five details by default with
concurrency two. Client ordering or duplicate IDs cannot increase that limit.

Candidate detail states are:

- `not_requested` — broad result only;
- `enriched` — fresh provider detail applied;
- `cached` — fresh detail cache applied;
- `failed` — detail failed, but the broad result remains selectable.

Enrichment returns the complete candidate set and never silently changes the
collector's explicit selection. Partial or total detail failure does not turn
a successful broad search into `empty`.

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
POST /api/numista/enrich             # Bounded progressive detail enrichment
GET  /api/numista/search?q=...       # Deprecated compatibility adapter
POST /api/coins/lookup               # Analyze photos; does not eagerly search Numista
GET  /api/admin/numista/health       # Admin-only redacted rolling health
```

The deprecated GET adapter keeps its legacy `{count,types}` response and maps
non-success/non-empty domain outcomes to HTTP 503. New clients use the typed
POST routes.

## Related Features

- [Coin Details](coin-details.md) — Structured catalog references
- [Coin Lookup](coin-lookup.md) — Photo analysis and NGC-first review
- [Quick Capture](../quick-capture.md) — Draft retention and promotion
- [Admin Settings](admin-settings.md) — Configure the Numista API key
- [API Reference](../api-reference.md#numista) — Typed request/response contract
- [ADR 0007](../adr/0007-shared-numista-lookup.md) — Shared boundary and migration

See also: [Numista.com](https://en.numista.com/)
