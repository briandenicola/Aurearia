# Native project skills

These are the canonical repository skills. Each directory has a native
`SKILL.md` with a unique name and task-specific description; load only the skill
needed for the selected work. Skills grant no setup, execution or release
permission. Follow the constitution and shared completion gates.

| Area | Skills |
|---|---|
| Schedulers | `adding-scheduled-job`, `adding-admin-schedule` |
| Go contracts/tests | `go-sqlite-test-isolation`, `testing-gorm-many-to-many-custom-timestamps`, `route-openapi-drift-test`, `webauthn-contract-tests` |
| Web state/navigation | `module-level-state-management`, `vue-router-parent-child-navigation`, `testing-coin-detail-section-pages`, `legacy-set-type-normalization` |
| Web presentation/media | `contextual-share-cards`, `museum-tray-reuse`, `svg-chart-patterns`, `user-initiated-camera` |
| Dependencies/integrations | `npm-audit-transitive-overrides`, `python-locking`, `toolchain-pinning`, `external-service-scraping-with-fixtures` |
| Software QC | `aurearia-software-qc-audit` |

[Migration inventory](../../.squad/artifacts/native-skill-migration-2026-09-21.json)
maps all 20 old entry points to 19 packages and original Git blobs. The duplicate
npm recipes share one destination. The repository QC skill has an Aurearia-
specific name to avoid shadowing a personal `post-major-work-qc-audit` skill;
no user-level skill is changed. Legacy `.squad/skills` files are links only.

Client discovery and SpecKit evaluation are documented in the
[P4 integration guide](../../docs/agentic-native-integration.md).
