import { computed, ref, nextTick, onMounted, onBeforeUnmount, type Ref } from 'vue'
import { useRoute } from 'vue-router'
import { agentChatStream, cancelCollectionProposal, commitCollectionProposal, createCoin, getApiErrorMessage, matchCategoryEra, proxyImage, scrapeImage, uploadImage, saveConversation, getPortfolioSummary, getAgentStatus, createCalendarEvent } from '@/api/client'
import type { CoinMutationPayload, CoinSuggestion, CoinShow, AgentChatAppContext, AgentChatMessage, Category, CollectionChatResponse, Material } from '@/types'
import { useDialog } from '@/composables/useDialog'
import { useCoinOptions } from '@/composables/useCoinOptions'
import { useCoinCopilot } from '@/composables/useCoinCopilot'
import { renderSafeChatMarkdown } from '@/composables/useMarkdown'

type ChatSuggestion = CoinSuggestion | CoinShow

export interface ChatMsg {
  role: 'user' | 'assistant'
  content: string
  suggestions?: ChatSuggestion[]
  collection?: CollectionChatResponse
  streaming?: boolean
  statusText?: string
}

interface UseCoinSearchChatOptions {
  loadConversation?: { id: number; title: string; messages: string } | null
  messagesEl: Ref<HTMLElement | undefined>
  inputBarEl: Ref<{ focus: () => void } | undefined>
  onAdded: () => void
}

interface AgentContextRoute {
  name?: unknown
  params: Record<string, unknown>
  fullPath: string
}

function positiveRouteId(value: unknown): number | undefined {
  const raw = Array.isArray(value) ? value[0] : value
  if (typeof raw !== 'string' || !/^[1-9]\d*$/.test(raw)) return undefined
  const parsed = Number(raw)
  return Number.isSafeInteger(parsed) ? parsed : undefined
}

export function buildAgentChatAppContext(route: AgentContextRoute): AgentChatAppContext {
  const id = positiveRouteId(route.params.id)
  const routeName = typeof route.name === 'string' ? route.name : ''
  return {
    route: route.fullPath.slice(0, 2000),
    activeCoinId: routeName.startsWith('coin-detail') ? id : undefined,
    activeDraftId: routeName === 'quick-capture-draft' ? id : undefined,
  }
}

const VALID_CATEGORIES = ['Roman', 'Greek', 'Byzantine', 'Modern', 'Other']
const VALID_MATERIALS = ['Gold', 'Silver', 'Bronze', 'Copper', 'Electrum', 'Other']
const VALID_ERAS = ['ancient', 'medieval', 'modern'] as const
const FIELD_LIMITS = {
  name: 200,
  denomination: 200,
  ruler: 200,
  notes: 5000,
  referenceUrl: 2000,
  referenceText: 2000,
  referenceCatalog: 32,
  referenceVolume: 64,
  referenceNumber: 128,
  referenceUri: 2000,
} as const
const VOLUME_REQUIRED_REFERENCE_CATALOGS = new Set(['RIC', 'RPC', 'SNG'])


export function normalizeSuggestionEra(value: string): 'ancient' | 'medieval' | 'modern' | '' {
  const normalized = value.trim().toLowerCase()
  if ((VALID_ERAS as readonly string[]).includes(normalized)) {
    return normalized as 'ancient' | 'medieval' | 'modern'
  }
  if (/\b(ancient|roman|greek|hellenistic|republic|imperial|provincial|ptolemaic|seleucid)\b/.test(normalized)) {
    return 'ancient'
  }
  if (/\b(medieval|byzantine)\b/.test(normalized)) {
    return 'medieval'
  }
  if (/\bmodern\b/.test(normalized)) {
    return 'modern'
  }
  return ''
}

/** A category/era candidate the app couldn't confidently resolve, needing a user choice. */
export interface CategoryEraConfirmRequest {
  fieldLabel: string
  suggestedValue: string
  options: string[]
}

