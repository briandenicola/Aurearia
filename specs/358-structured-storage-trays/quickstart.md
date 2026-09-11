# Quickstart and Validation: Structured Coin Storage Trays

## Scope

Run commands from `C:\Users\brian.denicolafamily\Code\AncientCoins` unless a command changes directory. This guide validates the future implementation; planning alone does not require running these gates.

## Automated validation

```powershell
# Confirm only intended files changed.
git --no-pager status --short
git --no-pager diff --check

# API build, architecture, unit/integration, and migration tests.
Push-Location src/api
go build ./...
go vet ./...
go test ./...
go test -run 'TestFeature358|Test.*StorageLocation|Test.*StorageSlot|Test.*Bulk.*Location|Test.*Duplicate' ./...
Pop-Location

# Race-sensitive repository/service tests. Requires a C toolchain.
task test-race

# Frontend CI-equivalent gates.
Push-Location src/web
npm ci
npm run lint
npm run type-check
npm run test
npm run build
npx playwright test e2e/workflows/storage-trays.spec.ts e2e/workflows/coin-form.spec.ts e2e/workflows/tray.spec.ts
Pop-Location

# Regenerate and verify Swagger/OpenAPI snapshots.
task openapi
git --no-pager diff --exit-code -- src/api/docs/docs.go src/api/docs/swagger.json src/api/docs/swagger.yaml docs/openapi.json
```

Python agent commands are not required unless implementation unexpectedly touches `src/agent/`; the planned design does not.

## Migration verification

The `src/api/database/feature358_migration_regression_test.go` fixture should:

1. Create the legacy `storage_locations` and `coins` shape.
2. Seed multiple owners, same names across owners, multiple coins in one Standard Location, and stable IDs/timestamps.
3. Run the production migration twice.
4. Assert all legacy locations are `standard`, dimensions are null, coin location IDs are unchanged, and slots are null.
5. Assert `idx_coins_storage_location_slot_unique` exists.
6. Seed a malformed duplicate non-null slot case and assert migration returns an error without deleting or remapping rows.

## Manual workflow

1. Start the application with `task up-all`.
2. In **Settings → Data Management → Storage Locations**:
   - create Standard `Safe`;
   - create Tray `Cabinet A`, 3×3;
   - verify types and `0 / 9`;
   - reject invalid 0, 21, and >400 dimensions;
   - verify case-insensitive duplicate names are rejected.
3. Add a coin:
   - select `Cabinet A`;
   - confirm coordinates (1,1)…(3,3);
   - choose row 2, column 3 and save.
4. Edit a second coin:
   - verify (2,3) is disabled;
   - choose another slot and save.
5. Edit the first coin:
   - verify its own (2,3) remains selectable;
   - move it to `Safe`;
   - reopen the second coin and verify (2,3) is available.
6. Bulk selection:
   - verify `Safe` is selectable;
   - verify `Cabinet A` is excluded or disabled with the per-coin-slot explanation;
   - assign selected coins to `Safe`, then clear them.
7. Duplicate:
   - duplicate a Standard-assigned coin and verify Standard location remains;
   - duplicate a tray-assigned coin and verify both location and slot are empty.
8. Settings lifecycle:
   - rename an occupied tray successfully;
   - reject resizing/deleting it with `tray_occupied`/`location_referenced`;
   - empty it, resize it, and delete it successfully.
9. Open **Collection → Storage Trays**:
   - verify every tray, including empty trays;
   - verify exact rows/columns, visible empty coordinates, and persisted placements;
   - activate an occupied well by mouse/touch, Enter, and Space;
   - verify empty wells are not buttons.
10. At narrow PWA width:
    - verify a 20-column tray stays one 20-column grid in a contained scroller;
    - verify no slot moves, disappears, packs, or paginates;
    - verify actionable targets remain at least 44×44.
11. Simulate a broken image and aggregate request failure; verify occupied fallback and retry guidance.
12. Visit existing `/tray`; verify Museum Tray drawer, responsive columns, swipe behavior, face toggle, size preference, and navigation are unchanged.

## API smoke examples

With a valid bearer token:

```powershell
$headers = @{ Authorization = "Bearer $env:AUREARIA_TOKEN"; "Content-Type" = "application/json" }

Invoke-RestMethod -Method Post -Uri http://localhost:8080/api/storage-locations -Headers $headers -Body '{"name":"Cabinet A","type":"tray","rows":3,"columns":3}'
Invoke-RestMethod -Method Get -Uri http://localhost:8080/api/storage-trays -Headers $headers
Invoke-RestMethod -Method Get -Uri http://localhost:8080/api/storage-locations/1/occupancy -Headers $headers
```

Verify 400 for invalid combinations, 404 for inaccessible IDs, and 409 with stable codes for occupied slot, occupied tray resize, and referenced deletion.

