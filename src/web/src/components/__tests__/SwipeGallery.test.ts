import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { readFileSync } from 'fs'
import { join } from 'path'
import SwipeGallery from '../SwipeGallery.vue'
import { useCoinsStore } from '@/stores/coins'
import { buildRomanDenariusCore } from '@/test/fixtures/coins'
import type { Coin } from '@/types'

vi.mock('vue-router', () => ({
  useRouter: () => ({ push: vi.fn() }),
}))

function buildCoins(count: number): Coin[] {
  return Array.from({ length: count }, (_, index) => buildRomanDenariusCore({
    id: index + 1,
    name: `Coin ${index + 1}`,
    images: [],
  }))
}

describe('SwipeGallery', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('shows the absolute position across paged PWA swipe results', () => {
    const store = useCoinsStore()
    store.galleryIndex = 0

    const wrapper = mount(SwipeGallery, {
      props: { coins: buildCoins(14), total: 64, page: 2, perPage: 50 },
    })

    expect(wrapper.find('.card-counter').text()).toBe('51 / 64')
  })

  it('loads the next page after swiping past the first page of coins', async () => {
    const store = useCoinsStore()
    store.galleryIndex = 49

    const wrapper = mount(SwipeGallery, {
      props: { coins: buildCoins(50), total: 64, page: 1, perPage: 50 },
    })

    await wrapper.findAll('.nav-btn')[1]!.trigger('click')
    await vi.advanceTimersByTimeAsync(300)

    expect(store.galleryIndex).toBe(0)
    expect(wrapper.emitted('page-change')).toEqual([[2]])
  })

  it('keeps navigation working when the final page has one coin', async () => {
    const store = useCoinsStore()
    store.galleryIndex = 0

    const wrapper = mount(SwipeGallery, {
      props: { coins: buildCoins(1), total: 51, page: 2, perPage: 50 },
    })

    expect(wrapper.find('.swipe-nav').exists()).toBe(true)

    await wrapper.findAll('.nav-btn')[1]!.trigger('click')
    await vi.advanceTimersByTimeAsync(300)

    expect(store.galleryIndex).toBe(0)
    expect(wrapper.emitted('page-change')).toEqual([[1]])
  })

  it('keeps swipe navigation buttons on a single line', () => {
    const source = readFileSync(join(process.cwd(), 'src', 'components', 'SwipeGallery.vue'), 'utf-8')

    expect(source).toContain('<span>Prev</span>')
    expect(source).toContain('<span>Next</span>')
    expect(source).toContain('flex-wrap: nowrap')
    expect(source).toContain('white-space: nowrap')
    expect(source).toContain('flex-shrink: 0')
    expect(source).toContain('min-width: 7.5rem')
    expect(source).toContain('flex-direction: row')
  })
})
