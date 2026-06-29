# Data Model: Quick Capture

## Entity: QuickCaptureDraft

Owner-scoped resumable intake record that is not a normal collection coin until promoted.

### Fields

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| `id` | uint | yes | Primary key |
| `userId` | uint | yes | Owner; every query is scoped to this user |
| `workingTitle` | string | no | Max 200; one of title/note/image is required for save |
| `dateRange` | string | no | Freeform capture such as `c. 330-335` |
| `era` | string | no | Reuse configured/allowed coin eras where practical; freeform date range remains separate |
| `acquisitionSource` | string | no | Maps to `Coin.purchaseLocation` at promotion |
| `purchasePrice` | decimal/float nullable | no | Maps to `Coin.purchasePrice`; frontend treats empty as null |
| `notes` | text | no | Max aligned with normal coin notes where practical |
| `status` | enum | yes | `active`, `promoting` (internal transient), `promoted`, `discarded` |
| `promotedCoinId` | uint nullable | no | Set exactly once on successful promotion |
| `promotedAt` | timestamp nullable | no | Set on successful promotion |
| `discardedAt` | timestamp nullable | no | Set when user discards/closes a draft |
| `createdAt` | timestamp | yes | GORM managed |
| `updatedAt` | timestamp | yes | GORM managed |

### Validation rules

- Owner is required.
- Active save requires at least one of:
  - non-empty `workingTitle`
  - non-empty `notes`
  - at least one valid draft image
- `purchasePrice` must be null or `>= 0`.
- Active drafts are listed; promoted/discarded drafts are hidden from default draft list.
- Non-owned draft reads/updates/promotions/discards return not found without leaking existence.

### State transitions

```text
active -> active      update/resume save
active -> promoting   promotion claim inside transaction
promoting -> promoted successful promotion with promotedCoinId
active -> discarded   explicit discard/close
promoting -> active   only by transaction rollback
promoted -> promoted  repeated promote returns existing coin id
discarded -> discarded repeated discard is a no-op/success message
```

## Entity: QuickCaptureDraftImage

Validated uploaded image attached to a quick-capture draft.

### Fields

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| `id` | uint | yes | Primary key |
| `draftId` | uint | yes | Parent draft |
| `userId` | uint | yes | Denormalized owner for media authorization |
| `filePath` | string | yes | Upload-relative path, same style as `CoinImage.filePath` |
| `imageType` | enum | yes | `obverse`, `reverse`, `detail`, `other` |
| `isPrimary` | bool | yes | Obverse should be primary when present |
| `displayOrder` | int | yes | Stable draft-list ordering |
| `createdAt` | timestamp | yes | GORM managed |

### Validation rules

- Draft must be owned by requesting user and active for upload/update.
- Supported extensions: `.jpg`, `.jpeg`, `.png`, `.gif`, `.webp`.
- MIME/content type is detected from file bytes; unsupported or empty files are rejected.
- Size limit matches normal coin image behavior.
- Draft image paths are authorized through the same private media route behavior as normal coin images.

## Entity: DraftLifecycleEvent

Audit/troubleshooting record for material draft transitions.

### Fields

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| `id` | uint | yes | Primary key |
| `draftId` | uint | yes | Parent draft |
| `userId` | uint | yes | Owner |
| `eventType` | enum | yes | `created`, `updated`, `image_added`, `image_removed`, `promotion_started`, `promoted`, `promotion_reused`, `discarded`, `promotion_failed_validation` |
| `message` | string | no | Safe diagnostic text only |
| `coinId` | uint nullable | no | Present for promotion events |
| `createdAt` | timestamp | yes | Event timestamp |

### Validation rules

- Events are written by service methods only.
- Event messages must not include raw internal errors, file-system paths beyond upload-relative media paths, or other users' data.

## Entity: Coin (existing)

Normal collection record created only through promotion after validation.

### Promotion field mapping

| Draft field | Coin field |
|-------------|------------|
| `workingTitle` | `name` |
| `era` | `era` |
| `dateRange` | appended to `notes` or mapped to an explicit date-range note until a normal coin date-range field exists |
| `acquisitionSource` | `purchaseLocation` |
| `purchasePrice` | `purchasePrice`; `currentValue` may default through existing create sanitization if frontend does so |
| `notes` | `notes` |
| draft obverse/reverse images | `CoinImage` rows for created coin |

### Promotion validation

- Normal coin minimum rules remain authoritative.
- Missing promotion-required fields return field-level validation details and leave the draft active/editable.
- Promotion creates normal coin rows with `isWishlist=false` and `isSold=false` unless an explicit future contract adds those fields; Quick Capture v1 is for new collection intake, not wishlist/sold creation.

## Relationships

- `User` 1 → many `QuickCaptureDraft`
- `QuickCaptureDraft` 1 → many `QuickCaptureDraftImage`
- `QuickCaptureDraft` 1 → many `DraftLifecycleEvent`
- `QuickCaptureDraft` 0/1 → 1 `Coin` through `promotedCoinId`
- `Coin` 1 → many existing `CoinImage` after promotion
