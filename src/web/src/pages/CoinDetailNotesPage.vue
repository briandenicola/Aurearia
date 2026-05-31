<template>
  <CoinDetailSectionPageShell section-title="Notes">
    <template #default="{ coin }">
      <div v-if="coin.notes" class="notes-content">
        <div class="section-content-card">
          <p>{{ coin.notes }}</p>
        </div>
      </div>
      <div v-else class="empty-state">
        <p>No notes recorded for this coin.</p>
        <router-link :to="`/edit/${coin.id}`" class="btn btn-secondary btn-sm">
          Add Notes
        </router-link>
      </div>
    </template>
  </CoinDetailSectionPageShell>
</template>

<script setup lang="ts">
import CoinDetailSectionPageShell from '@/components/coin/CoinDetailSectionPageShell.vue'
</script>

<style scoped>
.notes-content {
  margin-top: 0;
}

.notes-content p {
  font-size: 0.9rem;
  color: var(--text-secondary);
  white-space: pre-wrap;
  margin-bottom: 0;
}

.empty-state {
  text-align: center;
  padding: 3rem 2rem;
}

.empty-state p {
  color: var(--text-muted);
  margin-bottom: 1.5rem;
}
</style>
