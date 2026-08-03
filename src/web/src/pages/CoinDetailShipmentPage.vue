<template>
  <CoinDetailSectionPageShell section-title="Shipment Tracker">
    <template #default="{ coin }">
      <CoinShipmentSection :coin-id="coin.id" />
    </template>
  </CoinDetailSectionPageShell>
</template>

<script setup lang="ts">
import CoinDetailSectionPageShell from '@/components/coin/CoinDetailSectionPageShell.vue'
import CoinShipmentSection from '@/components/coin/CoinShipmentSection.vue'
</script>
