<template>
  <div class="info-grid">
    <div class="info-card" v-if="purchasePrice != null">
      <span class="info-label">Purchase Price</span>
      <span class="info-value">{{ formatCurrency(purchasePrice) }}</span>
    </div>
    <div class="info-card" v-if="currentValue != null">
      <span class="info-label">Current Value</span>
      <span class="info-value gold">{{ formatCurrency(currentValue) }}</span>
    </div>
    <div class="info-card" v-if="denomination">
      <span class="info-label">Denomination</span>
      <span class="info-value">{{ denomination }}</span>
    </div>
    <div class="info-card" v-if="era">
      <span class="info-label">Era</span>
      <span class="info-value">{{ era }}</span>
    </div>
    <div class="info-card" v-if="mint">
      <span class="info-label">Mint</span>
      <span class="info-value">{{ mint }}</span>
    </div>
    <div class="info-card" v-if="material">
      <span class="info-label">Material</span>
      <span class="info-value" :class="`material-${material.toLowerCase()}`">{{ material }}</span>
    </div>
    <div class="info-card" v-if="weightGrams != null">
      <span class="info-label">Weight</span>
      <span class="info-value">{{ weightGrams }}g</span>
    </div>
    <div class="info-card" v-if="diameterMm != null">
      <span class="info-label">Diameter</span>
      <span class="info-value">{{ diameterMm }}mm</span>
    </div>
    <div class="info-card" v-if="grade">
      <span class="info-label">Grade</span>
      <span class="info-value gold">{{ grade }}</span>
    </div>
    <div class="info-card" v-if="rarityRating">
      <span class="info-label">Rarity / RIC</span>
      <span class="info-value">{{ rarityRating }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { formatCurrency } from '@/utils/format'

defineProps<{
  purchasePrice?: number | null
  currentValue?: number | null
  denomination?: string | null
  era?: string | null
  mint?: string | null
  material?: string | null
  weightGrams?: number | null
  diameterMm?: number | null
  grade?: string | null
  rarityRating?: string | null
}>()
</script>

<style scoped>
.info-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 0.75rem;
  margin-bottom: 1.5rem;
}

.info-card {
  background: var(--bg-card);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
  padding: 0.75rem;
}

.info-label {
  display: block;
  font-size: 0.7rem;
  font-weight: 600;
  color: var(--text-muted);
  text-transform: uppercase;
  letter-spacing: 0.08em;
  margin-bottom: 0.2rem;
}

.info-value {
  font-size: 0.95rem;
  font-weight: 500;
}

.info-value.gold {
  color: var(--accent-gold);
}
</style>
