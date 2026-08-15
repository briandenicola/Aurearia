<template>
  <p v-if="uri" class="ocre-attribution">
    Coin type data:
    <SafeExternalLink :href="uri">Online Coins of the Roman Empire (OCRE)</SafeExternalLink>,
    American Numismatic Society &mdash; <SafeExternalLink href="https://opendatacommons.org/licenses/odbl/1-0/">ODbL 1.0</SafeExternalLink>.
  </p>
</template>

<script setup lang="ts">
import SafeExternalLink from '@/components/SafeExternalLink.vue'

defineProps<{
  // Canonical numismatics.org/ocre/id/... type link (ProviderAttribution.identifier).
  uri: string | null | undefined
}>()
</script>

<style scoped>
.ocre-attribution {
  margin: 0;
  font-size: 0.8125rem;
  color: var(--text-muted);
}

.ocre-attribution :deep(a) {
  color: var(--text-muted);
  text-decoration: underline;
}
</style>
