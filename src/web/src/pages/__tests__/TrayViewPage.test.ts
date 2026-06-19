import { flushPromises, shallowMount } from '@vue/test-utils'
import { nextTick } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import TrayViewPage from '@/pages/TrayViewPage.vue'
import { buildRomanDenariusCore } from '@/test/fixtures/coins'

const mockGetCoins = vi.fn()
const mockPush = vi.fn()

vi.mock('@/api/client', () => ({
  getCoins: (params?: Record<string, unknown>) => mockGetCoins(params),
}))

vi.mock('vue-router', () => ({
  useRouter: () => ({
    push: mockPush,
  }),
}))

vi.mock('@/composables/useTrayPreference', () => ({
  useTrayPreference: () => ({
    feltColor: 'red',
  }),
}))

const routerLinkStub = {
  props: ['to'],
  template: '<a :href="to"><slot /></a>',
}

describe('TrayViewPage', () => {
  beforeEach(() => {
    mockGetCoins.mockReset()
    mockPush.mockReset()
  })

  it('fetches every active collection page for tray drawers', async () => {
    const firstPage = Array.from({ length: 100 }, (_, index) =>
      buildRomanDenariusCore({ id: index + 1, name: `Coin ${index + 1}` }),
    )
    const secondPage = Array.from({ length: 20 }, (_, index) =>
      buildRomanDenariusCore({ id: index + 101, name: `Coin ${index + 101}` }),
    )
    mockGetCoins
      .mockResolvedValueOnce({ data: { coins: firstPage, total: 120 } })
      .mockResolvedValueOnce({ data: { coins: secondPage, total: 120 } })

    shallowMount(TrayViewPage, {
      global: {
        stubs: {
          RouterLink: routerLinkStub,
          MuseumTray: true,
          TrayControls: true,
        },
      },
    })
    await flushPromises()

    expect(mockGetCoins).toHaveBeenNthCalledWith(1, {
      wishlist: 'false',
      sold: 'false',
      page: 1,
      limit: 100,
      sort: 'name',
      order: 'asc',
    })
    expect(mockGetCoins).toHaveBeenNthCalledWith(2, {
      wishlist: 'false',
      sold: 'false',
      page: 2,
      limit: 100,
      sort: 'name',
      order: 'asc',
    })
    expect(mockGetCoins).toHaveBeenCalledTimes(2)
  })

  it('only sends coins with known diameter values to tray drawers', async () => {
    const measuredCoin = buildRomanDenariusCore({ id: 1, name: 'Measured Coin', diameterMm: 18 })
    const missingDiameterCoin = buildRomanDenariusCore({ id: 2, name: 'Missing Diameter', diameterMm: null })
    const zeroDiameterCoin = buildRomanDenariusCore({ id: 3, name: 'Zero Diameter', diameterMm: 0 })
    mockGetCoins.mockResolvedValueOnce({
      data: { coins: [measuredCoin, missingDiameterCoin, zeroDiameterCoin], total: 3 },
    })

    const wrapper = shallowMount(TrayViewPage, {
      global: {
        stubs: {
          RouterLink: routerLinkStub,
          MuseumTray: true,
          TrayControls: true,
        },
      },
    })
    await flushPromises()

    const tray = wrapper.findComponent({ name: 'MuseumTray' })
    expect(tray.props('coins')).toEqual([{
      id: measuredCoin.id,
      name: measuredCoin.name,
      diameterMm: measuredCoin.diameterMm,
      images: measuredCoin.images,
    }])
  })

  it('renders measured coins on the final desktop tray drawer', async () => {
    const measuredCoins = Array.from({ length: 37 }, (_, index) =>
      buildRomanDenariusCore({
        id: index + 1,
        name: `Measured Coin ${index + 1}`,
        diameterMm: 16 + (index % 4),
      }),
    )
    mockGetCoins.mockResolvedValueOnce({
      data: { coins: measuredCoins, total: measuredCoins.length },
    })

    const wrapper = shallowMount(TrayViewPage, {
      global: {
        stubs: {
          RouterLink: routerLinkStub,
          MuseumTray: true,
          TrayControls: true,
        },
      },
    })
    await flushPromises()

    const controls = wrapper.findComponent({ name: 'TrayControls' })
    controls.vm.$emit('next')
    await nextTick()
    controls.vm.$emit('next')
    await nextTick()
    controls.vm.$emit('next')
    await nextTick()

    const tray = wrapper.findComponent({ name: 'MuseumTray' })
    expect(controls.props('drawerIndex')).toBe(3)
    expect(controls.props('totalDrawers')).toBe(4)
    expect(tray.props('coins')).toHaveLength(1)
    expect(tray.props('coins')).toEqual([{
      id: measuredCoins[36]!.id,
      name: measuredCoins[36]!.name,
      diameterMm: measuredCoins[36]!.diameterMm,
      images: measuredCoins[36]!.images,
    }])
  })
})
