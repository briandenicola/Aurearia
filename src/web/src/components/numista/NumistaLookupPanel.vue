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
      <span id="numista-query-label" class="form-label">Search query</span>
      <textarea
        id="numista-query"
        v-model="query"
        class="form-textarea min-h-20"
        maxlength="500"
        rows="3"
        :disabled="searching"
        aria-labelledby="numista-query-label"
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
        v-if="enriching"
        type="button"
        class="btn btn-ghost btn-sm min-h-11"
        @click="cancelEnrichment()"
      >
        Cancel enrichment
      </button>
      <button
        v-else-if="canRetryEnrichment"
        type="button"
        class="btn btn-ghost btn-sm min-h-11"
        :disabled="searching"
        @click="retryEnrichment"
      >
        Retry details
      </button>
      <button
        v-if="selected"
        type="button"
        class="btn btn-ghost btn-sm min-h-11"
        :disabled="searching"
        aria-label="Remove selected Numista reference"
        @click="clearSelection"
      >
        Remove selection
      </button>
    </div>

    <p
      v-if="enrichmentMessage"
      class="m-0 text-sm text-text-secondary"
      aria-live="polite"
      aria-atomic="true"
    >
      {{ enrichmentMessage }}
    </p>

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
        <span class="grid h-16 w-16 grid-cols-2 gap-1">
          <img
            v-if="safeHttpsImage(candidate.obverseThumbnail)"
            :src="safeHttpsImage(candidate.obverseThumbnail) ?? ''"
            :alt="`Obverse thumbnail for ${candidate.title}`"
            class="h-16 min-w-0 rounded-sm object-contain"
          >
          <img
            v-if="safeHttpsImage(candidate.reverseThumbnail)"
            :src="safeHttpsImage(candidate.reverseThumbnail) ?? ''"
            :alt="`Reverse thumbnail for ${candidate.title}`"
            class="h-16 min-w-0 rounded-sm object-contain"
          >
        </span>
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
        :disabled="searching"
        @click="confirmSelection"
      >
        {{ confirmationLabel }}
      </button>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, nextTick, onUnmounted, ref, watch } from 'vue'
import { RouterLink } from 'vue-router'
import { enrichNumista, lookupNumista } from '@/api/client'
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
  NumistaEnrichmentRequest,
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
const searching = ref(false)
const enriching = ref(false)
const enrichmentMessage = ref('')
const enrichmentRetryAvailable = ref(false)
const outcome = ref<NumistaLookupOutcome | null>(null)
const selected = ref<NumistaCandidate | null>(props.initialSelection)
const selectedId = ref<number | null>(props.initialSelection?.id ?? null)
const statusRegion = ref<HTMLElement | null>(null)
let enrichmentController: AbortController | null = null

const candidates = computed(() => outcome.value?.candidates ?? [])
const selectionOutsideResults = computed(() => isSelectionOutsideResults(selected.value, candidates.value))
const panelState = computed(() => searching.value ? 'loading' : outcome.value?.status ?? 'idle')
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
  if (searching.value || !query.value.trim()) return true
  if (outcome.value?.status === 'unconfigured') return !statusGuidance.value.canRetry
  return false
})
const searchButtonLabel = computed(() => {
  if (searching.value) return 'Searching...'
  if (!outcome.value) return 'Search Numista'
  if (outcome.value.status === 'success') return 'Search again'
  if (outcome.value.status === 'unconfigured' && props.isAdmin) return 'Retry after configuration'
  if (statusGuidance.value.canRetry) return 'Retry lookup'
  return 'Search unavailable'
})
const canRetryEnrichment = computed(() => (
  enrichmentRetryAvailable.value
  && outcome.value?.status === 'success'
  && candidates.value.length > 0
))

watch(() => props.initialQuery, (value) => {
  if (!queryEdited.value && !outcome.value && !searching.value) query.value = value
})

watch(() => props.initialSelection, (value) => {
  selected.value = value
  selectedId.value = value?.id ?? null
})

async function search() {
  const effectiveQuery = query.value
  if (!effectiveQuery.trim() || searching.value) return

  cancelEnrichment(false)
  enrichmentMessage.value = ''
  enrichmentRetryAvailable.value = false
  searching.value = true
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

    await nextTick()
    if (shouldEnrich(response.data)) {
      void startEnrichment({
        query: effectiveQuery.trim(),
        path: props.path,
        evidence: props.evidence,
        candidates: response.data.candidates,
      })
    }
  } catch {
    outcome.value = {
      status: 'unavailable',
      effectiveQuery,
      candidates: [],
      stage: 'broad',
    }
  } finally {
    searching.value = false
    await nextTick()
    statusRegion.value?.focus()
  }
}

