import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, expect, it, vi } from 'vitest'
import { createMemoryHistory, createRouter } from 'vue-router'
import { ref } from 'vue'
import SetDetailPage from '@/pages/SetDetailPage.vue'
import { buildRomanDenariusCore } from '@/test/fixtures/coins'

const api = vi.hoisted(() => ({
  getSet: vi.fn(), getCoinsInSet: vi.fn(), getSetCompletion: vi.fn(), getCoins: vi.fn(),
  updateSet: vi.fn(), deleteSet: vi.fn(), addCoinToSet: vi.fn(), removeCoinFromSet: vi.fn(), reorderSetCoins: vi.fn(),
}))
const pin = vi.hoisted(() => vi.fn())
vi.mock('@/api/client', () => api)
vi.mock('@/composables/useQuickAccess', () => ({
  useQuickAccess: () => ({
    error: ref(''), refresh: vi.fn(), pin, unpin: vi.fn(), forget: vi.fn(),
    isPinned: () => ref(false), isBusy: () => ref(false),
  }),
}))
vi.mock('@/composables/usePinnedSets', () => ({ usePinnedSets: () => ({ pinLimitReached: ref(false), refresh: vi.fn() }) }))
vi.mock('@/composables/usePwa', () => ({ usePwa: () => ({ isPwa: ref(false) }) }))
vi.mock('@/composables/useToast', () => ({ useToast: () => ({ showToast: vi.fn() }) }))

beforeEach(() => {
  vi.resetAllMocks()
  vi.stubGlobal('confirm', vi.fn(() => true))
  api.getSet.mockImplementation(async (id: number) => ({ data: { id, name: `Set ${id}`, setType: 'standard', color: '#c9a84c' } }))
  api.getCoinsInSet.mockImplementation(async (id: number) => ({ data: { coins: [
    buildRomanDenariusCore({ id: id * 10, name: `Coin ${id}a` }),
    buildRomanDenariusCore({ id: id * 10 + 1, name: `Coin ${id}b` }),
  ] } }))
  api.getCoins.mockResolvedValue({ data: { coins: [], total: 0 } })
})
afterEach(() => { vi.unstubAllGlobals() })

async function openPage() {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/sets/:id', component: SetDetailPage },
      { path: '/sets', name: 'sets', component: { template: '<p>Sets</p>' } },
    ],
  })
  await router.push('/sets/1')
  await router.isReady()
  const wrapper = mount({ template: '<router-view />' }, {
    global: { plugins: [router], stubs: { MuseumTray: true, SetCompletionChecklist: true, TrayControls: true } },
  })
  await flushPromises()
  return { router, wrapper }
}

it('reuses the set page safely and targets the new set for edits, membership, ordering and deletion', async () => {
  const { router, wrapper } = await openPage()
  try {
    await router.push('/sets/2')
    await flushPromises()
    expect(wrapper.get('h1').text()).toBe('Set 2')
    await wrapper.get('[aria-label="Pin set to Quick Access"]').trigger('click')
    await flushPromises()
    expect(pin).toHaveBeenCalledWith('coin_set', 2)
    api.getCoins.mockResolvedValue({ data: { coins: [buildRomanDenariusCore({ id: 150 })], total: 1 } })
    await wrapper.get('[aria-haspopup="menu"]').trigger('click')
    await wrapper.findAll('button').find(button => button.text().includes('Add Coin'))!.trigger('click')
    await flushPromises()
    await wrapper.get('#coinToAdd').setValue('150')
    await wrapper.get('form').trigger('submit')
    await flushPromises()
    expect(api.addCoinToSet).toHaveBeenCalledWith(2, { coinId: 150 })
    await wrapper.get('[aria-label="Move Coin 2b earlier"]').trigger('click')
    await flushPromises()
    expect(api.reorderSetCoins).toHaveBeenCalledWith(2, { coinIds: [21, 20] })
    await wrapper.get('[aria-label="Remove Coin 2a from set"]').trigger('click')
    await flushPromises()
    expect(api.removeCoinFromSet).toHaveBeenCalledWith(2, 20)
    await wrapper.get('[aria-haspopup="menu"]').trigger('click')
    await wrapper.findAll('button').find(button => button.text().includes('Edit Set'))!.trigger('click')
    await wrapper.get('#editName').setValue('Renamed B')
    await wrapper.get('form').trigger('submit')
    await flushPromises()
    expect(api.updateSet).toHaveBeenCalledWith(2, expect.objectContaining({ name: 'Renamed B' }))
    await wrapper.get('[aria-haspopup="menu"]').trigger('click')
    await wrapper.findAll('button').find(button => button.text().includes('Delete Set'))!.trigger('click')
    await flushPromises()
    expect(api.deleteSet).toHaveBeenCalledWith(2)
  } finally { wrapper.unmount() }
})

