<template>
  <Transition name="bar-slide">
    <div v-if="visible" class="bulk-action-bar">
      <span class="bulk-count">{{ selectedCount }} coin{{ selectedCount === 1 ? '' : 's' }} selected</span>
      <div class="bulk-actions">
        <button class="bulk-btn bulk-btn-tag" @click="$emit('tag')">
          <TagIcon :size="16" /> Tag
        </button>
        <button class="bulk-btn bulk-btn-sell" @click="$emit('sell')">
          <DollarSign :size="16" /> Mark Sold
        </button>
        <button class="bulk-btn bulk-btn-delete" @click="$emit('delete')">
          <Trash2 :size="16" /> Delete
        </button>
      </div>
    </div>
  </Transition>
</template>

<script setup lang="ts">
import { Tag as TagIcon, DollarSign, Trash2 } from 'lucide-vue-next'

defineProps<{
  visible: boolean
  selectedCount: number
}>()

defineEmits<{
  tag: []
  sell: []
  delete: []
}>()
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

.bulk-btn-delete:hover {
  border-color: #ef4444;
  color: #ef4444;
}

.bulk-btn-sell:hover {
  border-color: #10b981;
  color: #10b981;
}

.bar-slide-enter-active,
.bar-slide-leave-active {
  transition: all 0.25s ease;
}
.bar-slide-enter-from,
.bar-slide-leave-to {
  opacity: 0;
  transform: translateX(-50%) translateY(20px);
}
</style>
