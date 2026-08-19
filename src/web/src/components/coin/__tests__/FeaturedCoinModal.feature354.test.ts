import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import FeaturedCoinModal from '../FeaturedCoinModal.vue'
import { buildRomanDenariusCore } from '@/test/fixtures/coins'
import type { FeaturedCoin } from '@/types'

// Independent QA regression coverage for spec 354 (Move to Collection,
// FR-D9), owned by Brutus (Tester/QA). Dedicated new file, additive to
// FeaturedCoinModal.test.ts (Aurelia's happy-path coverage of the
// Wishlist badge / basic Move to Collection dispatch) — this file targets
// edge cases her suite does not assert on: idempotent double-click
// prevention (mobile users double-tapping), error surfacing when the move
// PATCH fails, and the "already moved by another path" case where
// sourceType is still 'wishlist' but coin.isWishlist has already flipped
// to false (e.g. the user used Edit Coin in another tab).

const mocks = vi.hoisted(() => ({
  getFeaturedCoin: vi.fn(),
  updateCoin: vi.fn(),
  getApiErrorMessage: (error: unknown) => (error instanceof Error ? error.message : String(error)),
  shareCoinCard: vi.fn(),
  sharing: {
    __v_isRef: true,
    value: false,
  },
}))

vi.mock('@/api/client', () => ({
  getFeaturedCoin: mocks.getFeaturedCoin,
  updateCoin: mocks.updateCoin,
  getApiErrorMessage: mocks.getApiErrorMessage,
}))

vi.mock('@/composables/useCoinShareCard', () => ({
  useCoinShareCard: () => ({
    sharing: mocks.sharing,
    shareCoinCard: mocks.shareCoinCard,
  }),
}))

const routerLinkStub = {
  props: ['to'],
  template: '<a :href="to"><slot /></a>',
}

describe('FeaturedCoinModal — Move to Collection edge cases (spec 354)', () => {
  beforeEach(() => {
    mocks.getFeaturedCoin.mockReset()
    mocks.updateCoin.mockReset()
    mocks.shareCoinCard.mockReset()
    mocks.sharing.value = false
  })

  it('does not offer Move to Collection when the underlying coin was already moved by another path', async () => {
    // sourceType is still 'wishlist' (the pick was made while it was still
    // a wishlist item) but coin.isWishlist has since flipped to false via
    // some other flow (e.g. Edit Coin) — this must render byte-identical
    // to the owned-source case per the D9 code comment in the component.
    const coin = buildRomanDenariusCore({ id: 42, name: 'Trajan Denarius Core', isWishlist: false })
    mocks.getFeaturedCoin.mockResolvedValue({ data: buildFeaturedCoin({ sourceType: 'wishlist', coin }) })

    const wrapper = mountFeaturedCoinModal()
    await flushPromises()

    expect(findMoveButton(wrapper)).toBeUndefined()
  })

  it('prevents a duplicate PATCH when Move to Collection is double-clicked before the first request resolves', async () => {
    mocks.getFeaturedCoin.mockResolvedValue({ data: buildFeaturedCoin({ sourceType: 'wishlist' }) })
    let resolveUpdate!: (value: { data: Record<string, never> }) => void
    mocks.updateCoin.mockReturnValue(new Promise((resolve) => { resolveUpdate = resolve }))

    const wrapper = mountFeaturedCoinModal()
    await flushPromises()

    const button = findMoveButton(wrapper)!
    await button.trigger('click')
    await button.trigger('click')
    await button.trigger('click')

    expect(mocks.updateCoin).toHaveBeenCalledTimes(1)

    resolveUpdate({ data: {} })
    await flushPromises()

    expect(wrapper.text()).toContain('Moved to your collection')
  })

  it('surfaces an error message and re-offers the action when the move PATCH fails', async () => {
    mocks.getFeaturedCoin.mockResolvedValue({ data: buildFeaturedCoin({ sourceType: 'wishlist' }) })
    mocks.updateCoin.mockRejectedValue(new Error('Network error moving coin'))

    const wrapper = mountFeaturedCoinModal()
    await flushPromises()

    await findMoveButton(wrapper)!.trigger('click')
    await flushPromises()

    expect(wrapper.text()).toContain('Network error moving coin')
    expect(wrapper.text()).not.toContain('Moved to your collection')
    // The action must still be available for retry after a failure — the
    // button is not permanently removed just because one attempt failed.
    expect(findMoveButton(wrapper)).toBeDefined()
  })

  it('re-fetches and re-evaluates Move to Collection eligibility when featuredCoinId prop changes', async () => {
    mocks.getFeaturedCoin
      .mockResolvedValueOnce({ data: buildFeaturedCoin({ id: 1, sourceType: 'wishlist' }) })
      .mockResolvedValueOnce({ data: buildFeaturedCoin({ id: 2, sourceType: 'owned' }) })

    const wrapper = mountFeaturedCoinModal()
    await flushPromises()
    expect(findMoveButton(wrapper)).toBeDefined()

    await wrapper.setProps({ featuredCoinId: 2 })
    await flushPromises()

    expect(mocks.getFeaturedCoin).toHaveBeenCalledWith(2)
    expect(findMoveButton(wrapper)).toBeUndefined()
  })
})

function mountFeaturedCoinModal() {
  return mount(FeaturedCoinModal, {
    props: {
      featuredCoinId: 9001,
    },
    global: {
      stubs: {
        Teleport: true,
        RouterLink: routerLinkStub,
        AuthenticatedImage: true,
        Sparkles: true,
        Share2: true,
        X: true,
      },
    },
  })
}

function findMoveButton(wrapper: ReturnType<typeof mountFeaturedCoinModal>) {
  return wrapper.findAll('button').find((button) => button.text().includes('Move to Collection'))
}

function buildFeaturedCoin(overrides: Partial<FeaturedCoin> = {}): FeaturedCoin {
  const coin = overrides.coin ?? buildRomanDenariusCore({
    id: 42,
    name: 'Trajan Denarius Core',
    isWishlist: overrides.sourceType === 'wishlist',
  })

  return {
    id: 9001,
    userId: 101,
    coinId: coin.id,
    coin,
    summary: 'Trajan denarius summary with obverse and reverse details.',
    sourceType: 'owned',
    featuredAt: '2026-06-20T12:00:00Z',
    createdAt: '2026-06-20T12:00:00Z',
    ...overrides,
  }
}
