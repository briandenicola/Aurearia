import { describe, it, expect, vi, beforeEach } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import PublicShowcasePage from '@/pages/PublicShowcasePage.vue'
import { getPublicShowcase } from '@/api/client'

vi.mock('vue-router', () => ({
  useRoute: () => ({ params: { slug: 'featured-set' } }),
}))

vi.mock('@/api/client', () => ({
  getPublicShowcase: vi.fn(),
}))

const showcase = {
  title: 'Featured Set',
  description: 'Coins selected for public viewing',
  ownerName: 'Brian',
}

function publicCoin(overrides: Record<string, unknown> = {}) {
  return {
    id: 1,
    name: 'Aureus',
    diameterMm: 20,
    era: 'Roman Imperial',
    category: 'Roman',
    grade: 'VF',
    images: [{ id: 10, filePath: 'coins/aureus.webp', imageType: 'obverse', isPrimary: true }],
    ...overrides,
  }
}

function mockShowcase(coins = [publicCoin()]) {
  vi.mocked(getPublicShowcase).mockResolvedValue({
    data: { showcase, coins },
  } as Awaited<ReturnType<typeof getPublicShowcase>>)
}

async function mountLoaded() {
  const wrapper = mount(PublicShowcasePage)
  await flushPromises()
  return wrapper
}

describe('PublicShowcasePage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    localStorage.clear()
    mockShowcase()
  })

  it('loads a public showcase and renders coins in the tray layout instead of the old card grid', async () => {
    const wrapper = await mountLoaded()

    expect(getPublicShowcase).toHaveBeenCalledWith('featured-set')
    expect(wrapper.text()).toContain('Featured Set')
    expect(wrapper.find('.public-tray-section').exists()).toBe(true)
    expect(wrapper.find('.museum-tray').exists()).toBe(true)
    expect(wrapper.findAll('.tray-well')).toHaveLength(1)
    expect(wrapper.find('.coins-grid').exists()).toBe(false)
    expect(wrapper.find('.coin-card').exists()).toBe(false)
  })

  it('renders public showcase images through the showcase media route', async () => {
    const wrapper = await mountLoaded()

    const image = wrapper.find('img.well-coin')
    expect(image.attributes('src')).toBe('/api/showcase/featured-set/uploads/coins/aureus.webp')
    expect(image.attributes('alt')).toBe('Aureus')
  })

  it('keeps public tray coin labels available without exposing private coin-detail navigation', async () => {
    const wrapper = await mountLoaded()
    const well = wrapper.find('.tray-well')

    expect(well.attributes('aria-label')).toBe('Aureus')
    expect(well.attributes('role')).toBeUndefined()
    expect(well.attributes('tabindex')).toBeUndefined()
  })

  it('keeps tray drawer navigation available for larger public showcases', async () => {
    const coins = Array.from({ length: 13 }, (_, index) => publicCoin({
      id: index + 1,
      name: `Public Coin ${index + 1}`,
      images: [{ id: index + 10, filePath: `coins/${index + 1}.webp`, imageType: 'obverse' }],
    }))
    mockShowcase(coins)
    const wrapper = await mountLoaded()

    expect(wrapper.text()).toContain('Tray 1 of 2')
    expect(wrapper.findAll('.tray-well')).toHaveLength(12)

    await wrapper.findAll('button').find(button => button.text().includes('Next'))?.trigger('click')

    expect(wrapper.text()).toContain('Tray 2 of 2')
    expect(wrapper.findAll('.tray-well')).toHaveLength(1)
    expect(wrapper.find('.tray-well').attributes('aria-label')).toBe('Public Coin 13')
  })

  it('preserves loading, empty, and error states', async () => {
    vi.mocked(getPublicShowcase).mockReturnValue(new Promise(() => {}) as ReturnType<typeof getPublicShowcase>)
    const loadingWrapper = mount(PublicShowcasePage)
    expect(loadingWrapper.text()).toContain('Loading showcase...')

    mockShowcase([])
    const emptyWrapper = await mountLoaded()
    expect(emptyWrapper.text()).toContain('This showcase has no coins yet.')
    expect(emptyWrapper.find('.museum-tray').exists()).toBe(false)

    vi.mocked(getPublicShowcase).mockRejectedValue(new Error('not found'))
    const errorWrapper = await mountLoaded()
    expect(errorWrapper.text()).toContain('Showcase not found')
    expect(errorWrapper.text()).toContain('This showcase may have been removed or the link is incorrect.')
  })
})
