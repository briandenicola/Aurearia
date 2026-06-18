<template>
  <div class="detail-header">
    <div class="detail-navigation">
      <button class="btn btn-ghost btn-xs back-action" @click="router.push('/')">
        <ArrowLeft :size="14" />
        Back to Gallery
      </button>
      <button
        class="icon-action"
        :disabled="sharing"
        :title="sharing ? 'Sharing...' : 'Share'"
        :aria-label="sharing ? 'Sharing...' : 'Share'"
        @click="$emit('share')"
      >
        <Share2 :size="18" />
      </button>
    </div>
    <div class="detail-actions">
      <button
        v-if="!isWishlist && !isSold"
        class="icon-action"
        title="Sell"
        aria-label="Sell"
        @click="$emit('sell')"
      >
        <CircleDollarSign :size="18" />
      </button>
      <router-link :to="`/edit/${coinId}`" class="icon-action" title="Edit" aria-label="Edit">
        <Pencil :size="18" />
      </router-link>
      <button class="icon-action icon-action-danger" title="Delete" aria-label="Delete" @click="$emit('delete')">
        <Trash2 :size="18" />
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router'
import { ArrowLeft, CircleDollarSign, Pencil, Share2, Trash2 } from 'lucide-vue-next'

withDefaults(defineProps<{
  isWishlist: boolean
  isSold: boolean
  coinId: number
  sharing?: boolean
}>(), {
  sharing: false,
})

defineEmits<{
  share: []
  sell: []
  delete: []
}>()

const router = useRouter()
</script>

<style scoped>
.detail-header {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  align-items: center;
  gap: 0.75rem;
  margin-bottom: 0;
}

.detail-navigation {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 0.5rem;
  min-width: 0;
}

.detail-actions {
  display: flex;
  justify-content: flex-end;
  flex-wrap: wrap;
  gap: 0.5rem;
  min-width: 0;
}

.back-action {
  justify-self: start;
  white-space: nowrap;
}

.icon-action {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 2.25rem;
  height: 2.25rem;
  border-radius: var(--radius-full);
  border: 1px solid var(--border-subtle);
  background: var(--bg-card);
  color: var(--text-secondary);
  cursor: pointer;
  transition: background var(--transition-fast), border-color var(--transition-fast), color var(--transition-fast);
}

.icon-action:hover:not(:disabled) {
  background: var(--bg-card-hover);
  border-color: var(--border-accent);
  color: var(--accent-gold);
}

.icon-action:disabled {
  cursor: not-allowed;
  opacity: 0.55;
}

.icon-action-danger {
  color: var(--cat-byzantine);
  border-color: color-mix(in srgb, var(--cat-byzantine) 35%, transparent);
}

.icon-action-danger:hover:not(:disabled) {
  background: color-mix(in srgb, var(--cat-byzantine) 18%, transparent);
  color: var(--cat-byzantine);
  border-color: color-mix(in srgb, var(--cat-byzantine) 55%, transparent);
}

@media (max-width: 768px) {
  .detail-header {
    grid-template-columns: 1fr;
    gap: 0.5rem;
    margin-bottom: 1rem;
  }

  .detail-actions {
    justify-content: flex-start;
    gap: 0.35rem;
  }
}
</style>
