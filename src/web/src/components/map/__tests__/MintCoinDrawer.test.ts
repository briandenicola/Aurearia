import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import MintCoinDrawer from '@/components/map/MintCoinDrawer.vue'
import { groupCoinsByMint } from '@/utils/mintMap'
import { buildMintMapFixtureCoins } from '@/test/fixtures/coins'

const routerLinkStub = {
  props: ['to'],
  template: '<a :href="to"><slot /></a>',
}

describe('MintCoinDrawer', () => {
  it('lists only the selected mint group coins', () => {
    const group = groupCoinsByMint(buildMintMapFixtureCoins()).matched.find((item) => item.mint.id === 'rome') ?? null
    const wrapper = mount(MintCoinDrawer, {
      props: { open: true, group },
      global: { stubs: { RouterLink: routerLinkStub } },
    })

    expect(wrapper.text()).toContain('Trajan Denarius Core')
    expect(wrapper.text()).toContain('Roma Alias Denarius')
    expect(wrapper.text()).not.toContain('Byzantium Alias Solidus')
  })
})
