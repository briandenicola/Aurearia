<template>
  <section class="grid gap-3" aria-labelledby="numista-lookup-heading">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <h3 id="numista-lookup-heading" class="m-0 text-base font-medium text-text-primary">
        Numista Lookup
      </h3>
      <span v-if="outcome?.cache" class="chip-sm border border-border-subtle text-text-secondary">
        {{ outcome.cache.hit ? `Cached ${outcome.cache.ageSeconds}s ago` : 'Fresh results' }}
      </span>
    </div>

    <div>
      <label for="numista-query" class="form-label">Search query</label>
      <textarea
        id="numista-query"
        v-model="query"
        class="form-textarea min-h-20"
        maxlength="500"
        rows="3"
        :disabled="loading"
        aria-describedby="numista-query-help"
      />
      <p id="numista-query-help" class="mt-1 text-sm text-text-muted">
        Review or refine the attribution evidence before searching.
      </p>
    </div>

    <div class="flex flex-wrap gap-2">
      <button
        class="btn btn-primary btn-sm min-h-11"
        :disabled="loading || !query.trim()"
        @click="search"
      >
        {{ loading ? 'Searching...' : outcome ? 'Search again' : 'Search Numista' }}
      </button>
      <button
        v-if="selected"
        class="btn btn-ghost btn-sm min-h-11"
        :disabled="loading"
        @click="clearSelection"
      >
        Remove selection
      </button>
    </div>

    <div aria-live="polite" aria-atomic="true">
      <p v-if="loading" role="status" class="m-0 text-body text-text-secondary">
        Searching Numista for ranked candidates...
      </p>
      <div
        v-else-if="guidance"
        role="status"
        class="grid gap-1 rounded-sm border border-border-subtle bg-input p-3"
      >
        <strong class="text-body text-text-primary">{{ guidance.title }}</strong>
        <span class="text-body text-text-secondary">{{ guidance.message }}</span>
        <RouterLink
          v-if="guidance.settingsHref"
          :to="guidance.settingsHref"
          class="text-body text-gold"
        >
          Open Numista settings
        </RouterLink>
      </div>
    </div>

    <div
      v-if="selected && selectionOutsideResults"
      class="grid gap-1 rounded-sm border border-border-accent bg-input p-3"
      role="status"
    >
      <strong class="text-body text-text-primary">Selection retained from an earlier search</strong>
      <span class="text-body text-text-secondary">{{ selected.title }}</span>
    </div>

    <fieldset v-if="candidates.length" class="m-0 grid min-w-0 gap-3 border-0 p-0">
      <legend class="section-label mb-2">Ranked candidates</legend>
      <label
        v-for="candidate in candidates"
        :key="numistaCandidateIdentity(candidate)"
        class="grid min-h-11 cursor-pointer gap-3 rounded-sm border p-3 transition-colors sm:grid-cols-[auto_4rem_1fr]"
        :class="selected?.id === candidate.id ? 'border-gold bg-input' : 'border-border-subtle bg-card hover:border-border-accent'"
      >
        <input
          v-model="selectedId"
          type="radio"
          name="numista-candidate"
          class="mt-1 accent-gold"
          :value="candidate.id"
          @change="selectCandidate(candidate)"
        >
        <img
          v-if="candidate.obverseThumbnail"
          :src="candidate.obverseThumbnail"
          :alt="`Obverse thumbnail for ${candidate.title}`"
          class="h-16 w-16 rounded-sm object-contain"
        >
        <span v-else class="hidden h-16 w-16 sm:block" aria-hidden="true" />
        <span class="grid min-w-0 gap-2">
          <span class="flex min-w-0 flex-wrap items-start justify-between gap-2">
            <span class="min-w-0">
              <strong class="block break-words text-body text-text-primary">{{ candidate.title }}</strong>
              <span class="block text-sm text-text-secondary">{{ candidateSummary(candidate) }}</span>
            </span>
            <span class="chip-sm border border-border-subtle text-text-primary">
              {{ candidate.assessment.score }} · {{ candidate.assessment.band }}
            </span>
          </span>
          <span v-if="candidate.assessment.reasons.length" class="grid gap-1">
            <span
              v-for="reason in candidate.assessment.reasons"
              :key="`${reason.field}:${reason.code}`"
              class="text-sm"
              :class="reason.kind === 'conflict' ? 'text-warning' : 'text-text-secondary'"
            >
              {{ reason.kind === 'conflict' ? 'Conflict: ' : '' }}{{ reason.label }}
            </span>
          </span>
          <span v-else class="text-sm text-text-muted">No detailed relevance explanation is available.</span>
          <span class="flex flex-wrap items-center gap-2">
            <SafeExternalLink
              :href="candidate.canonicalUrl"
              target="_blank"
              rel="noopener noreferrer"
              class="text-sm text-gold"
              @click.stop
            >
              View catalog entry
            </SafeExternalLink>
            <span class="text-sm text-text-muted">{{ enrichmentLabel(candidate) }}</span>
          </span>
        </span>
      </label>
    </fieldset>

    <div v-if="selected" class="flex flex-wrap items-center justify-between gap-3 rounded-sm border border-border-accent bg-input p-3">
      <span class="min-w-0 text-body text-text-primary">
        Selected: <strong class="break-words">{{ selected.title }}</strong>
      </span>
      <button class="btn btn-primary btn-sm min-h-11" :disabled="loading" @click="confirmSelection">
        Add selected reference
      </button>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { lookupNumista } from '@/api/client'
