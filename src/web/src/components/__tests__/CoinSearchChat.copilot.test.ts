import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import CoinSearchChat from '../CoinSearchChat.vue'
import type { CoinCopilotRun, CoinCopilotSpecialistResult } from '@/types'
import type { CoinCopilotToolProgress } from '@/composables/useCoinCopilot'

const mocks = vi.hoisted(() => ({
  cancel: vi.fn(),
  resume: vi.fn(),
  addToWishlist: vi.fn(),
  active: false,
  run: null as CoinCopilotRun | null,
  tools: [] as CoinCopilotToolProgress[],
  clarification: null as null | {
    question: string
    inputType: 'text' | 'single_choice' | 'boolean'
    choices: string[]
    checkpointVersion: number
  },
}))

vi.mock('@/api/client', () => ({
  createNote: vi.fn(),
  getApiErrorMessage: () => '',
}))

vi.mock('@/composables/useDialog', () => ({
  useDialog: () => ({ showAlert: vi.fn() }),
}))

vi.mock('@/composables/useCoinSearchChat', () => ({
  useCoinSearchChat: () => ({
    messages: ref([{ role: 'user', content: 'Review my collection' }, { role: 'assistant', content: '', streaming: true }]),
    input: ref(''),
    loading: ref(false),
    addingIdx: ref(null),
    addedSet: ref(new Set()),
    savedShows: ref(new Set()),
    savingShow: ref(null),
    conversationId: ref(null),
    saving: ref(false),
    saveLabel: ref('Save'),
    providerConfigured: ref(true),
    categoryEraConfirmRequest: ref(null),
    copilotActive: ref(mocks.active),
    copilotRun: ref(mocks.run),
    copilotPlan: ref([
      { id: 'step_1', title: 'Summarize holdings', status: 'completed' },
      { id: 'step_2', title: 'Identify gaps', status: 'in_progress' },
    ]),
    copilotTools: ref(mocks.tools),
    copilotClarification: ref(mocks.clarification),
    copilotCanCancel: ref(Boolean(mocks.run && ['queued', 'running', 'paused'].includes(mocks.run.status))),
    copilotCanResume: ref(Boolean(mocks.run?.status === 'paused' && mocks.clarification)),
    copilotTruncated: ref(false),
    copilotError: ref(''),
    copilotCancelling: ref(false),
    copilotResuming: ref(false),
    chooseCategoryEraConfirmation: vi.fn(),
    cancelCategoryEraConfirmation: vi.fn(),
    sendMessage: vi.fn(),
    cancelCopilotRun: mocks.cancel,
    resumeCopilotRun: mocks.resume,
    sendExample: vi.fn(),
    sendPortfolioAnalysis: vi.fn(),
    handleSave: vi.fn(),
    addToWishlist: mocks.addToWishlist,
    confirmCollectionProposal: vi.fn(),
    cancelCollectionProposalMessage: vi.fn(),
    pickDisambiguationCandidate: vi.fn(),
    formatMessage: (message: string) => message,
    isCoinShowResults: () => false,
    saveShowToCalendar: vi.fn(),
  }),
}))

function activeRun(status: CoinCopilotRun['status']): CoinCopilotRun {
  return {
    id: 'ccr_1',
    threadId: 'cct_1',
    status,
    goal: 'Review my collection',
    checkpointVersion: 4,
    lastSeq: 3,
    attempt: 1,
    finalAnswer: null,
    failureCode: null,
    failureMessage: null,
    resumeDeadline: status === 'paused' ? '2026-09-24T00:00:00Z' : null,
    usage: { iterations: 1, toolCalls: 1, inputTokens: 1, outputTokens: 1 },
    createdAt: '',
    updatedAt: '',
  }
}

function mountChat() {
  return mount(CoinSearchChat, {
    global: {
      stubs: {
        CoinSuggestionGrid: true,
        CoinShowResultsGrid: true,
        CategoryEraConfirmModal: true,
      },
    },
  })
}

function specialistResult(
  outcome: CoinCopilotSpecialistResult['outcome'],
): CoinCopilotSpecialistResult {
  return {
    capability: 'market_search',
    outcome,
    items: outcome === 'complete' || outcome === 'partial' ? [{
      kind: 'dealer_listing',
      title: 'Domitian denarius',
      sourceUrl: 'https://www.cngcoins.com/Coin.aspx?CoinID=400001',
      observedAt: '2026-09-18T12:00:00Z',
      confidence: 'high',
      verificationState: 'verified',
      description: 'Silver denarius with Minerva reverse',
      dealerName: 'Classical Numismatic Group',
      listedPrice: 275,
      currency: 'USD',
      availability: 'available',
      ruler: 'Domitian',
      denomination: 'Denarius',
      era: 'Roman Imperial',
      material: 'Silver',
      facts: ['Dealer: Classical Numismatic Group', 'Price: USD 275'],
      matchedAttributes: [],
      materialDifferences: [],
    }] : [],
    trend: null,
    warnings: outcome === 'partial' || outcome === 'unavailable'
      ? ['One configured source was unavailable.']
      : [],
    truncation: {
      truncated: false,
      originalBytes: 100,
      persistedBytes: 100,
      digest: 'a'.repeat(64),
      omittedItems: 0,
    },
  }
}

