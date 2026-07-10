<template>
  <div v-if="listingStatus" class="mt-4 rounded-md border border-border-subtle bg-surface px-4 py-3">
    <div class="mb-[0.35rem] flex items-center justify-between gap-3">
      <span
        class="inline-block rounded-full px-[0.6rem] py-[0.2rem] text-sm font-semibold"
        :class="{
          'bg-gain/15 text-gain': listingStatus === 'available',
          'bg-loss/15 text-loss': listingStatus === 'unavailable',
          'bg-warning/15 text-warning': listingStatus === 'unknown',
        }"
      >{{ listingStatus === 'available' ? 'Available' : listingStatus === 'unavailable' ? 'Unavailable' : 'Unknown' }}</span>
      <button class="btn btn-ghost btn-xs" @click="handleDismiss">Dismiss</button>
    </div>
    <p v-if="listingCheckReason" class="my-1 text-[0.82rem] text-text-secondary">{{ listingCheckReason }}</p>
    <p v-if="listingCheckedAt" class="m-0 text-sm text-text-muted">Last checked: {{ formatDate(listingCheckedAt) }}</p>
  </div>
</template>

<script setup lang="ts">
import { updateListingStatus } from '@/api/client'

const props = defineProps<{
  coinId: number
  listingStatus?: string | null
  listingCheckReason?: string | null
  listingCheckedAt?: string | null
}>()

const emit = defineEmits<{
  dismissed: []
}>()

function formatDate(dateStr: string) {
  return new Date(dateStr).toLocaleDateString(undefined, {
    month: 'short', day: 'numeric', year: 'numeric', hour: '2-digit', minute: '2-digit',
  })
}

async function handleDismiss() {
  try {
    await updateListingStatus(props.coinId, '')
    emit('dismissed')
  } catch {
    // silently fail
  }
}
</script>
