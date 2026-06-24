import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import FeaturedCoinModal from '../FeaturedCoinModal.vue'
import { buildRomanDenariusCore } from '@/test/fixtures/coins'
import type { FeaturedCoin } from '@/types'

const mocks = vi.hoisted(() => ({
  getFeaturedCoin: vi.fn(),
  shareCoinCard: vi.fn(),
  sharing: {
    __v_isRef: true,
    value: false,
  },
}))

vi.mock('@/api/client', () => ({
  getFeaturedCoin: mocks.getFeaturedCoin,
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

describe('FeaturedCoinModal', () => {
  beforeEach(() => {
    mocks.getFeaturedCoin.mockReset()
    mocks.shareCoinCard.mockReset()
    mocks.shareCoinCard.mockResolvedValue({ mode: 'downloaded' })
    mocks.sharing.value = false
  })

  it('renders a share action after loading the featured coin', async () => {
    mocks.getFeaturedCoin.mockResolvedValue({ data: buildFeaturedCoin() })

    const wrapper = mountFeaturedCoinModal()
    await flushPromises()

    expect(mocks.getFeaturedCoin).toHaveBeenCalledWith(9001)
    expect(wrapper.text()).toContain('Coin of the Day')
    expect(wrapper.text()).toContain('Trajan Denarius Core')
    expect(findShareButton(wrapper)?.text()).toContain('Share')
  })

  it('shares the underlying coin with the Coin of the Day summary context', async () => {
    const obverseReverseSummary = 'Obverse: laureate portrait of Trajan. Reverse: Victory standing with wreath.'
    const featured = buildFeaturedCoin({
      summary: obverseReverseSummary,
    })
    mocks.getFeaturedCoin.mockResolvedValue({ data: featured })

    const wrapper = mountFeaturedCoinModal()
    await flushPromises()
    await findShareButton(wrapper)!.trigger('click')
    await flushPromises()

    expect(mocks.shareCoinCard).toHaveBeenCalledTimes(1)
    expect(mocks.shareCoinCard.mock.calls[0]?.[0]).toEqual(featured.coin)
    expect(mocks.shareCoinCard.mock.calls[0]?.[1]).toEqual({
      context: {
        heading: 'Coin of the Day',
        summary: obverseReverseSummary,
      },
    })
  })

  it('disables and communicates the share action while sharing', async () => {
    mocks.sharing.value = true
    mocks.getFeaturedCoin.mockResolvedValue({ data: buildFeaturedCoin() })

    const wrapper = mountFeaturedCoinModal()
    await flushPromises()

    const shareButton = findShareButton(wrapper)
    expect(shareButton?.text()).toContain('Sharing...')
    expect(shareButton?.attributes('disabled')).toBeDefined()
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

function findShareButton(wrapper: ReturnType<typeof mountFeaturedCoinModal>) {
  return wrapper.findAll('button').find((button) => button.text().includes('Shar'))
}

function buildFeaturedCoin(overrides: Partial<FeaturedCoin> = {}): FeaturedCoin {
  const coin = buildRomanDenariusCore({
    id: 42,
    name: 'Trajan Denarius Core',
  })

  return {
    id: 9001,
    userId: 101,
    coinId: coin.id,
    coin,
    summary: 'Trajan denarius summary with obverse and reverse details.',
    featuredAt: '2026-06-20T12:00:00Z',
    createdAt: '2026-06-20T12:00:00Z',
    ...overrides,
  }
}