function specialistTool(result: CoinCopilotSpecialistResult): CoinCopilotToolProgress {
  return {
    toolCallId: `call_${result.outcome}`,
    toolName: 'market_search',
    stepId: 'step_1',
    status: 'succeeded',
    resultSummary: 'Market search complete.',
    truncated: false,
    specialistResult: result,
  }
}

function priceTrendResult(): CoinCopilotSpecialistResult {
  const sourceUrl = 'https://www.numisbids.com/n.php?p=lot&sid=8000&lot=1'
  return {
    capability: 'price_trends',
    outcome: 'complete',
    items: [{
      kind: 'sale_observation',
      title: 'Domitian denarius sold at auction',
      sourceUrl,
      observedAt: '2026-09-18T12:00:00Z',
      confidence: 'high',
      verificationState: 'verified',
      facts: ['Amount: USD 275', 'Price basis: Hammer'],
      matchedAttributes: [],
      materialDifferences: [],
    }],
    trend: {
      state: 'rising',
      sampleSize: 3,
      dateFrom: '2026-05-01',
      dateTo: '2026-09-01',
      currency: 'USD',
      priceBasis: 'hammer',
      low: 200,
      median: 250,
      high: 300,
      confidence: 'high',
      limitations: ['The sample contains three verified auction sales.'],
      supportingSourceIds: [sourceUrl],
    },
    warnings: [],
    truncation: {
      truncated: false,
      originalBytes: 300,
      persistedBytes: 300,
      digest: 'b'.repeat(64),
      omittedItems: 0,
    },
  }
}

