# Phase 1 Data Model: Optional Nomisma.org Authority Linking

## Entity: `MintLocation` (extended, existing table `mint_locations`)

Existing fields (unchanged — see `src/api/models/mint_location.go`):

| Field | Type | Notes |
|-------|------|-------|
| `ID` | `uint` | primary key |
| `UserID` | `*uint` | nil = global (admin-curated); non-nil = private owner. **Unchanged by this feature.** |
| `DisplayName` | `string` | **Unchanged.** |
| `NormalizedName` | `string` | **Unchanged.** |
| `Lat` / `Lng` | `float64` | **Unchanged.** |
| `Region` | `string` | **Unchanged.** |
| `Aliases` | `StringList` | **Unchanged.** |
| `CreatedAt` / `UpdatedAt` | `time.Time` | **Unchanged.** |

New fields (additive, all nullable/omit-empty — absence means "not linked"):

| Field | Type | GORM tag | JSON | Notes |
|-------|------|----------|------|-------|
| `NomismaURI` | `*string` | `type:varchar(256)` | `nomismaUri,omitempty` | Durable Nomisma concept URI (e.g. `http://nomisma.org/id/roma`). Set only via the confirm/link action. |
| `NomismaLabel` | `string` | `type:varchar(256)` | `nomismaLabel,omitempty` | Matched label captured at confirmation time (display copy only — not re-fetched live). Empty when `NomismaURI` is nil. |
| `NomismaLinkedAt` | `*time.Time` | (no special tag) | `nomismaLinkedAt,omitempty` | Confirmation timestamp (server clock, UTC), set/cleared atomically with `NomismaURI`. |

### Invariants

- `NomismaURI`, `NomismaLabel` (non-empty), and `NomismaLinkedAt` are set or
  cleared **together** — never a partial state. Enforced in
  `MintLocationService`, not at the DB layer (SQLite has no native
  multi-column conditional constraint here; a service-level invariant
  mirrors how `ensureLookupKeysAvailable` already centralizes cross-field
  validation for this model).
- Only rows with `UserID == nil` (global) may ever have a non-nil
  `NomismaURI`. Enforced by only exposing link/unlink through
  `LinkNomismaGlobal`/`UnlinkNomismaGlobal` service methods that internally
  reuse the existing `UserID != nil → ErrMintLocationNotFound` guard used by
  `UpdateGlobal`/`DeleteGlobal`. A private `MintLocation` can never reach a
  linked state, including via direct API calls (FR-006).
- No uniqueness constraint on `NomismaURI` — multiple global mint locations
  may share the same concept URI (see research.md §6; spec Edge Cases).
- `NomismaURI`/`NomismaLabel`/`NomismaLinkedAt` are **never** mutated by
  `UpdateGlobal` (the general display-field editor for name/coordinates/
  region/aliases) — only by the dedicated link/unlink methods. This is the
  mechanism that satisfies FR-005's "without altering the MintLocation's
  name, coordinates, aliases" guarantee: the two operations are structurally
  separate `repo.Update(...)` calls touching disjoint column sets.
- Existing unique index `idx_mint_location_owner_name` on
  `(user_id, normalized_name)` is untouched; new columns are not part of any
  index (search/filter by Nomisma link is not a requirement in this MVP).

### Migration

Additive `AutoMigrate(&models.MintLocation{}, ...)` — no new migration file
mechanism exists in this codebase (see `database.go`'s single `AutoMigrate`
call for all models); GORM adds the three new nullable columns to the
existing SQLite table without data loss, matching the pattern already used
for other additive fields on `MintLocation` (e.g. `UserID` from feature 338).
No backfill is required or possible — every row starts unlinked
(`NomismaURI IS NULL`), which is the correct default per FR-003's "absent
fields mean not linked."

### Rollback

An older API version ignores the three new nullable columns entirely
(SQLite tolerates extra columns unknown to an older `SELECT *`-free GORM
model — GORM selects by struct field, not `SELECT *`, so a rolled-back
binary simply never reads/writes them). Existing coordinates, aliases,
ownership, and coin associations are unaffected either direction. No
destructive migration is introduced or required for rollback.

---

## Value object: `NomismaCandidate` (transient, not persisted)

Returned by `NomismaClient.Search` and the `/admin/mint-locations/nomisma/search`
handler response — never written to the database. Represents one raw
reconciliation result for the admin to review.

| Field | Type | Notes |
|-------|------|-------|
| `URI` | `string` | Candidate's durable Nomisma concept URI |
| `Label` | `string` | Candidate's display label from Nomisma |
| `Score` | `float64` | Nomisma's own reconciliation confidence score — decision support only, never auto-applied (FR-002) |
| `Match` | `bool` | Nomisma's own "this is confidently the match" flag — still requires explicit admin confirmation regardless of value (FR-002/FR-009) |

`NomismaCandidate` is the frontend-facing DTO too (same shape sent as JSON);
there is no separate internal/external DTO split needed here (unlike
Numista's provider-DTO-vs-application-DTO separation) because Nomisma's
reconciliation response is already minimal and stable.

---

## Value object: `NomismaSearchOutcome` (transient)

Typed wrapper distinguishing the FR-007/FR-008/FR-009 outcomes so the
handler/frontend never have to string-match an error:

| Kind | Meaning | HTTP surfacing |
|------|---------|-----------------|
| `ok` | One or more candidates returned | `200` with `candidates: [...]` |
| `no_match` | Zero candidates (valid, not an error) | `200` with `candidates: []`, `status: "no_match"` |
| `unavailable` | Timeout, network error, non-2xx from Nomisma, or malformed response | `200` with `candidates: []`, `status: "unavailable"` (never a 5xx — the mint location and normal CRUD remain fully usable per FR-007, so this is not treated as a caller-facing API failure) |
| `invalid_request` | Empty/whitespace-only query (validated before any Nomisma call) | `400` |

This mirrors the existing `Geocode` handler convention (`h.geocode.Search`
failures already collapse to an empty-candidates 200 response, "so the
frontend can fall back... without a dead end") rather than the more elaborate
`NumistaError`/telemetry-publication machinery, which is disproportionate for
a single admin-triggered search action (Principle IV; see research.md §3).

---

## Relationships

```text
MintLocation (global, UserID == nil)
  └── optional 1:1 → Nomisma concept (external, referenced by URI only)
        no local table; represented entirely by NomismaURI/NomismaLabel/
        NomismaLinkedAt on the MintLocation row itself
```

No new foreign keys, join tables, or repository interfaces are introduced.
`Coin` → `MintLocation` (existing FK from feature 338) is entirely unaffected
— a coin's link to its mint location is unchanged regardless of whether that
mint location has a Nomisma link.
