# Quickstart: Refine Coin Details Page for PWA and Desktop (#219)

## Prerequisites

- Web app running locally (`task up` from repo root or `npm run dev` from `src/web`).
- Authenticated user with at least one coin containing:
  - both obverse and reverse images
  - journal entries
  - notes
  - AI analysis data

## Scenario 1: Streamlined overview layout

1. Open `/coin/{id}` for a coin with two images.
2. Verify both obverse and reverse render by default in overview media.
3. Verify metadata appears in row/table format (not boxed info cards).
4. Verify overview remains visually clean and section density is reduced.

## Scenario 2: Settings-style section navigation

1. On overview, verify row links exist for:
   - `Journal`
   - `Notes`
   - `Actions`
   - `AI Analysis`
2. Click each row and verify route changes to section page (`/coin/{id}/...`).
3. Confirm each section page loads with the same coin context.
4. Navigate back and verify overview remains intact.

## Scenario 3: Capability parity after section move

1. Journal page: add and delete a journal entry.
2. Notes page: verify existing notes render correctly (including empty state when absent).
3. Actions page: run at least one existing action workflow.
4. AI Analysis page: trigger/refresh analysis and verify update flow still works.

## Scenario 4: Responsive behavior checks

1. Desktop viewport (`>=769px`):
   - Verify overview/table spacing and section links are aligned and readable.
   - Verify desktop-specific sticky behavior (if present) remains desktop-only.
2. PWA/mobile viewport (`<=768px`):
   - Verify no sticky behavior is introduced.
   - Verify overview and section pages remain ergonomic and scroll-safe.

## Validation Commands

```bash
# Frontend build/type-check parity
cd /home/brian/code/coin-collection-app/src/web
npm run build

# Optional regression guard if backend files were touched
cd /home/brian/code/coin-collection-app/src/api
go test ./...
```
