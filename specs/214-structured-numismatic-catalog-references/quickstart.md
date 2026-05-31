# Quickstart Validation — #214 Structured Numismatic References

Use this runbook to validate #214 end-to-end after changes land.

## 1. Backend API checks

1. Start API and open Swagger (`/swagger/index.html`).
2. Create a coin with `era` set to `ancient` and at least one structured reference.
3. Verify `GET /api/coins/:id` returns `references[]`.
4. Verify `GET /api/coins?era=ancient` filters correctly.
5. Verify `POST /api/coins/:id/references` rejects invalid payloads:
   - Missing `catalog`
   - Missing `number`
   - Missing `volume` when required by catalog rules
   - Duplicate reference on the same coin

## 2. Frontend UI checks

1. In **Add/Edit Coin**, confirm era is constrained to `ancient|medieval|modern|unspecified`.
2. In **Collection**:
   - Desktop header shows era filter chips.
   - PWA filter menu shows era filter chips.
   - Selecting an era updates list results.
3. In **Coin Detail**, use **Catalog References** section to:
   - Add a reference
   - Edit a reference
   - Delete a reference
   - Confirm changes persist on refresh

## 3. AI discovery interoperability checks

1. Run coin search chat and request listings with known catalog references.
2. Confirm suggestion payload includes `candidateReferences` when available.
3. Add suggestion to wishlist and verify created coin includes structured `references`.

## 4. Export parity checks

1. Run `GET /api/user/export` and inspect `coins.json`:
   - `era` present
   - `references` array present
   - legacy `referenceUrl` / `referenceText` still present
2. Run `GET /api/user/export/catalog` and verify PDF details show structured references summary.

## 5. Commands used for local validation

```bash
# API
cd src/api && go test ./...

# Web
cd src/web && npm run type-check

# Agent
cd src/agent && python -m pytest tests/ -v && ruff check app/ tests/

# Swagger / OpenAPI sync
task openapi
```
