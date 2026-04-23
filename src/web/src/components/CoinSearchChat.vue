<template>
  <div class="chat-overlay" @click.self="$emit('close')">
    <div class="chat-drawer">
      <ChatHeader
        :has-messages="messages.length > 0"
        :saving="saving"
        :conversation-id="conversationId"
        :save-label="saveLabel"
        @save="handleSave"
        @close="$emit('close')"
      />

      <!-- Unconfigured provider banner -->
      <div v-if="!providerConfigured" class="provider-banner">
        <AlertTriangle :size="16" />
        <span>AI provider not configured. <a href="/admin" @click="$emit('close')">Go to Admin Settings</a> to select Anthropic or Ollama.</span>
      </div>

      <div class="chat-messages" ref="messagesEl">
        <ChatIntroPanel
          v-if="messages.length === 0"
          @send="sendExample"
          @send-portfolio="sendPortfolioAnalysis"
        />

        <template v-for="(msg, i) in messages" :key="i">
          <div class="chat-bubble" :class="[msg.role, { streaming: msg.streaming }]">
            <div v-if="msg.streaming && msg.statusText && !msg.content" class="bubble-content status-text">
              <span class="status-indicator"></span>{{ msg.statusText }}
            </div>
            <div v-else class="bubble-content" v-html="formatMessage(msg.content)"></div>
          </div>

          <!-- Coin Show results -->
          <CoinShowResultsGrid
            v-if="msg.role === 'assistant' && msg.suggestions?.length && isCoinShowResults(msg.suggestions)"
            :shows="(msg.suggestions as CoinShow[])"
            :saved-shows="savedShows"
            :saving-show="savingShow"
            @save-show="saveShowToCalendar"
          />

          <!-- Coin suggestions after assistant message -->
          <CoinSuggestionGrid
            v-if="msg.role === 'assistant' && msg.suggestions?.length && !isCoinShowResults(msg.suggestions)"
            :suggestions="(msg.suggestions as CoinSuggestion[])"
            :added-set="addedSet"
            :adding-idx="addingIdx"
            :message-index="i"
            @add-to-wishlist="addToWishlist"
          />
        </template>

        <div v-if="loading && !messages[messages.length-1]?.streaming" class="chat-bubble assistant">
          <div class="bubble-content thinking">
            <span class="dot"></span><span class="dot"></span><span class="dot"></span>
          </div>
        </div>
      </div>

      <ChatInputBar
        v-model="input"
        :loading="loading"
        :provider-configured="providerConfigured"
        ref="inputBarEl"
        @send="sendMessage"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, nextTick, onMounted, onBeforeUnmount } from 'vue'
import { agentChatStream, createCoin, proxyImage, scrapeImage, uploadImage, saveConversation, getPortfolioSummary, getAgentStatus, createCalendarEvent } from '@/api/client'
import type { CoinSuggestion, CoinShow, AgentChatMessage, Category, Material } from '@/types'
import { AlertTriangle } from 'lucide-vue-next'
import DOMPurify from 'dompurify'
import { useDialog } from '@/composables/useDialog'
import MarkdownIt from 'markdown-it'
import ChatHeader from '@/components/chat/ChatHeader.vue'
import ChatIntroPanel from '@/components/chat/ChatIntroPanel.vue'
import ChatInputBar from '@/components/chat/ChatInputBar.vue'
import CoinShowResultsGrid from '@/components/chat/CoinShowResultsGrid.vue'
import CoinSuggestionGrid from '@/components/chat/CoinSuggestionGrid.vue'

type ChatSuggestion = CoinSuggestion | CoinShow

interface ChatMsg {
  role: 'user' | 'assistant'
  content: string
  suggestions?: ChatSuggestion[]
  streaming?: boolean
  statusText?: string
}

const props = defineProps<{
  loadConversation?: { id: number; title: string; messages: string } | null
}>()

const emit = defineEmits<{
  close: []
  added: []
}>()

const { showAlert } = useDialog()
const messages = ref<ChatMsg[]>([])
const input = ref('')
const loading = ref(false)
const addingIdx = ref<string | null>(null)
const addedSet = ref<Set<string>>(new Set())
const savedShows = ref<Set<string>>(new Set())
const savingShow = ref<string | null>(null)
const messagesEl = ref<HTMLElement>()
const inputBarEl = ref<InstanceType<typeof ChatInputBar>>()
const conversationId = ref<number | null>(null)
const saving = ref(false)
const scrapedImages = ref<Map<string, string>>(new Map())
const saveLabel = ref('Save')
const providerConfigured = ref(true)  // assume configured until checked

const VALID_CATEGORIES = ['Roman', 'Greek', 'Byzantine', 'Modern', 'Other']
const VALID_MATERIALS = ['Gold', 'Silver', 'Bronze', 'Copper', 'Electrum', 'Other']

function scrollToBottom() {
  nextTick(() => {
    if (messagesEl.value) {
      messagesEl.value.scrollTop = messagesEl.value.scrollHeight
    }
  })
}

