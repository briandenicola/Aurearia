<template>
  <section class="grid min-w-0 gap-4 overflow-hidden" aria-label="Deep Analysis report">
    <div v-if="report.partialSuccess" class="rounded-sm border border-byzantine bg-card p-3" role="status">
      <p class="m-0 text-sm font-semibold text-byzantine">Partial results</p>
      <p class="m-0 text-sm text-text-secondary">
        Not every provider could contribute. Review the coverage below before confirming any fields.
      </p>
    </div>

    <div class="grid min-w-0 gap-2">
      <h3 class="m-0 text-lg font-semibold text-text-primary">Narrative</h3>
      <p class="m-0 whitespace-pre-line break-words text-body text-text-secondary [overflow-wrap:anywhere]">{{ safeNarrative }}</p>
    </div>

    <DeepProviderCoverageList :coverage="report.coverage" />

    <div v-if="report.disagreements?.length" class="grid gap-2">
      <h3 class="m-0 text-lg font-semibold text-text-primary">Disagreements</h3>
      <ul class="m-0 grid gap-3 p-0" style="list-style: none;">
        <li
          v-for="disagreement in report.disagreements"
          :key="disagreement.field"
          class="grid min-w-0 gap-1 rounded-sm border border-border-subtle bg-card p-3"
        >
          <p class="m-0 text-sm font-semibold text-text-primary">
            {{ disagreement.field }}
            <span class="text-text-secondary">({{ disagreement.resolution }})</span>
          </p>
          <ul class="m-0 grid gap-1 p-0" style="list-style: none;">
            <li v-for="(claim, index) in disagreement.claims" :key="index" class="break-words text-sm text-text-secondary [overflow-wrap:anywhere]">
              {{ claim.value }}
              <a :href="claim.citation" target="_blank" rel="noopener noreferrer" class="text-gold underline">source</a>
            </li>
          </ul>
        </li>
      </ul>
    </div>

    <div v-if="report.unresolvedQuestions?.length" class="grid gap-2">
      <h3 class="m-0 text-lg font-semibold text-text-primary">Unresolved questions</h3>
      <ul class="m-0 grid gap-1 pl-5">
        <li v-for="(question, index) in report.unresolvedQuestions" :key="index" class="text-sm text-text-secondary">
          {{ question }}
        </li>
      </ul>
    </div>

    <div v-if="report.attributions?.length" class="grid gap-2">
      <h3 class="m-0 text-lg font-semibold text-text-primary">Attribution &amp; licensing</h3>
      <ul class="m-0 grid gap-1 p-0" style="list-style: none;">
        <li
          v-for="entry in report.attributions"
          :key="entry.provider"
          class="min-w-0 overflow-hidden rounded-sm border border-border-subtle bg-card p-2"
        >
          <OCREAttribution v-if="entry.provider === 'ocre'" :uri="entry.identifier" />
          <p v-else class="m-0 break-words text-sm text-text-secondary [overflow-wrap:anywhere]">
            {{ entry.text }}
            <SafeExternalLink
              v-if="entry.identifier"
              :href="entry.identifier"
              class="text-gold underline"
            >
              source
            </SafeExternalLink>
          </p>
        </li>
      </ul>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { DeepReport } from '@/types'
import DeepProviderCoverageList from './DeepProviderCoverageList.vue'
import OCREAttribution from './OCREAttribution.vue'
import SafeExternalLink from '@/components/SafeExternalLink.vue'

const props = defineProps<{ report: DeepReport }>()

const safeNarrative = computed(() => {
  const narrative = props.report.narrative?.trim() ?? ''
  const transportMarkers = [
    /['"]type['"]\s*:\s*['"]thinking['"]/i,
    /['"]signature['"]\s*:/i,
    /data:image\/[^;]+;base64,/i,
    /[A-Za-z0-9+/]{500,}={0,2}/,
  ]
  if (!narrative || transportMarkers.some((marker) => marker.test(narrative))) {
    return 'The narrative summary could not be displayed safely. Review the structured findings and provider coverage below, or retry the analysis.'
  }
  return narrative
})

</script>
