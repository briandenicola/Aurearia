# Testing CoinDetail Section Pages (CoinDetailSectionPageShell + useCoinDetailContext)

## Problem

Section pages under `/coin/:id/*` use two interlocking context sources:

1. `useCoinDetailContext()` composable — returns `{ coin: ComputedRef<Coin | null> }` from the coins store
2. `CoinDetailSectionPageShell` — renders the layout, back button, coin banner, and exposes coin via **scoped slot** (`#default="{ coin: coinData }"`)

Both point to the same coin. Mocking one without the other leaves either the computed properties or the template slot with stale/null data.

## Pattern

### 1. Shared `coinRef` at module level

```typescript
const coinRef = ref<Partial<Coin>>({ id: 42, isWishlist: false, isSold: false, /* ... */ })
```

Declare it outside any `describe` so it persists across tests. Mutate it in `beforeEach` or individual `it` blocks.

### 2. Mock both context sources to use the same ref

```typescript
vi.mock('@/composables/useCoinDetailContext', () => ({
  useCoinDetailContext: () => ({ coin: coinRef }),
}))

const ShellStub = {
  name: 'CoinDetailSectionPageShell',
  props: ['sectionTitle'],
  computed: { coin() { return coinRef.value } },
  template: '<div><slot :coin="coin" /></div>',
}
```

Because the stub's `computed.coin` reads `coinRef.value` reactively, updating `coinRef.value` before mounting causes both the `coin` script ref and the slot prop `coinData` to reflect the new value.

### 3. Mount options

```typescript
const defaultMountOptions = {
  global: { stubs: { CoinDetailSectionPageShell: ShellStub } },
}
```

Also mock `vue-router`:
```typescript
vi.mock('vue-router', () => ({
  useRoute: () => ({ params: { id: '42' } }),
}))
```

### 4. Reset in beforeEach

```typescript
beforeEach(() => {
  coinRef.value = { id: 42, isWishlist: false, isSold: false, purchasePrice: 100, purchaseDate: '...' }
  vi.mocked(myCoinApiCall).mockResolvedValue({ data: [] } as never)
})
```

### 5. Per-test coin state

```typescript
it('shows gate for wishlist coins', async () => {
  coinRef.value = { ...coinRef.value, isWishlist: true }
  const wrapper = mount(MyCoinSectionPage, defaultMountOptions)
  await flushPromises()
  // ...
})
```

## Loading state timing note

If the component sets `loading = true` inside `onMounted` (async), the component renders with `loading = false` initially. Add `await wrapper.vm.$nextTick()` after `mount()` before asserting loading state, since `onMounted` callbacks run as microtasks after the initial render.

## Example

See `src/web/src/pages/__tests__/CoinDetailValuationPage.test.ts`.
