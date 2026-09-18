<template>
  <section
    class="w-full rounded-sm border border-border-accent bg-card p-3"
    aria-labelledby="copilot-clarification-heading"
    data-testid="copilot-clarification"
  >
    <p class="section-label">Coin Copilot needs clarification</p>
    <h3 id="copilot-clarification-heading" class="m-0 text-base text-text-primary">
      {{ clarification.question }}
    </h3>

    <div v-if="clarification.inputType === 'single_choice'" class="mt-3 flex flex-wrap gap-2">
      <button
        v-for="choice in clarification.choices"
        :key="choice"
        type="button"
        class="chip min-h-[44px]"
        :class="{ active: answer === choice }"
        @click="answer = choice"
      >
        {{ choice }}
      </button>
    </div>

    <div v-else-if="clarification.inputType === 'boolean'" class="mt-3 flex gap-2">
      <button type="button" class="chip min-h-[44px]" :class="{ active: answer === 'Yes' }" @click="answer = 'Yes'">Yes</button>
      <button type="button" class="chip min-h-[44px]" :class="{ active: answer === 'No' }" @click="answer = 'No'">No</button>
    </div>

    <label v-else class="form-group mt-3">
      <span class="form-label">Your answer</span>
      <textarea v-model="answer" class="form-input min-h-[96px]" maxlength="4000" rows="3"></textarea>
    </label>

    <p v-if="error" class="mt-2 text-sm text-[var(--color-negative)]" role="alert">{{ error }}</p>
    <div class="mt-3 flex flex-wrap justify-end gap-2">
      <button
        type="button"
        class="btn btn-sm btn-ghost min-h-[44px]"
        :disabled="submitting || cancelling"
        @click="$emit('cancel')"
      >
        Cancel run
      </button>
      <button
        type="button"
        class="btn btn-sm btn-primary min-h-[44px]"
        :disabled="!answer.trim() || submitting || cancelling"
        @click="$emit('resume', answer.trim())"
      >
        {{ submitting ? 'Resuming...' : 'Continue' }}
      </button>
    </div>
  </section>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import type { CoinCopilotClarification } from '@/types'

const props = defineProps<{
  clarification: CoinCopilotClarification
  submitting: boolean
  cancelling: boolean
  error: string
}>()

defineEmits<{
  resume: [answer: string]
  cancel: []
}>()

const answer = ref('')

watch(() => props.clarification.checkpointVersion, () => {
  answer.value = ''
})
</script>