/**
 * Resolves a single AI-suggested category/era value against the live
 * admin-defined list: an exact match is used as-is; failing that, the
 * backend's normalized/fuzzy matcher is tried; failing that, the caller is
 * asked to confirm (map to an existing value, or keep it as suggested).
 * Returns null if the user cancels rather than choosing.
 */
async function resolveOneField(
  fieldLabel: 'Category' | 'Era',
  rawValue: string,
  knownOptions: string[],
  requestConfirmation: (request: CategoryEraConfirmRequest) => Promise<string | null>,
): Promise<string | null> {
  const trimmed = rawValue.trim()
  if (!trimmed) return ''
  const exact = knownOptions.find(o => o.toLowerCase() === trimmed.toLowerCase())
  if (exact) return exact
  try {
    const res = await matchCategoryEra(fieldLabel.toLowerCase() as 'category' | 'era', trimmed)
    if (res.data.matched && res.data.match) return res.data.match
  } catch {
    // Network failure - fall through to asking the user rather than guessing.
  }
  return requestConfirmation({ fieldLabel, suggestedValue: trimmed, options: knownOptions })
}

/**
 * Resolves both category and era for an AI coin suggestion before it's
 * saved. Era first tries the existing keyword heuristic (e.g. "Byzantine"
 * implies the medieval era) when that guess is itself one of the known
 * era values; otherwise both fields go through the same exact/fuzzy/confirm
 * pipeline. Returns null if the user cancels a confirmation.
 */
export async function resolveCategoryAndEra(
  coin: CoinSuggestion,
  categoryOptions: string[],
  eraOptions: string[],
  requestConfirmation: (request: CategoryEraConfirmRequest) => Promise<string | null>,
): Promise<{ category: string; era: string } | null> {
  const category = await resolveOneField('Category', coin.category || '', categoryOptions, requestConfirmation)
  if (category === null) return null

  const heuristicEra = normalizeSuggestionEra(coin.era || '')
  const era = heuristicEra && eraOptions.includes(heuristicEra)
    ? heuristicEra
    : await resolveOneField('Era', coin.era || '', eraOptions, requestConfirmation)
  if (era === null) return null

  return { category, era }
}

export function parseSuggestionPrice(price: string): number | null {
  if (!price) return null
  const match = price.match(/[\d,]+(?:\.\d+)?/)
  if (!match) return null
  const parsed = parseFloat(match[0].replace(/,/g, ''))
  return Number.isFinite(parsed) ? parsed : null
}

function limitText(value: unknown, max: number): string {
  const text = typeof value === 'string' ? value : value == null ? '' : String(value)
  return text.trim().slice(0, max)
}

function shouldKeepCandidateReference(ref: { catalog?: string; number?: string; volume?: string }): boolean {
  const catalog = limitText(ref.catalog, FIELD_LIMITS.referenceCatalog).toUpperCase()
  if (!catalog || !ref.number?.trim()) return false
  return !VOLUME_REQUIRED_REFERENCE_CATALOGS.has(catalog) || !!ref.volume?.trim()
}

/**
 * Builds the create-coin payload for an AI-suggested wishlist coin.
 * `resolved` carries category/era values already reconciled against the
 * live admin-defined lists (see resolveCategoryAndEra); when omitted, this
 * falls back to the legacy default-list-only behavior, which is what the
 * unit tests below exercise directly without needing to mock the resolver.
 */
