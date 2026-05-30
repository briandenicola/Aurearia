<template>
  <div class="health-trend card">
    <div class="trend-header">
      <h4>30-Day Trend</h4>
      <div v-if="trend.direction === 'unavailable'" class="trend-badge unavailable">
        Insufficient Data
      </div>
      <div v-else class="trend-badge" :class="`trend-${trend.direction}`">
        <component :is="trendIcon" :size="14" />
        {{ formatTrend() }}
      </div>
    </div>
    <div v-if="trend.direction !== 'unavailable'" class="trend-details">
      <div class="trend-item">
        <span class="trend-label">Change</span>
        <span class="trend-value">{{ formatDelta() }}</span>
      </div>
    </div>
    <p v-else class="trend-help">
      Trend data will be available after 30 days of collection tracking.
    </p>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { TrendingUp, TrendingDown, Minus } from 'lucide-vue-next'
import type { CollectionHealthTrend } from '@/types'

const props = defineProps<{
  trend: CollectionHealthTrend
}>()

const trendIcon = computed(() => {
  switch (props.trend.direction) {
    case 'up':
      return TrendingUp
    case 'down':
      return TrendingDown
    default:
      return Minus
  }
})

function formatTrend(): string {
  switch (props.trend.direction) {
    case 'up':
      return 'Improving'
    case 'down':
      return 'Declining'
    case 'flat':
      return 'Stable'
    default:
      return 'No Data'
  }
}

function formatDelta(): string {
  if (props.trend.delta === null) return 'N/A'
  const sign = props.trend.delta > 0 ? '+' : ''
  return `${sign}${props.trend.delta}`
}
</script>

<style scoped>
.health-trend {
  background: var(--bg-card);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  padding: 1rem;
  box-shadow: var(--shadow-card);
}

.trend-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 1rem;
}

.trend-header h4 {
  color: var(--text-heading);
  font-size: 0.9rem;
  margin: 0;
}

.trend-badge {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  padding: 0.2rem 0.7rem;
  border-radius: var(--radius-full);
  font-size: 0.75rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  border: 1px solid;
}

.trend-up {
  color: #27ae60;
  border-color: rgba(39, 174, 96, 0.3);
  background: rgba(39, 174, 96, 0.15);
}

.trend-down {
  color: #e74c3c;
  border-color: rgba(231, 76, 60, 0.3);
  background: rgba(231, 76, 60, 0.15);
}

.trend-flat {
  color: var(--text-secondary);
  border-color: var(--border-subtle);
  background: var(--accent-gold-glow);
}

.trend-unavailable {
  color: var(--text-muted);
  border-color: var(--border-subtle);
  background: transparent;
}

.trend-details {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.75rem;
  background: var(--bg-input);
  border-radius: var(--radius-sm);
  justify-content: center;
}

.trend-item {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.trend-label {
  font-size: 0.7rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: var(--text-muted);
}

.trend-value {
  font-size: 1.5rem;
  font-weight: 700;
  font-family: 'Cinzel', serif;
  color: var(--accent-gold);
}

.trend-arrow {
  font-size: 1.2rem;
  color: var(--text-muted);
  margin: 0 0.25rem;
}

.trend-delta {
  margin-left: auto;
  font-size: 1rem;
  font-weight: 700;
  padding: 0.25rem 0.6rem;
  border-radius: var(--radius-sm);
}

.delta-up {
  color: #27ae60;
  background: rgba(39, 174, 96, 0.15);
}

.delta-down {
  color: #e74c3c;
  background: rgba(231, 76, 60, 0.15);
}

.delta-flat {
  color: var(--text-secondary);
  background: var(--accent-gold-glow);
}

.trend-help {
  font-size: 0.85rem;
  color: var(--text-secondary);
  line-height: 1.5;
  margin: 0;
}

@media (max-width: 768px) {
  .trend-details {
    gap: 0.5rem;
  }

  .trend-value {
    font-size: 1rem;
  }

  .trend-delta {
    font-size: 0.9rem;
  }
}
</style>
