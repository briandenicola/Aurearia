<template>
  <div class="fixed inset-0 z-[1400] flex h-dvh justify-end bg-black/50" @click.self="$emit('close')">
    <div class="flex h-full w-full max-w-full flex-col bg-surface shadow-[-4px_0_20px_rgba(0,0,0,0.3)] sm:w-[480px]">
      <ChatHeader
        :has-messages="messages.length > 0"
        :saving="saving"
        :conversation-id="conversationId"
        :save-label="saveLabel"
        @save="handleSave"
        @close="$emit('close')"
      />

      <div v-if="!providerConfigured" class="shrink-0 flex items-center gap-2 border-b border-warning/30 bg-warning/10 px-4 py-3 text-body text-warning">
        <AlertTriangle :size="16" />
        <span>AI provider not configured. <a href="/admin" class="font-semibold text-gold underline" @click="$emit('close')">Go to Admin Settings</a> to select Anthropic or Ollama.</span>
      </div>

      <div ref="messagesEl" class="flex flex-1 flex-col gap-3 overflow-y-auto p-4">
        <ChatIntroPanel
          v-if="messages.length === 0"
          @send="sendExample"
          @send-portfolio="sendPortfolioAnalysis"
        />

        <template v-for="(msg, i) in messages" :key="i">
          <div
            class="max-w-[85%] rounded-md px-[0.85rem] py-[0.65rem] text-[0.88rem] leading-[1.5] break-words"
            :class="msg.role === 'user' ? 'self-end bg-[linear-gradient(135deg,var(--accent-gold),var(--accent-bronze))] text-surface' : 'self-start border border-border-subtle bg-card text-text-primary'"
          >
            <div v-if="msg.streaming && msg.statusText && !msg.content" class="flex items-center gap-2 italic text-text-secondary">
              <span class="h-1.5 w-1.5 rounded-full bg-gold animate-pulse"></span>{{ msg.statusText }}
            </div>
            <div
              v-else
              class="bubble-content break-words [&_a]:text-gold [&_a]:underline [&_blockquote]:my-2 [&_blockquote]:border-l-[3px] [&_blockquote]:border-gold [&_blockquote]:py-1 [&_blockquote]:pl-3 [&_blockquote]:text-text-secondary [&_code]:rounded-[3px] [&_code]:bg-white/[0.06] [&_code]:px-[0.35em] [&_code]:py-[0.1em] [&_code]:text-[0.88em] [&_h1]:my-2 [&_h1]:text-[1em] [&_h1]:font-semibold [&_h2]:my-2 [&_h2]:text-[1em] [&_h2]:font-semibold [&_h3]:my-2 [&_h3]:text-[1em] [&_h3]:font-semibold [&_h4]:my-2 [&_h4]:text-[1em] [&_h4]:font-semibold [&_li]:mb-[0.2em] [&_ol]:my-2 [&_ol]:ml-5 [&_ol]:p-0 [&_p]:mb-[0.5em] [&_p:last-child]:mb-0 [&_pre]:my-2 [&_pre]:overflow-x-auto [&_pre]:rounded-sm [&_pre]:bg-white/[0.06] [&_pre]:px-[0.8em] [&_pre]:py-[0.6em] [&_ul]:my-2 [&_ul]:ml-5 [&_ul]:p-0"
              v-html="formatMessage(msg.content)"
            ></div>
            <span v-if="msg.streaming && (!msg.statusText || msg.content)" class="ml-1 inline-block text-gold animate-pulse">▊</span>
          </div>

          <CoinShowResultsGrid
            v-if="msg.role === 'assistant' && msg.suggestions?.length && isCoinShowResults(msg.suggestions)"
            :shows="(msg.suggestions as CoinShow[])"
            :saved-shows="savedShows"
            :saving-show="savingShow"
            @save-show="saveShowToCalendar"
          />

          <CoinSuggestionGrid
            v-if="msg.role === 'assistant' && msg.suggestions?.length && !isCoinShowResults(msg.suggestions)"
            :suggestions="(msg.suggestions as CoinSuggestion[])"
            :added-set="addedSet"
            :adding-idx="addingIdx"
            :message-index="i"
            @add-to-wishlist="addToWishlist"
          />

          <div v-if="msg.role === 'assistant' && msg.collection" class="-mt-[0.3rem] mb-[0.3rem] flex max-w-[85%] flex-col gap-[0.6rem] self-start rounded-sm border border-border-subtle bg-card p-3">
            <div v-if="msg.collection.kind === 'proposal' && msg.collection.proposal">
              <div class="section-label">Pending collection update</div>
              <div class="text-body text-text-primary">#{{ msg.collection.proposal.coinId }} {{ msg.collection.proposal.coinName }}</div>
              <div class="flex flex-col gap-[0.35rem]">
                <div v-for="field in msg.collection.proposal.changedFields" :key="field" class="flex justify-between gap-3 text-chip text-text-secondary">
                  <span>{{ field }}</span>
                  <strong class="font-medium text-gold">{{ formatProposalChange(msg.collection, field) }}</strong>
                </div>
              </div>
              <div class="text-sm text-text-muted">Expires {{ formatExpiry(msg.collection.proposal.expiresAt) }}</div>
              <div class="flex gap-2">
                <button class="btn btn-xs btn-primary" @click="confirmCollectionProposal(msg)">Confirm update</button>
                <button class="btn btn-xs btn-ghost" @click="cancelCollectionProposalMessage(msg)">Cancel</button>
              </div>
            </div>

            <div v-else-if="msg.collection.kind === 'disambiguation' && msg.collection.disambiguation">
              <div class="section-label">Choose a coin</div>
              <div class="flex flex-wrap gap-[0.35rem]">
                <button
                  v-for="candidate in msg.collection.disambiguation.candidates"
                  :key="candidate.id"
                  class="btn btn-xs btn-secondary"
                  @click="pickDisambiguationCandidate(candidate.id)"
                >
                  #{{ candidate.id }} {{ candidate.name }}
                </button>
              </div>
            </div>

            <div v-else-if="msg.collection.kind === 'read_result'">
              <div v-if="msg.collection.readResult?.aggregate" class="flex flex-wrap gap-[0.35rem]">
                <span class="chip-sm">Active: {{ msg.collection.readResult.aggregate.totalCoins }}</span>
                <span class="chip-sm">Wishlist: {{ msg.collection.readResult.aggregate.totalWishlist }}</span>
                <span class="chip-sm">Sold: {{ msg.collection.readResult.aggregate.totalSold }}</span>
                <span class="chip-sm">Value: {{ formatUsd(msg.collection.readResult.aggregate.totalCurrentUsd) }}</span>
              </div>
              <div v-if="msg.collection.readResult?.coins?.length" class="flex flex-wrap gap-[0.35rem]">
                <span v-for="coin in msg.collection.readResult.coins" :key="coin.id" class="chip-sm">
                  #{{ coin.id }} {{ coin.name }}
                </span>
              </div>
            </div>
          </div>
        </template>

        <div v-if="loading && !messages[messages.length - 1]?.streaming" class="max-w-[85%] self-start rounded-md border border-border-subtle bg-card px-[0.85rem] py-[0.65rem] text-text-primary">
          <div class="flex items-center gap-1.5 italic text-text-muted">
            <span class="h-1.5 w-1.5 rounded-full bg-gold animate-pulse"></span>
            <span class="h-1.5 w-1.5 rounded-full bg-gold animate-pulse [animation-delay:0.2s]"></span>
            <span class="h-1.5 w-1.5 rounded-full bg-gold animate-pulse [animation-delay:0.4s]"></span>
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
import { onMounted, ref, watch } from 'vue'
import type { CoinSuggestion, CoinShow } from '@/types'
import { AlertTriangle } from 'lucide-vue-next'
import { useCoinSearchChat } from '@/composables/useCoinSearchChat'
import ChatHeader from '@/components/chat/ChatHeader.vue'
import ChatIntroPanel from '@/components/chat/ChatIntroPanel.vue'
import ChatInputBar from '@/components/chat/ChatInputBar.vue'
import CoinShowResultsGrid from '@/components/chat/CoinShowResultsGrid.vue'
import CoinSuggestionGrid from '@/components/chat/CoinSuggestionGrid.vue'

