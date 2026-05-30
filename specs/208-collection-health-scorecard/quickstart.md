# Quickstart — Collection Health Scorecard (v1)

## Goal

Implement and validate feature #208:

1. Collection-level health score + grade
2. Per-coin health score + missing checklist
3. Needs Attention queue with quick actions
4. 30-day trend indicator
5. Admin aggregate health panel

## Prerequisites

- API, web, and DB environment available (`docker-compose.yaml` or local services).
- Authenticated user with sample coins in mixed completeness states.
- Optional admin account for aggregate panel verification.

## Implementation Order (recommended)

1. **Backend scoring core**
   - Add health scoring service/repository logic.
   - Add grade mapping + checklist generation.
2. **Trend persistence**
   - Add `collection_health_snapshots` model + migration.
   - Add daily snapshot scheduler/upsert job.
3. **API endpoints**
   - `GET /api/stats/health`
   - `GET /api/coins/health?scope=needs_attention`
   - `GET /api/admin/health/summary` (admin-only)
4. **Frontend integration**
   - Add TS types + API client methods.
   - Render scorecard on dashboard and coin list/detail.
   - Add Needs Attention queue UI + quick actions.
   - Add admin aggregate panel card/table.
5. **Tests & docs**
   - Unit tests for score math and grading.
   - Integration tests for new endpoints/auth gates.
   - UI tests for queue ordering and trend display.
   - Document formula and thresholds.

## Validation Checklist

### Functional checks

- Dashboard shows `score`, `grade`, and 30-day delta.
- Per-coin rows show score + grade + checklist.
- Needs Attention queue sorted lowest score first.
- Quick actions route to existing edit/image/valuation/analysis workflows.
- Admin panel shows median, low-score %, and top missing fields.

### Quality gate commands

From repository root:

```bash
task test
cd src/api && go vet ./... && go test ./...
cd ../web && npm run build
```

Run strict frontend type checks in web container / CI-equivalent flow per constitution:

```bash
cd src/web
npm run lint
```

## Test Data Suggestions

- Coin A: complete metadata + both images + fresh valuation + AI fields (expect A/B).
- Coin B: partial metadata + one image + stale valuation + no AI fields (expect D/F).
- Coin C: no valuation history but complete metadata/images (valuation gap highlighted).

## Definition of Done for this feature

- All five scope bullets from issue #208 are visible and testable.
- Scoring formula and thresholds are documented and deterministic.
- Admin endpoint is protected by existing admin middleware.
- No unresolved clarifications remain in planning artifacts.
