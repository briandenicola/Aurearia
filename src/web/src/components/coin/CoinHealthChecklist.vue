<template>
  <div class="coin-health-checklist card">
    <div class="checklist-header">
      <div class="health-score-badge" :class="`grade-${grade.toLowerCase()}`">
        <span class="score-label">Health Score</span>
        <span class="score-value">{{ score }}</span>
        <span class="score-grade">{{ grade }}</span>
      </div>
    </div>

    <div v-if="missingItems.length === 0" class="checklist-complete">
      <CircleCheck :size="20" />
      <span>All quality checks passed</span>
    </div>

    <div v-else class="checklist-content">
      <div class="checklist-title">
        <AlertCircle :size="16" />
        Needs Attention ({{ missingItems.length }})
      </div>

      <div class="checklist-items">
        <div
          v-for="(item, idx) in missingItems"
          :key="idx"
          class="checklist-item"
          :class="`severity-${item.severity}`"
        >
          <div class="item-icon">
            <component :is="getSeverityIcon(item.severity)" :size="14" />
          </div>
          <div class="item-details">
            <div class="item-key">{{ item.label }}</div>
            <div class="item-dimension">{{ formatDimension(item.dimension) }}</div>
          </div>
          <button
            v-if="item.actionHint"
            class="btn-xs btn-ghost quick-action"
            @click="handleQuickAction(item.actionHint)"
          >
            {{ formatQuickAction(item.actionHint) }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { AlertCircle, CircleCheck, AlertTriangle, Info, AlertOctagon } from 'lucide-vue-next'
import type { HealthGrade, MissingChecklistItem, HealthQuickAction } from '@/types'

defineProps<{
  score: number
  grade: HealthGrade
  missingItems: MissingChecklistItem[]
}>()

const emit = defineEmits<{
  quickAction: [action: HealthQuickAction]
}>()

function getSeverityIcon(severity: string) {
  switch (severity) {
    case 'high':
      return AlertOctagon
    case 'medium':
      return AlertTriangle
    default:
      return Info
  }
}

function formatDimension(dimension: string): string {
  const labels: Record<string, string> = {
    metadata: 'Metadata',
    images: 'Images',
    valuation: 'Valuation',
    ai: 'AI Analysis',
  }
  return labels[dimension] || dimension
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

function handleQuickAction(action: HealthQuickAction) {
  emit('quickAction', action)
}
</script>

<style scoped>
.coin-health-checklist {
  background: var(--bg-card);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  padding: 1rem;
  box-shadow: var(--shadow-card);
}

.checklist-header {
  margin-bottom: 1rem;
}

.health-score-badge {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem 1rem;
  border-radius: var(--radius-sm);
  border: 1px solid;
  font-size: 0.85rem;
  font-weight: 600;
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

.score-label {
  font-size: 0.7rem;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  opacity: 0.8;
}

.score-value {
  font-size: 1.1rem;
  font-weight: 700;
}

.score-grade {
  font-size: 0.8rem;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.checklist-complete {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.75rem;
  background: rgba(39, 174, 96, 0.15);
  border: 1px solid rgba(39, 174, 96, 0.3);
  border-radius: var(--radius-sm);
  color: #27ae60;
  font-size: 0.85rem;
  font-weight: 500;
}

.checklist-content {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.checklist-title {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.85rem;
  font-weight: 600;
  color: var(--text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.checklist-items {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.checklist-item {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.75rem;
  background: var(--bg-input);
  border-radius: var(--radius-sm);
  border: 1px solid var(--border-subtle);
}

.severity-high {
  border-left: 3px solid #e74c3c;
}

.severity-medium {
  border-left: 3px solid #f39c12;
}

.severity-low {
  border-left: 3px solid #3498db;
}

.item-icon {
  flex-shrink: 0;
  color: var(--text-muted);
}

.severity-high .item-icon {
  color: #e74c3c;
}

.severity-medium .item-icon {
  color: #f39c12;
}

.severity-low .item-icon {
  color: #3498db;
}

.item-details {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 0.15rem;
}

.item-key {
  font-size: 0.85rem;
  font-weight: 500;
  color: var(--text-primary);
}

.item-dimension {
  font-size: 0.75rem;
  color: var(--text-muted);
}

.quick-action {
  flex-shrink: 0;
}

@media (max-width: 768px) {
  .checklist-item {
    gap: 0.5rem;
    padding: 0.6rem;
  }

  .item-key {
    font-size: 0.8rem;
  }

  .quick-action {
    font-size: 0.7rem;
    padding: 0.2rem 0.5rem;
  }
}
</style>