const props = defineProps<{
  loadConversation?: { id: number; title: string; messages: string } | null
  initialPrompt?: string | null
}>()

const emit = defineEmits<{
  close: []
  added: []
}>()

const messagesEl = ref<HTMLElement>()
const inputBarEl = ref<InstanceType<typeof ChatInputBar>>()
const sentInitialPrompts = new Set<string>()

const {
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
} = useCoinSearchChat({
  loadConversation: props.loadConversation,
  messagesEl,
  inputBarEl,
  onAdded: () => emit('added'),
})

function sendInitialPrompt(prompt?: string | null) {
  const text = prompt?.trim()
  if (!text || sentInitialPrompts.has(text)) return
  sentInitialPrompts.add(text)
  sendExample(text)
}

onMounted(() => {
  sendInitialPrompt(props.initialPrompt)
})

watch(() => props.initialPrompt, (prompt) => {
  sendInitialPrompt(prompt)
})

function formatProposalChange(collection: { proposal?: { changes: Record<string, unknown> } }, field: string): string {
  const value = collection.proposal?.changes?.[field]
  if (typeof value === 'number') return formatUsd(value)
  if (Array.isArray(value)) return value.join(', ')
  if (value === null || value === undefined) return 'n/a'
  return String(value)
}

function formatExpiry(value: string): string {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleString()
}

function formatUsd(value: number): string {
  return `$${value.toFixed(2)}`
}
</script>

<style scoped>
/* Markdown inside chat bubbles */
/*
 * :deep() audit — markdown-rendered content
 * Target: HTML elements emitted by markdown-it at runtime inside .bubble-content.
 * These selectors are intentional and irreplaceable with Tailwind utilities
 * because the nodes are generated after render from sanitized Markdown, not
 * authored in Vue templates; Vue's scoped-style hash cannot reach them.
 */
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
  padding: 0.25em 0 0.25em 0.75em;
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
</style>