describe('CoinSearchChat Coin Copilot drawer integration', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mocks.active = false
    mocks.run = null
    mocks.clarification = null
    mocks.tools = [{
      toolCallId: 'call_1',
      toolName: 'collection_summary',
      stepId: 'step_1',
      status: 'succeeded',
      resultSummary: 'Collection summary returned.',
      truncated: false,
    }]
  })

  it('preserves the legacy drawer without Copilot UI when fallback mode is active', () => {
    const wrapper = mountChat()

    expect(wrapper.text()).toContain('Coin Agent')
    expect(wrapper.find('[data-testid="copilot-run-progress"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="copilot-clarification"]').exists()).toBe(false)
    expect(wrapper.findAll('button').some(button => button.text() === 'Save')).toBe(true)
  })

  it('keeps an accepted run readable and cancellable after capability falls back', async () => {
    mocks.active = false
    mocks.run = activeRun('running')
    const wrapper = mountChat()

    const progress = wrapper.get('[data-testid="copilot-run-progress"]')
    expect(progress.text()).toContain('Beta')
    await progress.get('button').trigger('click')
    expect(mocks.cancel).toHaveBeenCalledTimes(1)
    expect(wrapper.findAll('button').some(button => button.text() === 'Save')).toBe(false)
  })

  it('offers conversation save again once the Coin Copilot run has finished', () => {
    mocks.active = true
    mocks.run = activeRun('completed')
    const wrapper = mountChat()

    expect(wrapper.findAll('button').some(button => button.text() === 'Save')).toBe(true)
  })

  it('renders compact beta plan/tool progress and cancellation in the same responsive drawer', async () => {
    mocks.active = true
    mocks.run = activeRun('running')
    const wrapper = mountChat()

    const progress = wrapper.get('[data-testid="copilot-run-progress"]')
    expect(progress.text()).toContain('Beta')
    expect(progress.text()).toContain('Summarize holdings')
    expect(progress.text()).toContain('Identify gaps')
    expect(progress.text()).toContain('Collection Summary')
    const cancel = progress.get('button')
    expect(cancel.classes()).toContain('min-h-[44px]')
    await cancel.trigger('click')
    expect(mocks.cancel).toHaveBeenCalledTimes(1)
    expect(wrapper.classes()).toContain('fixed')
    expect(wrapper.findAll('button').some(button => button.text() === 'Save')).toBe(false)
  })

  it('renders an accessible clarification card and resumes with the selected answer', async () => {
    mocks.active = true
    mocks.run = activeRun('paused')
    mocks.clarification = {
      question: 'Should sold coins be included?',
      inputType: 'single_choice',
      choices: ['Owned only', 'Include sold'],
      checkpointVersion: 4,
    }
    const wrapper = mountChat()

    const card = wrapper.get('[data-testid="copilot-clarification"]')
    expect(card.attributes('aria-labelledby')).toBe('copilot-clarification-heading')
    await card.findAll('button').find(button => button.text() === 'Owned only')!.trigger('click')
    await card.findAll('button').find(button => button.text() === 'Continue')!.trigger('click')
    expect(mocks.resume).toHaveBeenCalledWith('Owned only')
  })

  it.each([
    ['complete', 'Complete'],
    ['partial', 'Partial'],
    ['no_match', 'No matching evidence'],
    ['unavailable', 'Sources unavailable'],
  ] as const)('renders accessible %s specialist evidence', (outcome, label) => {
    mocks.active = true
    mocks.run = activeRun('running')
    mocks.tools = [specialistTool(specialistResult(outcome))]

    const progress = mountChat().get('[data-testid="copilot-run-progress"]')
    const specialist = progress.get('[data-testid="copilot-specialist-result"]')
    expect(specialist.text()).toContain(label)

    if (outcome === 'complete' || outcome === 'partial') {
      expect(specialist.text()).toContain('Domitian denarius')
      expect(specialist.text()).toContain('Observed Sep 18, 2026')
      expect(specialist.text()).toContain('High confidence')
      expect(specialist.text()).toContain('Verified')
      const source = specialist.get('a')
      expect(source.attributes()).toMatchObject({
        href: 'https://www.cngcoins.com/Coin.aspx?CoinID=400001',
        target: '_blank',
        rel: 'noopener noreferrer',
      })
    } else {
      expect(specialist.find('a').exists()).toBe(false)
    }
  })

  it('rechecks and adapts an eligible dealer listing before invoking the existing wishlist action', async () => {
    mocks.active = true
    mocks.run = activeRun('completed')
    const result = specialistResult('complete')
    mocks.tools = [specialistTool(result)]
    const wrapper = mountChat()

    await wrapper.get('[data-testid="copilot-specialist-result"] button').trigger('click')

    expect(mocks.addToWishlist).toHaveBeenCalledWith({
      name: 'Domitian denarius',
      description: 'Silver denarius with Minerva reverse',
      category: '',
      era: 'Roman Imperial',
      ruler: 'Domitian',
      material: 'Silver',
      denomination: 'Denarius',
      estPrice: 'USD 275',
      imageUrl: '',
      sourceUrl: 'https://www.cngcoins.com/Coin.aspx?CoinID=400001',
      sourceName: 'Classical Numismatic Group',
    }, 'copilot:call_complete:https://www.cngcoins.com/Coin.aspx?CoinID=400001')
  })

  it('rejects a forged ineligible wishlist event from the child component', async () => {
    mocks.active = true
    mocks.run = activeRun('completed')
    const result = specialistResult('complete')
    mocks.tools = [specialistTool(result)]
    const wrapper = mountChat()
    const progress = wrapper.findComponent({ name: 'CopilotRunProgress' })

    progress.vm.$emit(
      'addToWishlist',
      'market_search',
      { ...result.items[0], availability: 'sold' },
      'forged-key',
    )
    await wrapper.vm.$nextTick()

    expect(mocks.addToWishlist).not.toHaveBeenCalled()
  })

  it('renders cited trend direction, sample context, limitations, and safe supporting links', () => {
    mocks.active = true
    mocks.run = activeRun('running')
    const result = priceTrendResult()
    mocks.tools = [{
      ...specialistTool(result),
      toolCallId: 'call_trend',
      toolName: 'price_trends',
    }]

    const specialist = mountChat().get('[data-testid="copilot-specialist-result"]')
    const trend = specialist.get('[data-testid="copilot-price-trend"]')
    expect(trend.text()).toContain('Rising')
    expect(trend.text()).toContain('3 verified sales')
    expect(trend.text()).toContain('May 1, 2026')
    expect(trend.text()).toContain('Sep 1, 2026')
    expect(trend.text()).toContain('USD')
    expect(trend.text()).toContain('Hammer')
    expect(trend.text()).toContain('200')
    expect(trend.text()).toContain('250')
    expect(trend.text()).toContain('300')
    expect(trend.text()).toContain('The sample contains three verified auction sales.')
    expect(specialist.get('a').attributes()).toMatchObject({
      href: result.items[0]?.sourceUrl,
      target: '_blank',
      rel: 'noopener noreferrer',
    })
  })

  it('renders unknown trend limitations without converting incomparable evidence', () => {
    mocks.active = true
    mocks.run = activeRun('running')
    const result = priceTrendResult()
    result.outcome = 'partial'
    result.trend = {
      ...result.trend!,
      state: 'unknown',
      sampleSize: 2,
      low: null,
      median: null,
      high: null,
      limitations: ['Fewer than three comparable verified sales were available.'],
    }
    result.warnings = ['EUR premium-inclusive observations were kept separate.']
    mocks.tools = [{
      ...specialistTool(result),
      toolCallId: 'call_unknown_trend',
      toolName: 'price_trends',
    }]

    const specialist = mountChat().get('[data-testid="copilot-specialist-result"]')
    expect(specialist.text()).toContain('Unknown')
    expect(specialist.text()).toContain('Fewer than three comparable verified sales were available.')
    expect(specialist.text()).toContain('EUR premium-inclusive observations were kept separate.')
    expect(specialist.text()).not.toContain('Range')
  })
})
