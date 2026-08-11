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
- A usable NGC result remains primary. **Also search Numista** reveals the
  optional editable lookup without making an eager request.
- Draft save persists only the explicitly selected Numista reference.
- Unrelated draft edits and validation failures preserve the retained
  selection until it is explicitly replaced or removed.
- Draft cards display `Numista #<identifier>` from the existing owner-scoped
  draft-list response. Drafts without a selection display no Numista chip.
- Collection or wish-list promotion copies exactly the retained selection
  once. No selection means no generated Numista reference.

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
