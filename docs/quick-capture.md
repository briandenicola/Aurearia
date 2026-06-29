# Quick Capture

Quick Capture is a mobile/PWA-first workflow for saving sparse coin intake drafts without creating normal collection coins until an explicit promotion step.

## Workflow

1. Open **Quick Capture** from navigation.
2. Add a working title or note, and optionally obverse/reverse/detail photos plus purchase context.
3. Save the draft. Active drafts are excluded from normal collection, wishlist, sold, stats, and health counts.
4. Open **Quick Capture Drafts** to resume, edit images/fields, or discard drafts.
5. Promote only after required normal coin fields are complete. Quick Capture v1 promotions create active collection coins (`isWishlist=false`, `isSold=false`) and repeated promotion returns the existing coin instead of creating duplicates.

## Regression notes

- Draft media uses the authenticated media path and is owner-scoped.
- Promoted coins use the existing add/edit/image/display contracts after promotion.
- Quick Capture v1 is manual/deterministic and does not expand AI intake or Python agent behavior.
