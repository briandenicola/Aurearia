<template>
  <div class="rounded-sm border border-border-subtle bg-card px-4 py-3">
    <div
      v-for="row in rows"
      :key="row.key"
      class="metadata-row"
      :class="{ 'w-full': row.fullWidth }"
    >
      <span v-if="!row.fullWidth" class="row-label">{{ row.label }}</span>

      <!-- Purchase location row with Store: prefix -->
      <template v-if="row.key === 'purchaseLocation'">
        <span class="text-sm italic text-text-muted">Store: </span>
        <span
          v-if="!row.url"
          class="row-value"
          :class="[row.valueClass, row.fullWidth ? 'italic !text-text-secondary !text-left' : '']"
        >{{ row.value }}</span>
        <SafeExternalLink
          v-else
          :href="row.url"
          class="row-value transition-colors"
          :class="[
            row.valueClass,
            row.fullWidth ? '!text-gold italic !text-left hover:text-bronze' : '',
          ]"
        >
          {{ row.value }}
        </SafeExternalLink>
      </template>

      <!-- Standard rows -->
      <template v-else>
        <span
          v-if="!row.url"
          class="row-value"
          :class="[row.valueClass, row.fullWidth ? 'italic !text-text-secondary !text-left' : '']"
        >{{ row.value }}</span>
        <SafeExternalLink
          v-else
          :href="row.url"
          class="row-value transition-colors"
          :class="[
            row.valueClass,
            row.fullWidth ? '!text-gold italic !text-left hover:text-bronze' : '',
          ]"
        >
          {{ row.value }}
        </SafeExternalLink>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { CoinDetailMetadataRow } from '@/types'
import SafeExternalLink from '@/components/SafeExternalLink.vue'

defineProps<{
  rows: CoinDetailMetadataRow[]
}>()
</script>
