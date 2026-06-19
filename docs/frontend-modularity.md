# Frontend Modularity — Policy & Safe Extraction Seams

> **Policy:** Do not split oversized frontend modules for their own sake. Extract only when actively refactoring the owned workflow for a product feature, security fix, or UI consistency initiative.

**Issue reference:** #314 ([Conversation](https://github.com/briandenicola/coin-collection-app/issues/314))  
**Decision record:** this policy document and PR checklist guardrail  
**Related:** Principle IV (Simple, Complete, Proportional Changes) in `.specify/memory/constitution.md`

---

## Table of Contents

- [Why This Policy](#why-this-policy)
- [Oversized Modules & Extraction Seams](#oversized-modules--extraction-seams)
- [When to Extract](#when-to-extract)
- [How to Extract](#how-to-extract)
- [Checklist for Reviewers](#checklist-for-reviewers)

---

## Why This Policy

Oversized modules create maintenance friction, but pre-emptive extraction without a driver workflow is **low-signal refactoring** — it violates Principle IV.

**Real problems with these modules:**
- ❌ Hard to review when touched (1,000+ lines = high cognitive load)
- ❌ Tight coupling to active workflows makes casual changes risky
- ❌ Extraction without the workflow is speculation

**Our solution:**
- ✅ Extract only during workflow changes (feature work, UX redesign, security patch)
- ✅ Unit test the extracted behavior, not just the result
- ✅ Record the guardrail so future teams don't repeat the pre-refactoring trap

---

## Oversized Modules & Extraction Seams

### 1. AddCoinPage.vue (1,307 lines)

**Owned workflows:**
- Manual coin intake form
- Agentic (AI-assisted) intake with image analysis
- Camera capture with focus overlay
- Form state, validation, and submission

**Current structure:**
```vue
<template>
  <!-- Entry mode toggle (manual vs agentic) -->
  <!-- Agentic section: loading, camera, capture slots -->
  <!-- Manual section: flat form fields -->
</template>

<script setup>
  // Camera state (stream, ready, error, metadata)
  // Capture state (obverse, reverse files)
  // Form state (coin object)
  // Event handlers (capture, clear, submit)
  // Intake/analysis logic (async)
</script>
```

**Safe extraction seams:**

| Seam | Extract to | Condition |
|------|-----------|-----------|
| Camera capture + focus overlay logic | `composables/useAddCoinCamera.ts` | Camera UX work, PWA mobile viewport updates |
| Form field state + validation | `composables/useAddCoinForm.ts` | Form redesign, validation rule changes |
| Agentic intake workflow | Keep in page (for now) | Too tightly coupled; revisit after Coin Lookup feature stabilizes |

**Regression tests to maintain:**
- Manual mode: end-to-end form fill → submit → coin created
- Agentic mode: camera capture → AI analysis → form pre-fill → submit
- Mobile: camera focus guide visible, capture buttons accessible, form responsive

**When to extract:**
- PR title mentions camera, mobile UX, or form validation changes
- New feature: alternative capture modes (gallery upload, URL import)

---

### 2. AdminSchedulesSection.vue (1,134 lines)

**Owned workflows:**
- Availability check scheduling (wishlist URL health)
- Auction-ending run scheduling (watch for ending lots)
- Collection health snapshot scheduling (scoring)
- Valuation run scheduling (portfolio value updates)

**Current structure:**
```vue
<template>
  <!-- Four separate scheduler configuration subsections -->
  <!-- Each subsection: enable toggle, schedule form, run history table -->
</template>

<script setup>
  // Settings state (availability, auction, health, valuation)
  // Success/error messages per section
  // Form submission handlers
  // Run history fetch + pagination
</script>
```

**Safe extraction seams:**

| Seam | Extract to | Condition |
|------|-----------|-----------|
| Scheduler table (run history, status, actions) | `components/admin/SchedulerRunsTable.vue` | Admin dashboard redesign, table sorting/filtering work |
| Individual scheduler configuration form | `components/admin/SchedulerForm.vue` | Form UX work, validation changes |
| Run detail modal/drawer | `components/admin/SchedulerRunDetail.vue` | Detail view enhancement, logs/results display work |

**Regression tests to maintain:**
- Availability check: enable/disable toggle → setting saved → next run scheduled at correct time
- Auction-ending: threshold edit → validation → save → run history reflects new config
- Health snapshot: time picker → submission → confirm settings persisted
- Valuation run: manual trigger → success message → run appears in history

**When to extract:**
- Admin panel redesign (dashboard modernization)
- Scheduler feature parity work (e.g., adding pauses, run limits)
- Form UX work (date/time picker improvements)

---

### 3. CoinLookupPage.vue (1,097 lines)

**Owned workflows:**
- Camera capture for coin identification
- Image preview grid management (add, remove, reorder)
- Numista search API + results display
- Quick add to wishlist or collection

**Current structure:**
```vue
<template>
  <!-- Capture state: camera -->
  <!-- Results state: search results + save modal -->
</template>

<script setup>
  // Camera state
  // Captured images state
  // Search results state
  // Handlers for capture, search, add/save
</script>
```

**Safe extraction seams:**

| Seam | Extract to | Condition |
|------|-----------|-----------|
| Image preview grid (display, remove, reorder) | `components/ImagePreviewGrid.vue` | Any image UX work, preview consistency across app |
| Numista results display + formatting | Keep in page (for now) | Results API may change; extract after API stabilizes |

**Regression tests to maintain:**
- Capture → results display → click "add to wishlist" → coin added
- Upload from gallery → results → save to collection → coin in collection

**When to extract:**
- Image gallery/preview component work (e.g., progressive loading, carousel)
- Lookup feature enhancement (e.g., multi-coin batch identification)

---

### 4. App.vue (819 lines)

**Owned workflows:**
- Top-level navigation and routing
- Sidebar menu with collapsible sections (Stats, Collection)
- Sidebar reordering (drag-to-reorder menu items)
- Notification badge (unread count)
- Theme/layout management

**Current structure:**
```vue
<template>
  <!-- Nav bar: logo, brand, actions -->
  <!-- Sidebar: drag-reorder edit mode, nav items, submenus, badges -->
  <!-- Main router-view -->
</template>

<script setup>
  // Sidebar state (open, editMode, expandedSections, orderedItems)
  // Notification state (unread count)
  // Route/auth state
  // Drag-reorder handlers (sortable)
</script>
```

**Safe extraction seams:**

| Seam | Extract to | Condition |
|------|-----------|-----------|
| Sidebar reorder logic | `composables/useSidebarReorder.ts` | Nav UX work, menu personalization feature |
| Nav bar layout | Could stay inline | Rarely changed; extract only if nav redesign is planned |

**Regression tests to maintain:**
- Auth state → nav visible; logged out → login page
- Sidebar open → click menu item → route changes, sidebar closes (mobile)
- Drag reorder → new order persists across page reload
- Notification badge updates in real-time

**When to extract:**
- Navigation redesign (e.g., bottom nav for mobile, top nav restructure)
- Sidebar personalization feature (save menu order per user)

---

### 5. client.ts (780 lines)

**Owned workflows:**
- Authentication (login, register, token refresh)
- Coin CRUD (list, get, create, update, delete, purchase, sell)
- Tags, sets, bulk operations
- Admin endpoints (users, settings, catalogs, schedules)
- Agent chat (SSE streaming)
- Auctions, notifications, valuation, availability checks

**Current structure:**
```ts
// Axios instance + interceptors (JWT, 401 refresh queue)

// Auth: login, register, refresh
// Coins: CRUD + bulk operations
// Tags, Sets, Storage Locations
// Catalogs, Mint Locations
// Auctions, Bulk Operations
// Notifications, Agent, Valuation, Availability
// ...
```

**Safe extraction seams:**

| Seam | Extract to | Condition |
|------|-----------|-----------|
| Coin CRUD endpoints | `api/coin.ts` | Coin API versioning, bulk refactor of coin workflows |
| Admin endpoints | `api/admin.ts` | Admin panel restructure, admin feature parity |
| Agent endpoints | `api/agent.ts` | Agent service upgrade, LLM provider changes |
| Auction endpoints | `api/auction.ts` | Auction feature expansion |

**Regression tests to maintain:**
- For each domain group being extracted: at least 1 call-site integration test in the component using it (e.g., if extracting coin.ts, test AddCoinPage calls extracted functions)

**When to extract:**
- API versioning or major contract changes
- Multi-domain refactoring (e.g., auth overhaul affecting many endpoints)
- Organizing by feature boundaries (coin domain, admin domain, etc.)

---

## When to Extract

### ✅ DO Extract

1. **PR title/spec touches the owned workflow**
   - "Refactor camera UX" → extract `useAddCoinCamera`
   - "Admin dashboard redesign" → extract admin components
   - "Coin API versioning" → extract `api/coin.ts`

2. **Component is actively undergoing UX or logic changes**
   - Adding new form fields? Extract form state first.
   - Changing sidebar menu? Extract reorder logic first.

3. **Extracted piece has clear, testable responsibility**
   - "Camera initialization and stream state" ✅
   - "All of AddCoinPage logic" ❌

4. **Tests can accompany the extraction**
   - New composable → ≥1 unit test for state/methods
   - New component → ≥1 test for rendering or events
   - API module split → ≥1 integration test per endpoint domain

### ❌ DON'T Extract

1. **Pre-refactoring** — "This file is too big, let's split it"
2. **Speculative extraction** — "We might need this later"
3. **Component is not being actively changed** — Extraction adds risk with no driver benefit
4. **No tests for the extracted behavior** — Extraction without tests = untestable code

---

## How to Extract

### 1. Identify the Seam (Refer to Tables Above)

Pick the safe seam from the module's section above. Don't invent new abstractions.

### 2. Write Tests First (TDD for Extraction)

For a composable:
```ts
// tests/composables/useAddCoinCamera.spec.ts
describe('useAddCoinCamera', () => {
  it('initializes camera stream on mount', () => { ... })
  it('stops stream on unmount', () => { ... })
  it('captures frame from video element', () => { ... })
})
```

For a component:
```ts
// tests/components/SchedulerRunsTable.spec.ts
describe('SchedulerRunsTable.vue', () => {
  it('renders run rows with status', () => { ... })
  it('emits cancel-run on cancel button click', () => { ... })
})
```

### 3. Extract the Code

Move the logic to the new file. Update imports in the parent component.

### 4. Regression Test the Parent Workflow

End-to-end test on the page or component that uses the extracted code:

```ts
// tests/e2e/addCoin.spec.ts (Playwright)
test('Manual coin intake: form fill → submit → coin created', async ({ page }) => {
  await page.goto('/add')
  await page.fill('input[name="name"]', 'Denarius')
  await page.click('button[type="submit"]')
  await expect(page).toHaveURL('/collection')
  // Verify coin appears in list
})
```

### 5. PR Checklist

- [ ] Extracted code has ≥1 unit test
- [ ] Parent workflow end-to-end test still passes
- [ ] Imports updated correctly in parent component
- [ ] No dead code left in original file
- [ ] Commit message ties to the driver issue/PR (e.g., "Extract camera logic for mobile UX work #XYZ")

---

## Checklist for Reviewers

When reviewing a PR that extracts from one of the oversized modules:

- [ ] Extraction is driven by a workflow change (feature, UX, security), not pre-refactoring
- [ ] New code (extracted composable/component) has ≥1 unit test
- [ ] Parent component's end-to-end workflow is tested (no regression)
- [ ] Blast radius checked: Are other components affected? Are they tested?
- [ ] Extracted code is reusable elsewhere, not component-specific (or explicitly justified as local)
- [ ] Principle IV (Simple, Complete, Proportional) satisfied: extraction solves the real problem, not a symptom

**If extraction lacks tests or is pre-refactoring:** Request changes per Strict Lockout (§18.2). Don't merge extraction without justifying the driver workflow.

---

## References

- **Issue #314:** [GitHub Issue](https://github.com/briandenicola/coin-collection-app/issues/314)
- **Principle IV:** Constitution `§17 Quality Gate` + Principle IV (Simple, Complete, Proportional)
- **Test Infrastructure:** `docs/testing.md` — Vitest, Playwright, test patterns
- **Architecture:** `docs/ARCHITECTURE.md` — Vue 3 Frontend section
