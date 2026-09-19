# Implementation Plan: Coin Copilot Collector Curator

**Branch**: `beta` (existing worktree; do not create or switch branches)
**Date**: 2026-09-19
**Spec**: [spec.md](./spec.md)

## Summary

Implement Feature 363 as three small extensions of the shipped Coin Copilot
workflow:

1. persist one optional, private collector profile per authenticated owner;
2. pass that bounded profile as advisory context when Coin Copilot composes the
   existing `collection_summary`, `portfolio_review`, and `gap_analysis`
   capabilities; and
3. restore the existing `useCoinSearchChat.addToWishlist` interaction on
   eligible cards rendered by `CopilotRunProgress.vue`.

The wishlist interaction remains a direct user action. It converts an eligible
typed dealer result to the existing `CoinSuggestion` shape, then reuses
`resolveCategoryAndEra`, `buildWishlistCoinPayload`, `createCoin`, the current
best-effort image attachment sequence, and current duplicate/repeated-click
protections. The AI receives no write tool or credential.

There is no watchlist evaluator, provenance workflow, auction change, staged
wishlist action, action/audit persistence, new provider, browser, agent,
orchestration layer, or persistence platform.

## Technical Context

**Language/Version**: Go 1.26.1; Python 3.12; TypeScript/Vue 3
**Primary Dependencies**: existing Gin/GORM/SQLite API, FastAPI/Pydantic
stateless agent, Vue/Vite/PWA client
**Storage**: one additive `collector_profiles` table only; existing `coins`,
Coin Copilot run/checkpoint/event storage, and image storage remain unchanged
**Testing**: Go unit/handler/repository/architecture tests; Python
pytest/ruff for changed Coin Copilot prompt/request behavior; Vitest/Vue Test
Utils and strict Vue type checking
**Target Platform**: existing self-hosted single-node deployment and
desktop/mobile installed PWA
**Performance Goals**: one owner-scoped profile read per relevant Copilot run;
no extra provider fan-out; wishlist save has the same latency as the shipped
legacy suggestion flow
**Constraints**: owner-derived scope; bounded profile values; Python remains
stateless/read-only; only verified and currently available `dealer_listing`
items from `market_search` are eligible; auction source/config/model/route/
service/UI files are untouched
**Scale/Scope**: personal-scale application, fewer than 10 concurrent users;
one profile row per owner

There are no unresolved `NEEDS CLARIFICATION` items.

## Constitution Check

*GATE: Passed before research and rechecked after design.*

| Authority/gate | Design response | Result |
|---|---|---|
| §0 hierarchy | Constitution 3.1.0, `docs/prd.md`, and the rewritten Feature 363 spec govern this plan. Obsolete broader design artifacts are removed. | PASS |
| Principle I | Profile work follows handler → service → repository → database. The existing coin handler/service/repository path remains the wishlist authority. | PASS |
| Principle II | Vue calls Go only. Go supplies owner-scoped profile context to the existing stateless Python Coin Copilot request. Python receives no database or write access. | PASS |
| Principle III | The profile API and additive specialist-card fields have explicit Go/TypeScript/Pydantic contracts. Existing public handlers retain Swagger coverage. | PASS |
| Principle IV | One profile table, one settings section, one bounded context addition, and restoration of an existing UI save flow are proportional. No parallel action platform is introduced. | PASS |
| Principle V | Owner ID is always derived from authentication. Profile data is excluded from public/follower responses and logs. Wishlist creation remains owner-scoped through `POST /api/coins`. | PASS |
| Principle VI | Reuse Settings, `CopilotRunProgress`, current buttons/modals, design tokens, Lucide icons, and PWA-safe layouts. | PASS |
| §§17/21 | Target exact profile isolation, read-only curator composition, dealer eligibility, mapping, duplicate, cancellation, and image-failure paths before full gates. | PASS |
| Principle VIII | No ADR is required: the reduced design follows existing storage and service boundaries and introduces no material architecture decision. | PASS |

### Post-design re-evaluation

PASS. The only durable addition is an ordinary owner-scoped profile table.
Curator guidance uses existing Coin Copilot tools and persistence. Wishlist
creation uses the existing coin and image APIs. No waiver or replacement ADR
is justified.

## Current-Code Findings and Smallest Integration Seams

