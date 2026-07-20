import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import EmperorTrackerPage from '@/pages/EmperorTrackerPage.vue'
import type { EmperorTrackerResult, User } from '@/types'

const mockGetProgress = vi.fn()
const mockPush = vi.fn()

const authUser: Partial<User> = { emperorTrackerEnabled: true }

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({ user: authUser }),
}))

vi.mock('vue-router', () => ({
  useRouter: () => ({ push: mockPush }),
}))

vi.mock('@/api/client', () => ({
  getEmperorTrackerProgress: () => mockGetProgress(),
  getApiErrorMessage: (error: unknown) => {
    const maybeError = error as { message?: string }
    return maybeError.message ?? ''
  },
}))

function mountPage() {
  return mount(EmperorTrackerPage, {
    global: {
      stubs: { RouterLink: { template: '<a><slot /></a>' } },
    },
  })
}

const fullResult: EmperorTrackerResult = {
  emperor: {
    roles: ['emperor'],
    owned: 1,
    total: 2,
    percentage: 50,
    dynasties: [
      {
        dynasty: 'Julio-Claudian',
        owned: 1,
        total: 2,
        figures: [
          {
            figure: { id: 1, name: 'Augustus', aliases: [], role: 'emperor', region: 'west', dynasty: 'Julio-Claudian', reignStart: -27, reignEnd: 14, sortOrder: 1, rarityTier: 'common' },
            coin: { id: 42, name: 'My Augustus', diameterMm: 18, images: [] } as never,
          },
          {
            figure: { id: 2, name: 'Tiberius', aliases: [], role: 'emperor', region: 'west', dynasty: 'Julio-Claudian', reignStart: 14, reignEnd: 37, sortOrder: 2, rarityTier: 'common' },
            coin: null,
          },
        ],
      },
    ],
  },
  suggestions: [
    { id: 2, name: 'Tiberius', aliases: [], role: 'emperor', region: 'west', dynasty: 'Julio-Claudian', reignStart: 14, reignEnd: 37, sortOrder: 2, rarityTier: 'common' },
  ],
}

describe('EmperorTrackerPage', () => {
  beforeEach(() => {
    mockGetProgress.mockReset()
    mockPush.mockReset()
    authUser.emperorTrackerEnabled = true
  })

  it('shows an enable-in-settings prompt when the account has not opted in', async () => {
    authUser.emperorTrackerEnabled = false
    mockGetProgress.mockRejectedValue({ response: { status: 403 } })
    const wrapper = mountPage()
    await flushPromises()

    expect(wrapper.text()).toContain("isn't enabled yet")
    expect(wrapper.text()).toContain('Go to Settings')
  })

  it('shows the overall completion stat and per-dynasty breakdown', async () => {
    mockGetProgress.mockResolvedValue({ data: fullResult })
    const wrapper = mountPage()
    await flushPromises()

    expect(wrapper.text()).toContain('1 of 2')
    expect(wrapper.text()).toContain('50')
    expect(wrapper.text()).toContain('Julio-Claudian')
    expect(wrapper.text()).toContain('Augustus')
    expect(wrapper.text()).toContain('Tiberius')
  })

  it('shows the what-to-pursue-next suggestions list', async () => {
    mockGetProgress.mockResolvedValue({ data: fullResult })
    const wrapper = mountPage()
    await flushPromises()

    expect(wrapper.text()).toContain('What to Pursue Next')
    expect(wrapper.text()).toContain('Tiberius')
  })

  it('does not render optional category sections when absent from the response', async () => {
    mockGetProgress.mockResolvedValue({ data: fullResult })
    const wrapper = mountPage()
    await flushPromises()

    expect(wrapper.text()).not.toContain('Usurpers —')
    expect(wrapper.text()).not.toContain('Empresses —')
    expect(wrapper.text()).not.toContain('Other Figures —')
  })

  it('renders an optional category section when present in the response', async () => {
    mockGetProgress.mockResolvedValue({
      data: {
        ...fullResult,
        empresses: {
          roles: ['empress'],
          owned: 0,
          total: 1,
          percentage: 0,
          dynasties: [
            {
              dynasty: 'Julio-Claudian',
              owned: 0,
              total: 1,
              figures: [
                {
                  figure: { id: 3, name: 'Livia', aliases: [], role: 'empress', region: 'west', dynasty: 'Julio-Claudian', reignStart: -27, reignEnd: 14, sortOrder: 3, rarityTier: 'common' },
                  coin: null,
                },
              ],
            },
          ],
        },
      },
    })
    const wrapper = mountPage()
    await flushPromises()

    expect(wrapper.text()).toContain('Empresses —')
    expect(wrapper.text()).toContain('Livia')
  })

  it('shows an error message on unexpected failure', async () => {
    mockGetProgress.mockRejectedValue({ message: 'network down' })
    const wrapper = mountPage()
    await flushPromises()

    expect(wrapper.text()).toContain('network down')
  })
})
