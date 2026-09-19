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
    facts: [],
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
      originalBytes: 100,
      persistedBytes: 100,
      digest: 'a'.repeat(64),
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
  it('emits an eligible typed dealer listing only after an explicit button click', async () => {
    const wrapper = mountProgress()
    const button = wrapper.get('button')

    expect(button.element.tagName).toBe('BUTTON')
    expect(button.attributes('type')).toBe('button')
    expect(wrapper.emitted('addToWishlist')).toBeUndefined()

    await button.trigger('click')

    expect(wrapper.emitted('addToWishlist')).toEqual([[
      'market_search',
      evidence(),
      'copilot:call_market:https://www.cngcoins.com/Coin.aspx?CoinID=400001',
    ]])
  })

  it.each([
    ['wrong capability', evidence(), 'auction_search'],
    ['auction evidence', evidence({ kind: 'auction_lot' }), 'market_search'],
    ['partial verification', evidence({ verificationState: 'partial' }), 'market_search'],
    ['not available', evidence({ availability: 'sold' }), 'market_search'],
    ['non-canonical availability', evidence({ availability: 'AVAILABLE' as 'available' }), 'market_search'],
    ['missing title', evidence({ title: ' ' }), 'market_search'],
    ['missing URL', evidence({ sourceUrl: ' ' }), 'market_search'],
    ['misleading facts', evidence({ availability: undefined, facts: ['Availability: available'] }), 'market_search'],
  ] as const)('hides the action for %s', (_label, item, capability) => {
    expect(mountProgress(item, capability).find('button').exists()).toBe(false)
  })

  it('disables an already-added listing', () => {
    const key = 'copilot:call_market:https://www.cngcoins.com/Coin.aspx?CoinID=400001'
    const button = mountProgress(evidence(), 'market_search', {
      addedSet: new Set([key]),
    }).get('button')

    expect(button.attributes('disabled')).toBeDefined()
    expect(button.text()).toBe('Added to Wishlist')
  })
})