| Existing symbol/path | Planned use |
|---|---|
| `src/web/src/composables/useCoinSearchChat.ts:useCoinSearchChat` | Keep the single UI-owned mutation flow and expose/reuse `addToWishlist` for Copilot result cards. |
| `resolveCategoryAndEra` and `CategoryEraConfirmModal.vue` | Reconcile live category/era options; cancellation creates nothing. |
| `buildWishlistCoinPayload` | Preserve the existing allowlisted mapping, safe category/material fallback, text bounds, supported references, `isWishlist: true`, and parsed `currentValue`. |
| `createCoin` in `src/web/src/api/endpoints/coins.ts` | Continue using `POST /api/coins`; add no wishlist-action endpoint. |
| `scrapeImage`, `proxyImage`, `uploadImage` in `useCoinSearchChat.addToWishlist` | Preserve best-effort post-create image attachment; image failure does not undo or repeat coin creation. |
| `addingIdx` and `addedSet` in `useCoinSearchChat` | Preserve in-flight/repeated-click protection and success state. |
| `CoinRepository.FindWishlistByReferenceURL` | Reuse the existing owner-scoped source-URL duplicate lookup in the canonical create transaction for wishlist coins with a reference URL. |
| `CoinService.CreateCoin`, `prepareCoinForCreate`, `createPreparedCoinInTx` | Preserve canonical validation, transaction, structured references, and value snapshot behavior. |
| `CopilotSpecialistEvidence` | Source of typed `dealer_listing`, `verification_state`, `availability`, identity fields, price, currency, and provenance. |
| `ProjectCopilotSpecialistResult` | Add only the typed dealer fields the browser needs; do not parse the rendered `facts` strings. |
| `CoinCopilotSpecialistEvidence` in `src/web/src/types/agent.ts` | Extend the existing public discriminated evidence type with optional dealer fields needed to build a `CoinSuggestion`. |
| `CopilotRunProgress.vue` | Render the explicit button only when capability is `market_search`, kind is `dealer_listing`, verification is `verified`, and availability is `available`. Never render it for `auction_search`/`auction_lot`. |
| `CoinSearchChat.vue` | Pass the existing wishlist callback/state into `CopilotRunProgress` and keep the existing category/era modal at the parent level. |
| `collection_summary`, `portfolio_review`, `gap_analysis` | Remain the complete analysis set for curator guidance. No new curator agent or provider is added. |
| `SettingsPage.vue` and settings component conventions | Add a compact Collector Profile section using the existing API client and design system. Do not use global admin settings. |

### Important current-contract gap

The internal specialist result already contains typed dealer fields, but
`CopilotSpecialistPublicEvidence` currently projects only title, URL,
verification metadata, and display `facts`. Eligibility and mapping must not
parse human-readable fact strings. The smallest contract change is to project
the existing dealer fields (`description`, `dealerName`, `listedPrice`,
`currency`, `availability`, `ruler`, `denomination`, `era`, `material`) as
optional typed fields. No auction behavior is changed.

## Project Structure

### Planning artifacts

```text
specs/363-collector-curator-watchlist-provenance/
├── spec.md
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
└── contracts/
    ├── collector-workflows.md
    └── openapi-impact.md
```

`contracts/wishlist-action.openapi.yaml` and ADR 0018 are deleted because the
staged-action architecture was rejected.

### Expected implementation touch points

```text
src/api/
├── models/collector_profile.go
├── repository/collector_profile_repository.go
├── services/collector_profile_service.go
├── handlers/collector_profile.go
├── services/coin_copilot_contract.go
├── services/coin_copilot_worker.go
├── services/coin_service.go
├── repository/coin_repository.go
├── database/database.go
├── routes_protected.go
└── main.go

src/agent/
├── app/models/requests.py
├── app/teams/coin_copilot.py
└── tests/

src/web/src/
├── api/endpoints/collectorProfile.ts
├── types/collectorProfile.ts
├── types/agent.ts
├── components/settings/CollectorProfileSection.vue
├── pages/SettingsPage.vue
├── components/chat/CopilotRunProgress.vue
├── components/CoinSearchChat.vue
└── composables/useCoinSearchChat.ts
```

No file under an auction subsystem is an implementation touch point.

## Phase 0 Research Conclusions

Research is recorded in [research.md](./research.md). The decisions are:

- one profile row per owner, with bounded optional fields and JSON arrays;
- authenticated GET/PUT profile routes, atomic full replacement, and neutral
  defaults when absent;
- optional profile context added to the existing Coin Copilot execution
  request;
- curator instructions compose only the three shipped read-only capabilities;
- the existing public specialist projection gains typed dealer fields;
- the current `addToWishlist` path is reused without any action staging API;
- existing owner/source duplicate lookup is applied in the canonical create
  transaction rather than adding action state or identity tables.

## Phase 1 Design and Contracts

### Collector profile

- Add `CollectorProfile` with a unique `UserID`.
- Store optional budget min/max and currency plus bounded JSON arrays for
  preferred periods/categories, excluded categories, preferred dealers, and
  collecting goals.
- `GET /api/collector-profile` returns neutral empty values when absent.
- `PUT /api/collector-profile` validates and replaces the complete profile in
  one repository transaction.
