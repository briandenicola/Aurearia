<template>
  <div class="chat-overlay" @click.self="$emit('close')">
    <div class="chat-drawer">
      <div class="chat-header">
        <h1><Bot :size="20" /> Coin Search Agent</h1>
        <div class="chat-header-actions">
          <button
            v-if="messages.length > 0"
            class="chat-save"
            :disabled="saving"
            @click="handleSave"
            :title="conversationId ? 'Update saved conversation' : 'Save conversation'"
          >
            <Save :size="16" />
            {{ saveLabel }}
          </button>
          <button class="chat-close" @click="$emit('close')"><X :size="18" /></button>
        </div>
      </div>

      <!-- Unconfigured provider banner -->
      <div v-if="!providerConfigured" class="provider-banner">
        <AlertTriangle :size="16" />
        <span>AI provider not configured. <a href="/admin" @click="$emit('close')">Go to Admin Settings</a> to select Anthropic or Ollama.</span>
      </div>

      <div class="chat-messages" ref="messagesEl">
        <div v-if="messages.length === 0" class="chat-intro">
          <Bot :size="32" />
          <p>Search for coins, find upcoming shows, or get a portfolio analysis -- ask me anything about collecting.</p>
          <div class="chat-examples">
            <button class="example-btn" @click="sendExample('Find me Roman silver denarii of Julius Caesar')">
              Roman denarii of Julius Caesar
            </button>
            <button class="example-btn" @click="sendExample('I\'m looking for Byzantine gold solidi under $1000')">
              Byzantine gold solidi under $1000
            </button>
            <button class="example-btn" @click="sendExample('Show me ancient Greek tetradrachms from Athens')">
              Greek tetradrachms from Athens
            </button>
            <button class="example-btn" @click="sendExample('What ancient coin shows are coming up near me?')">
              Upcoming coin shows near me
            </button>
            <button class="example-btn" @click="sendPortfolioAnalysis">
              Analyze my portfolio
            </button>
          </div>
        </div>

        <template v-for="(msg, i) in messages" :key="i">
          <div class="chat-bubble" :class="[msg.role, { streaming: msg.streaming }]">
            <div v-if="msg.streaming && msg.statusText && !msg.content" class="bubble-content status-text">
              <span class="status-indicator"></span>{{ msg.statusText }}
            </div>
            <div v-else class="bubble-content" v-html="formatMessage(msg.content)"></div>
          </div>

          <!-- Coin Show results -->
          <div v-if="msg.role === 'assistant' && msg.suggestions?.length && isCoinShowResults(msg.suggestions)" class="suggestions-grid">
            <div v-for="(show, j) in (msg.suggestions as CoinShow[])" :key="j" class="show-card">
              <div class="show-body">
                <a v-if="show.url" :href="show.url" target="_blank" rel="noopener" class="show-name-link">
                  <h4>{{ show.name }} <ExternalLink :size="12" /></h4>
                </a>
                <h4 v-else>{{ show.name }}</h4>
                <div class="show-details">
                  <span v-if="show.dates" class="show-detail"><Calendar :size="13" /> {{ show.dates }}</span>
                  <span v-if="show.venue" class="show-detail"><MapPin :size="13" /> {{ show.venue }}</span>
                  <span v-if="show.location" class="show-detail-sub">{{ show.location }}</span>
                  <span v-if="show.entryFee" class="show-detail"><Ticket :size="13" /> {{ show.entryFee }}</span>
                </div>
                <p v-if="show.description" class="show-desc">{{ show.description }}</p>
                <div v-if="show.notableDealers?.length" class="show-dealers">
                  <span v-for="(dealer, k) in show.notableDealers" :key="k" class="meta-tag">{{ dealer }}</span>
                </div>
                <button
                  class="save-cal-btn"
                  :disabled="savedShows.has(showKey(show)) || savingShow === showKey(show)"
                  @click="saveShowToCalendar(show)"
                >
                  <CalendarPlus :size="13" />
                  {{ savedShows.has(showKey(show)) ? 'Saved' : savingShow === showKey(show) ? 'Saving...' : 'Save to Calendar' }}
                </button>
              </div>
            </div>
          </div>

          <!-- Coin suggestions after assistant message -->
          <div v-if="msg.role === 'assistant' && msg.suggestions?.length && !isCoinShowResults(msg.suggestions)" class="suggestions-grid">
            <div v-for="(coin, j) in (msg.suggestions as CoinSuggestion[])" :key="j" class="suggestion-card">
              <div class="suggestion-img" v-if="getSuggestionImageUrl(coin)">
                <img :src="getSuggestionImageUrl(coin)" :alt="coin.name" @error="handleImgError" />
              </div>
              <div class="suggestion-body">
                <h4>{{ coin.name }}</h4>
                <p class="suggestion-desc">{{ coin.description }}</p>
                <div class="suggestion-meta">
                  <span v-if="coin.era" class="meta-tag">{{ coin.era }}</span>
                  <span v-if="coin.material" class="meta-tag">{{ coin.material }}</span>
                  <span v-if="coin.denomination" class="meta-tag">{{ coin.denomination }}</span>
                </div>
                <div class="suggestion-price" v-if="coin.estPrice">{{ coin.estPrice }}</div>
                <div class="suggestion-actions">
                  <a v-if="coin.sourceUrl" :href="coin.sourceUrl" target="_blank" rel="noopener" class="source-link">
                    <ExternalLink :size="12" /> {{ coin.sourceName || 'Source' }}
                  </a>
                  <button
                    v-if="coin.era || coin.material || coin.denomination"
                    class="btn btn-primary btn-sm add-btn"
                    :disabled="addingIdx === `${i}-${j}`"
                    @click="addToWishlist(coin, `${i}-${j}`)"
                  >
                    <CirclePlus :size="14" />
                    {{ addedSet.has(`${i}-${j}`) ? 'Added!' : addingIdx === `${i}-${j}` ? 'Adding...' : 'Add to Wishlist' }}
                  </button>
                </div>
              </div>
            </div>
          </div>
        </template>

        <div v-if="loading && !messages[messages.length-1]?.streaming" class="chat-bubble assistant">
          <div class="bubble-content thinking">
            <span class="dot"></span><span class="dot"></span><span class="dot"></span>
          </div>
        </div>
      </div>

      <form class="chat-input-bar" @submit.prevent="sendMessage">
        <input
          v-model="input"
          class="chat-input"
          :placeholder="providerConfigured ? 'Describe the coins you\'re looking for...' : 'Configure AI provider in Admin Settings'"
          :disabled="loading || !providerConfigured"
          ref="inputEl"
        />
        <button type="submit" class="send-btn" :disabled="!input.trim() || loading || !providerConfigured">
          <SendHorizontal :size="18" />
        </button>
      </form>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, nextTick, onMounted, onBeforeUnmount } from 'vue'
