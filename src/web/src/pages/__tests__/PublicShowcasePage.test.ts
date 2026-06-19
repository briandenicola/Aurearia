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

describe('PublicShowcasePage', () => {
  beforeEach(() => {
    vi.mocked(getPublicShowcase).mockResolvedValue({
      data: {
        showcase: { title: 'Featured Set' },
        coins: [
          {
            id: 1,
            name: 'Aureus',
            images: [{ id: 10, filePath: 'coins/aureus.webp', imageType: 'obverse' }],
          },
        ],
      },
    } as Awaited<ReturnType<typeof getPublicShowcase>>)
  })

  it('renders coin images through the public showcase media route', async () => {
    const wrapper = mount(PublicShowcasePage)

    await flushPromises()

    expect(wrapper.find('img.coin-image').attributes('src')).toBe(
      '/api/showcase/featured-set/uploads/coins/aureus.webp',
    )
  })
})
