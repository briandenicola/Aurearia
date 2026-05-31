# Quickstart — Agentic Collection Updates (Epic F012)

## Goal

Implement the 4 epic features with one shared tool spine:

1. Structured Catalog References
2. Agentic Coin Entry
3. Collection Chat (read + confirm-gated write)
4. External Collection Tool Server parity

## Implementation Order (recommended)

1. **Feature 1 — Structured Catalog References**
   - Add `CoinReference` model/migrations/repository/service/handlers.
   - Add UI editing + display support.
   - Keep `referenceText`/`referenceURL` backward-compatible.
2. **Feature 3a — Collection Chat read spine**
   - Implement transport-agnostic tool layer read ops.
   - Wire collection mode in existing chat surface.
3. **Feature 2 — Agentic Coin Entry**
   - Add `coin_intake` team and draft/confirm APIs.
   - Build draft review UX and commit flow.
4. **Feature 3b — Collection Chat write slice**
   - Add propose/commit two-phase updates with allowlist.
   - Add disambiguation and audit journaling.
5. **Feature 4 — External Tool Server**
   - Expose same tool layer via external adapter endpoints.
   - Enforce API-key scopes and proposal-token commit model.

## Discrete Issues for Tracking

See `issues.md` for full issue breakdown and acceptance checklist for each feature card.

## Validation Checklist

### Functional checks

- Structured references can be created, validated, queried, and rendered.
- Intake drafts are editable and only commit after explicit confirmation.
- Collection chat reads are user-scoped and deterministic.
- Chat and external write flows require `propose_update` then `commit_update`.
- All commits produce journal entries with source tags.

### Quality gate commands

From repository root:

```bash
task test
cd src/api && go vet ./... && go test ./...
cd ../web && npm run build
cd ../agent && ruff check app/ tests/ && pytest tests/ -v
```

## Definition of Done

- Four issue cards complete with acceptance criteria met.
- One shared Go collection tool layer is used by in-app and external adapters.
- Read + write parity exists between in-app chat and external clients.
- No unresolved clarifications remain in planning artifacts.
