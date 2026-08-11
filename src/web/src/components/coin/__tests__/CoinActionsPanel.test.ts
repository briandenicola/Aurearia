import { flushPromises, shallowMount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import CoinActionsPanel from '../CoinActionsPanel.vue'

vi.mock('@/api/client', () => ({
  uploadImage: vi.fn(),
  proxyImage: vi.fn(),
  estimateCoinValue: vi.fn(),
  updateCoin: vi.fn(),
  getAIJob: vi.fn(),
  getCoinAIJobs: vi.fn().mockResolvedValue({ data: [] }),
}))

vi.mock('@/composables/useDialog', () => ({
  useDialog: () => ({ showAlert: vi.fn() }),
}))

vi.mock('@/composables/useNotifications', () => ({
  useNotifications: () => ({ refresh: vi.fn() }),
}))

vi.mock('@/composables/useToast', () => ({
  useToast: () => ({ showToast: vi.fn() }),
}))

describe('CoinActionsPanel Numista transition', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('removes the full lookup panel and offers at most one compact link to Catalog References', async () => {
    const wrapper = shallowMount(CoinActionsPanel, {
      props: {
        coinId: 42,
        coinName: 'Trajan denarius',
        coinMaterial: 'Silver',
        imageCount: 0,
        isPwa: false,
      },
      global: {
        stubs: {
          CoinNumistaPanel: {
            template: '<section data-testid="full-numista-panel"><h3>Numista Lookup</h3></section>',
          },
          RouterLink: {
            props: ['to'],
            template: '<a :href="to"><slot /></a>',
          },
        },
      },
    })
    await flushPromises()

    expect(wrapper.find('[data-testid="full-numista-panel"]').exists()).toBe(false)
    expect(wrapper.text()).not.toContain('Numista Lookup')
    expect(wrapper.findAll('[data-lookup-state]')).toHaveLength(0)

    const contextualLinks = wrapper.findAll('a').filter(link =>
      /Catalog References|Search Numista/i.test(link.text()),
    )
    expect(contextualLinks.length).toBeLessThanOrEqual(1)
    expect(contextualLinks).toHaveLength(1)
    expect(contextualLinks[0]!.attributes('href')).toBe('/coin/42#catalog-references')
    expect(contextualLinks[0]!.classes()).toContain('btn-sm')
  })
})
