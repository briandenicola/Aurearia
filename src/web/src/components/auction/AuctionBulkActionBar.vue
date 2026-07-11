<template>
  <Transition
    enter-active-class="transition duration-[250ms] ease-out"
    enter-from-class="translate-y-5 opacity-0"
    enter-to-class="translate-y-0 opacity-100"
    leave-active-class="transition duration-[250ms] ease-in"
    leave-from-class="translate-y-0 opacity-100"
    leave-to-class="translate-y-5 opacity-0"
  >
    <div
      v-if="selectedCount > 0"
      class="fixed bottom-6 left-1/2 z-[200] flex w-[calc(100%-1.5rem)] max-w-max -translate-x-1/2 flex-col gap-3 rounded-md border border-gold-dim bg-card px-5 py-3 shadow-[0_8px_30px_rgba(0,0,0,0.5)] md:w-auto md:flex-row md:items-center md:gap-4"
    >
      <span class="text-body font-medium text-text-secondary">{{ selectedCount }} lot{{ selectedCount === 1 ? '' : 's' }} selected</span>
      <div class="flex flex-col gap-2 md:flex-row md:items-center">
        <select v-model="localEventId" class="form-input text-[0.82rem] md:min-w-40">
          <option value="">Unlink Event</option>
          <option v-for="evt in calendarEvents" :key="evt.id" :value="evt.id">
            {{ evt.title }}
          </option>
        </select>
        <button class="inline-flex items-center justify-center gap-[0.35rem] rounded-sm border border-border-subtle bg-surface px-3 py-1.5 text-chip text-text-primary transition hover:border-gold hover:text-gold" @click="$emit('link-event', localEventId)">
          <CalendarDays :size="16" /> {{ localEventId === '' ? 'Unlink' : 'Link' }}
        </button>
      </div>
    </div>
  </Transition>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { CalendarDays } from 'lucide-vue-next'

defineProps<{
  selectedCount: number
  calendarEvents: Array<{ id: number; title: string; auctionHouse: string; startDate: string | null }>
}>()

defineEmits<{
  'link-event': [eventId: number | string]
}>()

const localEventId = ref<number | string>('')
</script>