import { agentChatStream, createCoin, proxyImage, scrapeImage, uploadImage, saveConversation, getPortfolioSummary, getAgentStatus, createCalendarEvent } from '@/api/client'
import type { CoinSuggestion, CoinShow, AgentChatMessage, Category, Material } from '@/types'
import { Bot, X, SendHorizontal, CirclePlus, ExternalLink, Save, AlertTriangle, Calendar, MapPin, Ticket, CalendarPlus } from 'lucide-vue-next'
import DOMPurify from 'dompurify'
import { useDialog } from '@/composables/useDialog'
import MarkdownIt from 'markdown-it'

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
const inputEl = ref<HTMLInputElement>()
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

function proxyImageUrl(url: string): string {
  if (!url) return ''
  return `/api/proxy-image?url=${encodeURIComponent(url)}`
}

function getSuggestionImageUrl(coin: CoinSuggestion): string {
  // Always prefer scraped image from sourceUrl (og:image is most reliable)
  if (coin.sourceUrl) {
    const cached = scrapedImages.value.get(coin.sourceUrl)
    if (cached) return proxyImageUrl(cached)
    if (cached === undefined) {
      scrapedImages.value.set(coin.sourceUrl, '')
      scrapeImage(coin.sourceUrl).then((res) => {
        if (res.data.imageUrl) {
          console.log('[agent] Scraped image from', coin.sourceUrl, '→', res.data.imageUrl)
          scrapedImages.value.set(coin.sourceUrl, res.data.imageUrl)
        } else if (coin.imageUrl) {
          // Scrape returned nothing — fall back to agent-provided URL
          console.log('[agent] Scrape empty, using agent imageUrl:', coin.imageUrl)
          scrapedImages.value.set(coin.sourceUrl, coin.imageUrl)
        }
      }).catch(() => {
        // Scrape failed — fall back to agent-provided URL
        if (coin.imageUrl) {
          console.log('[agent] Scrape failed, using agent imageUrl:', coin.imageUrl)
          scrapedImages.value.set(coin.sourceUrl, coin.imageUrl)
        }
      })
    }
    return ''
  }

  // No sourceUrl — try agent imageUrl directly
  if (coin.imageUrl) return proxyImageUrl(coin.imageUrl)
  return ''
}

