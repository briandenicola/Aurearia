# Catalog Registry Backend: CRUD + Reference Field Rename

**Date:** 2026-06-01  
**Context:** Backend changes coupling three related concerns: reference field semantics, AI confidence removal, and catalog management

## Changes

### 1. CoinReference.Certainty → InvoiceNumber
Repurposed the unused `certainty` field (originally for AI confidence scoring) as a manual invoice number field. The AI agent no longer emits certainty scores, so the field was available for reuse.

- **Model:** `varchar(64)` to allow longer invoice numbers (was 32)
- **Migration:** Idempotent column rename in `database.go` (checks existence via `PRAGMA table_info`)
- **JSON tag:** `invoiceNumber` (camelCase for frontend)

Legacy imports no longer set `certainty = "legacy-import"` — that metadata is not needed.

### 2. Remove AI Certainty/Confidence Concept
The user no longer tracks AI confidence on candidate references. Removed from:
- Go proxy structs (`CandidateReferenceProxy`, `CandidateReferenceDTORef`)
- Python models (`CandidateReference`)
- Agent prompts and normalization logic

The `ValueEstimate.confidence` and `AvailabilityVerdict.confidence` fields remain — those are different contexts (valuation and availability checks).

### 3. Catalog Registry Admin Management
Added full CRUD for `CatalogRegistry` with layered architecture:

- **Repository:** `Create`, `Update`, `Delete`, `FindByID`, `CountReferencesUsing` (checks `coin_references` usage)
- **Service:** `CatalogRegistryService` with validation (era ∈ {ancient, medieval, modern}, code required, duplicate check, in-use check on delete)
- **Handler:** `CatalogRegistryHandler` with Swagger annotations. Protected route `GET /catalogs` for read, admin routes `POST/PUT/DELETE /admin/catalogs/:id` for management.
- **Seed additions:** PRICE, BM, VENÈRA (preserves diacritic — `strings.ToUpper("venèra")` → "VENÈRA")

Sentinel errors: `ErrCatalogNotFound`, `ErrCatalogDuplicate`, `ErrCatalogInUse`, `ErrCatalogInvalidEra`, `ErrCatalogCodeRequired`, `ErrCatalogNameRequired`.

## Verification
- `go build ./...` ✅
- `go vet ./...` ✅
- `go test ./...` ✅ (architecture_test passes)
- `ruff check app/ tests/` ✅
- `pytest tests/ -v` ✅ (60/60 passed)

## Architecture Compliance
- **Principle I (Layered Architecture):** Handler → Service → Repository → Database. No `database` import outside `main.go`.
- **Principle X (Architecture Testing):** `architecture_test.go` confirms import rules enforced.
- **Principle VIII (Commits):** Co-authored-by trailer present.

## Notes
- The invoice number is optional — users enter it manually when they have a purchase invoice to track.
- The catalog code is stored uppercase and validated on input; the diacritic in VENÈRA is preserved per Go's `strings.ToUpper`.
- The migration is safe to run multiple times (idempotent column check).
