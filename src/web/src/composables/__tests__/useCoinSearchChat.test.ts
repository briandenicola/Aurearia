import { flushPromises, mount } from '@vue/test-utils'
import { defineComponent, ref } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { CoinSuggestion } from '@/types'
import {
  buildAgentChatAppContext,
  buildWishlistCoinPayload,
  normalizeSuggestionEra,
  resolveCategoryAndEra,
  useCoinSearchChat,
} from '../useCoinSearchChat'

const mockMatchCategoryEra = vi.fn()
const mockAgentChatStream = vi.fn()
const mockGetCoinCopilotCapability = vi.fn()
const mockCreateCoin = vi.fn()
const mockScrapeImage = vi.fn()
const mockProxyImage = vi.fn()
const mockUploadImage = vi.fn()
const mockShowAlert = vi.fn()
const mockGetApiErrorMessage = vi.fn()
vi.mock('@/api/client', () => ({
  matchCategoryEra: (type: 'category' | 'era', value: string) => mockMatchCategoryEra(type, value),
  getCoinCopilotCapability: () => mockGetCoinCopilotCapability(),
  getAgentStatus: vi.fn(async () => ({ data: { configured: true } })),
  agentChatStream: (...args: unknown[]) => mockAgentChatStream(...args),
  createCoin: (...args: unknown[]) => mockCreateCoin(...args),
  scrapeImage: (...args: unknown[]) => mockScrapeImage(...args),
  proxyImage: (...args: unknown[]) => mockProxyImage(...args),
  uploadImage: (...args: unknown[]) => mockUploadImage(...args),
  getApiErrorMessage: (...args: unknown[]) => mockGetApiErrorMessage(...args),
}))

vi.mock('vue-router', () => ({
  useRoute: () => ({ params: {}, fullPath: '/collection' }),
}))

vi.mock('@/composables/useDialog', () => ({
  useDialog: () => ({ showAlert: mockShowAlert }),
}))

vi.mock('@/composables/useCoinOptions', () => ({
  useCoinOptions: () => ({
    categoryOptions: ref(['Roman', 'Greek', 'Byzantine', 'Modern', 'Other']),
    eraOptions: ref(['ancient', 'medieval', 'modern']),
    loadOptions: vi.fn(),
  }),
}))

function makeSuggestion(overrides: Partial<CoinSuggestion> = {}): CoinSuggestion {
  return {
    name: 'Trajan Denarius',
    description: 'Silver denarius of Trajan',
    category: 'Roman',
    era: 'ancient',
    ruler: 'Trajan',
    material: 'Silver',
    denomination: 'Denarius',
    estPrice: '$125',
    imageUrl: '',
    sourceUrl: 'https://example.com/coin',
    sourceName: 'Example Dealer',
    ...overrides,
  }
}

function mountChat() {
  let chat: ReturnType<typeof useCoinSearchChat> | undefined
  const onAdded = vi.fn()
  const Host = defineComponent({
    setup() {
      chat = useCoinSearchChat({
        messagesEl: ref(),
        inputBarEl: ref(),
        onAdded,
      })
      return () => null
    },
  })
  mount(Host)
  return { chat: chat!, onAdded }
}