function handleImgError(e: Event) {
  const img = e.target as HTMLImageElement
  console.warn('[agent] Image failed to load:', img.src)
  img.style.display = 'none'
}

onMounted(async () => {
  inputEl.value?.focus()
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

.chat-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0.75rem 1rem;
  border-bottom: 1px solid var(--border-subtle);
  flex-shrink: 0;
}

.chat-header h1 {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 1.4rem;
  margin: 0;
  color: var(--accent-gold);
}

.chat-header-actions {
  display: flex;
  align-items: center;
  gap: 0.5rem;
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

.chat-save {
  display: flex;
  align-items: center;
  gap: 0.3rem;
  background: none;
  border: 1px solid var(--border-subtle);
  color: var(--text-secondary);
  cursor: pointer;
  padding: 0.3rem 0.6rem;
  border-radius: var(--radius-sm);
  font-size: 0.75rem;
  transition: all var(--transition-fast);
}

.chat-save:hover:not(:disabled) {
  color: var(--accent-gold);
  border-color: var(--accent-gold);
}

.chat-save:disabled {
  opacity: 0.5;
  cursor: default;
}

.chat-close {
  background: none;
  border: none;
  color: var(--text-muted);
  cursor: pointer;
  padding: 0.25rem;
  border-radius: var(--radius-sm);
  transition: all var(--transition-fast);
}

.chat-close:hover {
  color: var(--text-primary);
  background: var(--bg-card);
}

.chat-messages {
  flex: 1;
  overflow-y: auto;
  padding: 1rem;
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.chat-intro {
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
  padding: 1rem;
  color: var(--text-secondary);
  gap: 0.75rem;
  flex-shrink: 1;
  overflow-y: auto;
}

.chat-intro p {
  max-width: 300px;
  line-height: 1.5;
}

.chat-examples {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  margin-top: 0.5rem;
  width: 100%;
}

.example-btn {
  background: var(--bg-card);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
  padding: 0.6rem 0.75rem;
  color: var(--text-secondary);
  font-size: 0.82rem;
  cursor: pointer;
  text-align: left;
  transition: all var(--transition-fast);
}

.example-btn:hover {
  border-color: var(--accent-gold);
  color: var(--accent-gold);
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

.suggestions-grid {
  display: flex;
  flex-direction: column;
  gap: 0.6rem;
  width: 100%;
}

.suggestion-card {
  background: var(--bg-card);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  overflow: hidden;
  display: flex;
  transition: border-color var(--transition-fast);
}

.suggestion-card:hover {
  border-color: var(--accent-gold);
}

.suggestion-img {
  width: 80px;
  min-height: 80px;
  flex-shrink: 0;
  overflow: hidden;
  background: var(--bg-body);
}

.suggestion-img img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.suggestion-body {
  padding: 0.6rem 0.75rem;
  flex: 1;
  min-width: 0;
}

.suggestion-body h4 {
  font-size: 0.85rem;
  margin: 0 0 0.25rem;
  color: var(--text-primary);
  line-height: 1.3;
}

.suggestion-desc {
  font-size: 0.78rem;
  color: var(--text-secondary);
  margin: 0 0 0.4rem;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.suggestion-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 0.3rem;
  margin-bottom: 0.4rem;
}

.meta-tag {
  font-size: 0.7rem;
  padding: 0.1rem 0.4rem;
  border-radius: var(--radius-full);
  background: var(--bg-body);
  color: var(--text-muted);
  border: 1px solid var(--border-subtle);
}

.suggestion-price {
  font-size: 0.82rem;
  font-weight: 600;
  color: var(--accent-gold);
  margin-bottom: 0.4rem;
}

.suggestion-actions {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
}

.source-link {
  font-size: 0.72rem;
  color: var(--text-muted);
  text-decoration: none;
  display: flex;
  align-items: center;
  gap: 0.2rem;
  transition: color var(--transition-fast);
}

.source-link:hover {
  color: var(--accent-gold);
}

/* Coin Show cards */
.show-card {
  background: var(--bg-card);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  overflow: hidden;
  transition: border-color var(--transition-fast);
}

.show-card:hover {
  border-color: var(--accent-gold);
}

.show-body {
  padding: 0.7rem 0.85rem;
}

.show-body h4 {
  font-size: 0.88rem;
  margin: 0 0 0.4rem;
  color: var(--text-primary);
  line-height: 1.3;
}

.show-name-link {
  text-decoration: none;
  color: inherit;
}

.show-name-link h4 {
  display: flex;
  align-items: center;
  gap: 0.35rem;
  color: var(--accent-gold);
  transition: color var(--transition-fast);
}

.show-name-link:hover h4 {
  color: var(--accent-bronze);
}

.show-details {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
  margin-bottom: 0.4rem;
}

.show-detail {
  display: flex;
  align-items: center;
  gap: 0.35rem;
  font-size: 0.8rem;
  color: var(--text-secondary);
}

.show-detail-sub {
  font-size: 0.75rem;
  color: var(--text-muted);
  padding-left: 1.5rem;
}

.show-desc {
  font-size: 0.78rem;
  color: var(--text-secondary);
  margin: 0 0 0.4rem;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.show-dealers {
  display: flex;
  flex-wrap: wrap;
  gap: 0.25rem;
}

.save-cal-btn {
  display: flex;
  align-items: center;
  gap: 0.35rem;
  font-size: 0.75rem;
  padding: 0.3rem 0.65rem;
  margin-top: 0.5rem;
  background: transparent;
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
  color: var(--text-secondary);
  cursor: pointer;
  transition: all var(--transition-fast);
  font-family: inherit;
}

.save-cal-btn:hover:not(:disabled) {
  border-color: var(--accent-gold-dim);
  color: var(--accent-gold);
  background: var(--accent-gold-glow);
}

.save-cal-btn:disabled {
  opacity: 0.6;
  cursor: default;
}

.add-btn {
  display: flex;
  align-items: center;
  gap: 0.3rem;
  font-size: 0.72rem;
  padding: 0.3rem 0.6rem;
  flex-shrink: 0;
}

.chat-input-bar {
  display: flex;
  gap: 0.5rem;
  padding: 0.75rem 1rem;
  border-top: 1px solid var(--border-subtle);
  flex-shrink: 0;
  background: var(--bg-primary);
}

.chat-input {
  flex: 1;
  background: var(--bg-input);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
  padding: 0.6rem 0.75rem;
  color: var(--text-primary);
  font-size: 0.88rem;
  outline: none;
  transition: border-color var(--transition-fast);
}

.chat-input:focus {
  border-color: var(--accent-gold);
}

.send-btn {
  background: linear-gradient(135deg, var(--accent-gold), var(--accent-bronze));
  border: none;
  border-radius: var(--radius-sm);
  color: var(--bg-primary);
  padding: 0.5rem 0.75rem;
  cursor: pointer;
  transition: all var(--transition-fast);
  display: flex;
  align-items: center;
}

.send-btn:hover:not(:disabled) {
  box-shadow: 0 0 12px var(--accent-gold-dim);
}

.send-btn:disabled {
  opacity: 0.4;
  cursor: default;
}

@media (max-width: 640px) {
  .chat-drawer {
    width: 100%;
  }

  .suggestion-card {
    flex-direction: column;
  }

  .suggestion-img {
    width: 100%;
    height: 120px;
  }
}
</style>