export function buildWishlistCoinPayload(
  coin: CoinSuggestion,
  resolved?: { category?: string; era?: string },
): CoinMutationPayload {
  const category = resolved?.category
    ? (resolved.category as Category)
    : VALID_CATEGORIES.includes(coin.category) ? coin.category as Category : 'Other'
  const material = VALID_MATERIALS.includes(coin.material) ? coin.material as Material : 'Other'
  const candidateReferences = (coin.candidateReferences ?? [])
    .filter(shouldKeepCandidateReference)
    .map((ref) => ({
      catalog: limitText(ref.catalog, FIELD_LIMITS.referenceCatalog),
      volume: limitText(ref.volume, FIELD_LIMITS.referenceVolume),
      number: limitText(ref.number, FIELD_LIMITS.referenceNumber),
      uri: limitText(ref.uri, FIELD_LIMITS.referenceUri),
    }))

  const payload: CoinMutationPayload = {
    name: limitText(coin.name, FIELD_LIMITS.name) || 'Agent coin suggestion',
    category,
    material,
    denomination: limitText(coin.denomination, FIELD_LIMITS.denomination),
    ruler: limitText(coin.ruler, FIELD_LIMITS.ruler),
    era: resolved?.era !== undefined ? resolved.era : normalizeSuggestionEra(coin.era || ''),
    notes: limitText(coin.description, FIELD_LIMITS.notes),
    referenceUrl: limitText(coin.sourceUrl, FIELD_LIMITS.referenceUrl),
    referenceText: limitText(coin.sourceName, FIELD_LIMITS.referenceText),
    isWishlist: true,
    currentValue: parseSuggestionPrice(coin.estPrice),
  }
  if (candidateReferences.length > 0) {
    payload.references = candidateReferences
  }
  return payload
}

