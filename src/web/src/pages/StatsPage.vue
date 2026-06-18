<template>
  <div class="container stats-landing">
    <header class="page-header stats-landing-header">
      <div>
        <p class="section-label">Collection Insights</p>
        <h1>Stats</h1>
        <p class="page-intro">Choose a focused view for collection geography, chronology, and distribution.</p>
      </div>
    </header>

    <section class="stats-card-grid" aria-label="Stats views">
      <router-link
        v-for="card in statsCards"
        :key="card.to"
        class="stats-nav-card card"
        :to="card.to"
      >
        <component :is="card.icon" class="card-icon" :size="24" />
        <span class="section-label">{{ card.label }}</span>
        <h2>{{ card.title }}</h2>
        <p>{{ card.description }}</p>
        <span class="card-action">Open {{ card.title }}</span>
      </router-link>
    </section>
  </div>
</template>

<script setup lang="ts">
import { markRaw } from 'vue'
import { Activity, BarChart3, Clock, HeartPulse, LineChart, MapPinned } from 'lucide-vue-next'

const statsCards = [
  {
    label: 'Collection Geography',
    title: 'Mint Map',
    description: 'Plot matched mint names from your active collection on a real OpenStreetMap view.',
    to: '/stats/mint-map',
    icon: markRaw(MapPinned),
  },
  {
    label: 'Acquisition History',
    title: 'Timeline',
    description: 'Review collection, sold, and all-coin acquisition history by month.',
    to: '/stats/timeline',
    icon: markRaw(Clock),
  },
  {
    label: 'Collection Makeup',
    title: 'Collection Distribution',
    description: 'Compare category, material, era, ruler, price, and heat-map distribution.',
    to: '/stats/distribution',
    icon: markRaw(BarChart3),
  },
  {
    label: 'Collection Quality',
    title: 'Collection Health',
    description: 'Inspect the health scorecard and 30-day trend in the distribution view.',
    to: '/stats/distribution#collection-health',
    icon: markRaw(HeartPulse),
  },
  {
    label: 'Portfolio Trend',
    title: 'Value Over Time',
    description: 'Review invested and current value history in the distribution view.',
    to: '/stats/distribution#value-over-time',
    icon: markRaw(LineChart),
  },
  {
    label: 'At a Glance',
    title: 'Summary Cards',
    description: 'See the existing collection summary metrics in the distribution view.',
    to: '/stats/distribution#summary',
    icon: markRaw(Activity),
  },
] as const
</script>

<style scoped>
.stats-landing {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.stats-landing-header {
  margin-bottom: 0;
}

.page-intro {
  margin: 0.35rem 0 0;
  color: var(--text-secondary);
  font-size: 0.9rem;
}

.stats-card-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
  gap: 1rem;
}

.stats-nav-card {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  padding: 1.5rem;
  color: var(--text-primary);
  text-decoration: none;
  transition: transform var(--transition-fast), box-shadow var(--transition-fast), border-color var(--transition-fast);
}

.stats-nav-card:hover,
.stats-nav-card:focus-visible {
  border-color: var(--accent-gold);
  box-shadow: var(--shadow-glow);
  transform: translateY(-2px);
}

.card-icon {
  color: var(--accent-gold);
}

.stats-nav-card h2 {
  margin: 0;
}

.stats-nav-card p {
  flex: 1;
  margin: 0;
  color: var(--text-secondary);
  font-size: 0.9rem;
  line-height: 1.5;
}

.card-action {
  color: var(--accent-gold);
  font-size: 0.85rem;
  font-weight: 600;
}
</style>
