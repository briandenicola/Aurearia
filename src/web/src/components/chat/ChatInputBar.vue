<template>
  <form class="flex shrink-0 gap-2 border-t border-border-subtle bg-surface px-4 py-3" @submit.prevent="$emit('send')">
    <input
      ref="inputEl"
      v-model="model"
      class="form-input flex-1"
      :placeholder="providerConfigured ? 'Describe the coins you\'re looking for...' : 'Configure AI provider in Admin Settings'"
      :disabled="loading || !providerConfigured"
    />
    <button
      type="submit"
      class="btn btn-primary shrink-0 px-3 disabled:cursor-default disabled:opacity-40"
      :disabled="!model.trim() || loading || !providerConfigured"
    >
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
