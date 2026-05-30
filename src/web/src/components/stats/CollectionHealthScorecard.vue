<template>
  <div class="health-scorecard card">
    <div class="card-header">
      <h3>Collection Health</h3>
    </div>
    <div class="scorecard-content">
      <div class="score-display">
        <div class="score-value">{{ summary.score }}</div>
        <div class="score-grade" :class="`grade-${summary.grade.toLowerCase()}`">
          Grade {{ summary.grade }}
        </div>
      </div>
      
      <div class="dimensions-grid">
        <div
          v-for="(value, key) in summary.dimensions"
          :key="key"
          class="dimension-item"
        >
          <div class="dimension-label">{{ formatDimensionLabel(key) }}</div>
          <div class="dimension-bar">
            <div
              class="dimension-fill"
              :class="`fill-${key}`"
              :style="{ width: `${value}%` }"
            ></div>
          </div>
          <div class="dimension-value">{{ value }}%</div>
        </div>
      </div>

      <div class="weights-info">
        <div class="info-label">Scoring Weights</div>
        <div class="weights-grid">
          <span
            v-for="(value, key) in summary.weights"
            :key="key"
            class="weight-chip"
          >
            {{ formatDimensionLabel(key) }}: {{ value }}%
          </span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { CollectionHealthSummary } from '@/types'

defineProps<{
  summary: CollectionHealthSummary
}>()

function formatDimensionLabel(key: string): string {
  const labels: Record<string, string> = {
    metadata: 'Metadata',
    images: 'Images',
    valuation: 'Valuation',
    ai: 'AI Analysis',
  }
  return labels[key] || key
}
</script>

<style scoped>
.health-scorecard {
  background: var(--bg-card);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  padding: 1.5rem;
  box-shadow: var(--shadow-card);
}

.card-header h3 {
  margin-bottom: 1.25rem;
  color: var(--text-heading);
  font-size: 1.2rem;
}

.scorecard-content {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.score-display {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.5rem;
  padding: 1rem;
  background: var(--bg-input);
  border-radius: var(--radius-sm);
}

.score-value {
  font-size: 3rem;
  font-weight: 600;
  font-family: 'Cinzel', serif;
  color: var(--accent-gold);
}

.score-grade {
  font-size: 0.9rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  padding: 0.25rem 0.8rem;
  border-radius: var(--radius-full);
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

.dimensions-grid {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.dimension-item {
  display: grid;
  grid-template-columns: 100px 1fr 50px;
  align-items: center;
  gap: 0.75rem;
}

.dimension-label {
  font-size: 0.85rem;
  color: var(--text-secondary);
  font-weight: 500;
}

.dimension-bar {
  height: 8px;
  background: var(--bg-input);
  border-radius: var(--radius-full);
  overflow: hidden;
}

.dimension-fill {
  height: 100%;
  border-radius: var(--radius-full);
  transition: width var(--transition-med);
}

.fill-metadata {
  background: linear-gradient(90deg, var(--accent-gold), var(--accent-bronze));
}

.fill-images {
  background: linear-gradient(90deg, #3498db, #2980b9);
}

.fill-valuation {
  background: linear-gradient(90deg, #27ae60, #229954);
}

.fill-ai {
  background: linear-gradient(90deg, #9b59b6, #8e44ad);
}

.dimension-value {
  font-size: 0.85rem;
  font-weight: 600;
  color: var(--text-primary);
  text-align: right;
}

.weights-info {
  padding-top: 0.75rem;
  border-top: 1px solid var(--border-subtle);
}

.info-label {
  font-size: 0.7rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: var(--text-muted);
  margin-bottom: 0.5rem;
}

.weights-grid {
  display: flex;
  flex-wrap: wrap;
  gap: 0.35rem;
}

.weight-chip {
  font-size: 0.75rem;
  padding: 0.15rem 0.5rem;
  border-radius: var(--radius-full);
  border: 1px solid var(--border-subtle);
  background: var(--accent-gold-glow);
  color: var(--text-secondary);
}

@media (max-width: 768px) {
  .dimension-item {
    grid-template-columns: 80px 1fr 45px;
    gap: 0.5rem;
  }

  .score-value {
    font-size: 2.5rem;
  }
}
</style>
