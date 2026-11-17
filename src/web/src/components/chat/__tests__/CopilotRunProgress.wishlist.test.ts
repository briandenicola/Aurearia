import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import CopilotRunProgress from '@/components/chat/CopilotRunProgress.vue'
import { isCopilotWishlistProposal } from '@/api/endpoints/agent'
import type { CoinCopilotRun, CopilotWishlistProposal } from '@/types'
import type { CoinCopilotToolProgress } from '@/composables/useCoinCopilot'

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

const SOURCE = 'https://www.hjbltd.com/#!/inventory/item-detail/ancient-coins/119010'

const run: CoinCopilotRun = {
  id: 'ccr_wishlist',
  threadId: 'cct_wishlist',
  status: 'completed',
  goal: 'Add this coin to my wishlist',
  checkpointVersion: 1,
  lastSeq: 1,
  attempt: 1,
  finalAnswer: null,
  failureCode: null,
  failureMessage: null,
  resumeDeadline: null,
  usage: { iterations: 2, toolCalls: 1, inputTokens: 10, outputTokens: 10 },
  createdAt: '2026-11-17T18:00:00Z',
  updatedAt: '2026-11-17T18:01:00Z',
}

function field(value: string) {
  return { value, confidence: 0.9, evidence: [value] }
}

function readyProposal(): CopilotWishlistProposal {
  return {
    schemaVersion: 1,
    outcome: 'ready',
    sourceUrl: SOURCE,
    pageTitle: 'Theodosius I. AE 3',
    imageUrl: 'https://images.hjbltd.com/ancient-coins/cc119010a.jpg?cfcache=2',
    hypothesis: {
      name: field('Theodosius I. AE 3'),
      ruler: field('Theodosius I'),
      denomination: field('AE 3'),
      material: field('Bronze'),
      weightGrams: field('3.35'),
      listedPrice: field('200.00'),
      currency: field('USD'),
      observations: '',
      legible: true,
    } as CopilotWishlistProposal['hypothesis'],
  }
}

function mountWith(proposal: CopilotWishlistProposal) {
  const tool: CoinCopilotToolProgress = {
    toolCallId: 'call_wishlist',
    toolName: 'wishlist_url_proposal',
    stepId: 'step-1',
    status: 'succeeded',
    resultSummary: 'Wishlist proposal ready for your review.',
    wishlistProposal: proposal,
  }
  return mount(CopilotRunProgress, {
    props: {
      run,
      plan: [],
      tools: [tool],
      canCancel: false,
      cancelling: false,
      truncated: false,
      addingIdx: null,
      addedSet: new Set<string>(),
    },
    global: { stubs: { RouterLink: { props: ['to'], template: '<a :href="to"><slot /></a>' } } },
  })
}

describe('Copilot wishlist proposal', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mocks.proxy.mockRejectedValue(new Error('no image'))
    mocks.create.mockResolvedValue({ data: { id: 77 } })
    mocks.settings.mockResolvedValue({ data: {} })
  })

  it('prefills the review form and adds only after the owner confirms', async () => {
    const wrapper = mountWith(readyProposal())
    await flushPromises()

    const section = wrapper.get('[data-testid="copilot-wishlist-proposal"]')
    expect(section.find('form.url-row').exists()).toBe(false)
    expect(section.get('a[href]').attributes('href')).toBe(SOURCE)
    expect((section.get('input[required]').element as HTMLInputElement).value).toBe('Theodosius I. AE 3')
    expect(mocks.analyze).not.toHaveBeenCalled()
    expect(mocks.create).not.toHaveBeenCalled()

    await section.get('form.proposal').trigger('submit')
    await flushPromises()

    expect(mocks.create).toHaveBeenCalledTimes(1)
    expect(mocks.create.mock.calls[0][0]).toMatchObject({
      name: 'Theodosius I. AE 3',
      ruler: 'Theodosius I',
      referenceUrl: SOURCE,
      currentValue: 200,
      isWishlist: true,
    })
    expect(section.text()).toContain('Wishlist coin added.')
  })

  it('links to the existing coin for a duplicate instead of offering to add it', async () => {
    const wrapper = mountWith({ schemaVersion: 1, outcome: 'duplicate', sourceUrl: SOURCE, existingCoinId: 12 })
    await flushPromises()

    const section = wrapper.get('[data-testid="copilot-wishlist-proposal"]')
    expect(section.text()).toContain('already on your wishlist')
    expect(section.find('form.proposal').exists()).toBe(false)
    expect(section.find('a[href="/coin/12"]').exists()).toBe(true)
  })

  it('explains an unreadable page without a form', async () => {
    const wrapper = mountWith({
      schemaVersion: 1,
      outcome: 'unreadable',
      sourceUrl: 'https://dealer.example/lot/42',
      warnings: ['The listing page could not be read.'],
    })
    await flushPromises()

    const section = wrapper.get('[data-testid="copilot-wishlist-proposal"]')
    expect(section.text()).toContain('could not be read')
    expect(section.find('form').exists()).toBe(false)
    expect(section.find('a[href="/wishlist"]').exists()).toBe(true)
  })
})

describe('isCopilotWishlistProposal', () => {
  it('accepts every outcome shape the API publishes', () => {
    expect(isCopilotWishlistProposal(readyProposal())).toBe(true)
    expect(isCopilotWishlistProposal({ ...readyProposal(), sourceUrl: 'http://dealer.example/lot/1' })).toBe(true)
    expect(isCopilotWishlistProposal({ schemaVersion: 1, outcome: 'duplicate', sourceUrl: SOURCE, existingCoinId: 3 })).toBe(true)
    expect(isCopilotWishlistProposal({ schemaVersion: 1, outcome: 'invalid_url', sourceUrl: '', warnings: ['x'] })).toBe(true)
  })

  it.each([
    ['unknown outcome', { ...readyProposal(), outcome: 'added' }],
    ['script source', { ...readyProposal(), sourceUrl: 'javascript:alert(1)' }],
    ['private source', { ...readyProposal(), sourceUrl: 'http://192.168.1.10/lot' }],
    ['private image', { ...readyProposal(), imageUrl: 'http://localhost/x.jpg' }],
    ['ready without fields', { ...readyProposal(), hypothesis: undefined }],
    ['duplicate without coin', { schemaVersion: 1, outcome: 'duplicate', sourceUrl: SOURCE }],
    ['failure with fields', { ...readyProposal(), outcome: 'unreadable' }],
    ['wrong schema', { ...readyProposal(), schemaVersion: 2 }],
  ])('rejects %s', (_name, value) => {
    expect(isCopilotWishlistProposal(value)).toBe(false)
  })
})