- Add a dedicated Settings section; do not add fields to admin `AppSetting` or
  expose profile values through `GET /auth/me`.

### Curator guidance

- Capture one bounded profile value at Coin Copilot run start or tool execution
  and pass it in the existing Go → Python request.
- Update the existing Coin Copilot prompt/planning rules so a curator request
  uses `collection_summary`, `portfolio_review`, and `gap_analysis`.
- Keep tool implementations and run persistence unchanged except for the
  optional context field.
- Require the final response to distinguish observed collection facts,
  suggestions, profile influences, and limitations.

### Dealer-only Add to Wishlist

- Extend the existing public specialist projection with typed dealer fields.
- In `CopilotRunProgress`, calculate eligibility from typed fields only:
  `capability === "market_search"`, `kind === "dealer_listing"`,
  `verificationState === "verified"`, and `availability === "available"`.
- Adapt an eligible item to `CoinSuggestion` and emit the existing
  `addToWishlist` callback. The button click is the only trigger.
- Reuse `resolveCategoryAndEra`, `buildWishlistCoinPayload`, `createCoin`, and
  image attachment exactly once.
- Reuse the owner-scoped reference-URL duplicate query within the canonical
  create transaction; keep `addingIdx`/`addedSet` for browser repeat clicks.
- Return clear existing error/success states; distinguish an image warning
  after successful creation without retrying creation.

## Implementation Phases

### Phase 1 — Private collector profile

1. Add model, migration, repository, validation service, thin authenticated
   GET/PUT handler, protected routes, and Swagger.
2. Add typed client functions and a compact Settings section.
3. Test valid save/reload/edit/clear, invalid atomic rejection, owner
   isolation, absent neutral defaults, and public/follower non-disclosure.

**Estimated size**: 2–3 implementation days.

### Phase 2 — Read-only curator context

1. Add optional bounded profile context to the existing Coin Copilot
   Go/Pydantic request contract.
2. Update the existing planner/supervisor guidance to compose
   `collection_summary`, `portfolio_review`, and `gap_analysis` for curator
   requests.
3. Test empty profile, profile-influenced explanation, sparse/contradictory
   facts, replay/cancel, and zero domain writes.

**Estimated size**: 1–2 implementation days.

### Phase 3 — Restore dealer-result Add to Wishlist

1. Add typed dealer fields to the existing public specialist projection and
   TypeScript type.
2. Wire the existing `addToWishlist` callback/state from `CoinSearchChat` into
   `CopilotRunProgress`.
3. Render the button only for verified, available dealer results; explicitly
   exclude every auction and non-eligible state.
4. Apply the existing reference-URL duplicate lookup inside the canonical
   wishlist create transaction and preserve category/era and image behavior.
5. Test eligibility, explicit-click-only mutation, mapping, cancellation,
   repeated clicks/duplicate URL, and non-fatal image failure.

**Estimated size**: 1–2 implementation days.

### Phase 4 — Focused regression and documentation sync

Run the targeted suites in [quickstart.md](./quickstart.md), regenerate
OpenAPI for the profile routes and additive result fields, then run the normal
full quality gates. Confirm no auction file or behavior changed.

**Estimated size**: 0.5–1 implementation day.

## Excluded Paths and Designs

Implementation must not:

- change auction models, repositories, services, handlers, routes,
  configuration, providers, or UI;
- add watchlist evaluation/ranking or provenance/forensic analysis;
- add `/api/wishlist-actions/*` or any stage/revise/revoke/confirm endpoint;
- add wishlist action, outcome, audit, or lifecycle tables;
- add a new agent, graph, provider, browser, scheduler, persistence service, or
  generic write tool;
- let text, model output, replay, card rendering, or tool execution call
  `createCoin`;
- map auction results, purchase/acquisition fields, storage, sold state,
  draft state, or unsupported facts into a wishlist coin.

## Complexity Tracking

No constitutional violation or waiver exists.

| Addition | Why it is necessary | Why it remains proportional |
|---|---|---|
| One `collector_profiles` table | Profile data is private owner state, not a global setting or account identity field. | One row per owner; no goals table, history, snapshots, audit, or workflow state. |
| Additive specialist public fields | The UI needs typed eligibility and mapping data already present internally. | No new provider call or specialist capability; fields are projected from the validated result. |
| Transactional wishlist URL duplicate check | The active spec requires repeated/retried saves not to create another wishlist coin. | Reuses `FindWishlistByReferenceURL`, `WithTx`, and `CreateCoin`; no action table or new endpoint. |

## Stop Point

Planning ends after these artifacts are internally consistent. Do not modify
application code, run builds, create a branch, commit, push, or alter auction
subsystem files during this planning task. `tasks.md` must be regenerated from
this reduced plan before implementation because the existing task list
describes the rejected architecture.
