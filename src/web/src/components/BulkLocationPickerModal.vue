<template>
  <Teleport to="body">
    <div v-if="open" class="fixed inset-0 z-[300] flex items-center justify-center bg-[rgba(0,0,0,0.6)] px-4" @click="$emit('close')">
      <div class="w-full max-w-[320px] rounded-md border border-border-subtle bg-card p-6 shadow-[0_12px_40px_rgba(0,0,0,0.5)]" @click.stop>
        <h3 class="mb-3 text-base text-heading">Assign Location</h3>
        <div v-if="locations.length" class="flex max-h-[300px] flex-col gap-1.5 overflow-y-auto">
          <button
            class="flex items-center gap-2 rounded-sm border border-border-subtle bg-surface px-3 py-2 text-left text-body italic text-text-secondary transition-colors hover:border-gold hover:text-gold"
            @click="$emit('select', null)"
          >
            <span class="flex shrink-0 items-center justify-center">—</span>
            No location
          </button>
          <button
            v-for="location in locations"
            :key="location.id"
            class="flex items-center gap-2 rounded-sm border border-border-subtle bg-surface px-3 py-2 text-left text-body text-text-primary transition-colors hover:border-gold hover:text-gold"
            @click="$emit('select', location.id)"
          >
            <span class="flex shrink-0 items-center justify-center"><MapPin :size="14" /></span>
            {{ location.name }}
          </button>
        </div>
        <p v-else class="text-body text-text-muted">No storage locations. Create them in Settings first.</p>
        <button class="btn btn-secondary btn-sm mt-3" @click="$emit('close')">Cancel</button>
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
