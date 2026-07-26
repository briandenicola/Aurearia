# Implementation Plan: Mint ↔ Location Integration

**Branch**: `338-mint-location-integration` | **Date**: 2026-07-26 | **Spec**: `specs/338-mint-location-integration/spec.md`
**Input**: Feature specification from `specs/338-mint-location-integration/spec.md`

## Summary

Elevate `Coin.Mint` from a free-text string to a real, optional foreign key
into `MintLocation`, and extend `MintLocation` to support private,
user-created entries alongside the existing admin-curated global list. Wire
the resulting managed list into every UI surface that currently accepts a
free-text mint, add a "Create new mint" flow that geocodes the name
server-side (OSM Nominatim) and lets the user confirm/adjust the pin before
saving, and backfill existing coins against the mint-location list on
rollout, with a non-blocking nudge for anything left unlinked.

## Technical Context

**Language/Version**: Go (Gin, GORM) backend; Vue 3 + TypeScript (Vite) frontend
**Primary Dependencies**: GORM (SQLite), Leaflet + OpenStreetMap tiles (already in use)
**Storage**: SQLite via `glebarez/sqlite`, schema evolved via `AutoMigrate` + hand-written idempotent one-off migrations (no migration framework)
**Testing**: Existing Go test suite (`*_test.go` alongside services/repositories/handlers) + existing frontend test setup
**Target Platform**: Self-hosted web app (Go API + Vue SPA)
**Project Type**: Web application (backend `src/api` + frontend `src/web`)
**Performance Goals**: N/A beyond existing coin CRUD latency; geocode calls are user-initiated, not on any hot path
**Constraints**: No external service may receive coin, collection, or account data — geocoding sends only the typed mint name
**Scale/Scope**: ~17 seeded global mints today, private mints grow per-user; existing coin volume per user is small (personal collections)

## Constitution Check

- No new project/service boundary introduced — this is additive to the
  existing `src/api` monolith and `src/web` SPA, consistent with existing
  patterns (`StorageLocation` is the direct precedent for a per-user managed
  FK list).
- No new persistent external dependency — Nominatim is called over HTTPS,
  no SDK/library dependency added, no API key/secret to manage.
- Privacy constraint from the original Mint Map spec (`specs/225-mint-map-view/spec.md`)
  is preserved: no coin/user/account data leaves the server; only the
  user-typed mint name is sent to Nominatim, and only when the user explicitly
  triggers "Create new mint."

## Project Structure

### Documentation (this feature)

```text
specs/338-mint-location-integration/
├── plan.md              # this file
└── spec.md              # feature specification
```

(`research.md` / `data-model.md` / `quickstart.md` / `contracts/` / `tasks.md`
omitted for now — research was done inline in conversation rather than as a
separate artifact; `tasks.md` to be generated via `/speckit.tasks` when
implementation begins.)

### Source Code (repository root)

```text
src/api/
├── models/
│   ├── mint_location.go        # add UserID *uint; scope uniqueness per-owner
│   └── coin.go                 # add MintLocationID *uint (FK, mirrors StorageLocationID)
├── repository/
│   ├── mint_location_repository.go   # scope List/lookup by global-or-owned
│   └── coin_repository.go            # no structural change; Mint stays denormalized
├── services/
│   ├── mint_location_service.go      # ownership checks, uniqueness scoping, in-use delete guard
│   └── geocode_service.go            # NEW: Nominatim client (name -> candidates)
├── handlers/
│   ├── mint_location.go        # add self-service (non-admin) create/update/delete + geocode endpoint
│   └── coins.go                # accept mintLocationId on create/update; populate denormalized Mint
└── database/
    └── database.go             # one-off backfill migration (match existing Coin.Mint -> MintLocation)

src/web/
├── src/components/
│   ├── CoinForm.vue                  # mint free-text input -> <select> (mirrors storageLocationId picker)
│   ├── CreateMintModal.vue           # NEW: name entry -> geocode -> map confirm/adjust -> save
│   ├── UnlinkedMintBanner.vue        # NEW: non-blocking nudge on coin edit for unmatched legacy mint
│   └── map/                          # MintMapLeaflet.vue, MintPin.vue etc. — reused inside CreateMintModal
├── src/pages/MintMapPage.vue         # prefer coin.mintLocationId grouping; fuzzy match becomes fallback only
├── src/utils/mintMap.ts              # keep normalizeMintName as legacy-fallback path only
├── src/api/client.ts                 # add createMintLocation (self-service), geocodeMintName
└── src/data/ancientMints.ts          # DELETE (confirmed unused dead code)
```

**Structure Decision**: Existing single Go API + Vue SPA layout is unchanged;
this feature is additive within `src/api` and `src/web`, following the
`StorageLocation` precedent (per-user managed list with a real FK on `Coin`)
rather than introducing any new architectural pattern.

## Design Decisions (resolved during review)

1. **Geocoding**: hybrid — server-side Nominatim lookup by name, result(s)
   always shown on a map for the user to confirm or drag into place before
   saving; empty/failed lookup falls back to manual pin placement in the same
   modal, never a dead end.
2. **Private mint visibility**: private mints are visible only to their
   creator (dropdown, coin form, Mint Map, aggregates) — never shared
   globally. Promotion of a private mint to the global list is deferred to a
   future pass (not in scope here).
3. **Unlinked legacy mints**: surfaced via a non-blocking, dismissible banner
   on the coin edit view — never blocks save or view.

## Rollout Phases

1. **Phase 1 — Backend schema & API**: `MintLocation.UserID`, `Coin.MintLocationID`,
   ownership-scoped CRUD + uniqueness rules, in-use delete guard, Nominatim
   geocode endpoint, backfill migration.
2. **Phase 2 — Coin form**: dropdown (Unknown / My Mints / Mints / Create new),
   create-mint modal with geocode-then-confirm map UI.
3. **Phase 3 — Other surfaces & map**: wishlist search-alert criteria and
   smart-set criteria pick from the same managed list; `MintMapPage` grouping
   becomes FK-first with fuzzy-match fallback for legacy data; unlinked-mint
   banner added to coin edit view.
4. **Phase 4 — Cleanup**: delete dead `ancientMints.ts`; update
   `specs/225-mint-map-view/spec.md`'s now-superseded "out of scope" notes to
   point at this feature.

## Complexity Tracking

No constitution violations requiring justification — this reuses the existing
`StorageLocation` FK-list pattern and does not introduce a new project,
service, or persistent third-party dependency.
