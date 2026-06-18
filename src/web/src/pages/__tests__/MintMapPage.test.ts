import { shallowMount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { Coin } from '@/types'
import MintMapPage from '@/pages/MintMapPage.vue'
import { buildMintMapFixtureCoins } from '@/test/fixtures/coins'

const mockStore = {
  coins: [] as Coin[],
  loading: false,
  fetchCoins: vi.fn(),
}

vi.mock('@/stores/coins', () => ({
  useCoinsStore: () => mockStore,
}))

const routerLinkStub = {
  props: ['to'],
  template: '<a :href="to"><slot /></a>',
}

describe('MintMapPage', () => {
  beforeEach(() => {
    mockStore.coins = []
    mockStore.loading = false
    mockStore.fetchCoins.mockReset()
    mockStore.fetchCoins.mockResolvedValue(undefined)
  })

  it('fetches the default active collection when the store is empty', () => {
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

    expect(mockStore.fetchCoins).toHaveBeenCalledWith({ wishlist: 'false', sold: 'false' })
  })

  it('renders summary counts and the unattributed bucket', () => {
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

    expect(wrapper.text()).toContain('Mapped Coins')
    expect(wrapper.text()).toContain('4')
    expect(wrapper.findComponent({ name: 'UnattributedMintBucket' }).exists()).toBe(true)
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

    await wrapper.find('.select-rome').trigger('click')

    expect(wrapper.text()).toContain('Rome')
    expect(wrapper.text()).toContain('Trajan Denarius Core')
    expect(wrapper.text()).not.toContain('Byzantium Alias Solidus')
  })
})
