# Quick Capture

Quick Capture is a mobile/PWA-first workflow for saving sparse coin intake
drafts without creating normal collection or wish-list coins until an explicit
promotion step.

## Workflow

1. Open **Quick Capture** from navigation, or use **Identify Coin** to analyze
   captured or uploaded photos.
2. Add a working title or note and, optionally, obverse, reverse, or detail
   photos plus purchase context.
3. In Identify Coin, select **Analyze Photos** to review extracted details.
4. When Numista lookup is available in the review context, edit and explicitly
   submit the query. Select at most one candidate, or leave it unselected.
5. Select **Save as Draft**. Active drafts remain excluded from collection,
   wish-list, sold, statistics, and health counts.
6. Open **Quick Capture** to resume, edit images or fields, replace or remove a
   retained Numista selection, or discard the draft.
7. Promote only after required normal coin fields are complete. Promotion can
   create an active collection coin or a wish-list coin. Repeated promotion
   returns the existing coin rather than creating a duplicate.

## Numista References

Numista lookup remains contextual to Identify Coin and draft review; it is not
a standalone navigation destination.

- Photo analysis proposes evidence and an editable query but does not issue a
  Numista request before explicit search submission.
- The editable broad-search query is preserved as submitted. Progressive
  detail enrichment uses the same query with surrounding whitespace trimmed,
  preventing a second query identity from being introduced after broad paint.
- A usable NGC result remains primary. **Also search Numista** reveals the
  optional editable lookup without making an eager request.
- Broad candidates appear before bounded detail enrichment. Detail failure
  leaves broad candidates selectable, and enrichment never changes the
  explicit selection automatically.
- Draft save persists only the explicitly selected Numista reference as
  `Catalog: Numista`, its positive identifier, and the server-validated
  canonical Numista URL.
- Unrelated draft edits and validation failures preserve the retained
  selection until it is explicitly replaced or removed.
- Draft update omission preserves the selection; a replacement ID/URL pair
  replaces it; the explicit clear field removes it.
- Draft cards display `Numista #<identifier>` from the existing owner-scoped
  draft-list response. Drafts without a selection display no Numista chip.
- Collection or wish-list promotion copies exactly the retained selection
  inside the existing promotion transaction. Repeated promotion reuses the
  promoted coin and does not duplicate the reference. No selection means no
  generated Numista reference.

Expected lookup states are `success`, `empty`, `unconfigured`,
`quota-limited`, `timeout`, and `unavailable`. Status changes, retry, cache
reuse, and a selected reference that is absent from later results do not clear
the editable query or retained selection.

## Regression Notes

- Draft media uses the authenticated media path and is owner-scoped.
- Draft-list results and retained Numista references are owner-scoped.
- Promoted coins use the existing add, edit, image, reference, and display
  contracts.
- Manual catalog-reference management remains available on saved coins under
  **Catalog References**.
- Existing Quick Capture, Identify Coin, saved-coin, and Actions routes remain
  compatible; no Numista route or top-level navigation item is added.
- Quick Capture is deterministic and does not expand Python agent behavior.

See [Numista Catalog Lookup](features/numista-integration.md) for cache
freshness, enrichment states, status guidance, and the typed API contract.
