<template>
  <form class="chat-input-bar" @submit.prevent="$emit('send')">
    <input
      ref="inputEl"
      v-model="model"
      class="chat-input"
      :placeholder="providerConfigured ? 'Describe the coins you\'re looking for...' : 'Configure AI provider in Admin Settings'"
      :disabled="loading || !providerConfigured"
    />
    <button type="submit" class="send-btn" :disabled="!model.trim() || loading || !providerConfigured">
      <SendHorizontal :size="18" />
    </button>
  </form>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { SendHorizontal } from 'lucide-vue-next'

const model = defineModel<string>({ required: true })

defineProps<{
  loading: boolean
  providerConfigured: boolean
}>()

defineEmits<{
  send: []
}>()

const inputEl = ref<HTMLInputElement>()

defineExpose({ focus: () => inputEl.value?.focus() })
</script>

<style scoped>
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
</style>
