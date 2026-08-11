<template>
  <section class="grid gap-3" aria-labelledby="numista-lookup-heading">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <h3 id="numista-lookup-heading" class="m-0 text-base font-medium text-text-primary">
        Numista Lookup
      </h3>
      <div class="flex flex-wrap items-center justify-end gap-2">
        <span class="chip-sm border border-border-subtle text-text-secondary">
          Status: {{ statusGuidance.label }}
        </span>
        <span v-if="cacheFreshnessText" class="chip-sm border border-border-subtle text-text-secondary">
          {{ cacheFreshnessText }}
        </span>
      </div>
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
        @input="queryEdited = true"
      />
      <p id="numista-query-help" class="mt-1 text-sm text-text-muted">
        {{ query.trim()
          ? 'Review or refine the attribution evidence before searching.'
          : 'Enter at least one search term to enable Numista lookup.' }}
      </p>
    </div>

    <div class="flex flex-wrap gap-2">
      <button
        type="button"
        class="btn btn-primary btn-sm min-h-11"
        :disabled="searchDisabled"
        @click="search"
      >
        {{ searchButtonLabel }}
      </button>
      <button
        v-if="selected"
        type="button"
        class="btn btn-ghost btn-sm min-h-11"
        :disabled="loading"
        aria-label="Remove selected Numista reference"
        @click="clearSelection"
      >
        Remove selection
      </button>
    </div>

    <div
      ref="statusRegion"
      tabindex="-1"
      aria-live="polite"
      aria-atomic="true"
      class="outline-none"
    >
      <div
        role="status"
        class="grid gap-1 rounded-sm border border-border-subtle bg-input p-3"
        :data-lookup-state="panelState"
      >
        <strong class="text-body text-text-primary">{{ statusGuidance.title }}</strong>
        <span class="text-body text-text-secondary">{{ statusGuidance.message }}</span>
        <RouterLink
          v-if="statusGuidance.settingsHref"
          :to="statusGuidance.settingsHref"
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
      <button
        v-if="showConfirmation"
        type="button"
        class="btn btn-primary btn-sm min-h-11"
        :disabled="loading"
        @click="confirmSelection"
      >
        {{ confirmationLabel }}
      </button>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { RouterLink } from 'vue-router'
import { lookupNumista } from '@/api/client'
import SafeExternalLink from '@/components/SafeExternalLink.vue'
import {
  getNumistaCacheFreshnessText,
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
  initialSelection?: NumistaCandidate | null
  showConfirmation?: boolean
  confirmationLabel?: string
}>(), {
  path: 'direct',
  isAdmin: false,
  initialSelection: null,
  showConfirmation: true,
  confirmationLabel: 'Add selected reference',
})

const emit = defineEmits<{
  confirmed: [candidate: NumistaCandidate]
  selectionChanged: [candidate: NumistaCandidate | null]
}>()

const query = ref(props.initialQuery)
const queryEdited = ref(false)
const loading = ref(false)
const outcome = ref<NumistaLookupOutcome | null>(null)
const selected = ref<NumistaCandidate | null>(props.initialSelection)
const selectedId = ref<number | null>(props.initialSelection?.id ?? null)
const statusRegion = ref<HTMLElement | null>(null)

const candidates = computed(() => outcome.value?.candidates ?? [])
const selectionOutsideResults = computed(() => isSelectionOutsideResults(selected.value, candidates.value))
const panelState = computed(() => loading.value ? 'loading' : outcome.value?.status ?? 'idle')
const statusGuidance = computed(() => getNumistaStatusGuidance(
  panelState.value,
  props.isAdmin,
  outcome.value?.retryAfterSeconds,
))
const cacheFreshnessText = computed(() => getNumistaCacheFreshnessText(
  outcome.value?.status ?? null,
  outcome.value?.cache,
))
const searchDisabled = computed(() => {
  if (loading.value || !query.value.trim()) return true
  if (outcome.value?.status === 'unconfigured') return !statusGuidance.value.canRetry
  return false
})
const searchButtonLabel = computed(() => {
  if (loading.value) return 'Searching...'
  if (!outcome.value) return 'Search Numista'
  if (outcome.value.status === 'success') return 'Search again'
  if (outcome.value.status === 'unconfigured' && props.isAdmin) return 'Retry after configuration'
  if (statusGuidance.value.canRetry) return 'Retry lookup'
  return 'Search unavailable'
})

watch(() => props.initialQuery, (value) => {
  if (!queryEdited.value && !outcome.value && !loading.value) query.value = value
})

watch(() => props.initialSelection, (value) => {
  selected.value = value
  selectedId.value = value?.id ?? null
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
    query.value = response.data.effectiveQuery || effectiveQuery
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
    await nextTick()
    statusRegion.value?.focus()
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
