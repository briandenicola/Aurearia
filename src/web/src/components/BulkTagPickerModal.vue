<template>
  <Teleport to="body">
    <div v-if="open" class="fixed inset-0 z-[300] flex items-center justify-center bg-[rgba(0,0,0,0.6)] px-4" @click="$emit('close')">
      <div class="w-full max-w-[320px] rounded-md border border-border-subtle bg-card p-6 shadow-[0_12px_40px_rgba(0,0,0,0.5)]" @click.stop>
        <h3 class="mb-3 text-base text-heading">Apply Set</h3>
        <div v-if="tags.length" class="flex max-h-[300px] flex-col gap-1.5 overflow-y-auto">
          <button
            v-for="tag in tags"
            :key="tag.id"
            class="flex items-center gap-2 rounded-sm border border-border-subtle bg-surface px-3 py-2 text-left text-body text-text-primary transition-colors hover:border-gold hover:text-gold"
            @click="$emit('select', tag.filterValue)"
          >
            <span class="h-3 w-3 shrink-0 rounded-full" :style="{ background: tag.color }"></span>
            {{ tag.name }}
          </button>
        </div>
        <p v-else class="text-body text-text-muted">No sets available. Create sets first.</p>
        <button class="btn btn-secondary btn-sm mt-3" @click="$emit('close')">Cancel</button>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import type { CollectionSetOption } from '@/types'

defineProps<{
  open: boolean
  tags: CollectionSetOption[]
}>()

defineEmits<{
  select: [target: string]
  close: []
}>()
</script>
