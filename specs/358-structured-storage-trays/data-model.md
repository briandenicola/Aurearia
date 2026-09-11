# Data Model: Structured Coin Storage Trays

## StorageLocation

Existing owner-scoped table: `storage_locations`.

| Field | Type | Null | Rules |
|---|---|---:|---|
| `id` | uint PK | no | Existing stable identifier |
| `user_id` | uint | no | Owner scope; existing `(user_id,name)` index retained/reviewed |
| `name` | string | no | Trimmed; 1–100 chars; case-insensitive unique per owner across both types |
| `type` | string enum | no | `standard` or `tray`; omitted/legacy value resolves to `standard` |
| `rows` | integer | yes | Null for Standard; required 1–20 for Tray |
| `columns` | integer | yes | Null for Standard; required 1–20 for Tray |
| `sort_order` | integer | no | Existing ordering behavior |
| timestamps | time | no | Existing behavior |

Derived values for a tray:

- `capacity = rows × columns` (1–400)
- `occupied = count(coins where storage_location_id=id and storage_slot is not null)`

### Validation invariants

1. Standard: `rows IS NULL AND columns IS NULL`.
2. Tray: both dimensions present, each 1–20, product ≤400.
3. Name is nonblank after trim and case-insensitively unique within `user_id`.
4. Type is immutable when referenced; v1 should not expose conversion between Standard and Tray.
5. Empty tray dimensions may change; occupied tray dimensions may not.
6. Any referenced location may not be deleted.

### State transitions

```text
create standard ──rename──> standard
create tray(empty) ──rename/resize──> tray(empty)
tray(empty) ──coin assignment──> tray(occupied)
tray(occupied) ──rename──> tray(occupied)
tray(occupied) ──last coin leaves──> tray(empty)
standard/tray(empty, unreferenced) ──delete──> deleted
```

Rejected: occupied resize, referenced delete, type conversion.

## Coin

Existing table: `coins`.

| Field | Type | Null | Rules |
|---|---|---:|---|
| `storage_location_id` | uint | yes | Existing owner-scoped relationship; `constraint:-` remains |
| `storage_slot` | integer | yes | New; one-based; valid only with an owned Tray |

### Assignment invariants

| Location | Slot | Result |
|---|---|---|
| null | null | valid, unassigned |
| null | non-null | 400 validation error |
| Standard | null | valid |
| Standard | non-null | 400 validation error |
| Tray | null | 400 validation error |
| Tray | `1..capacity`, free/current | valid |
| Tray | out of range | 400 validation error |
| Tray | occupied by another coin | 409 `slot_occupied` |
| Other owner's location | any | 404 |

Authoritative index:

```sql
CREATE UNIQUE INDEX idx_coins_storage_location_slot_unique
ON coins(storage_location_id, storage_slot)
WHERE storage_slot IS NOT NULL;
```

Location and slot are one assignment value. They are written together. Explicitly clearing location clears slot. Omitted update fields preserve the existing assignment.

## TrayPosition (derived, not persisted)

| Field | Type | Rule |
|---|---|---|
| `slot` | integer | `1..capacity` |
| `row` | integer | `floor((slot-1)/columns)+1` |
| `column` | integer | `((slot-1)%columns)+1` |
| `occupied` | boolean | Whether aggregate has a coin for this slot |
| `coin` | nullable minimal coin | Present only when occupied |

Every slot is materialized by the UI loop even when no coin row exists.

## TrayAggregate (read model)

| Field | Type | Notes |
|---|---|---|
| `id` | uint | Tray location ID |
| `name` | string | Owner-visible name |
| `rows`, `columns` | integer | Persisted dimensions |
| `capacity` | integer | Derived |
| `occupied` | integer | Count |
| `coins` | array | Only positioned coins; empty wells are derived |

Minimal positioned coin:

| Field | Type | Notes |
|---|---|---|
| `id` | uint | Navigation target |
| `name` | string | Accessible/display label |
| `diameterMm` | number/null | Optional visual sizing input |
| `storageSlot` | integer | Required in aggregate |
| `image` | object/null | Minimal `filePath`/`imageType` private-media reference |

The aggregate excludes price, provenance, notes, AI analysis, tags, sets, and other full-coin data.

## Occupancy read model

```text
locationId, rows, columns, capacity,
occupiedSlots: integer[],
currentCoinSlot: integer|null
```

`coinId`, when supplied, must identify an owner coin. The response does not map occupied slots to unrelated coin IDs or names.

## Migration mapping

```text
legacy StorageLocation -> type=standard, rows=NULL, columns=NULL
legacy Coin            -> storage_slot=NULL; storage_location_id unchanged
```

Migration must preserve all primary keys, owners, names, sort order, timestamps, and coin associations. It is idempotent and fails visibly if preexisting inconsistent non-null slots prevent index creation.

