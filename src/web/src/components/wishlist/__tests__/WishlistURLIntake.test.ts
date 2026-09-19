import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import WishlistURLIntake from '../WishlistURLIntake.vue'

const mocks = vi.hoisted(() => ({
  analyze: vi.fn(),
  create: vi.fn(),
  proxy: vi.fn(),
  upload: vi.fn(),
  settings: vi.fn(),
}))

vi.mock('@/api/client', () => ({
  analyzeWishlistURL: mocks.analyze,
  createCoin: mocks.create,
  proxyImage: mocks.proxy,
  uploadImage: mocks.upload,
  getApiErrorMessage: () => 'Request failed',
  getAppSettings: mocks.settings,
}))

function mountComponent() {
  return mount(WishlistURLIntake, {
    global: {
      stubs: {
        RouterLink: { template: '<a><slot /></a>' },
      },
    },
  })
}

describe('WishlistURLIntake', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mocks.proxy.mockRejectedValue(new Error('no image'))
    mocks.create.mockResolvedValue({ data: { id: 77 } })
    mocks.settings.mockResolvedValue({ data: {} })
  })

  it('keeps duplicate results read-only and links to the existing coin', async () => {
    mocks.analyze.mockResolvedValue({
      data: {
        schemaVersion: 1,
        outcome: 'duplicate',
        sourceUrl: 'https://dealer.example/lot/7',
        existingCoinId: 42,
      },
    })
    const wrapper = mountComponent()
    await wrapper.get('button').trigger('click')
    await wrapper.get('input[type="url"]').setValue('https://dealer.example/lot/7')
    await wrapper.get('form').trigger('submit')
    await flushPromises()

    expect(wrapper.text()).toContain('already on your wishlist')
    expect(wrapper.text()).toContain('View existing wishlist coin')
    expect(mocks.create).not.toHaveBeenCalled()
  })

  it('requires explicit confirmation before canonical coin creation', async () => {
    mocks.analyze.mockResolvedValue({
      data: {
        schemaVersion: 1,
        outcome: 'ready',
        sourceUrl: 'https://dealer.example/lot/9',
        hypothesis: {
          name: { value: 'Hadrian Denarius', confidence: 0.9, evidence: ['Hadrian Denarius'] },
          category: { value: 'Roman', confidence: 0.9, evidence: ['Roman'] },
          material: { value: 'Silver', confidence: 0.8, evidence: ['Silver'] },
          listedPrice: { value: '125', confidence: 0.9, evidence: ['USD 125'] },
          dealerName: { value: 'Dealer', confidence: 0.9, evidence: ['Dealer'] },
          observations: '',
          legible: true,
        },
      },
    })
    const wrapper = mountComponent()
    await wrapper.get('button').trigger('click')
    await wrapper.get('input[type="url"]').setValue('https://dealer.example/lot/9')
    await wrapper.get('form').trigger('submit')
    await flushPromises()

    expect(mocks.create).not.toHaveBeenCalled()
    await wrapper.get('.proposal').trigger('submit')
    await flushPromises()

    expect(mocks.create).toHaveBeenCalledWith(expect.objectContaining({
      name: 'Hadrian Denarius',
      category: 'Roman',
      material: 'Silver',
      currentValue: 125,
      referenceUrl: 'https://dealer.example/lot/9',
      isWishlist: true,
    }))
    expect(wrapper.emitted('created')).toHaveLength(1)
  })

  it('cancels an in-flight analysis without creating a proposal', async () => {
    mocks.analyze.mockImplementation((_url: string, signal: AbortSignal) => new Promise((_resolve, reject) => {
      signal.addEventListener('abort', () => reject(new DOMException('Aborted', 'AbortError')))
    }))
    const wrapper = mountComponent()
    await wrapper.get('button').trigger('click')
    await wrapper.get('input[type="url"]').setValue('https://dealer.example/lot/10')
    await wrapper.get('form').trigger('submit')
    await wrapper.get('.url-row button[type="button"]').trigger('click')
    await flushPromises()

    expect(wrapper.text()).toContain('Analysis cancelled')
    expect(mocks.create).not.toHaveBeenCalled()
  })
})
