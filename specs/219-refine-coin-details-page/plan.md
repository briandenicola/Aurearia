# Implementation Plan: Refine Coin Details Page for PWA and Desktop

**Branch**: `219-refine-coin-details-page` | **Date**: 2026-05-31 | **Spec**: `/specs/219-refine-coin-details-page/spec.md`  
**Input**: Feature specification from `/specs/219-refine-coin-details-page/spec.md`

## Summary

Implement issue #219 by transforming the coin detail page into an overview-first experience: dual-side media by default, row/table metadata in place of boxed cards, and dedicated section pages for Journal, Notes, Actions, and AI Analysis with settings-style link navigation.

## Technical Context

**Language/Version**: TypeScript (Vue 3), Go 1.26.x (no planned API schema changes)  
**Primary Dependencies**: Vue Router, Pinia, Axios API client, existing coin detail components  
**Storage**: N/A (UI route/layout refactor; existing coin/journal/analysis persistence reused)  
**Testing**: `npm run build` (primary), `go test ./...` if backend wiring touched  
**Target Platform**: Web + PWA (mobile standalone and desktop browser)  
**Project Type**: Web application (frontend UX refinement)  
**Performance Goals**: Maintain current coin-detail perceived load behavior; no additional blocking request required for overview render  
**Constraints**: Must honor design tokens/classes, preserve existing section behavior, maintain PWA non-sticky and desktop-specific sticky rules  
**Scale/Scope**: Coin detail route family and related web components only

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Gate | Status | Notes |
|------|--------|-------|
| Principle V (Design Token System) | PASS | Plan enforces token/class-only styling for new table rows and link surfaces. |
| Principle IX (UI/UX Consistency) | PASS | Redesign keeps dark theme, icon system, and consistent section hierarchy. |
| Principle XIII (PWA/Mobile Interaction Rules) | PASS | Explicitly preserves no-sticky behavior in PWA and desktop-only sticky affordances. |
| Principle VII (Schema-Driven Contracts) | PASS | Route/navigation behavior documented in feature-level UI contract. |
| §17 Quality Gate | PASS | Frontend build/type-check command included in quickstart validation. |

## Project Structure

### Documentation (this feature)

```text
specs/219-refine-coin-details-page/
├── plan.md
├── spec.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── coin-detail-ui.contract.yaml
└── tasks.md
```

### Source Code (repository root)

```text
src/web/src/
├── router/
│   └── index.ts
├── pages/
│   ├── CoinDetailPage.vue
│   ├── CoinDetailJournalPage.vue       # new
│   ├── CoinDetailNotesPage.vue         # new
│   ├── CoinDetailActionsPage.vue       # new
│   └── CoinDetailAnalysisPage.vue      # new
├── components/coin/
│   ├── CoinInfoGrid.vue                # replaced/adapted by row metadata format
│   ├── CoinActivityJournal.vue
│   ├── CoinActionsPanel.vue
│   ├── CoinAIAnalysis.vue
│   └── CoinDetailSectionLinks.vue      # new settings-style row links
└── types/index.ts                      # if navigation/view-model types are added
```

**Structure Decision**: Keep scope fully in frontend route/component layer, introducing coin detail section pages and a shared settings-style section-link surface while reusing existing feature components to preserve behavior.

## Phase 0 Research (Completed)

`research.md` resolves planning decisions for:

1. Overview + dedicated-section information architecture.
2. Row/table metadata presentation replacing boxed detail cards.
3. Route strategy for section pages and deep-link behavior.
4. PWA/desktop rule preservation.
5. Component reuse strategy to minimize regression risk.

## Phase 1 Design Outputs (Completed)

1. `data-model.md` defines overview view-model, metadata rows, section links, and section page context.
2. `contracts/coin-detail-ui.contract.yaml` defines route/navigation contract for overview and section pages.
3. `quickstart.md` defines practical validation scenarios for overview UX, navigation, capability parity, and responsive behavior.
4. Agent context updated via `.specify/scripts/powershell/update-agent-context.ps1 -AgentType copilot`.

## Post-Design Constitution Check

| Gate | Status | Notes |
|------|--------|-------|
| Principle V (Design Token System) | PASS | Design artifacts specify token/class alignment and reject raw styling drift. |
| Principle IX (UI/UX Consistency) | PASS | New IA and section links provide consistent and elegant detail-page hierarchy. |
| Principle XIII (PWA/Mobile Interaction Rules) | PASS | PWA/desktop behavior requirements preserved explicitly in contract + quickstart. |
| Principle VII (Schema-Driven Contracts) | PASS | UI route contract captures expected behavior and deep-link semantics. |
| §17 Quality Gate | PASS | Validation steps include frontend production build/type parity. |

## Complexity Tracking

No constitution violations or waivers identified at planning time.