describe('useCoinSearchChat wishlist payload', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockGetCoinCopilotCapability.mockResolvedValue({
      data: {
        mode: 'legacy',
        enabled: true,
        modelToolCallingSupported: false,
        reason: 'model_tool_calling_unsupported',
      },
    })
    mockCreateCoin.mockResolvedValue({ data: { id: 42 } })
    mockScrapeImage.mockResolvedValue({ data: { imageUrl: '' } })
    mockGetApiErrorMessage.mockReturnValue('')
  })

  it('emits authoritative ids only for their exact detail routes', () => {
    expect(buildAgentChatAppContext({
      name: 'quick-capture-draft',
      params: { id: '42' },
      fullPath: '/quick-capture/drafts/42',
    })).toEqual({
      route: '/quick-capture/drafts/42',
      activeCoinId: undefined,
      activeDraftId: 42,
    })
    expect(buildAgentChatAppContext({
      name: 'coin-detail',
      params: { id: '7' },
      fullPath: '/coin/7',
    })).toEqual({
      route: '/coin/7',
      activeCoinId: 7,
      activeDraftId: undefined,
    })
  })

  it('does not infer a target from unrelated routes, invalid ids, or prompt-like route text', () => {
    for (const route of [
      { name: 'quick-capture-drafts', params: { id: '42' }, fullPath: '/quick-capture/drafts' },
      { name: 'quick-capture-draft', params: { id: '0' }, fullPath: '/quick-capture/drafts/0' },
      { name: 'collection', params: { id: '42' }, fullPath: '/collection?prompt=use%20draft%2042' },
    ]) {
      const context = buildAgentChatAppContext(route)
      expect(context.activeCoinId).toBeUndefined()
      expect(context.activeDraftId).toBeUndefined()
    }
  })

  it('normalizes AI era labels that the coin API would reject', () => {
    expect(normalizeSuggestionEra('Roman Imperial')).toBe('ancient')
    expect(normalizeSuggestionEra('Byzantine')).toBe('medieval')
    expect(normalizeSuggestionEra('Modern commemorative')).toBe('modern')
    expect(normalizeSuggestionEra('Unknown period')).toBe('')
  })

  describe('useCoinSearchChat legacy fallback', () => {
    beforeEach(() => {
      vi.clearAllMocks()
      mockGetCoinCopilotCapability.mockResolvedValue({
        data: {
          mode: 'legacy',
          enabled: true,
          modelToolCallingSupported: false,
          reason: 'model_tool_calling_unsupported',
        },
      })
      mockAgentChatStream.mockImplementation(async (
        _message: string,
        _history: unknown[],
        _onText: unknown,
        onDone: (message: string, suggestions: unknown[]) => void,
      ) => {
        onDone('Legacy response', [])
      })
    })

    it('uses the unchanged legacy stream when the feature/model capability falls back', async () => {
      let chat: ReturnType<typeof useCoinSearchChat> | undefined
      const Host = defineComponent({
        setup() {
          chat = useCoinSearchChat({
            messagesEl: ref(),
            inputBarEl: ref(),
            onAdded: vi.fn(),
          })
          return () => null
        },
      })
      mount(Host)
      await flushPromises()
      chat!.input.value = 'Find a Roman coin'

      await chat!.sendMessage()

      expect(mockAgentChatStream).toHaveBeenCalledTimes(1)
      expect(chat!.messages.value.at(-1)?.content).toBe('Legacy response')
    })
  })

  it('builds a create-coin payload with only supported era values', () => {
    const payload = buildWishlistCoinPayload(makeSuggestion({
      era: 'Roman Imperial',
      category: 'Unclassified',
      material: 'Billon',
      estPrice: 'Estimate $1,250.50',
      candidateReferences: [
        { catalog: ' RIC ', number: ' 123 ', volume: ' II ', uri: ' https://example.com/ric ' },
        { catalog: 'RIC', number: '456' },
        { catalog: 'RPC', number: '   ' },
        { catalog: 'SNG', number: '789', volume: '' },
        { catalog: 'SEAR', number: '101' },
      ],
    }))

    expect(payload).toMatchObject({
      name: 'Trajan Denarius',
      category: 'Other',
      material: 'Other',
      era: 'ancient',
      isWishlist: true,
      currentValue: 1250.5,
      references: [
        {
          catalog: 'RIC',
          number: '123',
          volume: 'II',
          uri: 'https://example.com/ric',
        },
        {
          catalog: 'SEAR',
          number: '101',
          volume: '',
          uri: '',
        },
      ],
    })
  })

  it('does not require or send catalog references for wishlist suggestions', () => {
    const payload = buildWishlistCoinPayload(makeSuggestion({
      candidateReferences: [
        { catalog: '', number: '123' },
        { catalog: ' RIC ', number: '456', volume: '' },
        { catalog: 'SEAR', number: '' },
      ],
    }))

    expect(payload).toMatchObject({
      name: 'Trajan Denarius',
      isWishlist: true,
    })
    expect(payload.references).toBeUndefined()
  })

  it('truncates agent text fields to backend create-coin limits', () => {
    const payload = buildWishlistCoinPayload(makeSuggestion({
      name: 'A'.repeat(250),
      description: 'B'.repeat(6000),
      denomination: 'C'.repeat(250),
      ruler: 'D'.repeat(250),
      sourceUrl: `https://example.com/${'e'.repeat(2100)}`,
      sourceName: 'F'.repeat(2100),
    }))

    expect(payload.name).toHaveLength(200)
    expect(payload.notes).toHaveLength(5000)
    expect(payload.denomination).toHaveLength(200)
    expect(payload.ruler).toHaveLength(200)
    expect(payload.referenceUrl).toHaveLength(2000)
    expect(payload.referenceText).toHaveLength(2000)
  })

  it('creates one allowlisted wishlist coin and ignores a repeated action', async () => {
    const { chat, onAdded } = mountChat()
    const suggestion = makeSuggestion()

    await chat.addToWishlist(suggestion, 'dealer-1')
    await chat.addToWishlist(suggestion, 'dealer-1')

    expect(mockCreateCoin).toHaveBeenCalledTimes(1)
    expect(mockCreateCoin).toHaveBeenCalledWith(expect.objectContaining({
      name: 'Trajan Denarius',
      denomination: 'Denarius',
      ruler: 'Trajan',
      notes: 'Silver denarius of Trajan',
      referenceUrl: 'https://example.com/coin',
      referenceText: 'Example Dealer',
      isWishlist: true,
      currentValue: 125,
    }))
    const payload = mockCreateCoin.mock.calls[0]?.[0] as Record<string, unknown>
    for (const forbidden of [
      'purchasePrice', 'purchaseDate', 'purchaseLocation', 'vendorSku', 'vendorInvoice',
      'storageLocationId', 'storageSlot', 'isSold', 'soldPrice', 'soldDate', 'soldTo',
      'aiAnalysis', 'obverseAnalysis', 'reverseAnalysis',
    ]) {
      expect(payload).not.toHaveProperty(forbidden)
    }
    expect(mockScrapeImage).toHaveBeenCalledTimes(1)
    expect(onAdded).toHaveBeenCalledTimes(1)
    expect(chat.addedSet.value.has('dealer-1')).toBe(true)
  })

  it('blocks a repeated click while the first create is still pending', async () => {
    let finishCreate!: (value: { data: { id: number } }) => void
    mockCreateCoin.mockReturnValue(new Promise(resolve => {
      finishCreate = resolve
    }))
    const { chat } = mountChat()
    const suggestion = makeSuggestion()

    const first = chat.addToWishlist(suggestion, 'dealer-pending')
    const repeated = chat.addToWishlist(suggestion, 'dealer-pending')
    await flushPromises()
    expect(mockCreateCoin).toHaveBeenCalledTimes(1)

    finishCreate({ data: { id: 42 } })
    await Promise.all([first, repeated])
  })

  it('does not create when category or era confirmation is cancelled', async () => {
    mockMatchCategoryEra.mockResolvedValue({ data: { matched: false, match: '' } })
    const { chat } = mountChat()

    const pending = chat.addToWishlist(makeSuggestion({ category: 'Gaulish' }), 'dealer-cancel')
    await flushPromises()
    expect(chat.categoryEraConfirmRequest.value?.fieldLabel).toBe('Category')
    chat.cancelCategoryEraConfirmation()
    await pending

    expect(mockCreateCoin).not.toHaveBeenCalled()
  })

  it('keeps the created wishlist coin when best-effort image attachment fails', async () => {
    vi.spyOn(console, 'warn').mockImplementation(() => {})
    mockScrapeImage.mockRejectedValue(new Error('image unavailable'))
    mockProxyImage.mockRejectedValue(new Error('image unavailable'))
    const { chat, onAdded } = mountChat()

    await chat.addToWishlist(makeSuggestion({ imageUrl: 'https://example.com/coin.jpg' }), 'dealer-image')

    expect(mockCreateCoin).toHaveBeenCalledTimes(1)
    expect(mockUploadImage).not.toHaveBeenCalled()
    expect(onAdded).toHaveBeenCalledTimes(1)
    expect(chat.addedSet.value.has('dealer-image')).toBe(true)
  })

  it('shows duplicate feedback and leaves the listing available for retry', async () => {
    mockCreateCoin.mockRejectedValue({
      response: { status: 409, data: { code: 'wishlist_duplicate' } },
    })
    mockGetApiErrorMessage.mockReturnValue('A wishlist coin already uses this source URL')
    const { chat, onAdded } = mountChat()

    await chat.addToWishlist(makeSuggestion(), 'dealer-duplicate')

    expect(mockCreateCoin).toHaveBeenCalledTimes(1)
    expect(mockShowAlert).toHaveBeenCalledWith(
      'Failed to add coin to wishlist: A wishlist coin already uses this source URL',
      { title: 'Error' },
    )
    expect(onAdded).not.toHaveBeenCalled()
    expect(chat.addedSet.value.has('dealer-duplicate')).toBe(false)
  })
})

