# Implementation Plan: Quick Capture

**Branch**: `feature/async-ai-jobs` (working tree; `SPECIFY_FEATURE=336-quick-capture` used for this plan without switching branches) | **Date**: 2026-06-29 | **Spec**: `specs/336-quick-capture/spec.md`
**Input**: Feature specification from `/specs/336-quick-capture/spec.md`

**Scope boundary**: Quick Capture v1 is deterministic/manual only. It creates separate quick-capture draft/intake records and promotes them explicitly into normal `Coin` records. Existing AI intake (`/coins/intake/*`) remains unchanged and is not expanded for this MVP.

## Summary

Implement a mobile/PWA-first Quick Capture workflow for authenticated collectors to save sparse draft records with obverse/reverse photos and purchase context, resume or discard those drafts, and explicitly promote a valid draft into exactly one normal coin. The implementation adds a dedicated Go API model/repository/service/handler stack for `QuickCaptureDraft`, `DraftImage`, and lifecycle events, reuses existing coin/image validation and authenticated media display patterns, and adds Vue routes/pages/API client types for capture, draft list/resume, and promotion. Drafts remain outside normal `coins` rows until promotion, so collection counts, wishlist/sold totals, collection views, and health scoring remain unchanged until a transactional, idempotent promotion creates the normal coin.

## Technical Context

**Language/Version**: Go 1.26.1 API, Vue 3 + TypeScript + Pinia + Vite PWA frontend; Python agent unchanged and out of v1 scope.  
**Primary Dependencies**: Gin, GORM, SQLite pure-Go driver, existing JWT auth middleware, existing upload/image service patterns, axios client in `src/web/src/api/client.ts`, `lucide-vue-next`, existing design tokens/global CSS.  
**Storage**: SQLite via GORM AutoMigrate; upload files under configured `UPLOAD_DIR`; new dedicated quick-capture draft tables rather than pre-creating `coins` rows.  
**Testing**: `go test ./...`, `go vet ./...`, `npm run build` / `vue-tsc --build`, Vitest component/API-client tests; no Python test changes expected because AI enrichment is deferred.  
**Target Platform**: Self-hosted REST API + Vue SPA/PWA; mobile/PWA viewport (375px) is primary, desktop route remains functional.  
**Project Type**: Web application with Go API backend and Vue PWA frontend.  
**Performance Goals**: Save a draft with at least one photo or identifying note in under 60 seconds during usability testing; draft list loads first page (default 50) with preview metadata without scanning `coins`; promotion writes one coin and related image rows in one database transaction.  
**Constraints**: Handler → Service → Repository → Database; all GORM access in `src/api/repository/`; multi-step promotion is transactional and idempotent; authenticated owner scope on every draft operation; image extension/signature/size rules align with normal coin images; no AI enrichment/automatic attribution/agent calls in v1; drafts do not affect normal collection contracts before promotion.  
**Scale/Scope**: One owner-scoped draft queue, compact capture/resume/promote UI, obverse/reverse plus optional supplemental image handling, targeted regression coverage for collection counts, wishlist/sold totals, health scoring, existing add/edit image flows, and repeated promotion.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Gate | Status | Evidence / Plan Response |
|------|--------|--------------------------|
| Principle I — Clear Layered Architecture | PASS | Add `models`, `repository`, `services`, and `handlers` files; handlers parse/bind only; services own validation, lifecycle, promotion, and image orchestration; repositories own all GORM queries/scopes. Promotion uses `db.Transaction`. |
| Principle II — Service Boundary Separation | PASS | Quick Capture v1 is manual/deterministic and does not call the Python agent or browser-to-agent routes. Existing `/coins/intake/*` remains unchanged. |
| Principle III — Strict Types and Explicit Contracts | PASS | Add typed Go request/response structs with Swagger annotations, typed TypeScript interfaces and API-client methods, and `contracts/quick-capture-api.md`. No `any`/`@ts-ignore`. |
| Principle IV — Simple Complete Changes | PASS | Implement only manual drafts/resume/discard/promote; no AI enrichment, no broad add/edit rewrite, no parallel visual system. |
| Principle V — Security/Auth/Privacy by Default | PASS | Every draft query scopes by `user_id`; non-owned drafts return not found; uploads reuse allowlist + magic-byte validation; internal errors stay generic. |
| Principle VI — Consistent UX | PASS | Reuse design tokens/global classes, `lucide-vue-next`, `AuthenticatedImage`, existing PWA camera/upload conventions, responsive layouts. No emoji UI text. |
| §17/§21 — Workflow Contract Regression | PASS | Plan includes targeted tests for draft exclusion from counts/health/views, wishlist/sold totals, image validation/display, add/edit regressions, and idempotent promotion. |

**Post-design re-check**: PASS. The Phase 1 design artifacts keep drafts in separate tables, preserve existing collection/list/count contracts until promotion, and specify transaction + idempotency gates for promotion.

