# Skill: Vue Router Parent-Child Navigation

## Context

When implementing form/edit pages that save and return to a parent detail/list view, incorrect router navigation can pollute the browser history stack with duplicate entries, causing unexpected back-button behavior.

**Symptom:** User edits a coin, saves, returns to detail view, then clicks back — lands on the edit page again instead of the gallery.

## Pattern

**Parent-child relationship:** A page (Parent) links to a form/edit page (Child). After the Child saves, it should return to Parent without leaving itself in the history stack.

## Rules

### ✅ Child → Parent After Save: Use `router.back()`

When a form/edit page successfully saves and should return to its parent:

```typescript
// EditCoinPage.vue (or any edit/form page)
async function handleSubmit() {
  saving.value = true
  try {
    await updateCoin(form.id!, form)
    // ... handle image uploads, etc.
    
    router.back()  // ✅ Pops the Edit entry, returns to original parent
  } catch {
    await showAlert('Failed to update', { title: 'Error' })
  } finally {
    saving.value = false
  }
}
```

**Why it works:**
- Stack before save: `[Gallery, Detail, Edit]`
- Stack after `router.back()`: `[Gallery, Detail]`
- Detail's back button now correctly goes to Gallery

### ❌ Avoid: `router.replace()` with path string

```typescript
// DON'T do this:
router.replace(`/coin/${coinId}`)  // ❌ Creates duplicate Detail entry
```

**Why it fails:**
- Vue Router treats the path-based replace as a NEW Detail entry
- Stack becomes: `[Gallery, Detail_old, Detail_new]`
- Back button lands on `Detail_old`, making it appear stuck

### ✅ Cancel Button: Also use `router.back()`

Form cancel buttons should mirror the save behavior:

```vue
<!-- CoinForm.vue -->
<button type="button" class="btn btn-secondary" @click="$router.back()">Cancel</button>
```

### ✅ Parent → Child: Use `<router-link>` or `router.push()`

Navigating FROM parent TO child should use push (default behavior):

```vue
<!-- CoinDetailHeaderActions.vue -->
<router-link :to="`/edit/${coinId}`" class="btn btn-secondary btn-sm">Edit</router-link>
```

Or:
```typescript
router.push(`/edit/${coinId}`)
```

### When to Use `router.replace()`

Use `replace` only when you want to **replace the current entry** without being able to return to it:

