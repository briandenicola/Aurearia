# Module-Level State Management in Vue Composables

**Category:** Frontend Architecture  
**Last Updated:** 2026-06-01  
**Author:** Aurelia

## Problem

Module-level refs exported from Vue composables persist across component lifecycle and navigation, causing state leaks when components unmount but shared refs stay dirty.

## Pattern

### Avoid Module-Level Refs for Component-Scoped State

**Anti-pattern:**
```typescript
// ❌ useBulkSelect.ts
import { ref } from 'vue'

const bulkSelectActive = ref(false) // MODULE-LEVEL — persists forever

export function useBulkSelect() {
  return { bulkSelectActive }
}
```

**Problem:** When a component using this composable unmounts, the module-level ref stays in memory with its last value. If another component (or fresh instance of the same component) later reads this ref, it sees stale state.

### Solution 1: Explicit Lifecycle Management

If module-level state is required (e.g., for global UI coordination), the owning component MUST clean up on unmount:

```typescript
// Component using the module-level ref
import { onMounted, onUnmounted } from 'vue'
import { useBulkSelect } from '@/composables/useBulkSelect'

const { bulkSelectActive } = useBulkSelect()

onMounted(() => {
  // Defensive: ensure clean state on mount
  bulkSelectActive.value = false
})

onUnmounted(() => {
  // REQUIRED: clean up when navigating away
  bulkSelectActive.value = false
})
```

**When to use:** Global flags that coordinate behavior across components (e.g., hiding UI elements when a mode is active).

### Solution 2: Pinia Store (Recommended for Shared State)

For state that must be shared across components but should reset properly:

```typescript
// stores/bulkSelect.ts
import { defineStore } from 'pinia'

export const useBulkSelectStore = defineStore('bulkSelect', {
  state: () => ({
    active: false
  }),
  actions: {
    activate() { this.active = true },
    deactivate() { this.active = false }
  }
})
```

**Benefits:**
- Explicit reset methods
- DevTools integration
- SSR-friendly
- Testable

### Solution 3: Component-Scoped Only (Simplest)

If state doesn't need to be shared, keep it local:

```typescript
// Inside component
const selectMode = ref(false)
// No composable export, no cleanup needed
```

**When to use:** State that only affects the current component.

## When Module-Level Refs Are Appropriate

- **Config/preferences that should persist** (e.g., `usePwa()` detection, theme)
- **Singletons with explicit lifecycle** (e.g., WebSocket connection, auth state)
- **Truly global flags that reset via explicit API calls** (document cleanup contract)

## Red Flags

🚨 **Danger Signs:**
- Module-level ref gates UI interactions (clicks, navigation)
- Module-level ref represents "current mode" or "active state"
- No `onUnmounted()` cleanup in consumer component
- Ref value affects components other than the one that set it

## Testing for State Leaks

1. Navigate to page A, activate mode
2. Navigate away to page B
3. Return to page A
4. Check if mode is still active (should be off)

If step 4 shows stale state, you have a module-level leak.

## Real-World Example

**Bug:** After activating bulk select mode in CollectionPage and navigating away, the agent FAB stayed hidden indefinitely because `bulkSelectActive` (module-level) remained `true`.

**Fix:** Added `onMounted()` reset and `onUnmounted()` cleanup to CollectionPage.

**Files:** 
- `src/web/src/composables/useBulkSelect.ts` — module-level ref (unchanged)
- `src/web/src/pages/CollectionPage.vue` — lifecycle hooks added

**Decision Note:** `.squad/decisions/inbox/aurelia-pwa-stuck-tap-fix.md`

## References

- Vue Composition API: [Reactive State](https://vuejs.org/guide/essentials/reactivity-fundamentals.html)
- Pinia: [State Management](https://pinia.vuejs.org/core-concepts/state.html)
- Constitution Principle IV: Strict Typing & Build Parity
