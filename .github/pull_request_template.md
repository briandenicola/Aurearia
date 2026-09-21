## Summary
<!-- 1-3 sentences: what changes and why -->
<!-- ADR 0019 consumer draft: prospective policy awaits section 22 acceptance. -->

## Constitution self-check
- Principle(s) touched: <!-- e.g., I (Clear Layered Architecture), IV (Simple Complete Changes) -->
- Operational section(s): <!-- e.g., §17 Quality Gate, §21 DoD -->
- ADR added? <!-- yes/no — required for material design choices, §22 -->
- Principle IV check: <!-- simple, complete, proportional; note workflow/sibling paths tested -->
- Workflow contract(s): <!-- user workflow(s), API/UI/config contracts touched -->
- Blast radius / sibling workflows checked: <!-- e.g., Add Coin, Edit Coin, Admin settings, wishlist -->

## Linked work
- Issue: #
- Authorizing spec / issue / process plan:
- Lane and non-goals:
- Task/evidence source: <!-- do not infer acceptance from checked tasks -->
- Tested/reviewed commit or dirty-tree identity:
- State: <!-- implemented / verified / accepted / released -->
- Outstanding reviewer blocks and required owner approvals:

## Definition of Done (§21)
<!-- Use docs/testing.md section 6. N/A needs a reason; unavailable evidence is pending. -->
- [ ] 1. Approved scope/lane recorded and applicable builds pass; dependency installation is not a build check
- [ ] 2. Applicable architecture/contract tests pass using verified selectors
- [ ] 3. Applicable Go, full web package, and locked-environment agent tests pass
- [ ] 4. Type checks pass (`vue-tsc --build`, Go compile)
- [ ] 5. Applicable linters clean, including zero-warning frontend lint
- [ ] 6. Bug fixes include a targeted regression test for the exact failing user path, or document why automation is deferred
- [ ] 7. Shared workflow/config/API contract changes list blast radius and sibling workflows checked
- [ ] 8. User/admin-configured UI values are accepted by every API path the UI can submit them to, or blocked with a clear UI message
- [ ] 9. New service methods have ≥1 unit test
- [ ] 10. Public handlers have Swagger annotations
- [ ] 11. If API changed: `task openapi` run and `docs/openapi.json` updated
- [ ] 12. If material design choice: ADR proposed/accepted through the required process, never inferred from code
- [ ] 13. Authoritative tasks/issue state reconciled with evidence and deferrals
- [ ] 14. `.squad/decisions/inbox/` written if cross-cutting decision made
- [ ] 15. Simple Complete Changes self-check complete (Principle IV)
- [ ] 15a. If touched oversized module (see [#314](https://github.com/briandenicola/coin-collection-app/issues/314)): extraction seams reviewed + regression tests maintained
- [ ] 16. Secrets scan clean (no credentials in diff)
- [ ] 17. Conventional commit messages and required `Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>` trailer
- [ ] 18. Independent review evidence, block clearance, and required owner approval bound to the candidate

## Verification and release boundary
- Commands / results / CI evidence:
- Exact acceptance and sibling workflow evidence:
- Unavailable checks / manual exceptions:
- Owner authorization for main/release: <!-- pending unless explicitly granted for this candidate -->

<!-- Race/security/compatibility/container requirements remain applicable.
     Green CI, beta pushes, and this checklist are not release approval. -->

## Notes for reviewer
<!-- Anything reviewer should pay special attention to -->