export function useCoinSearchChat(options: UseCoinSearchChatOptions) {
  const route = useRoute()
  const { showAlert, showConfirm } = useDialog()
  const { categoryOptions, eraOptions, loadOptions: loadCoinOptions } = useCoinOptions()

  const messages = ref<ChatMsg[]>([])
  const input = ref('')
  const loading = ref(false)
  const addingIdx = ref<string | null>(null)
  const addedSet = ref<Set<string>>(new Set())
  const savedShows = ref<Set<string>>(new Set())
  const savingShow = ref<string | null>(null)
  const conversationId = ref<number | null>(null)
  const saving = ref(false)
  const scrapedImages = ref<Map<string, string>>(new Map())
  const saveLabel = ref('Save')
  const providerConfigured = ref(true)
  const categoryEraConfirmRequest = ref<CategoryEraConfirmRequest | null>(null)
  const copilotCancelling = ref(false)
  const copilotResuming = ref(false)
  let pendingCategoryEraConfirm: ((value: string | null) => void) | null = null
  let saveLabelTimer: ReturnType<typeof setTimeout> | null = null
  let copilotAssistantIdx: number | null = null
  let generation = 0
  let legacyController: AbortController | null = null
  const initializing = ref(true)
  const resetting = ref(false)
  const resolvingMode = ref(false)

  const copilot = useCoinCopilot({
    onRecoveredThread(thread) {
      if (options.loadConversation || messages.value.length > 0) return
      messages.value = thread.messages.map(message => ({
        role: message.role,
        content: message.content,
      }))
      const currentRun = thread.runs.find(candidate => candidate.id === copilot.run.value?.id)
      if (currentRun && !['completed', 'failed', 'cancelled'].includes(currentRun.status)) {
        copilotAssistantIdx = messages.value.length
        messages.value.push({
          role: 'assistant',
          content: '',
          streaming: currentRun.status !== 'paused',
          statusText: currentRun.status === 'paused' ? 'Waiting for your answer' : 'Reconnecting to Coin Copilot...',
        })
      }
      scrollToBottom()
    },
  })

  const newChatDisabled = computed(() =>
    initializing.value || resetting.value || resolvingMode.value || copilot.starting.value ||
    saving.value || addingIdx.value !== null || savingShow.value !== null || copilotCancelling.value)

  async function newChat(): Promise<boolean> {
    if (newChatDisabled.value) return false
    resetting.value = true
    try {
      if (loading.value || copilot.canCancel.value || copilot.run.value?.status === 'cancel_requested') {
        if (!await showConfirm('Cancel the current request and start a new chat?', {
          title: 'New Chat', confirmLabel: 'Cancel and start new', cancelLabel: 'Keep chat',
        })) return false
        if (copilot.canCancel.value && !await copilot.cancel()) return false
      }
      generation += 1
      legacyController?.abort()
      legacyController = null
      copilot.clearRun()
      cancelCategoryEraConfirmation()
      if (saveLabelTimer) clearTimeout(saveLabelTimer)
      saveLabelTimer = null
      copilotAssistantIdx = null
      messages.value = []
      input.value = ''
      loading.value = false
      conversationId.value = null
      saveLabel.value = 'Save'
      addedSet.value = new Set()
      savedShows.value = new Set()
      scrapedImages.value = new Map()
      copilotResuming.value = false
      await nextTick()
      options.inputBarEl.value?.focus()
      return true
    } finally {
      resetting.value = false
    }
  }

  function requestCategoryEraConfirmation(request: CategoryEraConfirmRequest): Promise<string | null> {
    return new Promise((resolve) => {
      pendingCategoryEraConfirm = resolve
      categoryEraConfirmRequest.value = request
    })
  }

  function chooseCategoryEraConfirmation(value: string) {
    categoryEraConfirmRequest.value = null
    pendingCategoryEraConfirm?.(value)
    pendingCategoryEraConfirm = null
  }

  function cancelCategoryEraConfirmation() {
    categoryEraConfirmRequest.value = null
    pendingCategoryEraConfirm?.(null)
    pendingCategoryEraConfirm = null
  }

  function scrollToBottom() {
    nextTick(() => {
      if (options.messagesEl.value) {
        options.messagesEl.value.scrollTop = options.messagesEl.value.scrollHeight
      }
    })
  }

  function buildHistory(): AgentChatMessage[] {
    return messages.value
      .filter(m => m.role === 'user' || m.role === 'assistant')
      .map(m => ({ role: m.role, content: m.content }))
  }

  function buildAppContext(): AgentChatAppContext {
    return buildAgentChatAppContext(route)
  }

  async function sendLegacyMessage(text: string, history: AgentChatMessage[], assistantIdx: number) {
    const currentGeneration = generation
    legacyController = new AbortController()
    await agentChatStream(
      text,
      history,
      (chunk: string) => {
        if (currentGeneration !== generation) return
        const msg = messages.value[assistantIdx]!
        if (msg.statusText) msg.statusText = ''
        msg.content += chunk
        scrollToBottom()
      },
      (message: string, suggestions: CoinSuggestion[], collection?: CollectionChatResponse) => {
        if (currentGeneration !== generation) return
        const msg = messages.value[assistantIdx]!
        msg.content = message
        msg.suggestions = suggestions
        msg.collection = collection
        msg.streaming = false
        msg.statusText = ''
        loading.value = false
        scrollToBottom()
      },
      (error: string) => {
        if (currentGeneration !== generation) return
        const msg = messages.value[assistantIdx]!
        msg.content = error || 'Failed to get a response. Please try again.'
        msg.streaming = false
        msg.statusText = ''
        loading.value = false
        scrollToBottom()
      },
      (status: string) => {
        if (currentGeneration !== generation) return
        const msg = messages.value[assistantIdx]!
        if (!msg.content) {
          msg.statusText = status
          scrollToBottom()
        }
      },
      buildAppContext(),
      legacyController.signal,
    )
  }

  function settleCopilotMessage(result: { status: string; answer?: string; error?: string }) {
    const assistantIdx = copilotAssistantIdx
    if (assistantIdx === null) return
    const msg = messages.value[assistantIdx]
    if (!msg) return

    if (result.status === 'completed') {
      msg.content = result.answer || copilot.run.value?.finalAnswer || 'Coin Copilot completed without a response.'
    } else if (result.status === 'failed') {
      msg.content = result.error || copilot.run.value?.failureMessage || 'Coin Copilot could not complete this request.'
    } else if (result.status === 'cancelled') {
      msg.content = 'Coin Copilot run cancelled.'
    }
    msg.streaming = result.status === 'queued' || result.status === 'running' || result.status === 'cancel_requested'
    msg.statusText = result.status === 'paused' ? 'Waiting for your answer' : ''
    loading.value = Boolean(msg.streaming)
    scrollToBottom()
  }

  async function sendMessage() {
    const text = input.value.trim()
    if (!text || loading.value || resolvingMode.value || resetting.value) return

    const currentGeneration = generation
    resolvingMode.value = true
    const mode = await copilot.resolveCapability()
    resolvingMode.value = false
    if (currentGeneration !== generation) return
    messages.value.push({ role: 'user', content: text })
    const history = buildHistory().slice(0, -1)
    input.value = ''
    loading.value = true
    scrollToBottom()

    const assistantIdx = messages.value.length
    messages.value.push({ role: 'assistant', content: '', streaming: true })
    scrollToBottom()

    if (mode.mode !== 'copilot') {
      await sendLegacyMessage(text, history, assistantIdx)
      return
    }

    copilotAssistantIdx = assistantIdx
    const result = await copilot.start(text, buildAppContext(), copilot.run.value?.threadId)
    if (currentGeneration !== generation) return
    if (!result.accepted && result.fallback) {
      await sendLegacyMessage(text, history, assistantIdx)
      return
    }
    if (!result.accepted) {
      const msg = messages.value[assistantIdx]!
      msg.content = result.error
      msg.streaming = false
      msg.statusText = ''
      loading.value = false
      scrollToBottom()
      return
    }
    settleCopilotMessage(result)
  }

  async function cancelCopilotRun() {
    if (!copilot.canCancel.value || copilotCancelling.value) return
    copilotCancelling.value = true
    try {
      const cancelled = await copilot.cancel()
      if (cancelled && copilot.run.value?.status === 'cancelled') {
        settleCopilotMessage({ status: 'cancelled' })
      }
    } finally {
      copilotCancelling.value = false
    }
  }

  async function resumeCopilotRun(answer: string) {
    if (!copilot.canResume.value || copilotResuming.value) return
    copilotResuming.value = true
    const currentGeneration = generation
    loading.value = true
    const assistantIdx = copilotAssistantIdx
    if (assistantIdx !== null && messages.value[assistantIdx]) {
      messages.value[assistantIdx]!.streaming = true
      messages.value[assistantIdx]!.statusText = 'Resuming Coin Copilot...'
    }
    try {
      const resumed = await copilot.resume(answer)
      if (currentGeneration !== generation) return
      if (resumed && copilot.run.value) {
        settleCopilotMessage({
          status: copilot.run.value.status,
          answer: copilot.run.value.finalAnswer ?? undefined,
          error: copilot.run.value.failureMessage ?? (copilot.error.value || undefined),
        })
      } else {
        loading.value = false
        if (assistantIdx !== null && messages.value[assistantIdx]) {
          messages.value[assistantIdx]!.streaming = false
          messages.value[assistantIdx]!.statusText = 'Waiting for your answer'
        }
      }
    } finally {
      if (currentGeneration === generation) copilotResuming.value = false
    }
  }

  function sendExample(text: string) {
    input.value = text
    sendMessage()
  }

  async function sendPortfolioAnalysis() {
    const currentGeneration = generation
    try {
      const res = await getPortfolioSummary()
      if (currentGeneration !== generation) return
      const summary = res.data
      const missingProperties = Object.entries(summary.missingFields ?? {})
      const context = `Analyze my coin collection portfolio. Here is my collection summary:\n\n` +
        `Total Coins: ${summary.totalCoins ?? 0}\n` +
        `Total Value: $${summary.totalValue?.toFixed(2) ?? '0'}\n` +
        `Total Invested: $${summary.totalInvested?.toFixed(2) ?? '0'}\n` +
        `Categories: ${summary.categories?.map((c) => `${c.category} (${c.count})`).join(', ') || 'none'}\n` +
        `Materials: ${summary.materials?.map((m) => `${m.material} (${m.count})`).join(', ') || 'none'}\n` +
        `Eras: ${summary.eras?.map((e) => `${e.era} (${e.count})`).join(', ') || 'none'}\n` +
        `Top Rulers: ${summary.rulers?.map((r) => `${r.ruler} (${r.count})`).join(', ') || 'none'}\n` +
        `Missing Properties: ${missingProperties.length ? missingProperties.map(([field, count]) => `${field} (${count})`).join(', ') : 'none'}\n` +
        `Top Coins by Value: ${summary.topCoins?.map((c) => `${c.name} ($${c.currentValue?.toFixed(2) ?? '?'})`).join(', ') || 'none'}\n\n` +
        `Please analyze my collection, identify gaps, and suggest what I should consider adding.`
      input.value = context
      sendMessage()
    } catch {
      if (currentGeneration !== generation) return
      input.value = 'Analyze my coin collection portfolio and suggest areas for improvement.'
      sendMessage()
    }
  }

  async function handleSave() {
    if (messages.value.length === 0 || saving.value) return
    saving.value = true
    saveLabel.value = 'Saving...'

    try {
      const firstUserMsg = messages.value.find(m => m.role === 'user')
      const title = firstUserMsg?.content.substring(0, 100) || 'Untitled conversation'

      const res = await saveConversation({
        id: conversationId.value || undefined,
        title,
        messages: JSON.stringify(messages.value),
      })
      conversationId.value = res.data.id
      saveLabel.value = 'Saved!'
      if (saveLabelTimer) clearTimeout(saveLabelTimer)
      saveLabelTimer = setTimeout(() => { saveLabel.value = 'Save' }, 2000)
    } catch {
      saveLabel.value = 'Failed'
      if (saveLabelTimer) clearTimeout(saveLabelTimer)
      saveLabelTimer = setTimeout(() => { saveLabel.value = 'Save' }, 2000)
    } finally {
      saving.value = false
    }
  }

  async function addToWishlist(coin: CoinSuggestion, idx: string) {
    if (addingIdx.value !== null || addedSet.value.has(idx)) return
    addingIdx.value = idx
    try {
      const resolved = await resolveCategoryAndEra(
        coin,
        categoryOptions.value,
        eraOptions.value,
        requestCategoryEraConfirmation,
      )
      if (resolved === null) {
        // User cancelled the category/era confirmation - don't create the coin.
        return
      }
      const created = await createCoin(buildWishlistCoinPayload(coin, resolved))

      let imageAttached = false

      if (coin.sourceUrl) {
        try {
          let scrapedUrl = scrapedImages.value.get(coin.sourceUrl) || ''
          if (!scrapedUrl) {
            const scraped = await scrapeImage(coin.sourceUrl)
            scrapedUrl = scraped.data.imageUrl || ''
          }
          if (scrapedUrl) {
            // downloading scraped image
            const imgRes = await proxyImage(scrapedUrl)
            const blob = imgRes.data as Blob
            if (blob.size > 0) {
              const ext = blob.type.includes('png') ? '.png' : '.jpg'
              const file = new File([blob], `obverse${ext}`, { type: blob.type || 'image/jpeg' })
              await uploadImage(created.data.id, file, 'obverse', true)
              imageAttached = true
              // image attached via scraping
            }
          }
        } catch (err) {
          console.warn('[agent] Scrape-based image failed for', coin.sourceUrl, err)
        }
      }

      if (!imageAttached && coin.imageUrl) {
        try {
          // trying agent imageUrl
          const imgRes = await proxyImage(coin.imageUrl)
          const blob = imgRes.data as Blob
          if (blob.size > 0) {
            const ext = blob.type.includes('png') ? '.png' : '.jpg'
            const file = new File([blob], `obverse${ext}`, { type: blob.type || 'image/jpeg' })
            await uploadImage(created.data.id, file, 'obverse', true)
            imageAttached = true
            // image attached via agent imageUrl
          }
        } catch (err) {
          console.warn('[agent] Agent imageUrl download failed:', coin.imageUrl, err)
        }
      }

      if (!imageAttached) {
        console.warn('[agent] No image could be attached for coin:', coin.name)
      }

      addedSet.value.add(idx)
      options.onAdded()
    } catch (err) {
      const detail = getApiErrorMessage(err)
      await showAlert(detail ? `Failed to add coin to wishlist: ${detail}` : 'Failed to add coin to wishlist', { title: 'Error' })
    } finally {
      addingIdx.value = null
    }
  }

  function formatMessage(text: string): string {
    return renderSafeChatMarkdown(text)
  }

  function isCoinShowResults(suggestions: ChatSuggestion[]): boolean {
    if (!suggestions?.length) return false
    const first = suggestions[0]!
    return 'dates' in first || 'venue' in first
  }

  function showKey(show: CoinShow): string {
    return `${show.name}|${show.dates}`
  }

  function parseDateRange(dateStr: string): { start?: string; end?: string } {
    if (!dateStr) return {}

    const isoMatch = dateStr.match(/(\d{4}-\d{2}-\d{2})/)
    if (isoMatch) {
      return { start: isoMatch[1]! + 'T00:00:00Z' }
    }

    const rangeMatch = dateStr.match(/([A-Z][a-z]+)\s+(\d{1,2})\s*[-–]\s*(\d{1,2}),?\s*(\d{4})/)
    if (rangeMatch) {
      const [, month, startDay, endDay, year] = rangeMatch
      const s = new Date(`${month} ${startDay}, ${year}`)
      const e = new Date(`${month} ${endDay}, ${year}`)
      if (!isNaN(s.getTime())) {
        return {
          start: s.toISOString().split('T')[0]! + 'T00:00:00Z',
          end: !isNaN(e.getTime()) ? e.toISOString().split('T')[0]! + 'T00:00:00Z' : undefined,
        }
      }
    }

    const crossMonthMatch = dateStr.match(/([A-Z][a-z]+)\s+(\d{1,2})\s*[-–]\s*([A-Z][a-z]+)\s+(\d{1,2}),?\s*(\d{4})/)
    if (crossMonthMatch) {
      const [, month1, day1, month2, day2, year] = crossMonthMatch
      const s = new Date(`${month1} ${day1}, ${year}`)
      const e = new Date(`${month2} ${day2}, ${year}`)
      if (!isNaN(s.getTime())) {
        return {
          start: s.toISOString().split('T')[0]! + 'T00:00:00Z',
          end: !isNaN(e.getTime()) ? e.toISOString().split('T')[0]! + 'T00:00:00Z' : undefined,
        }
      }
    }

    const singleMatch = dateStr.match(/([A-Z][a-z]+)\s+(\d{1,2}),?\s*(\d{4})/)
    if (singleMatch) {
      const d = new Date(`${singleMatch[1]} ${singleMatch[2]}, ${singleMatch[3]}`)
      if (!isNaN(d.getTime())) {
        return { start: d.toISOString().split('T')[0]! + 'T00:00:00Z' }
      }
    }

    const d = new Date(dateStr)
    if (!isNaN(d.getTime())) {
      return { start: d.toISOString().split('T')[0]! + 'T00:00:00Z' }
    }
    return {}
  }

  async function saveShowToCalendar(show: CoinShow) {
    const key = showKey(show)
    if (savedShows.value.has(key)) return
    savingShow.value = key
    try {
      const { start, end } = parseDateRange(show.dates)
      const location = [show.venue, show.location].filter(Boolean).join(', ')
      await createCalendarEvent({
        title: show.name,
        startDate: start,
        endDate: end,
        url: show.url || undefined,
        notes: [location, show.entryFee ? `Entry: ${show.entryFee}` : '', show.description].filter(Boolean).join('\n'),
      })
      savedShows.value.add(key)
    } catch {
      await showAlert('Failed to save event to calendar')
    } finally {
      savingShow.value = null
    }
  }

  async function confirmCollectionProposal(msg: ChatMsg) {
    const proposal = msg.collection?.proposal
    if (!proposal) return
    const currentGeneration = generation

    try {
      const res = await commitCollectionProposal(proposal.proposalId, proposal.proposalToken)
      if (currentGeneration !== generation) return
      messages.value.push({
        role: 'assistant',
        content: res.data?.message || 'Update committed.',
      })
      msg.collection = {
        kind: 'read_result',
        message: 'Proposal committed.',
      }
      scrollToBottom()
    } catch {
      if (currentGeneration !== generation) return
      await showAlert('Failed to commit collection update proposal.', { title: 'Error' })
    }
  }

  async function cancelCollectionProposalMessage(msg: ChatMsg) {
    const proposal = msg.collection?.proposal
    if (!proposal) return
    const currentGeneration = generation

    try {
      const res = await cancelCollectionProposal(proposal.proposalId)
      if (currentGeneration !== generation) return
      messages.value.push({
        role: 'assistant',
        content: res.data?.message || 'Proposal cancelled.',
      })
      msg.collection = {
        kind: 'read_result',
        message: 'Proposal cancelled.',
      }
      scrollToBottom()
    } catch {
      if (currentGeneration !== generation) return
      await showAlert('Failed to cancel collection update proposal.', { title: 'Error' })
    }
  }

  function pickDisambiguationCandidate(coinId: number) {
    if (loading.value) return
    input.value = `Use coin #${coinId} for that update.`
    sendMessage()
  }

  function handleViewportResize() {
    const overlay = document.querySelector('.chat-overlay') as HTMLElement | null
    if (!overlay || !window.visualViewport) return
    const vv = window.visualViewport
    overlay.style.height = `${vv.height}px`
    overlay.style.top = `${vv.offsetTop}px`
  }

  onMounted(async () => {
    const currentGeneration = generation
    options.inputBarEl.value?.focus()
    loadCoinOptions()
    if (options.loadConversation) {
      conversationId.value = options.loadConversation.id
      try {
        messages.value = JSON.parse(options.loadConversation.messages)
        scrollToBottom()
      } catch { /* ignore parse errors */ }
    }
    try {
      const res = await getAgentStatus()
      providerConfigured.value = res.data.configured
    } catch {
      providerConfigured.value = true
    }
    await copilot.resolveCapability()
    if (currentGeneration !== generation) return
    if (!options.loadConversation) {
      await copilot.restoreActiveRun()
    }
    if (currentGeneration !== generation) return
    initializing.value = false
    if (window.visualViewport) {
      window.visualViewport.addEventListener('resize', handleViewportResize)
      window.visualViewport.addEventListener('scroll', handleViewportResize)
    }
  })

  onBeforeUnmount(() => {
    generation += 1
    legacyController?.abort()
    if (window.visualViewport) {
      window.visualViewport.removeEventListener('resize', handleViewportResize)
      window.visualViewport.removeEventListener('scroll', handleViewportResize)
    }
    if (saveLabelTimer) clearTimeout(saveLabelTimer)
    copilot.disconnect()
  })

  return {
    messages,
    input,
    loading,
    addingIdx,
    addedSet,
    savedShows,
    savingShow,
    conversationId,
    saving,
    saveLabel,
    providerConfigured,
    categoryEraConfirmRequest,
    copilotActive: copilot.active,
    copilotRun: copilot.run,
    copilotPlan: copilot.plan,
    copilotTools: copilot.tools,
    copilotClarification: copilot.clarification,
    copilotCanCancel: copilot.canCancel,
    copilotCanResume: copilot.canResume,
    copilotTruncated: copilot.truncated,
    copilotError: copilot.error,
    copilotCancelling,
    copilotResuming,
    newChat,
    newChatDisabled,
    chooseCategoryEraConfirmation,
    cancelCategoryEraConfirmation,
    sendMessage,
    cancelCopilotRun,
    resumeCopilotRun,
    sendExample,
    sendPortfolioAnalysis,
    handleSave,
    addToWishlist,
    confirmCollectionProposal,
    cancelCollectionProposalMessage,
    pickDisambiguationCandidate,
    formatMessage,
    isCoinShowResults,
    saveShowToCalendar,
  }
}
