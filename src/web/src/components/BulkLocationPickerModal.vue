<template>
  <Teleport to="body">
    <div v-if="open" class="modal-backdrop" @click="$emit('close')">
      <div class="modal-content location-picker-modal" @click.stop>
        <h3>Assign Location</h3>
        <div v-if="locations.length" class="location-picker-list">
          <button
            class="location-picker-item location-picker-clear"
            @click="$emit('select', null)"
          >
            <span class="location-icon">—</span>
            No location
          </button>
          <button
            v-for="location in locations"
            :key="location.id"
            class="location-picker-item"
            @click="$emit('select', location.id)"
          >
            <span class="location-icon"><MapPin :size="14" /></span>
            {{ location.name }}
          </button>
        </div>
        <p v-else class="empty-locations">No storage locations. Create them in Settings first.</p>
        <button class="btn btn-secondary btn-sm" style="margin-top: 0.75rem;" @click="$emit('close')">Cancel</button>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { MapPin } from 'lucide-vue-next'
import type { StorageLocation } from '@/types'

defineProps<{
  open: boolean
  locations: StorageLocation[]
}>()

defineEmits<{
  select: [locationId: number | null]
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

.location-picker-modal h3 {
  margin-bottom: 0.75rem;
  font-size: 1rem;
}

.location-picker-list {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
  max-height: 300px;
  overflow-y: auto;
}

.location-picker-item {
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

.location-picker-item:hover {
  border-color: var(--accent-gold);
  color: var(--accent-gold);
}

.location-picker-clear {
  font-style: italic;
  color: var(--text-secondary);
}

.location-picker-clear:hover {
  color: var(--accent-gold);
}

.location-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.empty-locations {
  color: var(--text-muted);
  font-size: 0.85rem;
}
</style>
