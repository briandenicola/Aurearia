import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import MintMapSvg from '@/components/map/MintMapSvg.vue'
import { groupCoinsByMint } from '@/utils/mintMap'
import { buildMintMapFixtureCoins } from '@/test/fixtures/coins'

describe('MintMapSvg', () => {
  const groups = groupCoinsByMint(buildMintMapFixtureCoins()).matched

  it('renders accessible pins with counts and selected state', () => {
    const wrapper = mount(MintMapSvg, {
      props: {
        groups,
        selectedMintId: 'rome',
      },
    })

    expect(wrapper.text()).toContain('2')
    expect(wrapper.find('[aria-label="Rome: 2 coins"]').exists()).toBe(true)
    expect(wrapper.find('.mint-pin.active').exists()).toBe(true)
  })

  it('emits selected mint from click and keyboard activation', async () => {
    const wrapper = mount(MintMapSvg, {
      props: { groups },
    })

    const pin = wrapper.find('[aria-label="Rome: 2 coins"]')
    await pin.trigger('click')
    await pin.trigger('keydown', { key: 'Enter' })

    expect(wrapper.emitted('select-mint')?.[0]?.[0]).toMatchObject({ mint: { id: 'rome' }, count: 2 })
    expect(wrapper.emitted('select-mint')).toHaveLength(2)
  })

  it('does not include network tile dependencies', () => {
    const wrapper = mount(MintMapSvg, {
      props: { groups },
    })

    expect(wrapper.find('img').exists()).toBe(false)
    expect(wrapper.find('image').exists()).toBe(false)
    expect(wrapper.html()).not.toContain('tile.openstreetmap')
    expect(wrapper.html()).not.toContain('mapbox')
  })
})
