import { ref, nextTick, onMounted, onBeforeUnmount, type Ref } from 'vue'
import { useRoute } from 'vue-router'
import { agentChatStream, cancelCollectionProposal, commitCollectionProposal, createCoin, proxyImage, scrapeImage, uploadImage, saveConversation, getPortfolioSummary, getAgentStatus, createCalendarEvent } from '@/api/client'
import type { CoinSuggestion, CoinShow, AgentChatAppContext, AgentChatMessage, Category, CollectionChatResponse, Material } from '@/types'
import { useDialog } from '@/composables/useDialog'
import DOMPurify from 'dompurify'
import MarkdownIt from 'markdown-it'

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

const VALID_CATEGORIES = ['Roman', 'Greek', 'Byzantine', 'Modern', 'Other']
const VALID_MATERIALS = ['Gold', 'Silver', 'Bronze', 'Copper', 'Electrum', 'Other']

const md = new MarkdownIt({ html: false, linkify: true, breaks: true })

export function useCoinSearchChat(options: UseCoinSearchChatOptions) {
  const route = useRoute()
  const { showAlert } = useDialog()

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
  let saveLabelTimer: ReturnType<typeof setTimeout> | null = null

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
    const idParam = route.params.id
    const activeCoinId = typeof idParam === 'string'
      ? Number.parseInt(idParam, 10)
      : Array.isArray(idParam) && typeof idParam[0] === 'string'
        ? Number.parseInt(idParam[0], 10)
        : undefined

    return {
      route: route.fullPath,
      activeCoinId: Number.isFinite(activeCoinId ?? NaN) ? activeCoinId : undefined,
    }
  }

  async function sendMessage() {
    const text = input.value.trim()
    if (!text || loading.value) return

    messages.value.push({ role: 'user', content: text })
    const history = buildHistory().slice(0, -1)
    input.value = ''
    loading.value = true
    scrollToBottom()

    const assistantIdx = messages.value.length
    messages.value.push({ role: 'assistant', content: '', streaming: true })
    scrollToBottom()

    await agentChatStream(
      text,
      history,
      (chunk: string) => {
        const msg = messages.value[assistantIdx]!
        if (msg.statusText) msg.statusText = ''
        msg.content += chunk
        scrollToBottom()
      },
      (message: string, suggestions: CoinSuggestion[], collection?: CollectionChatResponse) => {
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
        const msg = messages.value[assistantIdx]!
        msg.content = error || 'Failed to get a response. Please try again.'
        msg.streaming = false
        msg.statusText = ''
        loading.value = false
        scrollToBottom()
      },
      (status: string) => {
        const msg = messages.value[assistantIdx]!
        if (!msg.content) {
          msg.statusText = status
          scrollToBottom()
        }
      },
      buildAppContext(),
    )
  }

  function sendExample(text: string) {
    input.value = text
    sendMessage()
  }

  async function sendPortfolioAnalysis() {
    try {
      const res = await getPortfolioSummary()
      const summary = res.data
      const context = `Analyze my coin collection portfolio. Here is my collection summary:\n\n` +
        `Total Coins: ${summary.totalCoins ?? 0}\n` +
        `Total Value: $${summary.totalValue?.toFixed(2) ?? '0'}\n` +
        `Total Invested: $${summary.totalInvested?.toFixed(2) ?? '0'}\n` +
        `Categories: ${summary.categories?.map((c) => `${c.category} (${c.count})`).join(', ') || 'none'}\n` +
        `Materials: ${summary.materials?.map((m) => `${m.material} (${m.count})`).join(', ') || 'none'}\n` +
        `Eras: ${summary.eras?.map((e) => `${e.era} (${e.count})`).join(', ') || 'none'}\n` +
        `Top Rulers: ${summary.rulers?.map((r) => `${r.ruler} (${r.count})`).join(', ') || 'none'}\n` +
        `Top Coins by Value: ${summary.topCoins?.map((c) => `${c.name} ($${c.currentValue?.toFixed(2) ?? '?'})`).join(', ') || 'none'}\n\n` +
        `Please analyze my collection, identify gaps, and suggest what I should consider adding.`
      input.value = context
      sendMessage()
    } catch {
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
    if (addedSet.value.has(idx)) return
    addingIdx.value = idx
    try {
      const category = VALID_CATEGORIES.includes(coin.category) ? coin.category as Category : 'Other'
      const material = VALID_MATERIALS.includes(coin.material) ? coin.material as Material : 'Other'
      const candidateReferences = (coin.candidateReferences ?? [])
        .filter((ref) => !!ref.catalog?.trim() && !!ref.number?.trim())
        .map((ref) => ({
          catalog: ref.catalog.trim(),
          volume: ref.volume?.trim() || '',
          number: ref.number.trim(),
          certainty: ref.certainty?.trim() || '',
          uri: ref.uri?.trim() || '',
        }))

      const created = await createCoin({
        name: coin.name,
        category,
        material,
        denomination: coin.denomination || '',
        ruler: coin.ruler || '',
        era: coin.era || '',
        notes: coin.description || '',
        referenceUrl: coin.sourceUrl || '',
        referenceText: coin.sourceName || '',
        isWishlist: true,
        currentValue: parsePrice(coin.estPrice),
        references: candidateReferences,
      })

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
    } catch {
      await showAlert('Failed to add coin to wishlist', { title: 'Error' })
    } finally {
      addingIdx.value = null
    }
  }

  function parsePrice(price: string): number | null {
    if (!price) return null
    const match = price.match(/[\d,]+(?:\.\d+)?/)
    if (!match) return null
    return parseFloat(match[0].replace(/,/g, ''))
  }

  function formatMessage(text: string): string {
    if (!text) return ''
    const html = md.render(text)
    return DOMPurify.sanitize(html, {
      ALLOWED_TAGS: ['strong', 'em', 'br', 'p', 'ul', 'ol', 'li', 'a', 'h1', 'h2', 'h3', 'h4', 'code', 'pre', 'blockquote', 'hr'],
      ALLOWED_ATTR: ['href', 'target', 'rel'],
    })
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

    try {
      const res = await commitCollectionProposal(proposal.proposalId, proposal.proposalToken)
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
      await showAlert('Failed to commit collection update proposal.', { title: 'Error' })
    }
  }

  async function cancelCollectionProposalMessage(msg: ChatMsg) {
    const proposal = msg.collection?.proposal
    if (!proposal) return

    try {
      const res = await cancelCollectionProposal(proposal.proposalId)
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
    options.inputBarEl.value?.focus()
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
    if (window.visualViewport) {
      window.visualViewport.addEventListener('resize', handleViewportResize)
      window.visualViewport.addEventListener('scroll', handleViewportResize)
    }
  })

  onBeforeUnmount(() => {
    if (window.visualViewport) {
      window.visualViewport.removeEventListener('resize', handleViewportResize)
      window.visualViewport.removeEventListener('scroll', handleViewportResize)
    }
    if (saveLabelTimer) clearTimeout(saveLabelTimer)
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
    sendMessage,
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
