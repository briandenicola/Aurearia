import { flushPromises, shallowMount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import StorageTraysPage from '@/pages/StorageTraysPage.vue'
import { emptyThreeByThreeTray, occupiedThreeByThreeTray } from '@/test/fixtures/storageTrays'

const getStorageTrays = vi.fn()
const push = vi.fn()
vi.mock('@/api/client', () => ({ getStorageTrays: () => getStorageTrays() }))
vi.mock('vue-router', () => ({ useRouter: () => ({ push }) }))

describe('StorageTraysPage', () => {
  beforeEach(() => {
    getStorageTrays.mockReset()
    push.mockReset()
  })

  it('uses one aggregate request and renders empty and occupied trays', async () => {
    getStorageTrays.mockResolvedValue({ data: { trays: [emptyThreeByThreeTray, { ...occupiedThreeByThreeTray, id: 2 }] } })
    const wrapper = shallowMount(StorageTraysPage, {
      global: { stubs: { RouterLink: true, StorageTrayGrid: false } },
    })
    await flushPromises()
    expect(getStorageTrays).toHaveBeenCalledTimes(1)
    expect(wrapper.findAllComponents({ name: 'StorageTrayGrid' })).toHaveLength(2)
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