function deferred<T>() {
  let resolve!: (value: T) => void
  let reject!: (reason: Error) => void
  const promise = new Promise<T>((yes, no) => { resolve = yes; reject = no })
  return { promise, resolve, reject }
}

it('rejects delayed A loads even after A-to-B-to-A navigation', async () => {
  const old = deferred<{ data: { id: number; name: string; setType: string } }>()
  api.getSet.mockImplementationOnce(() => old.promise)
  const { router, wrapper } = await openPage()
  try {
    expect(wrapper.text()).toContain('Loading set details')
    await router.push('/sets/2')
    await flushPromises()
    expect(wrapper.get('h1').text()).toBe('Set 2')
    await router.push('/sets/1')
    await flushPromises()
    old.resolve({ data: { id: 1, name: 'Stale A', setType: 'standard' } })
    await flushPromises()
    expect(wrapper.get('h1').text()).toBe('Set 1')
  } finally { wrapper.unmount() }
})

it('shows loading failure and can retry without showing the previous set', async () => {
  const { router, wrapper } = await openPage()
  try {
    api.getSet.mockRejectedValueOnce(new Error('offline'))
    await router.push('/sets/2')
    await flushPromises()
    expect(wrapper.find('h1').exists()).toBe(false)
    expect(wrapper.get('[role="alert"]').text()).toContain('Unable to load')
    await wrapper.get('[role="alert"] button').trigger('click')
    await flushPromises()
    expect(wrapper.get('h1').text()).toBe('Set 2')
  } finally { wrapper.unmount() }
})

it.each(['edit', 'add', 'remove', 'reorder', 'delete', 'pin'])('ignores late %s completion after navigation', async action => {
  const pending = deferred<void>()
  const operation = { edit: api.updateSet, add: api.addCoinToSet, remove: api.removeCoinFromSet,
    reorder: api.reorderSetCoins, delete: api.deleteSet, pin }[action]!
  operation.mockImplementationOnce(() => pending.promise)
  const { router, wrapper } = await openPage()
  try {
    if (action === 'reorder') await wrapper.get('[aria-label="Move Coin 1b earlier"]').trigger('click')
    else if (action === 'remove') await wrapper.get('[aria-label="Remove Coin 1a from set"]').trigger('click')
    else if (action === 'pin') await wrapper.get('[aria-label="Pin set to Quick Access"]').trigger('click')
    else {
      await wrapper.get('[aria-haspopup="menu"]').trigger('click')
      const label = { edit: 'Edit Set', add: 'Add Coin', delete: 'Delete Set' }[action]!
      api.getCoins.mockResolvedValue({ data: { coins: [buildRomanDenariusCore({ id: 150 })], total: 1 } })
      await wrapper.findAll('button').find(button => button.text().includes(label))!.trigger('click')
      await flushPromises()
      if (action === 'edit') await wrapper.get('#editName').setValue('Old A edit')
      if (action === 'add') await wrapper.get('#coinToAdd').setValue('150')
      if (action !== 'delete') await wrapper.get('form').trigger('submit')
    }
    expect(operation.mock.calls[0]?.[action === 'pin' ? 1 : 0]).toBe(1)
    await router.push('/sets/2')
    await flushPromises()
    const loads = api.getSet.mock.calls.length
    if (action === 'reorder') pending.reject(new Error('late order failure'))
    else pending.resolve()
    await flushPromises()
    expect(router.currentRoute.value.path).toBe('/sets/2')
    expect(wrapper.get('h1').text()).toBe('Set 2')
    expect(wrapper.find('#editName').exists()).toBe(false)
    expect(wrapper.find('#coinToAdd').exists()).toBe(false)
    expect(wrapper.text()).not.toContain('Coin 1a')
    expect(api.getSet).toHaveBeenCalledTimes(loads)
  } finally { wrapper.unmount() }
})