function buildHistory(): AgentChatMessage[] {
  return messages.value
    .filter(m => m.role === 'user' || m.role === 'assistant')
    .map(m => ({ role: m.role, content: m.content }))
}

async function sendMessage() {
  const text = input.value.trim()
  if (!text || loading.value) return

  messages.value.push({ role: 'user', content: text })
  const history = buildHistory().slice(0, -1)
  input.value = ''
  loading.value = true
  scrollToBottom()

  // Add a streaming assistant bubble
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
    (message: string, suggestions: CoinSuggestion[]) => {
      const msg = messages.value[assistantIdx]!
      msg.content = message
      msg.suggestions = suggestions
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
    // Use first user message as title
    const firstUserMsg = messages.value.find(m => m.role === 'user')
    const title = firstUserMsg?.content.substring(0, 100) || 'Untitled conversation'

    const res = await saveConversation({
      id: conversationId.value || undefined,
      title,
      messages: JSON.stringify(messages.value),
    })
    conversationId.value = res.data.id
    saveLabel.value = 'Saved!'
    setTimeout(() => { saveLabel.value = 'Save' }, 2000)
  } catch {
    saveLabel.value = 'Failed'
    setTimeout(() => { saveLabel.value = 'Save' }, 2000)
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
    })

    // Try to download and attach coin image as obverse
    let imageAttached = false

    // Primary: scrape og:image from the listing page (most reliable)
    if (coin.sourceUrl) {
      try {
        // Check if we already scraped this URL during preview
        let scrapedUrl = scrapedImages.value.get(coin.sourceUrl) || ''
        if (!scrapedUrl) {
          const scraped = await scrapeImage(coin.sourceUrl)
          scrapedUrl = scraped.data.imageUrl || ''
        }
        if (scrapedUrl) {
          console.log('[agent] Downloading scraped image:', scrapedUrl)
          const imgRes = await proxyImage(scrapedUrl)
          const blob = imgRes.data as Blob
          if (blob.size > 0) {
            const ext = blob.type.includes('png') ? '.png' : '.jpg'
            const file = new File([blob], `obverse${ext}`, { type: blob.type || 'image/jpeg' })
            await uploadImage(created.data.id, file, 'obverse', true)
            imageAttached = true
            console.log('[agent] Image attached via scraping')
          }
        }
      } catch (err) {
        console.warn('[agent] Scrape-based image failed for', coin.sourceUrl, err)
      }
    }

    // Fallback: try agent-provided imageUrl directly
    if (!imageAttached && coin.imageUrl) {
      try {
        console.log('[agent] Trying agent imageUrl:', coin.imageUrl)
        const imgRes = await proxyImage(coin.imageUrl)
        const blob = imgRes.data as Blob
        if (blob.size > 0) {
          const ext = blob.type.includes('png') ? '.png' : '.jpg'
          const file = new File([blob], `obverse${ext}`, { type: blob.type || 'image/jpeg' })
          await uploadImage(created.data.id, file, 'obverse', true)
          imageAttached = true
          console.log('[agent] Image attached via agent imageUrl')
        }
      } catch (err) {
        console.warn('[agent] Agent imageUrl download failed:', coin.imageUrl, err)
      }
    }

    if (!imageAttached) {
      console.warn('[agent] No image could be attached for coin:', coin.name)
    }

    addedSet.value.add(idx)
    emit('added')
  } catch {
    await showAlert('Failed to add coin to wishlist', { title: 'Error' })
  } finally {
    addingIdx.value = null
  }
}

function parsePrice(price: string): number | null {
  if (!price) return null
  // Extract the first number from strings like "$150-300" or "$200"
  const match = price.match(/[\d,]+(?:\.\d+)?/)
  if (!match) return null
  return parseFloat(match[0].replace(/,/g, ''))
}

const md = new MarkdownIt({ html: false, linkify: true, breaks: true })

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

  // ISO format: "2026-05-15"
  const isoMatch = dateStr.match(/(\d{4}-\d{2}-\d{2})/)
  if (isoMatch) {
    return { start: isoMatch[1]! + 'T00:00:00Z' }
  }

  // "Month Day-Day, Year" e.g. "May 15-17, 2026"
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

  // "Month Day - Month Day, Year" e.g. "May 30 - June 1, 2026"
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

  // "Month Day, Year" e.g. "May 15, 2026"
  const singleMatch = dateStr.match(/([A-Z][a-z]+)\s+(\d{1,2}),?\s*(\d{4})/)
  if (singleMatch) {
    const d = new Date(`${singleMatch[1]} ${singleMatch[2]}, ${singleMatch[3]}`)
    if (!isNaN(d.getTime())) {
      return { start: d.toISOString().split('T')[0]! + 'T00:00:00Z' }
    }
  }

  // Fallback: try native Date parsing
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

