<template>
  <Teleport to="body">
    <div v-if="request" class="fixed inset-0 z-[1600] flex items-center justify-center bg-[rgba(0,0,0,0.6)] px-4" @click="cancel">
      <div class="w-full max-w-[360px] rounded-md border border-border-subtle bg-card p-6 shadow-[0_12px_40px_rgba(0,0,0,0.5)]" @click.stop>
        <h3 class="mb-2 text-base text-heading">Confirm {{ request.fieldLabel }}</h3>
        <p class="mb-3 text-body text-text-secondary">
          The suggested {{ request.fieldLabel.toLowerCase() }} "<strong class="text-text-primary">{{ request.suggestedValue }}</strong>"
          doesn't match any of your configured values. Use one of yours, or keep it as suggested.
        </p>
        <div class="flex max-h-[240px] flex-col gap-1.5 overflow-y-auto">
          <button
            v-for="option in request.options"
            :key="option"
            class="rounded-sm border border-border-subtle bg-surface px-3 py-2 text-left text-body text-text-primary transition-colors hover:border-gold hover:text-gold"
            @click="choose(option)"
          >
            {{ option }}
          </button>
        </div>
        <button
          class="btn btn-secondary btn-sm mt-3 w-full"
          @click="choose(request.suggestedValue)"
        >
          Keep "{{ request.suggestedValue }}"
        </button>
        <button class="btn btn-ghost btn-sm mt-2 w-full" @click="cancel">Cancel</button>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import type { CategoryEraConfirmRequest } from '@/composables/useCoinSearchChat'

defineProps<{
  request: CategoryEraConfirmRequest | null
}>()

const emit = defineEmits<{
  choose: [value: string]
  cancel: []
}>()

function choose(value: string) {
  emit('choose', value)
}

function cancel() {
  emit('cancel')
}
</script>
