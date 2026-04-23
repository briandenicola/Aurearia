<template>
  <div class="chat-intro">
    <Bot :size="32" />
    <p>Search for coins, find upcoming shows, or get a portfolio analysis -- ask me anything about collecting.</p>
    <div class="chat-examples">
      <button class="example-btn" @click="$emit('send', 'Find me Roman silver denarii of Julius Caesar')">
        Roman denarii of Julius Caesar
      </button>
      <button class="example-btn" @click="$emit('send', 'I\'m looking for Byzantine gold solidi under $1000')">
        Byzantine gold solidi under $1000
      </button>
      <button class="example-btn" @click="$emit('send', 'Show me ancient Greek tetradrachms from Athens')">
        Greek tetradrachms from Athens
      </button>
      <button class="example-btn" @click="$emit('send', 'What ancient coin shows are coming up near me?')">
        Upcoming coin shows near me
      </button>
      <button class="example-btn" @click="$emit('send-portfolio')">
        Analyze my portfolio
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { Bot } from 'lucide-vue-next'

defineEmits<{
  send: [message: string]
  'send-portfolio': []
}>()
</script>

<style scoped>
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
</style>
