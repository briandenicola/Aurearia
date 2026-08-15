# Quickstart: Optional Nomisma.org Authority Linking

This is a validation/rollout checklist for the implementing PR(s), not a
user manual. It maps directly to §21 Definition of Done and the planning
constraints given for this feature.

## 1. Backend build/test loop

```powershell
cd src\api
go vet ./...
go test ./... -run TestArchitecture   # layering unaffected
go test ./...                          # includes new nomisma_client_test.go,
                                        # nomisma_cache_test.go,
                                        # mint_location_service_test.go,
                                        # mint_location_handler_test.go
```

All Nomisma HTTP interaction in tests MUST go through `httptest.Server`
fixtures constructed the same way `NewGeocodeServiceForTest`/existing
Numista tests stub their providers — no test may reach the real
`nomisma.org` host (constraint: "external-client fixture tests without
relying on live Nomisma in CI").

## 2. Frontend build/test loop

```powershell
cd src\web
npm run build
npx vue-tsc --build
npm run test -- NomismaAttribution AdminCoinPropertiesSection MintCoinDrawer
```

## 3. Manual admin walkthrough (mirrors User Stories 1–3)

1. Log in as an admin, open the global mint-locations panel
   (`AdminCoinPropertiesSection.vue`).
2. Pick an unlinked global mint, trigger a Nomisma search with a real mint
   name → confirm candidates render with label + score, none pre-selected.
3. Confirm one candidate → mint location now shows
   "Source: Nomisma.org · CC BY 4.0" linking to the chosen concept and the
   CC BY 4.0 license page.
4. Open Mint Map, select that mint's pin → drawer shows the same
   attribution line.
5. Re-search and confirm a different candidate → old link is replaced;
   attribution now points at the new concept; name/coordinates/aliases
   unchanged (diff the mint location before/after — only the three Nomisma
   fields differ).
6. Unlink → attribution disappears; mint behaves exactly like a
   never-linked mint; coin associations unaffected.
7. Simulate zero-result search (obscure/gibberish query) → "no match found",
   no link created, mint fully usable.
8. Simulate Nomisma unavailability (point the client at a closed port /
   stub a 500 in a lower environment) → "lookup unavailable" state; mint
   location and all coin CRUD continue working; no 5xx surfaced to the
   admin UI.

## 4. Private-mint authorization checks (User Story 4)

- Attempt `GET/POST/DELETE /admin/mint-locations/{privateMintId}/nomisma`
  against a known private mint ID (owned by any user, including the acting
  admin's own private mint) → expect `404`, and confirm (via test double /
  request capture) that no outbound Nomisma HTTP call was made for that
  request.
- Confirm no search/link/unlink controls render in any UI path reachable
  from a private mint (coin form's "My Mints" entries, `CreateMintModal`,
  `SettingsDataSection`'s custom mint list).

## 5. Migration / backward compatibility

- Fresh `go run .` against an existing SQLite database with pre-existing
  `mint_locations` rows → `AutoMigrate` adds the three nullable columns
  without error; existing rows read back with `nomismaUri/-Label/-LinkedAt`
  absent (not linked).
- Roll the API binary back to the pre-feature version against a database
  that already has confirmed links → existing rows still load correctly
  (older GORM model simply doesn't select/write the new columns); no data
  loss; re-upgrading restores visibility of the previously confirmed links.
- Confirm feature 338's coin-mint backfill/legacy free-text reconciliation
  behavior is unaffected: rerun `backfillCoinMintLocations`-covering tests
  and confirm they're still green untouched.

## 6. Observability

- Confirm any new logging/telemetry around Nomisma search/link/unlink logs
  outcome/status/latency only — never the raw query text, matched label, or
  the requesting user's identity beyond what standard request logging
  already includes (mirrors ADR 0007's "never records keys, queries,
  evidence text... or user identity" rule, scoped down to this feature's
  much smaller surface).

## 7. Rollout / rollback

- **Rollout order**: backend first (additive columns + routes), then
  frontend (new UI is inert until the backend routes exist; existing mint
  panel keeps working with the old build against the new backend since the
  new fields are optional/omitted).
- **Rollback**: redeploy the previous backend/frontend build pair. No
  destructive migration exists to reverse; the three additive columns are
  simply unused by the older binary. Do not manually drop the columns during
  an emergency rollback — that decision (if ever needed) requires its own
  migration and ADR, per ADR 0007's precedent.

## 8. Required artifacts before marking this feature done

- [ ] `docs/adr/000X-nomisma-authority-linking.md` if the implementing PR
      finalizes the client/cache boundary as designed here (or documents a
      deviation).
- [ ] Swagger regenerated (`swag`) and root `openapi.yaml` updated for the
      three new admin routes.
- [ ] `.squad/decisions/inbox/` entry if implementation reveals any
      contradiction with this plan (per the "stop rather than silently
      change product behavior" constraint) — otherwise no entry is required
      for this planning pass.