onMounted(async () => {
  inputBarEl.value?.focus()
  if (props.loadConversation) {
    conversationId.value = props.loadConversation.id
    try {
      messages.value = JSON.parse(props.loadConversation.messages)
      scrollToBottom()
    } catch { /* ignore parse errors */ }
  }
  // Check if AI provider is configured
  try {
    const res = await getAgentStatus()
    providerConfigured.value = res.data.configured
  } catch {
    providerConfigured.value = true // don't block on network error
  }
  // Handle iOS keyboard resizing the visual viewport
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
})

function handleViewportResize() {
  const overlay = document.querySelector('.chat-overlay') as HTMLElement | null
  if (!overlay || !window.visualViewport) return
  const vv = window.visualViewport
  overlay.style.height = `${vv.height}px`
  overlay.style.top = `${vv.offsetTop}px`
}
</script>

<style scoped>
.chat-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.5);
  z-index: 300;
  display: flex;
  justify-content: flex-end;
  height: 100%;
  height: 100dvh;
}

.chat-drawer {
  width: 480px;
  max-width: 100%;
  height: 100%;
  background: var(--bg-primary);
  display: flex;
  flex-direction: column;
  box-shadow: -4px 0 20px rgba(0, 0, 0, 0.3);
}

.provider-banner {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.75rem 1rem;
  background: rgba(231, 176, 60, 0.1);
  border-bottom: 1px solid rgba(231, 176, 60, 0.3);
  color: #e7b03c;
  font-size: 0.85rem;
  flex-shrink: 0;
}

.provider-banner a {
  color: var(--accent-gold, #d4a843);
  text-decoration: underline;
  font-weight: 600;
}

.chat-messages {
  flex: 1;
  overflow-y: auto;
  padding: 1rem;
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.chat-bubble {
  max-width: 85%;
  padding: 0.65rem 0.85rem;
  border-radius: var(--radius-md);
  font-size: 0.88rem;
  line-height: 1.5;
  word-wrap: break-word;
}

.chat-bubble.user {
  align-self: flex-end;
  background: linear-gradient(135deg, var(--accent-gold), var(--accent-bronze));
  color: var(--bg-primary);
}

.chat-bubble.assistant {
  align-self: flex-start;
  background: var(--bg-card);
  border: 1px solid var(--border-subtle);
  color: var(--text-primary);
}

.thinking {
  display: flex;
  align-items: center;
  gap: 0.4rem;
  color: var(--text-muted);
  font-style: italic;
}

.dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--accent-gold);
  animation: pulse 1.2s ease-in-out infinite;
}

.dot:nth-child(2) { animation-delay: 0.2s; }
.dot:nth-child(3) { animation-delay: 0.4s; }

@keyframes pulse {
  0%, 80%, 100% { opacity: 0.3; transform: scale(0.8); }
  40% { opacity: 1; transform: scale(1); }
}

.chat-bubble.assistant.streaming .bubble-content::after {
  content: '▊';
  animation: blink 1s step-end infinite;
  color: var(--accent-gold);
}

/* Markdown inside chat bubbles */
.bubble-content :deep(p) {
  margin: 0 0 0.5em;
}
.bubble-content :deep(p:last-child) {
  margin-bottom: 0;
}
.bubble-content :deep(ul),
.bubble-content :deep(ol) {
  margin: 0.25em 0 0.5em 1.25em;
  padding: 0;
}
.bubble-content :deep(li) {
  margin-bottom: 0.2em;
}
.bubble-content :deep(a) {
  color: var(--accent-gold);
  text-decoration: underline;
}
.bubble-content :deep(code) {
  background: var(--bg-elevated, rgba(255,255,255,0.06));
  padding: 0.1em 0.35em;
  border-radius: 3px;
  font-size: 0.88em;
}
.bubble-content :deep(pre) {
  background: var(--bg-elevated, rgba(255,255,255,0.06));
  padding: 0.6em 0.8em;
  border-radius: var(--radius-sm);
  overflow-x: auto;
  margin: 0.5em 0;
}
.bubble-content :deep(blockquote) {
  border-left: 3px solid var(--accent-gold);
  margin: 0.5em 0;
  padding: 0.25em 0.75em;
  color: var(--text-secondary);
}
.bubble-content :deep(h1),
.bubble-content :deep(h2),
.bubble-content :deep(h3),
.bubble-content :deep(h4) {
  margin: 0.5em 0 0.25em;
  font-size: 1em;
  font-weight: 600;
}

.status-text {
  color: var(--text-secondary, #999);
  font-style: italic;
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.status-text .status-indicator {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--accent-gold);
  animation: pulse-dot 1.2s ease-in-out infinite;
}

@keyframes pulse-dot {
  0%, 100% { opacity: 0.3; transform: scale(0.8); }
  50% { opacity: 1; transform: scale(1.2); }
}

@keyframes blink {
  50% { opacity: 0; }
}

@media (max-width: 640px) {
  .chat-drawer {
    width: 100%;
  }
}
</style>
