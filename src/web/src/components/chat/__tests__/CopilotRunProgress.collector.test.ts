import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'

import CopilotRunProgress from '@/components/chat/CopilotRunProgress.vue'
import type {
  CoinCopilotRun,
  CoinCopilotSpecialistEvidence,
  CoinCopilotSpecialistResult,
} from '@/types'
import type { CoinCopilotToolProgress } from '@/composables/useCoinCopilot'

const run: CoinCopilotRun = {
  id: 'ccr_wishlist',
  threadId: 'cct_wishlist',
  status: 'completed',
  goal: 'Find a coin',
  checkpointVersion: 1,
  lastSeq: 1,
  attempt: 1,
  finalAnswer: null,
  failureCode: null,
  failureMessage: null,
  resumeDeadline: null,
  usage: { iterations: 1, toolCalls: 1, inputTokens: 1, outputTokens: 1 },
  createdAt: '2026-09-18T18:00:00Z',
  updatedAt: '2026-09-18T18:01:00Z',
}

function evidence(overrides: Partial<CoinCopilotSpecialistEvidence> = {}): CoinCopilotSpecialistEvidence {
  return {
    kind: 'dealer_listing',
    title: 'Domitian denarius',
    sourceUrl: 'https://www.cngcoins.com/Coin.aspx?CoinID=400001',
    observedAt: '2026-09-18T12:00:00Z',
    confidence: 'high',
    verificationState: 'verified',
    dealerName: 'Classical Numismatic Group',
    listedPrice: 275,
    currency: 'USD',
    availability: 'available',
    ruler: 'Domitian',
    denomination: 'Denarius',
    era: 'Roman Imperial',
    material: 'Silver',
    matchedAttributes: [],
    materialDifferences: [],
    ...overrides,
  }
}

function mountProgress(
  item = evidence(),
  capability: CoinCopilotSpecialistResult['capability'] = 'market_search',
  state: { addingIdx?: string | null; addedSet?: Set<string> } = {},
) {
  const result: CoinCopilotSpecialistResult = {
    capability,
    outcome: 'complete',
    items: [item],
    trend: null,
    warnings: [],
    truncation: {
      truncated: false,
      omittedItems: 0,
    },
  }
  const tool: CoinCopilotToolProgress = {
    toolCallId: 'call_market',
    toolName: capability,
    stepId: 'step_1',
    status: 'succeeded',
    specialistResult: result,
  }
  return mount(CopilotRunProgress, {
    props: {
      run,
      plan: [],
      tools: [tool],
      canCancel: false,
      cancelling: false,
      truncated: false,
      addingIdx: state.addingIdx ?? null,
      addedSet: state.addedSet ?? new Set<string>(),
    },
    global: {
      stubs: {
        RouterLink: { template: '<a><slot /></a>' },
      },
    },
  })
}

describe('CopilotRunProgress collector wishlist action', () => {
  it('renders dealer listings with the same card the legacy chat uses', () => {
    const wrapper = mountProgress(evidence({ imageUrl: 'https://img.example/coin.jpg' }))
    const grid = wrapper.findComponent({ name: 'CoinSuggestionGrid' })

    expect(grid.exists()).toBe(true)
    expect(grid.props('suggestions')).toMatchObject([{
      name: 'Domitian denarius',
      imageUrl: 'https://img.example/coin.jpg',
      sourceName: 'Classical Numismatic Group',
      era: 'Roman Imperial',
      material: 'Silver',
      denomination: 'Denarius',
    }])
  })

  it('maps a grid wish list click back to the typed evidence item', async () => {
    const wrapper = mountProgress()
    const grid = wrapper.findComponent({ name: 'CoinSuggestionGrid' })

    grid.vm.$emit('add-to-wishlist', { name: 'Domitian denarius' }, '0-0')
    await wrapper.vm.$nextTick()

    expect(wrapper.emitted('addToWishlist')).toEqual([[
      'market_search',
      evidence(),
      'copilot:call_market:https://www.cngcoins.com/Coin.aspx?CoinID=400001',
    ]])
  })

  it('emits an eligible typed dealer listing only after an explicit button click', async () => {
    const wrapper = mountProgress()
    const button = wrapper.findAll('button').find(candidate => candidate.text().includes('Add to Wishlist'))

    expect(button).toBeDefined()
    expect(wrapper.emitted('addToWishlist')).toBeUndefined()

    await button!.trigger('click')

    expect(wrapper.emitted('addToWishlist')).toEqual([[
      'market_search',
      evidence(),
      'copilot:call_market:https://www.cngcoins.com/Coin.aspx?CoinID=400001',
    ]])
  })

  it('renders dealer listings with the same card the legacy chat uses', () => {
    const wrapper = mountProgress(evidence({ imageUrl: 'https://img.example/coin.jpg' }))
    const grid = wrapper.findComponent({ name: 'CoinSuggestionGrid' })

    expect(grid.exists()).toBe(true)
    expect(grid.props('suggestions')).toMatchObject([{
      name: 'Domitian denarius',
      imageUrl: 'https://img.example/coin.jpg',
      sourceName: 'Classical Numismatic Group',
      era: 'Roman Imperial',
      material: 'Silver',
      denomination: 'Denarius',
    }])
  })

  it('maps a grid wish list click back to the typed evidence item', async () => {
    const wrapper = mountProgress()
    const grid = wrapper.findComponent({ name: 'CoinSuggestionGrid' })

    grid.vm.$emit('add-to-wishlist', { name: 'Domitian denarius' }, '0-0')
    await wrapper.vm.$nextTick()

    expect(wrapper.emitted('addToWishlist')).toEqual([[
      'market_search',
      evidence(),
      'copilot:call_market:https://www.cngcoins.com/Coin.aspx?CoinID=400001',
    ]])
  })

  it('shows the action for a listing whose availability the page did not state', () => {
    const wrapper = mountProgress(evidence({ availability: 'unknown' }))
    expect(wrapper.find('button').exists()).toBe(true)
    expect(wrapper.get('[data-testid="dealer-uncertainty"]').text()).toContain('Availability is unknown')
    expect(wrapper.get('[data-testid="dealer-uncertainty"]').text()).toContain('Domitian')
  })

  it('shows the action for a partially verified listing read from search results', () => {
    const item = evidence({ verificationState: 'partial', confidence: 'medium', availability: 'unknown' })
    const wrapper = mountProgress(item)
    expect(wrapper.find('button').exists()).toBe(true)
    expect(wrapper.get('[data-testid="dealer-uncertainty"]').text()).toContain('only partially verified')
  })

  it.each([
    ['wrong capability', evidence(), 'auction_search'],
    ['auction evidence', evidence({ kind: 'auction_lot' }), 'market_search'],
    ['not available', evidence({ availability: 'sold' }), 'market_search'],
    ['non-canonical availability', evidence({ availability: 'AVAILABLE' as 'available' }), 'market_search'],
    ['missing title', evidence({ title: ' ' }), 'market_search'],
    ['missing URL', evidence({ sourceUrl: ' ' }), 'market_search'],
  ] as const)('hides the action for %s', (_label, item, capability) => {
    expect(mountProgress(item, capability).find('button').exists()).toBe(false)
  })

  it('disables an already-added listing', () => {
    const key = 'copilot:call_market:https://www.cngcoins.com/Coin.aspx?CoinID=400001'
    const wrapper = mountProgress(evidence(), 'market_search', { addedSet: new Set([key]) })
    const button = wrapper.findAll('button').find(candidate => candidate.text().includes('Added'))

    expect(button).toBeDefined()
    expect(button!.text()).toContain('Added!')
  })
})
