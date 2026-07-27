import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import MintListPanel from '@/components/map/MintListPanel.vue'
import { groupCoinsByMint } from '@/utils/mintMap'
import { buildMintMapFixtureCoins, buildTestMintLocations } from '@/test/fixtures/coins'

describe('MintListPanel', () => {
  const mintLocations = buildTestMintLocations()
  const groups = groupCoinsByMint(buildMintMapFixtureCoins(), mintLocations).matched

  it('renders a list item for each matched group', () => {
    const wrapper = mount(MintListPanel, { props: { groups, selectedMintId: null } })
    const items = wrapper.findAll('.mint-list-item')
    expect(items).toHaveLength(groups.length)
  })

  it('shows the mint display name and region in each list item', () => {
    const wrapper = mount(MintListPanel, { props: { groups, selectedMintId: null } })
    const rome = groups.find((g) => g.mint.id === 1)
    expect(wrapper.text()).toContain(rome?.mint.displayName)
    expect(wrapper.text()).toContain(rome?.mint.region)
  })

  it('shows coin count badge for each mint group', () => {
    const wrapper = mount(MintListPanel, { props: { groups, selectedMintId: null } })
    const rome = groups.find((g) => g.mint.id === 1)!
    const romeBadges = wrapper.findAll('.mint-list-badge')
    const romeItem = wrapper
      .findAll('.mint-list-item')
      .find((el) => el.text().includes(rome.mint.displayName))
    expect(romeItem).toBeDefined()
    expect(romeItem?.find('.mint-list-badge').text()).toBe(String(rome.count))
    expect(romeBadges.length).toBe(groups.length)
  })

  it('shows coin name preview of up to 2 coins for each group', () => {
    const wrapper = mount(MintListPanel, { props: { groups, selectedMintId: null } })
    const rome = groups.find((g) => g.mint.id === 1)!
    const romeItem = wrapper
      .findAll('.mint-list-item')
      .find((el) => el.text().includes(rome.mint.displayName))
    expect(romeItem).toBeDefined()
    // Rome has 2 coins — both names should appear in the preview
    const previewEl = romeItem?.find('.mint-list-preview')
    expect(previewEl?.exists()).toBe(true)
    expect(previewEl?.text()).toContain(rome.coins[0]?.name)
  })

  it('appends +N more when group has more than 2 coins', () => {
    const extraGroups = groups.map((g) =>
      g.mint.id === 1
        ? {
            ...g,
            coins: [
              ...g.coins,
              { ...g.coins[0]!, id: 9001, name: 'Third Coin' },
              { ...g.coins[0]!, id: 9002, name: 'Fourth Coin' },
            ],
            count: g.count + 2,
          }
        : g,
    )
    const wrapper = mount(MintListPanel, { props: { groups: extraGroups, selectedMintId: null } })
    const romeItem = wrapper
      .findAll('.mint-list-item')
      .find((el) => el.text().includes('Rome'))
    const preview = romeItem?.find('.mint-list-preview').text() ?? ''
    expect(preview).toMatch(/\+\d+ more/)
  })

  it('applies selected class and aria-pressed to the active mint', () => {
    const romeGroup = groups.find((g) => g.mint.id === 1)!
    const wrapper = mount(MintListPanel, { props: { groups, selectedMintId: romeGroup.mint.id } })
    const selected = wrapper
      .findAll('.mint-list-item')
      .find((el) => el.classes('mint-list-item--selected'))
    expect(selected).toBeDefined()
    expect(selected?.attributes('aria-pressed')).toBe('true')
    expect(selected?.text()).toContain(romeGroup.mint.displayName)
  })

  it('emits select-mint when a list item is clicked', async () => {
    const wrapper = mount(MintListPanel, { props: { groups, selectedMintId: null } })
    await wrapper.findAll('.mint-list-item')[0]?.trigger('click')
    const emitted = wrapper.emitted('select-mint')
    expect(emitted).toHaveLength(1)
    expect((emitted?.[0] as [unknown])[0]).toMatchObject({ mint: { id: groups[0]?.mint.id } })
  })

  it('shows location total count in the panel header', () => {
    const wrapper = mount(MintListPanel, { props: { groups, selectedMintId: null } })
    expect(wrapper.find('.mint-list-total').text()).toBe(String(groups.length))
  })

  it('renders no list items when groups is empty', () => {
    const wrapper = mount(MintListPanel, { props: { groups: [], selectedMintId: null } })
    expect(wrapper.findAll('.mint-list-item')).toHaveLength(0)
    expect(wrapper.find('.mint-list-total').text()).toBe('0')
  })
})
