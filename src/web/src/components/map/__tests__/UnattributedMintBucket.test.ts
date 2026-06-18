import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import UnattributedMintBucket from '@/components/map/UnattributedMintBucket.vue'
import { groupCoinsByMint } from '@/utils/mintMap'
import { buildMintMapFixtureCoins } from '@/test/fixtures/coins'

const routerLinkStub = {
  props: ['to'],
  template: '<a :href="to"><slot /></a>',
}

describe('UnattributedMintBucket', () => {
  it('surfaces unknown and unmatched mint coins', () => {
    const grouped = groupCoinsByMint(buildMintMapFixtureCoins())
    const wrapper = mount(UnattributedMintBucket, {
      props: {
        expanded: true,
        unknown: grouped.unknown,
        unmatched: grouped.unmatched,
      },
      global: { stubs: { RouterLink: routerLinkStub } },
    })

    expect(wrapper.text()).toContain('Unknown mint')
    expect(wrapper.text()).toContain('Unknown Mint Fraction')
    expect(wrapper.text()).toContain('Traveling Camp')
    expect(wrapper.text()).toContain('Unmatched Camp Mint Bronze')
  })
})
