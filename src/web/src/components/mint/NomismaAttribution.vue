<template>
  <p v-if="uri" class="nomisma-attribution">
    Source:
    <SafeExternalLink :href="uri">Nomisma.org</SafeExternalLink>
    ·
    <SafeExternalLink href="https://creativecommons.org/licenses/by/4.0/">CC BY 4.0</SafeExternalLink>
  </p>
</template>

<script setup lang="ts">
import SafeExternalLink from '@/components/SafeExternalLink.vue'

defineProps<{
  uri: string | null | undefined
}>()
</script>

<style scoped>
.nomisma-attribution {
  margin: 0;
  font-size: 0.8125rem;
  color: var(--text-muted);
}

.nomisma-attribution :deep(a) {
  color: var(--text-muted);
  text-decoration: underline;
}
</style>