- Login → Dashboard (don't allow back to login after authenticated)
- Redirect flows (e.g., `/process-image` → `/settings?tab=process`)

### ⚠️ Hub Pages with Multiple Subpages: Use Absolute Navigation

**NEW (2026-06-01):** When a parent page has multiple child subpages that return via "Back to Overview" buttons, the parent's own back button **must use absolute navigation** to avoid history pollution.

**Problem:** Subpages correctly use `router.push('/coin/:id')` to return to the parent hub, allowing continued exploration. This adds the hub page to the stack multiple times. If the hub's back button uses `router.back()`, it pops to the most recent subpage instead of the grandparent list.

**Example:** Coin Detail page with journal/health/analysis subpages:

```typescript
// ❌ WRONG: CoinDetailHeaderActions.vue
<button @click="router.back()">← Back</button>

// Stack after Gallery → Detail → Journal → "Back to Overview":
// [Gallery, Detail_1, Journal, Detail_2]
// Clicking Back on Detail_2 goes to Journal ❌

// ✅ CORRECT: Use absolute navigation to gallery
<button @click="router.push('/')">← Back to Gallery</button>

// Now Back always goes to Gallery regardless of subpage history ✅
```

**When to apply this pattern:**
- Parent page is a **hub** with multiple child subpages (detail sections, settings tabs, etc.)
- Subpages use "Back to Overview" buttons that push back to the hub
- Hub's back button should always return to a specific grandparent (gallery, list, etc.)

**Implementation (commit 6747a6d):**
- `CoinDetailHeaderActions.vue` changed from `router.back()` to `router.push('/')`
- Button label changed from "Back" to "Back to Gallery" for clarity

## Edge Case: Deep Linking

If a user lands directly on an edit page (e.g., via bookmark or external link), `router.back()` may have nowhere to go. Consider:

```typescript
async function handleSubmit() {
  // ... save logic
  
  if (window.history.length <= 2) {
    // User likely deep-linked; navigate explicitly
    router.replace(`/coin/${coinId}`)
  } else {
    router.back()
  }
}
```

**However:** In Ancient Coins, edit pages require auth and are not meant to be direct-entry points, so this is currently not needed. Document if/when we add such handling.

## Related Patterns in This Codebase

### Section Pages (journal, health, notes, actions, analysis)

These are **sibling pages**, not parent-child. They navigate to Detail overview explicitly:

```typescript
// useCoinDetailContext.ts
function navigateToOverview() {
  if (coinId.value) {
    router.push(`/coin/${coinId.value}`)  // ✅ Correct for sibling navigation
  }
}
```

**Stack when navigating from section pages:**
```
[Gallery, Detail, Journal] → (back to overview via push) → [Gallery, Detail, Journal, Detail]
```

This is intentional: sections are not children of Detail, they're alternate views of the same coin. The user can navigate back through Journal to Detail_original.

### Delete Flow

After deleting a coin, explicitly navigate to a safe parent:

```typescript
// CoinDetailPage.vue
async function handleDelete() {
  if (!coin.value || !await showConfirm('Delete this coin?')) return
  await deleteCoin(coin.value.id)
  router.push('/')  // ✅ Explicit navigation since current coin no longer exists
}
```

## Testing the Pattern

After implementing a parent-child form flow:

1. **Normal flow:**
   - Parent → Child → Save → Parent → Back → Grandparent ✅
2. **Cancel flow:**
   - Parent → Child → Cancel → Parent → Back → Grandparent ✅
3. **Deep link edge case (if applicable):**
   - Load Child directly → Save → lands safely ✅

## Verification Commands

```bash
cd src/web/
npm run type-check  # vue-tsc --build must pass
npm run build       # production build must pass
npm run lint        # eslint must pass (warnings ok if pre-existing)
```

## Constitution References

- **Principle IV (Strict Typing & Build Parity):** Ensure vue-tsc --build passes with stricter checks
- **Principle IX (UI/UX Consistency):** Back-button behavior must be predictable and consistent across the app
- **Principle XIII (PWA/Mobile Rules):** Navigation stack correctness is critical for mobile back-gesture UX

## When NOT to Use This Pattern

- **Redirects after auth:** Use `router.replace('/')` to prevent back to login
- **Sibling page navigation:** Use `router.push()` when navigating between peer pages (e.g., section pages within coin detail)
- **External navigation:** Use explicit paths when the parent route is dynamic/unknown

## Implementation Checklist

- [ ] Child page save handler uses `router.back()`
- [ ] Child page cancel button uses `$router.back()` or `router.back()`
- [ ] Parent → Child navigation uses `router.push()` or `<router-link>` (default)
- [ ] Test both save and cancel flows manually
- [ ] Verify `vue-tsc --build` passes
- [ ] Check that parent's back button returns to grandparent, not child

## Historical Context

**Issue:** After editing a coin from detail view, back button landed on edit page instead of gallery.

**Root cause:** EditCoinPage used `router.replace('/coin/:id')` which created a duplicate Detail entry.

**Fix:** Changed to `router.back()` (commit 9ca10ea, 2026-06-01).

**Related files:**
- `src/web/src/pages/EditCoinPage.vue` — edit form that returns to detail
- `src/web/src/components/coin/CoinDetailHeaderActions.vue` — detail header with back button
- `src/web/src/components/CoinForm.vue` — shared form with cancel button
- `src/web/src/router/index.ts` — route definitions

**Decision record:** `.squad/decisions/inbox/aurelia-back-nav-fix.md`
