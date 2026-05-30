<template>
  <div class="admin-health-section">
    <div class="section-header">
      <h2>Collection Health</h2>
      <p class="section-description">
        Aggregate quality metrics across all active user collections.
      </p>
    </div>

    <div v-if="loading" class="loading-overlay">
      <div class="spinner"></div>
    </div>

    <div v-else-if="error" class="error-message">
      <AlertCircle :size="20" />
      <span>{{ error }}</span>
    </div>

    <div v-else-if="summary" class="metrics-grid">
      <div class="metric-card">
        <div class="metric-icon median">
          <Activity :size="24" />
        </div>
        <div class="metric-content">
          <div class="metric-label">Median Score</div>
          <div class="metric-value">{{ summary.medianScore }}</div>
          <div class="metric-detail">
          Across {{ summary.eligibleCoinCount }} active coins
          </div>
        </div>
      </div>

      <div class="metric-card">
        <div class="metric-icon low-score">
          <AlertTriangle :size="24" />
        </div>
        <div class="metric-content">
          <div class="metric-label">Low-Score Coins</div>
          <div class="metric-value">{{ summary.lowScorePercentage }}%</div>
          <div class="metric-detail">
          Below {{ summary.lowScoreThreshold }}
          </div>
        </div>
      </div>

      <div class="metric-card wide">
        <div class="metric-icon missing">
          <FileWarning :size="24" />
        </div>
        <div class="metric-content">
          <div class="metric-label">Top Missing Fields</div>
          <div v-if="summary.topMissingFields.length === 0" class="metric-empty">
            No missing fields detected
          </div>
          <div v-else class="missing-fields-list">
            <div
              v-for="(field, idx) in summary.topMissingFields"
              :key="idx"
              class="missing-field-item"
            >
              <span class="field-key">{{ field.key }}</span>
              <span class="field-count chip-sm">{{ field.count }} coins</span>
            </div>
          </div>
        </div>
      </div>
    </div>

    <div v-else class="empty-state">
      <FileQuestion :size="48" />
      <p>No collection health data available</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { AlertCircle, Activity, AlertTriangle, FileWarning, FileQuestion } from 'lucide-vue-next'
import * as api from '@/api/client'
import type { AdminHealthSummaryResponse } from '@/types'

const summary = ref<AdminHealthSummaryResponse | null>(null)
const loading = ref(false)
const error = ref('')

onMounted(() => {
  fetchAdminHealth()
})

async function fetchAdminHealth() {
  loading.value = true
  error.value = ''
  try {
    const res = await api.getAdminHealthSummary()
    summary.value = res.data
  } catch (err) {
    error.value = 'Failed to load health metrics'
    console.error('Admin health fetch error:', err)
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.admin-health-section {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.section-header h2 {
  margin: 0 0 0.5rem 0;
  color: var(--text-heading);
  font-size: 1.5rem;
}

.section-description {
  margin: 0;
  font-size: 0.9rem;
  color: var(--text-secondary);
  line-height: 1.5;
}

.loading-overlay {
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

.error-message {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 1rem;
  background: rgba(231, 76, 60, 0.15);
  border: 1px solid rgba(231, 76, 60, 0.3);
  border-radius: var(--radius-sm);
  color: #e74c3c;
  font-size: 0.9rem;
}

.metrics-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
  gap: 1rem;
}

.metric-card {
  display: flex;
  gap: 1rem;
  padding: 1.5rem;
  background: var(--bg-card);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  box-shadow: var(--shadow-card);
  transition: all var(--transition-fast);
}

.metric-card.wide {
  grid-column: 1 / -1;
}

.metric-card:hover {
  border-color: var(--border-accent);
  box-shadow: var(--shadow-glow);
}

.metric-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 48px;
  height: 48px;
  border-radius: var(--radius-sm);
  flex-shrink: 0;
}

.metric-icon.median {
  background: rgba(52, 152, 219, 0.15);
  color: #3498db;
}

.metric-icon.low-score {
  background: rgba(243, 156, 18, 0.15);
  color: #f39c12;
}

.metric-icon.missing {
  background: rgba(231, 76, 60, 0.15);
  color: #e74c3c;
}

.metric-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
}

.metric-label {
  font-size: 0.7rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: var(--text-muted);
}

.metric-value {
  font-size: 2rem;
  font-weight: 700;
  font-family: 'Cinzel', serif;
  color: var(--accent-gold);
  line-height: 1;
}

.metric-detail {
  font-size: 0.85rem;
  color: var(--text-secondary);
}

.metric-empty {
  font-size: 0.85rem;
  color: var(--text-muted);
  font-style: italic;
}

.missing-fields-list {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  margin-top: 0.25rem;
}

.missing-field-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0.5rem 0.75rem;
  background: var(--bg-input);
  border-radius: var(--radius-sm);
  border: 1px solid var(--border-subtle);
}

.field-key {
  font-size: 0.85rem;
  font-weight: 500;
  color: var(--text-primary);
}

.field-count {
  font-size: 0.75rem;
  padding: 0.15rem 0.5rem;
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 1rem;
  padding: 3rem 2rem;
  background: var(--bg-card);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  color: var(--text-muted);
  text-align: center;
}

.empty-state p {
  margin: 0;
  font-size: 0.9rem;
  color: var(--text-secondary);
}

@media (max-width: 768px) {
  .metrics-grid {
    grid-template-columns: 1fr;
  }

  .metric-card.wide {
    grid-column: 1;
  }

  .metric-value {
    font-size: 1.75rem;
  }
}
</style>
