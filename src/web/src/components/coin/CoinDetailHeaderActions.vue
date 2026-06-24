<template>
  <div class="detail-header">
    <button class="btn btn-ghost btn-xs back-action" @click="router.push('/')">
      <ArrowLeft :size="14" />
      Back to Gallery
    </button>
    <div class="detail-actions">
      <button
        class="icon-action"
        :disabled="sharing"
        :title="sharing ? 'Sharing...' : 'Share'"
        :aria-label="sharing ? 'Sharing...' : 'Share'"
        @click="$emit('share')"
      >
        <Share2 :size="18" />
      </button>
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
      <button
        class="icon-action"
        :disabled="duplicating"
        :title="duplicating ? 'Duplicating...' : 'Duplicate'"
        :aria-label="duplicating ? 'Duplicating...' : 'Duplicate'"
        @click="$emit('duplicate')"
      >
        <Copy :size="18" />
      </button>
      <button class="icon-action" title="Delete" aria-label="Delete" @click="$emit('delete')">
        <Trash2 :size="18" />
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router'
import { ArrowLeft, CircleDollarSign, Copy, Pencil, Share2, Trash2 } from 'lucide-vue-next'

withDefaults(defineProps<{
  isWishlist: boolean
  isSold: boolean
  coinId: number
  sharing?: boolean
  duplicating?: boolean
}>(), {
  sharing: false,
  duplicating: false,
})

defineEmits<{
  share: []
  sell: []
  duplicate: []
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

.detail-actions {
  display: flex;
  justify-content: flex-end;
  align-items: center;
  gap: 0.35rem;
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
  padding: 0.4rem;
  border-radius: var(--radius-sm);
  border: none;
  background: transparent;
  color: var(--text-secondary);
  cursor: pointer;
  text-decoration: none;
  transition: background var(--transition-fast), color var(--transition-fast);
}

.icon-action:hover:not(:disabled) {
  background: var(--accent-gold-glow);
  color: var(--accent-gold);
}

.icon-action:disabled {
  cursor: not-allowed;
  opacity: 0.55;
}

@media (max-width: 768px) {
  .detail-header {
    grid-template-columns: minmax(0, 1fr) auto;
    gap: 0.35rem;
    margin-bottom: 1rem;
  }

  .detail-actions {
    gap: 0.2rem;
  }

  .icon-action {
    padding: 0.35rem;
  }
}
</style>
