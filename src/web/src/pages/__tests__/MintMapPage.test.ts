import { flushPromises, shallowMount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import MintMapPage from '@/pages/MintMapPage.vue'
import { buildMintMapFixtureCoins, buildRomanDenariusCore, buildTestMintLocations } from '@/test/fixtures/coins'

const mockGetMintLocations = vi.fn()
const mockGetCoins = vi.fn()
vi.mock('@/api/client', () => ({
  getMintLocations: () => mockGetMintLocations(),
  getCoins: (params?: Record<string, unknown>) => mockGetCoins(params),
}))

const routerLinkStub = {
  props: ['to'],
  template: '<a :href="to"><slot /></a>',
}

async function flushMountedPromises() {
  await flushPromises()
}

describe('MintMapPage', () => {
  beforeEach(() => {
    mockGetMintLocations.mockReset()
    mockGetMintLocations.mockResolvedValue({ data: { mintLocations: buildTestMintLocations() } })
    mockGetCoins.mockReset()
    const coins = buildMintMapFixtureCoins()
    mockGetCoins.mockResolvedValue({
      data: {
        coins,
        total: coins.length,
        page: 1,
        limit: 100,
      },
    })
  })

  it('fetches active collection coins for the map', async () => {
    shallowMount(MintMapPage, {
      global: {
        stubs: {
          RouterLink: routerLinkStub,
          MintMapLeaflet: true,
          MintCoinDrawer: true,
          UnattributedMintBucket: true,
        },
      },
    })
    await flushMountedPromises()

    expect(mockGetCoins).toHaveBeenCalledWith({ wishlist: 'false', sold: 'false', page: 1, limit: 100 })
  })

  it('fetches every active collection page instead of stopping at the first page', async () => {
    const firstPage = Array.from({ length: 100 }, (_, index) =>
      buildRomanDenariusCore({ id: index + 1, name: `Rome Coin ${index + 1}` }),
    )
    const secondPage = Array.from({ length: 20 }, (_, index) =>
      buildRomanDenariusCore({ id: index + 101, name: `Rome Coin ${index + 101}` }),
    )
    mockGetCoins
      .mockResolvedValueOnce({ data: { coins: firstPage, total: 120, page: 1, limit: 100 } })
      .mockResolvedValueOnce({ data: { coins: secondPage, total: 120, page: 2, limit: 100 } })

    const wrapper = shallowMount(MintMapPage, {
      global: {
        stubs: {
          RouterLink: routerLinkStub,
          MintMapLeaflet: true,
          MintCoinDrawer: true,
          UnattributedMintBucket: true,
        },
      },
    })
    await flushMountedPromises()

    expect(mockGetCoins).toHaveBeenNthCalledWith(1, { wishlist: 'false', sold: 'false', page: 1, limit: 100 })
    expect(mockGetCoins).toHaveBeenNthCalledWith(2, { wishlist: 'false', sold: 'false', page: 2, limit: 100 })
    expect(wrapper.find('.mapped-count').text()).toBe('120')
  })

  it('renders summary counts and the unattributed bucket', async () => {
    const wrapper = shallowMount(MintMapPage, {
      global: {
        stubs: {
          RouterLink: routerLinkStub,
          MintMapLeaflet: true,
          MintCoinDrawer: true,
          UnattributedMintBucket: true,
        },
      },
    })
    await flushMountedPromises()

    expect(wrapper.text()).toContain('Mapped Coins:')
    expect(wrapper.text()).toContain('4')
    expect(wrapper.findComponent({ name: 'UnattributedMintBucket' }).exists()).toBe(true)
  })

  it('renders the correct header with Map of Coins title and back link to /stats', () => {
    const wrapper = shallowMount(MintMapPage, {
      global: {
        stubs: {
          RouterLink: routerLinkStub,
          MintMapLeaflet: true,
          MintCoinDrawer: true,
          UnattributedMintBucket: true,
        },
      },
    })

    expect(wrapper.find('h1').text()).toBe('Map of Coins')
    const backLink = wrapper.find('a[href="/stats"]')
    expect(backLink.exists()).toBe(true)
    expect(backLink.attributes('aria-label')).toBe('Back to Stats')
  })

  it('renders a compact summary row with title-case label and count', async () => {
    const wrapper = shallowMount(MintMapPage, {
      global: {
        stubs: {
          RouterLink: routerLinkStub,
          MintMapLeaflet: true,
          MintCoinDrawer: true,
          UnattributedMintBucket: true,
        },
      },
    })
    await flushMountedPromises()

    const summaryRow = wrapper.find('.summary-row')
    expect(summaryRow.exists()).toBe(true)
    expect(summaryRow.find('.summary-label').text()).toBe('Mapped Coins:')
    expect(summaryRow.find('.mapped-count').text()).toBe('4')
  })

  it('opens a drawer with only the selected mint group', async () => {
    const wrapper = shallowMount(MintMapPage, {
      global: {
        stubs: {
          RouterLink: routerLinkStub,
          MintMapLeaflet: {
            props: ['groups', 'selectedMintId'],
            emits: ['select-mint'],
            template: '<button class="select-rome" @click="$emit(\'select-mint\', groups[0])">Select Rome</button>',
          },
          MintCoinDrawer: {
            props: ['open', 'group'],
            template: '<aside v-if="open">{{ group.mint.displayName }} {{ group.coins.map((coin) => coin.name).join(", ") }}</aside>',
          },
          UnattributedMintBucket: true,
        },
      },
    })
    await flushMountedPromises()

    await wrapper.find('.select-rome').trigger('click')

    expect(wrapper.text()).toContain('Rome')
    expect(wrapper.text()).toContain('Trajan Denarius Core')
    expect(wrapper.text()).not.toContain('Byzantium Alias Solidus')
  })
})
