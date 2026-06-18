import { shallowMount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { Coin } from '@/types'
import MintMapPage from '@/pages/MintMapPage.vue'
import { buildMintMapFixtureCoins, buildTestMintLocations } from '@/test/fixtures/coins'

const mockStore = {
  coins: [] as Coin[],
  loading: false,
  fetchCoins: vi.fn(),
}

vi.mock('@/stores/coins', () => ({
  useCoinsStore: () => mockStore,
}))

const mockGetMintLocations = vi.fn()
vi.mock('@/api/client', () => ({
  getMintLocations: () => mockGetMintLocations(),
}))

const routerLinkStub = {
  props: ['to'],
  template: '<a :href="to"><slot /></a>',
}

async function flushMountedPromises() {
  await Promise.resolve()
  await Promise.resolve()
}

describe('MintMapPage', () => {
  beforeEach(() => {
    mockStore.coins = []
    mockStore.loading = false
    mockStore.fetchCoins.mockReset()
    mockStore.fetchCoins.mockResolvedValue(undefined)
    mockGetMintLocations.mockReset()
    mockGetMintLocations.mockResolvedValue({ data: { mintLocations: buildTestMintLocations() } })
  })

  it('fetches the default active collection when the store is empty', async () => {
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

    expect(mockStore.fetchCoins).toHaveBeenCalledWith({ wishlist: 'false', sold: 'false' })
  })

  it('renders summary counts and the unattributed bucket', async () => {
    mockStore.coins = buildMintMapFixtureCoins()
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
    mockStore.coins = buildMintMapFixtureCoins()
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
    mockStore.coins = buildMintMapFixtureCoins()
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
    mockStore.coins = buildMintMapFixtureCoins()
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
