import { flushPromises, shallowMount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import CoinActionsPanel from '../CoinActionsPanel.vue'

const routerPush = vi.fn()

vi.mock('vue-router', () => ({
  RouterLink: {
    name: 'RouterLink',
    props: ['to'],
    template: '<a :href="to"><slot /></a>',
  },
  useRouter: () => ({
    push: routerPush,
  }),
}))

vi.mock('@/api/client', () => ({
  uploadImage: vi.fn(),
  proxyImage: vi.fn(),
  estimateCoinValue: vi.fn(),
  updateCoin: vi.fn(),
  getAIJob: vi.fn(),
  getCoinAIJobs: vi.fn().mockResolvedValue({ data: [] }),
  createDeepIdentificationJob: vi.fn(),
  getApiErrorMessage: vi.fn(() => 'error'),
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

  it('offers a Deep Analysis entry point that opens a saved-coin start panel reusing existing images', async () => {
    const wrapper = shallowMount(CoinActionsPanel, {
      props: {
        coinId: 42,
        coinName: 'Trajan denarius',
        coinMaterial: 'Silver',
        imageCount: 2,
        coinHasObverseImage: true,
        coinHasReverseImage: true,
        isPwa: false,
      },
      global: {
        stubs: {
          CoinNumistaPanel: { template: '<section />' },
          RouterLink: { props: ['to'], template: '<a :href="to"><slot /></a>' },
        },
      },
    })
    await flushPromises()

    expect(wrapper.findComponent({ name: 'DeepAnalysisEntryButton' }).exists()).toBe(true)
    expect(wrapper.findComponent({ name: 'DeepAnalysisStartPanel' }).exists()).toBe(false)

    await wrapper.findComponent({ name: 'DeepAnalysisEntryButton' }).vm.$emit('click')
    await flushPromises()

    const panel = wrapper.findComponent({ name: 'DeepAnalysisStartPanel' })
    expect(panel.exists()).toBe(true)
    expect(panel.props('coinId')).toBe(42)
    expect(panel.props('hasExistingObverse')).toBe(true)
    expect(panel.props('hasExistingReverse')).toBe(true)
  })

  it('does not issue any direct coin update when starting a saved-coin Deep Analysis job', async () => {
    const { updateCoin, createDeepIdentificationJob } = await import('@/api/client')
    vi.mocked(createDeepIdentificationJob).mockResolvedValue({ data: { job: { id: 7, status: 'queued' } } } as never)

    const wrapper = shallowMount(CoinActionsPanel, {
      props: {
        coinId: 42,
        coinName: 'Trajan denarius',
        coinMaterial: 'Silver',
        imageCount: 1,
        coinHasObverseImage: true,
        coinHasReverseImage: false,
        isPwa: false,
      },
      global: {
        stubs: {
          CoinNumistaPanel: { template: '<section />' },
          RouterLink: { props: ['to'], template: '<a :href="to"><slot /></a>' },
        },
      },
    })
    await flushPromises()
    await wrapper.findComponent({ name: 'DeepAnalysisEntryButton' }).vm.$emit('click')
    await flushPromises()

    const panel = wrapper.findComponent({ name: 'DeepAnalysisStartPanel' })
    await panel.vm.$emit('submit', { coinId: 42, obverseImage: null, reverseImage: new File([], 'r.png'), hintImages: [] })
    await flushPromises()

    expect(createDeepIdentificationJob).toHaveBeenCalledWith(expect.objectContaining({ coinId: 42 }))
    expect(updateCoin).not.toHaveBeenCalled()
    expect(routerPush).toHaveBeenCalledWith('/deep-analysis/7')
  })
})