function shouldEnrich(result: NumistaLookupOutcome): boolean {
  return result.status === 'success'
    && result.stage === 'broad'
    && result.candidates.some(candidate => candidate.enrichmentState === 'not_requested')
}

async function startEnrichment(request: NumistaEnrichmentRequest) {
  cancelEnrichment(false)
  const controller = new AbortController()
  enrichmentController = controller
  enriching.value = true
  enrichmentRetryAvailable.value = false
  enrichmentMessage.value = 'Enriching leading candidates with catalog details.'

  try {
    const response = await enrichNumista(request, controller.signal)
    if (enrichmentController !== controller) return

    const merged = mergeEnrichedCandidates(request.candidates, response.data.candidates)
    outcome.value = {
      ...response.data,
      status: 'success',
      candidates: merged,
      stage: 'enriched',
    }
    selected.value = retainNumistaSelection(selected.value, merged)
    selectedId.value = selected.value?.id ?? null

    const failedCount = merged.filter(candidate => candidate.enrichmentState === 'failed').length
    const detailedCount = merged.filter(candidate => (
      candidate.enrichmentState === 'enriched' || candidate.enrichmentState === 'cached'
    )).length
    enrichmentRetryAvailable.value = failedCount > 0
    enrichmentMessage.value = failedCount === 0
      ? 'Candidate details are ready.'
      : detailedCount === 0
        ? 'Details could not be loaded. Broad results remain selectable.'
        : `Details are ready for ${detailedCount} candidates; ${failedCount} remain broad results.`
  } catch (error) {
    if (enrichmentController !== controller) return
    if (isCancellation(error)) {
      enrichmentRetryAvailable.value = true
      enrichmentMessage.value = 'Enrichment cancelled. Broad results remain selectable.'
      return
    }

    const retained = request.candidates.map(candidate => (
      candidate.enrichmentState === 'not_requested'
        ? { ...candidate, enrichmentState: 'failed' as const }
        : candidate
    ))
    if (outcome.value) {
      outcome.value = { ...outcome.value, candidates: retained, stage: 'enriched' }
    }
    selected.value = retainNumistaSelection(selected.value, retained)
    selectedId.value = selected.value?.id ?? null
    enrichmentRetryAvailable.value = true
    enrichmentMessage.value = 'Details could not be loaded. Broad results remain selectable.'
  } finally {
    if (enrichmentController === controller) {
      enrichmentController = null
      enriching.value = false
    }
  }
}

function retryEnrichment() {
  const result = outcome.value
  if (!result || result.status !== 'success' || !result.candidates.length) return
  void startEnrichment({
    query: result.effectiveQuery || query.value,
    path: props.path,
    evidence: props.evidence,
    candidates: result.candidates.map(candidate => (
      candidate.enrichmentState === 'failed'
        ? { ...candidate, enrichmentState: 'not_requested' as const }
        : candidate
    )),
  })
}

function cancelEnrichment(announce = true) {
  if (!enrichmentController) return
  const controller = enrichmentController
  enrichmentController = null
  controller.abort()
  enriching.value = false
  enrichmentRetryAvailable.value = announce
  if (announce) enrichmentMessage.value = 'Enrichment cancelled. Broad results remain selectable.'
}

function mergeEnrichedCandidates(
  broadCandidates: NumistaCandidate[],
  enrichedCandidates: NumistaCandidate[],
): NumistaCandidate[] {
  const broadById = new Map(broadCandidates.map(candidate => [candidate.id, candidate]))
  const merged = enrichedCandidates
    .filter(candidate => broadById.has(candidate.id))
    .map(candidate => ({ ...broadById.get(candidate.id), ...candidate }))
  const returnedIds = new Set(merged.map(candidate => candidate.id))
  return [
    ...merged,
    ...broadCandidates
      .filter(candidate => !returnedIds.has(candidate.id))
      .map(candidate => ({ ...candidate, enrichmentState: 'failed' as const })),
  ]
}

function isCancellation(error: unknown): boolean {
  if (typeof error !== 'object' || error === null) return false
  const candidate = error as { code?: unknown; name?: unknown }
  return candidate.code === 'ERR_CANCELED'
    || candidate.name === 'CanceledError'
    || candidate.name === 'AbortError'
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

function safeHttpsImage(value: string | undefined): string | null {
  if (!value) return null
  try {
    const url = new URL(value)
    return url.protocol === 'https:' ? url.toString() : null
  } catch {
    return null
  }
}

onUnmounted(() => cancelEnrichment(false))
</script>