import SafeExternalLink from '@/components/SafeExternalLink.vue'
import {
  getNumistaStatusGuidance,
  isSelectionOutsideResults,
  numistaCandidateIdentity,
  retainNumistaSelection,
} from '@/utils/numistaLookup'
import type {
  NumistaCandidate,
  NumistaEvidence,
  NumistaLookupOutcome,
  NumistaLookupPath,
} from '@/types'

const props = withDefaults(defineProps<{
  initialQuery: string
  evidence: NumistaEvidence
  path?: NumistaLookupPath
  isAdmin?: boolean
}>(), {
  path: 'direct',
  isAdmin: false,
})

const emit = defineEmits<{
  confirmed: [candidate: NumistaCandidate]
  selectionChanged: [candidate: NumistaCandidate | null]
}>()

const query = ref(props.initialQuery)
const loading = ref(false)
const outcome = ref<NumistaLookupOutcome | null>(null)
const selected = ref<NumistaCandidate | null>(null)
const selectedId = ref<number | null>(null)

const candidates = computed(() => outcome.value?.candidates ?? [])
const selectionOutsideResults = computed(() => isSelectionOutsideResults(selected.value, candidates.value))
const guidance = computed(() => {
  const current = outcome.value
  return current
    ? getNumistaStatusGuidance(current.status, props.isAdmin, current.retryAfterSeconds)
    : null
})

watch(() => props.initialQuery, (value) => {
  if (!outcome.value && !loading.value) query.value = value
})

async function search() {
  const effectiveQuery = query.value
  if (!effectiveQuery.trim() || loading.value) return

  loading.value = true
  try {
    const response = await lookupNumista({
      query: effectiveQuery,
      path: props.path,
      evidence: props.evidence,
    })
    outcome.value = response.data
    query.value = response.data.effectiveQuery
    selected.value = retainNumistaSelection(selected.value, response.data.candidates)
    selectedId.value = selected.value?.id ?? null
  } catch {
    outcome.value = {
      status: 'unavailable',
      effectiveQuery,
      candidates: [],
      stage: 'broad',
    }
  } finally {
    loading.value = false
  }
}

function selectCandidate(candidate: NumistaCandidate) {
  selected.value = candidate
  selectedId.value = candidate.id
  emit('selectionChanged', candidate)
}

function clearSelection() {
  selected.value = null
  selectedId.value = null
  emit('selectionChanged', null)
}

function confirmSelection() {
  if (selected.value) emit('confirmed', selected.value)
}

function candidateSummary(candidate: NumistaCandidate): string {
  return [
    candidate.issuer,
    candidate.denomination,
    candidate.mint,
    candidate.yearDisplay,
    candidate.material,
  ].filter(Boolean).join(' · ')
}

function enrichmentLabel(candidate: NumistaCandidate): string {
  switch (candidate.enrichmentState) {
    case 'enriched': return 'Details enriched'
    case 'cached': return 'Cached details'
    case 'failed': return 'Detail unavailable'
    case 'not_requested': return 'Broad result'
  }
}
</script>