it('rolls back a failed reorder on the same set visibly', async () => {
  api.reorderSetCoins.mockRejectedValueOnce(new Error('Order unavailable'))
  const { wrapper } = await openPage()
  const log = vi.spyOn(console, 'error').mockImplementation(() => {})
  try {
    await wrapper.get('[aria-label="Move Coin 1b earlier"]').trigger('click')
    await flushPromises()
    expect(wrapper.text()).toContain('Order unavailable')
    const buttons = wrapper.findAll('[aria-label^="Move "][aria-label$=" earlier"]')
    expect(buttons[0]?.attributes('aria-label')).toBe('Move Coin 1a earlier')
  } finally { wrapper.unmount(); log.mockRestore() }
})

it.each(['assign', 'clear'])('keeps agentic slot %s scoped across navigation', async action => {
  api.getSet.mockImplementation(async (id: number) => ({ data: { id, name: `Set ${id}`, setType: 'agentic' } }))
  api.getSetCompletion.mockImplementation(async (id: number) => ({ data: { targetMatches: [
    { target: { id: id * 100, label: 'First slot' }, coin: buildRomanDenariusCore({ id: id * 10 }) },
    { target: { id: id * 100 + 1, label: 'Other slot' }, coin: buildRomanDenariusCore({ id: id * 10 + 1 }) },
  ] } }))
  api.getCoins.mockResolvedValue({ data: { coins: [
    buildRomanDenariusCore({ id: 20 }), buildRomanDenariusCore({ id: 21 }), buildRomanDenariusCore({ id: 150 }),
  ], total: 3 } })
  const { router, wrapper } = await openPage()
  const pending = deferred<void>()
  try {
    await router.push('/sets/2'); await flushPromises()
    wrapper.findComponent({ name: 'MuseumTray' }).vm.$emit('coin-clicked', 20)
    await flushPromises()
    expect(wrapper.find('option[value="21"]').exists()).toBe(false)
    expect(wrapper.get<HTMLSelectElement>('#slotCoinToAssign').element.value).toBe('20')
    if (action === 'assign') {
      api.addCoinToSet.mockImplementationOnce(() => pending.promise)
      await wrapper.get('#slotCoinToAssign').setValue('150')
      await wrapper.get('form').trigger('submit')
      expect(api.addCoinToSet).toHaveBeenCalledWith(2, { coinId: 150, targetId: 200 })
    } else {
      api.removeCoinFromSet.mockImplementationOnce(() => pending.promise)
      await wrapper.findAll('button').find(button => button.text().includes('Clear Slot'))!.trigger('click')
      expect(api.removeCoinFromSet).toHaveBeenCalledWith(2, 20)
    }
    await router.push('/sets/1'); await flushPromises()
    const loads = api.getSet.mock.calls.length
    pending.resolve(); await flushPromises()
    expect(wrapper.get('h1').text()).toBe('Set 1')
    expect(wrapper.find('#slotCoinToAssign').exists()).toBe(false)
    expect(api.getSet).toHaveBeenCalledTimes(loads)
  } finally { wrapper.unmount() }
})
