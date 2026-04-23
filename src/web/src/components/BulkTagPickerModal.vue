<template>
  <Teleport to="body">
    <div v-if="open" class="modal-backdrop" @click="$emit('close')">
      <div class="modal-content tag-picker-modal" @click.stop>
        <h3>Apply Tag</h3>
        <div v-if="tags.length" class="tag-picker-list">
          <button
            v-for="tag in tags"
            :key="tag.id"
            class="tag-picker-item"
            @click="$emit('select', tag.id)"
          >
            <span class="tag-swatch" :style="{ background: tag.color }"></span>
            {{ tag.name }}
          </button>
        </div>
        <p v-else class="empty-tags">No tags. Create tags in Settings first.</p>
        <button class="btn btn-secondary btn-sm" style="margin-top: 0.75rem;" @click="$emit('close')">Cancel</button>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import type { Tag } from '@/types'

defineProps<{
  open: boolean
  tags: Tag[]
}>()

defineEmits<{
  select: [tagId: number]
  close: []
}>()
</script>

<style scoped>
.modal-backdrop {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.6);
  z-index: 300;
  display: flex;
  align-items: center;
  justify-content: center;
}

.modal-content {
  background: var(--bg-card);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  padding: 1.5rem;
  max-width: 320px;
  width: 90%;
  box-shadow: 0 12px 40px rgba(0, 0, 0, 0.5);
}

.tag-picker-modal h3 {
  margin-bottom: 0.75rem;
  font-size: 1rem;
}

.tag-picker-list {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
  max-height: 300px;
  overflow-y: auto;
}

.tag-picker-item {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem 0.75rem;
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
  background: var(--bg-primary);
  color: var(--text-primary);
  cursor: pointer;
  font-size: 0.85rem;
  transition: all var(--transition-fast);
}

.tag-picker-item:hover {
  border-color: var(--accent-gold);
  color: var(--accent-gold);
}

.tag-swatch {
  width: 12px;
  height: 12px;
  border-radius: 50%;
  flex-shrink: 0;
}

.empty-tags {
  color: var(--text-muted);
  font-size: 0.85rem;
}
</style>
