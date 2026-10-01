<template>
  <div v-if="supported" class="grid gap-1.5">
    <BaseButton
      type="button"
      variant="secondary"
      size="sm"
      class="min-h-11 justify-self-start"
      :disabled="disabled"
      :aria-label="buttonLabel"
      :aria-pressed="status !== 'idle'"
      @click="toggle"
    >
      <Square v-if="status !== 'idle'" :size="16" aria-hidden="true" />
      <Mic v-else :size="16" aria-hidden="true" />
      {{ visibleLabel }}
    </BaseButton>
    <p class="m-0 text-sm text-text-muted">
      Voice recognition may send audio to your browser or device provider. Aurearia does not store audio.
    </p>
    <p v-if="error" class="m-0 text-sm text-loss" role="alert">{{ error }}</p>
  </div>
</template>

<script setup lang="ts">
import { computed, toRef } from 'vue'
import { Mic, Square } from 'lucide-vue-next'
import BaseButton from '@/components/ui/BaseButton.vue'
import { useVoiceDictation } from '@/composables/useVoiceDictation'

const props = withDefaults(defineProps<{
  disabled?: boolean
}>(), {
  disabled: false,
})

const emit = defineEmits<{
  transcript: [text: string]
}>()

const {
  supported,
  status,
  error,
  start,
  stop,
} = useVoiceDictation({
  disabled: toRef(props, 'disabled'),
  onFinalText: text => emit('transcript', text),
})

const buttonLabel = computed(() =>
  status.value === 'idle' ? 'Start voice dictation' : 'Stop voice dictation')
const visibleLabel = computed(() => {
  if (status.value === 'listening') return 'Listening'
  if (status.value === 'stopping') return 'Stopping'
  return 'Dictate notes'
})

function toggle() {
  if (status.value === 'idle') start()
  else stop()
}
</script>
