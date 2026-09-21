# Storage Location + Settings Batch

**Timestamp:** 2026-06-01T20:26:20Z
**Requested by:** briandenicola (Brian)
**Scribe:** session logging, decision merge, history handoff, commit

## Summary

- Cassius added the Storage Location API and coin assignment contract.
- Aurelia added Storage Location management in Settings/Data, coin form assignment, and detail display.
- Aurelia reorganized Settings so backups/imports/API keys live under `Backups & Keys` while `Data` keeps lookup metadata only.

## Validation Reported by Implementers

- Backend: `task openapi`, `go build ./...`, `go vet ./...`, `go test -v ./...` pass.
- Frontend: `npm run build` and `npm run lint` pass; `npm test` has unchanged pre-existing token-budget failures.

## Constitution Alignment

- Principle I / II: backend route/service/repository/model layering and DI preserved.
- Principle V / IX: UI follows design-token/global-class direction and Settings IA remains PWA-compatible.
- Principle VII / §17: OpenAPI regenerated and quality gates reported.
