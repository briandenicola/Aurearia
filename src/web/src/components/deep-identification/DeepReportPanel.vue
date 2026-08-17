<template>
  <section class="grid min-w-0 gap-4 overflow-hidden" aria-label="Deep Analysis report">
    <div v-if="report.partialSuccess" class="rounded-sm border border-byzantine bg-card p-3" role="status">
      <p class="m-0 text-sm font-semibold text-byzantine">Partial results</p>
      <p class="m-0 text-sm text-text-secondary">
        Not every provider could contribute. Review the coverage below before confirming any fields.
      </p>
    </div>

    <div
      v-if="report.quickLookupOutcome === 'unavailable'"
      class="rounded-sm border border-warning/30 bg-[color-mix(in_srgb,var(--text-warning)_10%,transparent)] p-3"
      role="status"
    >
      <p class="m-0 text-sm font-semibold text-warning">Quick lookup incomplete</p>
      <p class="m-0 text-sm text-text-secondary">
        The initial NGC/catalog quick lookup did not finish before this report was generated. Any
        missing supporting data below may be a result of that, not proof the coin has none &mdash;
        try running Deep Analysis again.
      </p>
    </div>
    <p v-else-if="report.quickLookupOutcome === 'no_data'" class="m-0 text-sm text-text-muted">
      Quick lookup completed but found no supporting cert or catalog data for this coin.
    </p>

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
              <a v-if="claim.citation" :href="claim.citation" target="_blank" rel="noopener noreferrer" class="text-gold underline">source</a>
              <span v-else class="chip-sm" title="This claim came only from the image analysis, not a catalog citation">
                From images
              </span>
            </li>
          </ul>
        </li>
      </ul>
    </div>

    <details
      v-if="report.image_hypothesis"
      class="min-w-0 overflow-hidden rounded-sm border border-border-subtle bg-card p-3"
    >
      <summary class="cursor-pointer select-none text-sm font-semibold uppercase tracking-[0.08em] text-text-muted">
        What the images alone said
      </summary>

      <div class="mt-3 grid min-w-0 gap-3">
        <p v-if="!report.image_hypothesis.legible" class="m-0 text-sm text-text-secondary">
          The images were not legible enough for the vision analysis to identify this coin. That is
          different from the analysis not running at all &mdash; it ran, but the coin's surfaces could
          not be read clearly enough to produce findings.
        </p>

        <template v-else>
          <p v-if="!hypothesisFieldEntries.length" class="m-0 text-sm text-text-muted">
            The images were legible, but the vision analysis did not find any coin details it was
            confident enough to report.
          </p>
          <ul v-else class="m-0 grid gap-2 p-0" style="list-style: none;">
            <li
              v-for="entry in hypothesisFieldEntries"
              :key="entry.name"
              class="grid min-w-0 gap-1"
            >
              <span class="text-xs font-semibold uppercase tracking-[0.08em] text-text-muted">{{ formatFieldName(entry.name) }}</span>
              <p class="m-0 flex flex-wrap items-center gap-2 break-words text-sm text-text-secondary [overflow-wrap:anywhere]">
                <span class="text-text-primary">{{ entry.value }}</span>
                <span class="chip-sm">{{ Math.round(entry.confidence * 100) }}% confidence</span>
              </p>
            </li>
          </ul>

          <p v-if="report.image_hypothesis.observations" class="m-0 break-words text-sm text-text-secondary [overflow-wrap:anywhere]">
            {{ report.image_hypothesis.observations }}
          </p>
        </template>
      </div>
    </details>

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
import type { CoinHypothesis, DeepReport } from '@/types'
import DeepProviderCoverageList from './DeepProviderCoverageList.vue'
import OCREAttribution from './OCREAttribution.vue'
import SafeExternalLink from '@/components/SafeExternalLink.vue'
import { formatFieldName } from '@/utils/format'

const props = defineProps<{ report: DeepReport }>()

// Display order mirrors contracts/vision-hypothesis.md §1 — every coin-field
// vocabulary key the hypothesis can carry, excluding the `observations`/
// `legible` metadata rendered separately below.
type HypothesisFieldName = Exclude<keyof CoinHypothesis, 'observations' | 'legible'>
const hypothesisFieldOrder: HypothesisFieldName[] = [
  'ruler',
  'denomination',
  'material',
  'mint',
  'dateRange',
  'era',
  'obverseInscription',
  'reverseInscription',
  'obverseDescription',
  'reverseDescription',
  'diameterMm',
  'weightGrams',
  'notes',
  'coin_type',
]

const hypothesisFieldEntries = computed(() => {
  const hypothesis = props.report.image_hypothesis
  if (!hypothesis) return []
  return hypothesisFieldOrder.flatMap((name) => {
    const field = hypothesis[name]
    return field ? [{ name, value: field.value, confidence: field.confidence }] : []
  })
})

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