describe('resolveCategoryAndEra', () => {
  const categoryOptions = ['Roman', 'Greek', 'Byzantine', 'Modern', 'Other', 'Celtic']
  const eraOptions = ['ancient', 'medieval', 'modern']

  it('resolves an exact match without calling the backend or asking the user', async () => {
    const requestConfirmation = vi.fn()
    const result = await resolveCategoryAndEra(
      makeSuggestion({ category: 'roman', era: 'ancient' }),
      categoryOptions,
      eraOptions,
      requestConfirmation,
    )
    expect(result).toEqual({ category: 'Roman', era: 'ancient' })
    expect(mockMatchCategoryEra).not.toHaveBeenCalled()
    expect(requestConfirmation).not.toHaveBeenCalled()
  })

  it('short-circuits era via the existing keyword heuristic when it lands on a known era', async () => {
    const requestConfirmation = vi.fn()
    mockMatchCategoryEra.mockResolvedValue({ data: { matched: false, match: '' } })
    const result = await resolveCategoryAndEra(
      makeSuggestion({ category: 'Roman', era: 'Byzantine' }),
      categoryOptions,
      eraOptions,
      requestConfirmation,
    )
    expect(result).toEqual({ category: 'Roman', era: 'medieval' })
    expect(mockMatchCategoryEra).not.toHaveBeenCalledWith('era', expect.anything())
  })

  it('falls back to the backend fuzzy matcher for a near-miss category', async () => {
    mockMatchCategoryEra.mockResolvedValue({ data: { matched: true, match: 'Roman' } })
    const requestConfirmation = vi.fn()
    const result = await resolveCategoryAndEra(
      makeSuggestion({ category: 'Roman Republic', era: 'ancient' }),
      categoryOptions,
      eraOptions,
      requestConfirmation,
    )
    expect(result).toEqual({ category: 'Roman', era: 'ancient' })
    expect(mockMatchCategoryEra).toHaveBeenCalledWith('category', 'Roman Republic')
    expect(requestConfirmation).not.toHaveBeenCalled()
  })

  it('asks the user to confirm when nothing matches confidently', async () => {
    mockMatchCategoryEra.mockResolvedValue({ data: { matched: false, match: '' } })
    const requestConfirmation = vi.fn().mockResolvedValue('Celtic')
    const result = await resolveCategoryAndEra(
      makeSuggestion({ category: 'Gaulish', era: 'ancient' }),
      categoryOptions,
      eraOptions,
      requestConfirmation,
    )
    expect(result).toEqual({ category: 'Celtic', era: 'ancient' })
    expect(requestConfirmation).toHaveBeenCalledWith({
      fieldLabel: 'Category',
      suggestedValue: 'Gaulish',
      options: categoryOptions,
    })
  })

  it('returns null (never creates the coin) if the user cancels confirmation', async () => {
    mockMatchCategoryEra.mockResolvedValue({ data: { matched: false, match: '' } })
    const requestConfirmation = vi.fn().mockResolvedValue(null)
    const result = await resolveCategoryAndEra(
      makeSuggestion({ category: 'Gaulish', era: 'ancient' }),
      categoryOptions,
      eraOptions,
      requestConfirmation,
    )
    expect(result).toBeNull()
  })
})
