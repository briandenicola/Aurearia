import { flushPromises, shallowMount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import StorageTraysPage from '@/pages/StorageTraysPage.vue'
import { emptyThreeByThreeTray, occupiedThreeByThreeTray } from '@/test/fixtures/storageTrays'

const getStorageTrays = vi.fn()
const push = vi.fn()
vi.mock('@/api/client', () => ({ getStorageTrays: () => getStorageTrays() }))
vi.mock('vue-router', () => ({ useRouter: () => ({ push }) }))
vi.mock('@/composables/useTrayPreference', () => ({
  useTrayPreference: () => ({ feltColor: 'navy' }),
}))

describe('StorageTraysPage', () => {
  beforeEach(() => {
    getStorageTrays.mockReset()
    push.mockReset()
    localStorage.clear()
  })

  it('uses one aggregate request and renders empty and occupied trays', async () => {
    getStorageTrays.mockResolvedValue({ data: { trays: [emptyThreeByThreeTray, { ...occupiedThreeByThreeTray, id: 2 }] } })
    const wrapper = shallowMount(StorageTraysPage, {
      global: { stubs: { RouterLink: true, StorageTrayGrid: false } },
    })
    await flushPromises()
    expect(getStorageTrays).toHaveBeenCalledTimes(1)
    const grids = wrapper.findAllComponents({ name: 'StorageTrayGrid' })
    expect(grids).toHaveLength(2)
    expect(grids[0]?.props('feltTheme')).toBe('navy')
    expect(grids[0]?.props('sizeScale')).toBe(1)
    expect(wrapper.find('#storage-tray-size-slider').exists()).toBe(true)
  })

  it('restores and updates the shared museum tray zoom preference', async () => {
    localStorage.setItem('tray:sizeScale', '1.5')
    getStorageTrays.mockResolvedValue({ data: { trays: [occupiedThreeByThreeTray] } })
    const wrapper = shallowMount(StorageTraysPage, {
      global: { stubs: { RouterLink: true, StorageTrayGrid: false } },
    })
    await flushPromises()

    const grid = wrapper.findComponent({ name: 'StorageTrayGrid' })
    expect(grid.props('sizeScale')).toBe(1.5)

    await wrapper.get('#storage-tray-size-slider').setValue('2')
    expect(grid.props('sizeScale')).toBe(2)
    expect(localStorage.getItem('tray:sizeScale')).toBe('2')
  })

  it('offers a retry after a recoverable request failure', async () => {
    getStorageTrays.mockRejectedValueOnce(new Error('offline')).mockResolvedValueOnce({ data: { trays: [] } })
    const wrapper = shallowMount(StorageTraysPage, { global: { stubs: { RouterLink: true } } })
    await flushPromises()
    expect(wrapper.get('[role="alert"]').text()).toContain('could not be loaded')
    await wrapper.get('button').trigger('click')
    await flushPromises()
    expect(getStorageTrays).toHaveBeenCalledTimes(2)
    expect(wrapper.text()).toContain('No coin trays')
  })
})
