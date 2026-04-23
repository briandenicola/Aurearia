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
import { ref } from 'vue'
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
}>()

const emit = defineEmits<{
  close: []
  added: []
}>()

const messagesEl = ref<HTMLElement>()
const inputBarEl = ref<InstanceType<typeof ChatInputBar>>()

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
  formatMessage,
  isCoinShowResults,
  saveShowToCalendar,
} = useCoinSearchChat({
  loadConversation: props.loadConversation,
  messagesEl,
  inputBarEl,
  onAdded: () => emit('added'),
})
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
