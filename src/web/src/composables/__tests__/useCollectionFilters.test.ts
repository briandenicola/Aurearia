import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { defineComponent, h } from 'vue'
import { mount } from '@vue/test-utils'
import { COIN_ERAS } from '@/types'
import { getAppSettings, getSets, getTags } from '@/api/client'
import { useCollectionFilters } from '../useCollectionFilters'

vi.mock('@/api/client', () => ({
  getAppSettings: vi.fn(),
  getSets: vi.fn(),
  getTags: vi.fn(),
}))

describe('useCollectionFilters', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    localStorage.clear()
    sessionStorage.clear()
    vi.clearAllMocks()
  })

  it('keeps tag and set filters when admin settings are unavailable', async () => {
    vi.mocked(getTags).mockResolvedValue({
      data: {
        tags: [
          { id: 1, name: 'Favorites', color: '#c9a84c' },
        ],
      },
    } as Awaited<ReturnType<typeof getTags>>)
    vi.mocked(getSets).mockResolvedValue({
      data: {
        sets: [
          { id: 2, name: 'Twelve Caesars', color: '#b08d57', setType: 'open' },
        ],
      },
    } as Awaited<ReturnType<typeof getSets>>)
    vi.mocked(getAppSettings).mockRejectedValue({ response: { status: 403 } })

    let filters!: ReturnType<typeof useCollectionFilters>
    const wrapper = mount(defineComponent({
      setup() {
        filters = useCollectionFilters()
        return () => h('div')
      },
    }))
    await filters.fetchUserTags()

    expect(filters.userTags.value).toEqual([
      {
        id: 1,
        name: 'Favorites',
        color: '#c9a84c',
        filterValue: 'tag:1',
        source: 'tag',
      },
      {
        id: 2,
        name: 'Twelve Caesars',
        color: '#b08d57',
        filterValue: 'set:2',
        source: 'set',
      },
    ])
    expect(filters.eraOptions.value).toEqual(COIN_ERAS)
    wrapper.unmount()
  })
})
