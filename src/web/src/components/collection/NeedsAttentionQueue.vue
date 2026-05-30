<template>
  <div class="needs-attention-queue">
    <div class="queue-header">
      <h3>
        <AlertCircle :size="20" />
        Needs Attention
      </h3>
      <div v-if="total > 0" class="queue-count">{{ total }} coins</div>
    </div>

    <div v-if="loading" class="queue-loading">
      <div class="spinner"></div>
    </div>

    <div v-else-if="coins.length === 0" class="queue-empty">
      <CircleCheck :size="32" />
      <p>All coins are in good health</p>
    </div>

    <div v-else class="queue-content">
      <div class="queue-list">
        <div
          v-for="coin in coins"
          :key="coin.coinId"
          class="queue-item"
          @click="handleCoinClick(coin.coinId)"
        >
          <div class="coin-basic">
            <router-link :to="`/coins/${coin.coinId}`" class="coin-name">
              {{ coin.title || `Coin #${coin.coinId}` }}
            </router-link>
          </div>

          <div class="health-info">
            <div class="health-badge" :class="`grade-${coin.grade.toLowerCase()}`">
              {{ coin.score }}
              <span class="grade-letter">{{ coin.grade }}</span>
            </div>
            <div class="missing-count">
              {{ coin.missingItems.length }} issues
            </div>
          </div>

          <div class="quick-actions-inline">
            <button
              v-for="action in coin.quickActions.slice(0, 2)"
              :key="action"
              class="btn-xs btn-ghost"
              @click.stop="handleQuickAction(coin.coinId, action)"
            >
              {{ formatQuickAction(action) }}
            </button>
          </div>
        </div>
      </div>

      <div v-if="total > coins.length" class="queue-pagination">
        <button
          class="btn btn-sm btn-secondary"
          :disabled="page === 1"
          @click="emit('pageChange', page - 1)"
        >
          <ChevronLeft :size="16" /> Previous
        </button>
        <span class="page-info">
          Page {{ page }} of {{ Math.ceil(total / limit) }}
        </span>
        <button
          class="btn btn-sm btn-secondary"
          :disabled="page * limit >= total"
          @click="emit('pageChange', page + 1)"
        >
          Next <ChevronRight :size="16" />
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router'
import { AlertCircle, CircleCheck, ChevronLeft, ChevronRight } from 'lucide-vue-next'
import type { CoinHealthItem, HealthQuickAction } from '@/types'

defineProps<{
  coins: CoinHealthItem[]
  loading: boolean
  total: number
  page: number
  limit: number
}>()

const emit = defineEmits<{
  quickAction: [coinId: number, action: HealthQuickAction]
  pageChange: [page: number]
}>()

const router = useRouter()

function handleCoinClick(coinId: number) {
  router.push(`/coins/${coinId}`)
}

function formatQuickAction(action: HealthQuickAction): string {
  const labels: Record<HealthQuickAction, string> = {
    edit_metadata: 'Edit',
    upload_images: 'Upload',
    run_valuation: 'Valuate',
    run_ai_analysis: 'Analyze',
  }
  return labels[action] || action
}

function handleQuickAction(coinId: number, action: HealthQuickAction) {
  emit('quickAction', coinId, action)
}
</script>

<style scoped>
.needs-attention-queue {
  background: var(--bg-card);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  padding: 1.5rem;
  box-shadow: var(--shadow-card);
}

.queue-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 1.25rem;
}

.queue-header h3 {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  margin: 0;
  color: var(--text-heading);
  font-size: 1.2rem;
}

.queue-count {
  font-size: 0.85rem;
  color: var(--text-secondary);
  padding: 0.25rem 0.7rem;
  background: var(--accent-gold-glow);
  border-radius: var(--radius-full);
  border: 1px solid var(--border-subtle);
}

.queue-loading {
  display: flex;
  justify-content: center;
  padding: 3rem;
}

.spinner {
  width: 32px;
  height: 32px;
  border: 3px solid var(--border-subtle);
  border-top-color: var(--accent-gold);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.queue-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.75rem;
  padding: 3rem 2rem;
  color: var(--text-secondary);
  text-align: center;
}

.queue-empty p {
  margin: 0;
  font-size: 0.9rem;
}

.queue-list {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.queue-item {
  display: grid;
  grid-template-columns: 1fr auto auto;
  align-items: center;
  gap: 1rem;
  padding: 0.75rem;
  background: var(--bg-input);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition: all var(--transition-fast);
}

.queue-item:hover {
  border-color: var(--border-accent);
  background: var(--bg-card-hover);
  box-shadow: var(--shadow-glow);
}

.coin-basic {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.coin-name {
  font-size: 0.9rem;
  font-weight: 600;
  color: var(--text-primary);
  text-decoration: none;
  transition: color var(--transition-fast);
}

.coin-name:hover {
  color: var(--accent-gold);
}

.coin-category {
  font-size: 0.75rem;
  color: var(--text-muted);
}

.health-info {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 0.25rem;
}

.health-badge {
  display: flex;
  align-items: center;
  gap: 0.35rem;
  padding: 0.25rem 0.6rem;
  border-radius: var(--radius-full);
  font-size: 0.8rem;
  font-weight: 600;
  border: 1px solid;
}

.grade-a {
  color: #27ae60;
  border-color: rgba(39, 174, 96, 0.3);
  background: rgba(39, 174, 96, 0.15);
}

.grade-b {
  color: #3498db;
  border-color: rgba(52, 152, 219, 0.3);
  background: rgba(52, 152, 219, 0.15);
}

.grade-c {
  color: #f39c12;
  border-color: rgba(243, 156, 18, 0.3);
  background: rgba(243, 156, 18, 0.15);
}

.grade-d {
  color: #e67e22;
  border-color: rgba(230, 126, 34, 0.3);
  background: rgba(230, 126, 34, 0.15);
}

.grade-f {
  color: #e74c3c;
  border-color: rgba(231, 76, 60, 0.3);
  background: rgba(231, 76, 60, 0.15);
}

.grade-letter {
  font-size: 0.7rem;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.missing-count {
  font-size: 0.75rem;
  color: var(--text-muted);
}

.quick-actions-inline {
  display: flex;
  gap: 0.35rem;
}

.queue-pagination {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: 1rem;
  padding-top: 1rem;
  border-top: 1px solid var(--border-subtle);
}

.page-info {
  font-size: 0.85rem;
  color: var(--text-secondary);
}

@media (max-width: 768px) {
  .queue-item {
    grid-template-columns: 1fr;
    gap: 0.75rem;
  }

  .health-info {
    flex-direction: row;
    align-items: center;
    justify-content: space-between;
    width: 100%;
  }

  .quick-actions-inline {
    width: 100%;
    justify-content: flex-start;
  }

  .queue-pagination {
    flex-direction: column;
    gap: 0.75rem;
  }
}
</style>
