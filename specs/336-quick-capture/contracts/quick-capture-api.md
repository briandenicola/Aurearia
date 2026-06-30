# Contract: Quick Capture API

All routes are under `/api`, require bearer auth, use existing API rate limits/write rate limits as appropriate, and return `{ "error": "..." }` for generic errors unless a response schema is specified.

## Types

```ts
type QuickCaptureDraftStatus = 'active' | 'promoting' | 'promoted' | 'discarded'
type QuickCaptureImageType = 'obverse' | 'reverse' | 'detail' | 'other'

interface QuickCaptureDraftImage {
  id: number
  draftId: number
  filePath: string
  imageType: QuickCaptureImageType
  isPrimary: boolean
  displayOrder: number
  createdAt: string
}

interface QuickCaptureDraft {
  id: number
  userId: number
  workingTitle: string
  dateRange: string
  era: string
  acquisitionSource: string
  purchasePrice: number | null
  notes: string
  status: QuickCaptureDraftStatus
  promotedCoinId: number | null
  promotedAt: string | null
  discardedAt: string | null
  images: QuickCaptureDraftImage[]
  createdAt: string
  updatedAt: string
}

interface QuickCaptureDraftListResponse {
  drafts: QuickCaptureDraft[]
  total: number
  page: number
  limit: number
}

interface QuickCapturePromotionResponse {
  draftId: number
  status: 'promoted'
  coinId: number
  alreadyPromoted: boolean
  target: 'collection' | 'wishlist'
}
```

## `GET /quick-capture/drafts`

Lists the authenticated user's active drafts by default.

### Query

- `status` optional: `active`, `promoted`, `discarded`; default `active`
- `page` optional integer `>= 1`; default `1`
- `limit` optional integer `1..100`; default `50`

### 200

```json
{
  "drafts": [
    {
      "id": 12,
      "userId": 4,
      "workingTitle": "Unattributed denarius",
      "dateRange": "2nd century",
      "era": "ancient",
      "acquisitionSource": "Show table",
      "purchasePrice": 42.5,
      "notes": "Needs ruler check",
      "status": "active",
      "promotedCoinId": null,
      "promotedAt": null,
      "discardedAt": null,
      "images": [],
      "createdAt": "2026-06-29T15:00:00Z",
      "updatedAt": "2026-06-29T15:05:00Z"
    }
  ],
  "total": 1,
  "page": 1,
  "limit": 50
}
```

## `POST /quick-capture/drafts`

Creates a draft. Accepts `multipart/form-data` so first save can include photos.

### Form fields

- `workingTitle` optional string
- `dateRange` optional string
- `era` optional string
- `acquisitionSource` optional string
- `purchasePrice` optional number
- `notes` optional string
- `obverseImage` optional file
- `reverseImage` optional file
- `detailImages` optional repeated files

At least one of `workingTitle`, `notes`, or any valid image file is required.

### 201

Returns `QuickCaptureDraft`.

### 400

Validation failure, including unsupported image type, invalid image bytes, missing minimum identity, or invalid price.

```json
{
  "error": "Add a working title, note, or image before saving a draft"
}
```

## `GET /quick-capture/drafts/{id}`

Returns one owner-scoped draft with images.

### 200

Returns `QuickCaptureDraft`.

### 404

Returned for missing or non-owned drafts.

## `PUT /quick-capture/drafts/{id}`

Updates an active owner-scoped draft. Accepts `multipart/form-data` to support replacing fields and images in one save.

### Form fields

Same fields as create, plus:

- `removeImageIds` optional comma-separated image IDs owned by the draft
- `replaceObverse` optional boolean; when true and `obverseImage` is present, previous obverse images are removed
- `replaceReverse` optional boolean; when true and `reverseImage` is present, previous reverse images are removed

### 200

Returns updated `QuickCaptureDraft`.

### 400

Validation failure. Draft remains active and editable.

### 404

Missing, non-owned, promoted, or discarded draft.

## `POST /quick-capture/drafts/{id}/promote`

Explicitly promotes a valid active draft into one normal `Coin` in either the user's collection or wishlist.

### Request body

Optional `target` plus overrides in normal coin mutation shape. `target` defaults to `collection` when omitted, preserving the original Quick Capture promotion behavior. Use `wishlist` to create the promoted coin with `isWishlist: true`.

```json
{
  "confirm": true,
  "target": "collection",
  "overrides": {
    "name": "Augustus Denarius",
    "category": "Roman",
    "material": "Silver",
    "era": "ancient",
    "purchasePrice": 42.5,
    "purchaseLocation": "Show table",
    "notes": "Captured from Quick Capture draft"
  }
}
```

### 200

Returns `QuickCapturePromotionResponse`.

First success:

```json
{
  "draftId": 12,
  "status": "promoted",
  "coinId": 91,
  "alreadyPromoted": false,
  "target": "collection"
}
```

Repeated promote after success:

```json
{
  "draftId": 12,
  "status": "promoted",
  "coinId": 91,
  "alreadyPromoted": true,
  "target": "collection"
}
```

### 400

Missing `confirm: true`, invalid `target`, or normal-coin validation failure. Draft remains active/editable.

```json
{
  "error": "Complete required fields before promotion",
  "fields": {
    "name": "Name is required"
  }
}
```

### 404

Missing or non-owned draft.

### 409

Draft is discarded or currently being promoted by another request.

## `POST /quick-capture/drafts/{id}/discard`

Closes an unwanted active draft without creating a coin.

### 200

Returns updated `QuickCaptureDraft` with `status: "discarded"`.

### Idempotency

Repeated discard of an already discarded draft returns the discarded draft. Discarding a promoted draft returns `409`.

## Media display contract

Draft image `filePath` values are rendered through the existing `AuthenticatedImage` component and `/api/uploads/*` private media path. The media resolver must authorize a file if it belongs to either:

1. a `CoinImage` visible to the authenticated user, or
2. a `QuickCaptureDraftImage` whose draft is owned by the authenticated user.

## Regression coverage note

Frontend US4 preservation coverage intentionally uses targeted contract/source tests where broad page harnesses are brittle: normal collection filters continue to request `wishlist=false` and `sold=false`, wishlist/sold pages keep their existing filters, stats use the normal stats store, edit pages keep `getCoin`/`updateCoin`/image upload behavior, and navigation preserves Add Coin plus Identify Coin without adding Quick Capture AI-intake routes.
