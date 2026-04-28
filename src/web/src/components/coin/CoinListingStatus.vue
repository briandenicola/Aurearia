<template>
  <div v-if="listingStatus" class="listing-status-card">
    <div class="listing-status-header">
      <span
        class="listing-status-badge"
        :class="{
          'listing-available': listingStatus === 'available',
          'listing-unavailable': listingStatus === 'unavailable',
          'listing-unknown': listingStatus === 'unknown',
        }"
      >{{ listingStatus === 'available' ? 'Available' : listingStatus === 'unavailable' ? 'Unavailable' : 'Unknown' }}</span>
      <button class="btn btn-ghost btn-xs" @click="handleDismiss">Dismiss</button>
    </div>
    <p v-if="listingCheckReason" class="listing-reason">{{ listingCheckReason }}</p>
    <p v-if="listingCheckedAt" class="listing-checked-at">Last checked: {{ formatDate(listingCheckedAt) }}</p>
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

<style scoped>
.listing-status-card {
  margin-top: 1rem;
  padding: 0.75rem 1rem;
  background: var(--bg-primary);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
}

.listing-status-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 0.35rem;
}

.listing-status-badge {
  display: inline-block;
  padding: 0.2rem 0.6rem;
  border-radius: var(--radius-full);
  font-size: 0.75rem;
  font-weight: 600;
}

.listing-available {
  background: rgba(46, 204, 113, 0.15);
  color: #2ecc71;
}

.listing-unavailable {
  background: rgba(231, 76, 60, 0.15);
  color: #e74c3c;
}

.listing-unknown {
  background: rgba(241, 196, 15, 0.15);
  color: #f1c40f;
}

.listing-reason {
  font-size: 0.82rem;
  color: var(--text-secondary);
  margin: 0.25rem 0;
}

.listing-checked-at {
  font-size: 0.75rem;
  color: var(--text-muted);
  margin: 0;
}
</style>
