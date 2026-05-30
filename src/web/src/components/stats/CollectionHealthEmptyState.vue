<template>
  <div class="health-empty-state card">
    <div class="empty-icon">
      <FileQuestion :size="48" />
    </div>
    <h3>No Active Coins to Score</h3>
    <p>
      Collection health scoring requires active coins (not wishlist or sold items).
    </p>
    <p class="empty-hint">
      Add coins to your collection or mark wishlist items as acquired to start tracking
      collection health metrics.
    </p>
    <router-link to="/add" class="btn btn-primary">
      <CirclePlus :size="16" /> Add Your First Coin
    </router-link>
  </div>
</template>

<script setup lang="ts">
import { FileQuestion, CirclePlus } from 'lucide-vue-next'
</script>

<style scoped>
.health-empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 1rem;
  padding: 3rem 2rem;
  background: var(--bg-card);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  box-shadow: var(--shadow-card);
  text-align: center;
}

.empty-icon {
  color: var(--text-muted);
  opacity: 0.6;
  margin-bottom: 0.5rem;
}

.health-empty-state h3 {
  color: var(--text-heading);
  margin: 0;
  font-size: 1.2rem;
}

.health-empty-state p {
  color: var(--text-secondary);
  font-size: 0.9rem;
  line-height: 1.6;
  margin: 0;
  max-width: 500px;
}

.empty-hint {
  font-size: 0.85rem;
  color: var(--text-muted);
}

.health-empty-state .btn {
  margin-top: 0.5rem;
}

@media (max-width: 768px) {
  .health-empty-state {
    padding: 2rem 1.5rem;
  }
}
</style>
