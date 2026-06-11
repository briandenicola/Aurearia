## Summary
<!-- 1-3 sentences: what changes and why -->

## Constitution self-check
- Principle(s) touched: <!-- e.g., I (Clear Layered Architecture), IV (Simple Complete Changes) -->
- Operational section(s): <!-- e.g., §17 Quality Gate, §21 DoD -->
- ADR added? <!-- yes/no — required for material design choices, §22 -->
- Principle IV check: <!-- simple, complete, proportional; note workflow/sibling paths tested -->
- Workflow contract(s): <!-- user workflow(s), API/UI/config contracts touched -->
- Blast radius / sibling workflows checked: <!-- e.g., Add Coin, Edit Coin, Admin settings, wishlist -->

## Linked work
- Issue: #
- Spec: `specs/NNN-*/spec.md`
- Tasks completed: <!-- specs/NNN-*/tasks.md line(s) -->

## Definition of Done (§21)
- [ ] 1. Code compiles (`go build ./...`, `npm run build`, `pip install -e ".[dev]"` as applicable)
- [ ] 2. Architecture tests green (`go test -run TestArchitecture ./...`)
- [ ] 3. Unit tests pass (`go test ./...`, `pytest tests/`)
- [ ] 4. Type checks pass (`vue-tsc --build`, Go compile)
- [ ] 5. Linters clean (`go vet`, `ruff check`)
- [ ] 6. Bug fixes include a targeted regression test for the exact failing user path, or document why automation is deferred
- [ ] 7. Shared workflow/config/API contract changes list blast radius and sibling workflows checked
- [ ] 8. User/admin-configured UI values are accepted by every API path the UI can submit them to, or blocked with a clear UI message
- [ ] 9. New service methods have ≥1 unit test
- [ ] 10. Public handlers have Swagger annotations
- [ ] 11. If API changed: `task openapi` run and `docs/openapi.json` updated
- [ ] 12. If material design choice: ADR added in `docs/adr/` *(when Phase 3 lands)*
- [ ] 13. Active `specs/NNN-*/tasks.md` items checked off
- [ ] 14. `.squad/decisions/inbox/` written if cross-cutting decision made
- [ ] 15. Simple Complete Changes self-check complete (Principle IV)
- [ ] 16. Secrets scan clean (no credentials in diff)
- [ ] 17. Conventional commit messages
- [ ] 18. `Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>` trailer present when AI-assisted

## Notes for reviewer
<!-- Anything reviewer should pay special attention to -->