## Project Structure

### Documentation (this feature)

```text
specs/336-quick-capture/
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output (/speckit.plan command)
├── data-model.md        # Phase 1 output (/speckit.plan command)
├── quickstart.md        # Phase 1 output (/speckit.plan command)
├── contracts/           # Phase 1 output (/speckit.plan command)
│   └── quick-capture-api.md
└── tasks.md             # Phase 2 output (/speckit.tasks command - NOT created by /speckit.plan)
```

### Source Code (repository root)

```text
src/api/
├── models/
│   ├── quick_capture_draft.go
│   └── coin.go                         # unchanged normal Coin/CoinImage contract
├── repository/
│   ├── quick_capture_repository.go
│   ├── image_repository.go              # extend media lookup for draft images
│   └── scopes.go                        # reuse owner scopes; add draft scopes if useful
├── services/
│   ├── quick_capture_service.go
│   ├── image_service.go                 # extract/reuse image validation + file save helpers
│   └── coin_service.go                  # reuse coin validation on promotion
├── handlers/
│   ├── quick_capture.go
│   └── images.go                        # serve draft image media through existing authenticated media path
├── database/
│   └── database.go                      # AutoMigrate new quick capture models
└── main.go                              # wire repo/service/handler and protected routes

src/web/src/
├── api/
│   ├── client.ts                        # typed quick capture client calls
│   └── __tests__/client.test.ts
├── types/
│   └── index.ts                         # QuickCaptureDraft/DraftImage/request/response types
├── router/
│   └── index.ts                         # /quick-capture and /quick-capture/drafts routes
├── pages/
│   ├── QuickCapturePage.vue
│   ├── QuickCaptureDraftsPage.vue
│   └── QuickCaptureDraftPage.vue
├── components/quick-capture/
│   ├── QuickCaptureForm.vue
│   ├── QuickCaptureImageSlots.vue
│   ├── QuickCaptureDraftCard.vue
│   └── PromotionReadinessPanel.vue
└── App.vue                              # mobile/PWA nav entry; desktop sidebar route remains accessible

src/api/*_test.go and src/web/src/**/__tests__/
└── targeted unit, handler, service, API-client, and component regressions
```

**Structure Decision**: Use the existing two-service app structure (`src/api`, `src/web`) and add feature-specific files alongside existing coin/image/intake patterns. Do not modify the Python agent. Do not store drafts as incomplete `coins` rows because that would pollute existing collection counts, views, wishlist/sold totals, value snapshots, and health scoring.

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

No constitution violations.

## Phase 0 Research Summary

See `research.md`.

Key resolved decisions:

- Use new `quick_capture_drafts`, `quick_capture_draft_images`, and lifecycle event tables.
- Keep existing AI `CoinIntakeDraft` unchanged; it is AI-generated, 24-hour, and commit-oriented, so it does not satisfy resumable manual Quick Capture v1.
- Reuse/extract image validation and authenticated media display patterns rather than creating a parallel upload system.
- Promotion claims the draft transactionally and returns the existing promoted coin on repeated calls.

## Phase 1 Design Summary

See `data-model.md`, `contracts/quick-capture-api.md`, and `quickstart.md`.

Implementation highlights:

1. **Backend**:
   - Add `QuickCaptureDraft`, `QuickCaptureDraftImage`, and `DraftLifecycleEvent` models with owner, status, promoted coin link, and timestamps.
   - Add repository methods for owner-scoped list/get/update/discard, image association, promotion claim, and lifecycle event insertions.
   - Add `QuickCaptureService` to validate partial-save identity (title, note, or image), validate promotion readiness against normal coin rules, and orchestrate transactional promotion.
   - Reuse `CoinService` validation logic or extract shared validation so promoted coins obey existing normal create rules.
   - Extract image extension/signature/size validation from existing image upload paths so draft images accept the same extensions and magic-byte checks.
   - Extend authenticated media resolution to authorize draft image paths by draft ownership.

2. **Frontend**:
   - Add mobile-first `/quick-capture` page with photo-first obverse/reverse slots, fallback file upload, sparse fields, and explicit save states.
   - Add `/quick-capture/drafts` and `/quick-capture/drafts/:id` to list, resume, discard, and promote drafts.
   - Add `Quick Capture` PWA/sidebar navigation using existing `lucide-vue-next` icons, tokens, chips/buttons, and responsive layout patterns.
   - Use `AuthenticatedImage` for draft previews and new typed API client methods in `client.ts`.

3. **Regression coverage**:
   - Creating/updating drafts changes `/coins?wishlist=false&sold=false`, `/stats`, wishlist totals, sold totals, and health eligible counts by zero.
   - Promotion creates exactly one `Coin`, increments active collection count exactly once when not wishlist/sold, and repeated promotion returns/indicates the existing coin without duplicates.
   - Existing manual add/edit image upload and AI intake routes continue to pass unchanged.
