<template>
  <Transition name="bar-slide">
    <div v-if="selectedCount > 0" class="bulk-action-bar">
      <span class="bulk-count">{{ selectedCount }} lot{{ selectedCount === 1 ? '' : 's' }} selected</span>
      <div class="bulk-actions">
        <select v-model="localEventId" class="form-input bulk-event-select">
          <option value="">Unlink Event</option>
          <option v-for="evt in calendarEvents" :key="evt.id" :value="evt.id">
            {{ evt.title }}
          </option>
        </select>
        <button class="bulk-btn bulk-btn-link" @click="$emit('link-event', localEventId)">
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

<style scoped>
.bulk-action-bar {
  position: fixed;
  bottom: 1.5rem;
  left: 50%;
  transform: translateX(-50%);
  display: flex;
  align-items: center;
  gap: 1rem;
  background: var(--bg-card);
  border: 1px solid var(--accent-gold-dim);
  border-radius: var(--radius-md);
  padding: 0.75rem 1.25rem;
  box-shadow: 0 8px 30px rgba(0, 0, 0, 0.5);
  z-index: 200;
  white-space: nowrap;
}

.bulk-count {
  font-size: 0.85rem;
  color: var(--text-secondary);
  font-weight: 500;
}

.bulk-actions {
  display: flex;
  gap: 0.5rem;
  align-items: center;
}

.bulk-event-select {
  min-width: 160px;
  font-size: 0.82rem;
}

.bulk-btn {
  display: flex;
  align-items: center;
  gap: 0.35rem;
  padding: 0.4rem 0.75rem;
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
  background: var(--bg-primary);
  color: var(--text-primary);
  font-size: 0.8rem;
  cursor: pointer;
  transition: all var(--transition-fast);
}

.bulk-btn:hover {
  border-color: var(--accent-gold);
  color: var(--accent-gold);
}

/* Bar slide transition */
.bar-slide-enter-active,
.bar-slide-leave-active {
  transition: transform 0.25s ease, opacity 0.25s ease;
}

.bar-slide-enter-from,
.bar-slide-leave-to {
  transform: translateX(-50%) translateY(20px);
  opacity: 0;
}
</style>
