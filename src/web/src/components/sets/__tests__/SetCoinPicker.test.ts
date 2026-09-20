import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, expect, it, vi } from 'vitest'
import SetCoinPicker from '../SetCoinPicker.vue'
import { buildRomanDenariusCore } from '@/test/fixtures/coins'

const getCoins = vi.hoisted(() => vi.fn())
vi.mock('@/api/client', () => ({ getCoins }))
const coins = Array.from({ length: 151 }, (_, index) =>
  buildRomanDenariusCore({ id: index + 1, name: `Coin ${index + 1}` }))

beforeEach(() => {
  getCoins.mockReset()
  getCoins.mockImplementation(async ({ page, limit, search }) => {
    const matches = coins.filter(coin => !search || coin.name === search)
    return { data: { coins: matches.slice((page - 1) * limit, page * limit), total: matches.length } }
  })
})

function openPicker(extra = {}) {
  return mount(SetCoinPicker, { props: {
    modelValue: null, excludedIds: [], allowWishlist: false, searchId: 'search', selectId: 'coin', ...extra,
  } })
}

it('pages beyond coin 100, searches the server and retains selection across both', async () => {
  const wrapper = openPicker()
  try {
    await flushPromises()
    expect(wrapper.text()).toContain('Page 1 of 4 (151 matches)')
    const next = () => wrapper.findAll('button').find(button => button.text().includes('Next'))!
    await next().trigger('click'); await flushPromises()
    await next().trigger('click'); await flushPromises()
    expect(wrapper.get('option[value="150"]').text()).toContain('Coin 150')
    await wrapper.get('#coin').setValue('150')
    expect(wrapper.emitted('update:modelValue')?.[0]).toEqual([150])
    await wrapper.setProps({ modelValue: 150 })
    await next().trigger('click'); await flushPromises()
    expect(wrapper.get('option[value="150"]').exists()).toBe(true)
    await wrapper.get('#search').setValue('Coin 151'); await flushPromises()
    expect(getCoins).toHaveBeenLastCalledWith(expect.objectContaining({ page: 1, search: 'Coin 151', limit: 50 }))
    expect(wrapper.text()).toContain('Page 1 of 1 (1 matches)')
    expect(wrapper.get('option[value="151"]').exists()).toBe(true)
    expect(wrapper.get<HTMLSelectElement>('#coin').element.value).toBe('150')
  } finally { wrapper.unmount() }
})

it('excludes members or other assigned slots and sold coins, but allows goal wishlist coins', async () => {
  getCoins.mockResolvedValue({ data: { coins: [
    buildRomanDenariusCore({ id: 1 }), buildRomanDenariusCore({ id: 2, isSold: true }),
    buildRomanDenariusCore({ id: 3, isWishlist: true }), buildRomanDenariusCore({ id: 4 }),
  ], total: 4 } })
  const wrapper = openPicker({ excludedIds: [1], selectedCoin: coins[3], modelValue: 4 })
  try {
    await flushPromises()
    expect(wrapper.find('option[value="1"]').exists()).toBe(false)
    expect(wrapper.find('option[value="2"]').exists()).toBe(false)
    expect(wrapper.find('option[value="3"]').exists()).toBe(false)
    expect(wrapper.find('option[value="4"]').exists()).toBe(true)
    await wrapper.setProps({ allowWishlist: true })
    await flushPromises()
    expect(getCoins).toHaveBeenLastCalledWith(expect.objectContaining({ wishlist: undefined, sold: 'false' }))
    expect(wrapper.find('option[value="3"]').exists()).toBe(true)
  } finally { wrapper.unmount() }
})

it('discards stale search responses and surfaces retryable failures', async () => {
  let resolve!: (value: unknown) => void
  getCoins.mockImplementationOnce(() => new Promise(done => { resolve = done }))
  const wrapper = openPicker()
  try {
    expect(wrapper.text()).toContain('Loading coins')
    await wrapper.get('#search').setValue('Coin 151'); await flushPromises()
    resolve({ data: { coins: [coins[0]], total: 1 } }); await flushPromises()
    expect(wrapper.find('option[value="1"]').exists()).toBe(false)
    expect(wrapper.find('option[value="151"]').exists()).toBe(true)
    getCoins.mockRejectedValueOnce(new Error('offline'))
    await wrapper.get('#search').setValue('Coin 150'); await flushPromises()
    expect(wrapper.get('[role="alert"]').text()).toContain('Unable to load coins')
    await wrapper.get('[role="alert"] button').trigger('click'); await flushPromises()
    expect(wrapper.find('[role="alert"]').exists()).toBe(false)
    expect(wrapper.find('option[value="150"]').exists()).toBe(true)
  } finally { wrapper.unmount() }
})
