<template>
  <div class="mb-6 grid grid-cols-1 gap-3 sm:grid-cols-2">
    <div class="rounded-sm border border-border-subtle bg-card p-3" v-if="purchasePrice != null">
      <span class="section-label !mb-[0.2rem] block">Purchase Price</span>
      <span class="text-[0.95rem] font-medium text-text-primary">{{ formatCurrency(purchasePrice) }}</span>
    </div>
    <div class="rounded-sm border border-border-subtle bg-card p-3" v-if="currentValue != null">
      <span class="section-label !mb-[0.2rem] block">Current Value</span>
      <span class="text-[0.95rem] font-medium text-gold">{{ formatCurrency(currentValue) }}</span>
    </div>
    <div class="rounded-sm border border-border-subtle bg-card p-3" v-if="denomination">
      <span class="section-label !mb-[0.2rem] block">Denomination</span>
      <span class="text-[0.95rem] font-medium text-text-primary">{{ denomination }}</span>
    </div>
    <div class="rounded-sm border border-border-subtle bg-card p-3" v-if="era">
      <span class="section-label !mb-[0.2rem] block">Era</span>
      <span class="text-[0.95rem] font-medium text-text-primary">{{ era }}</span>
    </div>
    <div class="rounded-sm border border-border-subtle bg-card p-3" v-if="mint">
      <span class="section-label !mb-[0.2rem] block">Mint</span>
      <span class="text-[0.95rem] font-medium text-text-primary">{{ mint }}</span>
    </div>
    <div class="rounded-sm border border-border-subtle bg-card p-3" v-if="material">
      <span class="section-label !mb-[0.2rem] block">Material</span>
      <span class="text-[0.95rem] font-medium" :class="`material-${material.toLowerCase()}`">{{ material }}</span>
    </div>
    <div class="rounded-sm border border-border-subtle bg-card p-3" v-if="weightGrams != null">
      <span class="section-label !mb-[0.2rem] block">Weight</span>
      <span class="text-[0.95rem] font-medium text-text-primary">{{ weightGrams }}g</span>
    </div>
    <div class="rounded-sm border border-border-subtle bg-card p-3" v-if="diameterMm != null">
      <span class="section-label !mb-[0.2rem] block">Diameter</span>
      <span class="text-[0.95rem] font-medium text-text-primary">{{ diameterMm }}mm</span>
    </div>
    <div class="rounded-sm border border-border-subtle bg-card p-3" v-if="grade">
      <span class="section-label !mb-[0.2rem] block">Grade</span>
      <span class="text-[0.95rem] font-medium text-gold">{{ grade }}</span>
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
}>()
</script>
