<template>
  <section
    v-if="run"
    class="w-full rounded-sm border border-border-subtle bg-card p-3"
    aria-live="polite"
    data-testid="copilot-run-progress"
  >
    <div class="flex flex-wrap items-center justify-between gap-2">
      <div class="flex items-center gap-2">
        <span class="chip-sm border-gold text-gold">Beta</span>
        <span class="section-label mb-0">Coin Copilot</span>
      </div>
      <span class="text-sm text-text-secondary">{{ statusLabel }}</span>
    </div>

    <ol v-if="plan.length" class="mt-3 flex list-none flex-col gap-2 p-0">
      <li v-for="item in plan" :key="item.id" class="flex items-start gap-2 text-sm">
        <CheckCircle2 v-if="item.status === 'completed'" :size="16" class="mt-0.5 shrink-0 text-gold" />
        <CircleX v-else-if="item.status === 'failed'" :size="16" class="mt-0.5 shrink-0 text-[var(--color-negative)]" />
        <LoaderCircle v-else-if="item.status === 'in_progress'" :size="16" class="mt-0.5 shrink-0 animate-spin text-gold" />
        <Circle v-else :size="16" class="mt-0.5 shrink-0 text-text-muted" />
        <span :class="item.status === 'completed' ? 'text-text-secondary' : 'text-text-primary'">{{ item.title }}</span>
      </li>
    </ol>

    <div v-if="tools.length" class="mt-3 flex flex-col gap-2 border-t border-border-subtle pt-3">
      <div v-for="(tool, toolIndex) in tools" :key="tool.toolCallId" class="flex flex-col gap-2 text-sm text-text-secondary">
        <div class="flex items-start gap-2">
          <LoaderCircle v-if="tool.status === 'running'" :size="15" class="mt-0.5 shrink-0 animate-spin text-gold" />
          <CheckCircle2 v-else-if="tool.status === 'succeeded'" :size="15" class="mt-0.5 shrink-0 text-gold" />
          <CircleX v-else :size="15" class="mt-0.5 shrink-0 text-[var(--color-negative)]" />
          <span>
            <strong class="font-medium text-text-primary">{{ toolLabel(tool.toolName) }}</strong>
            <span v-if="tool.resultSummary"> — {{ tool.resultSummary }}</span>
            <span v-if="tool.truncated" class="text-text-muted"> Result details were shortened.</span>
          </span>
        </div>

        <section
          v-if="tool.specialistResult"
          class="ml-6 flex flex-col gap-2 rounded-sm border border-border-subtle bg-input p-3"
          :aria-label="`${toolLabel(tool.toolName)} evidence`"
          data-testid="copilot-specialist-result"
        >
          <div class="flex flex-wrap items-center justify-between gap-2">
            <span class="section-label mb-0">Source evidence</span>
            <span class="chip-sm">{{ outcomeLabel(tool.specialistResult.outcome) }}</span>
          </div>

          <section
            v-if="tool.specialistResult.trend"
            class="flex flex-col gap-2 border-t border-border-subtle pt-2"
            aria-label="Price trend summary"
            data-testid="copilot-price-trend"
          >
            <div class="flex flex-wrap items-center justify-between gap-2">
              <strong class="font-medium text-text-primary">
                {{ titleCase(tool.specialistResult.trend.state) }}
              </strong>
              <span class="text-xs text-text-muted">
                {{ tool.specialistResult.trend.sampleSize }} verified sales
              </span>
            </div>
            <p class="mb-0 text-xs text-text-secondary">
              <template v-if="tool.specialistResult.trend.dateFrom && tool.specialistResult.trend.dateTo">
                {{ formatDate(tool.specialistResult.trend.dateFrom) }} to
                {{ formatDate(tool.specialistResult.trend.dateTo) }} ·
              </template>
              <template v-if="tool.specialistResult.trend.currency">
                {{ tool.specialistResult.trend.currency }} ·
              </template>
              <template v-if="tool.specialistResult.trend.priceBasis">
                {{ titleCase(tool.specialistResult.trend.priceBasis) }}
              </template>
            </p>
            <p
              v-if="tool.specialistResult.trend.low !== null &&
                tool.specialistResult.trend.median !== null &&
                tool.specialistResult.trend.high !== null"
              class="mb-0 text-xs text-text-secondary"
            >
              Range {{ formatAmount(tool.specialistResult.trend.low, tool.specialistResult.trend.currency) }}
              to {{ formatAmount(tool.specialistResult.trend.high, tool.specialistResult.trend.currency) }};
              median {{ formatAmount(tool.specialistResult.trend.median, tool.specialistResult.trend.currency) }}
            </p>
            <ul
              v-if="tool.specialistResult.trend.limitations.length"
              class="mb-0 flex list-none flex-col gap-1 p-0 text-xs text-text-muted"
            >
              <li v-for="limitation in tool.specialistResult.trend.limitations" :key="limitation">
                {{ limitation }}
              </li>
            </ul>
          </section>

          <p
            v-if="dealerSuggestions(tool).length && hasPartialDealerEvidence(tool)"
            class="mb-0 text-xs text-text-muted"
          >
            Some listings were read from search results and are only partially verified.
          </p>

          <CoinSuggestionGrid
            v-if="dealerSuggestions(tool).length"
            :suggestions="dealerSuggestions(tool)"
            :added-set="gridAddedSet(tool, toolIndex)"
            :adding-idx="gridAddingIdx(tool, toolIndex)"
            :message-index="toolIndex"
            @add-to-wishlist="(_coin, key) => emitGridWishlist(tool, key)"
          />

          <article
            v-for="item in (tool.specialistResult.capability === 'market_search' ? [] : tool.specialistResult.items)"
            :key="item.sourceUrl"
            class="flex flex-col gap-2 border-t border-border-subtle pt-2"
          >
            <div class="flex gap-2">
              <img
                v-if="item.imageUrl"
                :src="item.imageUrl"
                alt=""
                loading="lazy"
                class="h-14 w-14 shrink-0 rounded-sm object-cover"
              />
              <a
                :href="item.sourceUrl"
                target="_blank"
                rel="noopener noreferrer"
                class="font-medium text-gold"
              >
                {{ item.title }}
              </a>
            </div>
            <p class="mb-0 text-xs text-text-muted">
              Observed {{ formatObservedAt(item.observedAt) }} ·
              {{ titleCase(item.confidence) }} confidence ·
              {{ titleCase(item.verificationState) }}
            </p>
            <ul v-if="itemFacts(item).length" class="mb-0 flex list-none flex-col gap-1 p-0 text-xs">
              <li v-for="fact in itemFacts(item)" :key="fact">{{ fact }}</li>
            </ul>
            <ul
              v-if="item.candidateReferences?.length"
              class="mb-0 flex list-none flex-wrap gap-1 p-0 text-xs text-text-muted"
            >
              <li v-for="reference in item.candidateReferences" :key="`${reference.catalog}${reference.number}`">
                {{ reference.catalog }} {{ reference.volume ?? '' }} {{ reference.number }}
              </li>
            </ul>
            <button
              v-if="isEligibleCopilotDealerListing(tool.specialistResult.capability, item)"
              type="button"
              class="btn btn-xs btn-primary min-h-[44px] self-start"
              :disabled="addingIdx === dealerListingKey(tool.toolCallId, item.sourceUrl) ||
                addedSet.has(dealerListingKey(tool.toolCallId, item.sourceUrl))"
              @click="$emit(
                'addToWishlist',
                tool.specialistResult.capability,
                item,
                dealerListingKey(tool.toolCallId, item.sourceUrl),
              )"
            >
              {{ addedSet.has(dealerListingKey(tool.toolCallId, item.sourceUrl))
                ? 'Added to Wishlist'
                : addingIdx === dealerListingKey(tool.toolCallId, item.sourceUrl)
                  ? 'Adding...'
                  : 'Add to Wishlist' }}
            </button>
          </article>

          <p v-if="tool.specialistResult.outcome === 'no_match'" class="mb-0 text-text-secondary">
            No matching evidence was found in the checked sources.
          </p>
          <p v-else-if="tool.specialistResult.outcome === 'unavailable'" class="mb-0 text-text-secondary">
            Sources unavailable. Try again later.
          </p>
          <ul v-if="tool.specialistResult.warnings.length" class="mb-0 flex list-none flex-col gap-1 p-0 text-xs text-text-muted">
            <li v-for="warning in tool.specialistResult.warnings" :key="warning">{{ warning }}</li>
          </ul>
        </section>

        <section
          v-if="tool.deepAnalysisHandoffResult"
          class="ml-6 flex flex-col gap-2 rounded-sm border border-border-subtle bg-input p-3"
          aria-label="Deep Analysis handoff"
          data-testid="copilot-deep-analysis-result"
        >
          <div class="flex flex-wrap items-center justify-between gap-2">
            <span class="section-label mb-0">Deep Analysis</span>
            <span class="chip-sm">{{ deepOutcomeLabel(tool.deepAnalysisHandoffResult.outcome) }}</span>
          </div>

          <template v-if="'schema_version' in tool.deepAnalysisHandoffResult">
            <p v-if="tool.deepAnalysisHandoffResult.target" class="mb-0 text-sm text-text-primary">
              {{ tool.deepAnalysisHandoffResult.target.display_label }}
            </p>
            <p v-if="tool.deepAnalysisHandoffResult.job" class="mb-0 text-xs text-text-secondary">
              {{ deepStatusLabel(tool.deepAnalysisHandoffResult.job.status) }}
            </p>
            <p v-if="tool.deepAnalysisHandoffResult.result?.narrative" class="mb-0 text-xs text-text-secondary">
              {{ tool.deepAnalysisHandoffResult.result.narrative }}
            </p>

            <section
              v-if="tool.deepAnalysisHandoffResult.result?.disagreements.length"
              class="border-t border-border-subtle pt-2"
            >
              <span class="section-label mb-0">Conflicts</span>
              <ul class="mb-0 mt-2 flex list-none flex-col gap-1 p-0 text-xs text-text-secondary">
                <li
                  v-for="conflict in tool.deepAnalysisHandoffResult.result.disagreements"
                  :key="`${conflict.field}:${conflict.summary}`"
                >
                  <strong class="font-medium text-text-primary">{{ conflict.field }}:</strong>
                  {{ conflict.summary }}
                </li>
              </ul>
            </section>

            <section
              v-if="tool.deepAnalysisHandoffResult.result?.coverage.length"
              class="border-t border-border-subtle pt-2"
            >
              <span class="section-label mb-0">Provider coverage</span>
              <ul class="mb-0 mt-2 flex list-none flex-wrap gap-1 p-0 text-xs text-text-secondary">
                <li
                  v-for="coverage in tool.deepAnalysisHandoffResult.result.coverage"
                  :key="coverage.provider"
                  class="chip-sm"
                >
                  {{ titleCase(coverage.provider) }}: {{ titleCase(coverage.status) }}
                </li>
              </ul>
            </section>

            <p
              v-if="tool.deepAnalysisHandoffResult.truncation?.truncated"
              class="mb-0 border-t border-border-subtle pt-2 text-xs text-text-muted"
              data-testid="copilot-deep-analysis-omissions"
            >
              Result shortened:
              {{ omittedCount(tool.deepAnalysisHandoffResult.truncation) }} items omitted.
            </p>

            <ul
              v-if="deepLimitations(tool.deepAnalysisHandoffResult).length"
              class="mb-0 flex list-none flex-col gap-1 border-t border-border-subtle p-0 pt-2 text-xs text-text-muted"
            >
              <li v-for="limitation in deepLimitations(tool.deepAnalysisHandoffResult)" :key="limitation">
                {{ limitation }}
              </li>
            </ul>

            <RouterLink
              v-if="validatedReviewUrl(tool.deepAnalysisHandoffResult)"
              :to="validatedReviewUrl(tool.deepAnalysisHandoffResult) ?? ''"
              class="btn btn-xs btn-primary min-h-[44px] self-start"
            >
              Open Deep Analysis
            </RouterLink>
          </template>
        </section>
      </div>
    </div>

    <p v-if="truncated" class="mb-0 mt-3 text-sm text-text-muted">
      Earlier progress expired, so this view resumed from the retained event history.
    </p>

    <div v-if="canCancel" class="mt-3 flex justify-end">
      <button
        type="button"
        class="btn btn-xs btn-ghost min-h-[44px]"
        :disabled="cancelling"
        @click="$emit('cancel')"
      >
        <Square :size="15" />
        {{ cancelling ? 'Cancelling...' : 'Cancel run' }}
      </button>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { CheckCircle2, Circle, CircleX, LoaderCircle, Square } from 'lucide-vue-next'
import type {
  CoinCopilotPlanItem,
  CoinCopilotRun,
  CoinCopilotSpecialistCapability,
  CoinCopilotSpecialistEvidence,
  DeepAnalysisHandoffResult,
  DeepAnalysisHandoffTruncation,
} from '@/types'
import type { CoinCopilotToolProgress } from '@/composables/useCoinCopilot'
import {
  copilotDealerListingKey,
  copilotDealerListingToSuggestion,
  isEligibleCopilotDealerListing,
} from '@/utils/copilotWishlist'
import CoinSuggestionGrid from '@/components/chat/CoinSuggestionGrid.vue'

const props = defineProps<{
  run: CoinCopilotRun | null
  plan: CoinCopilotPlanItem[]
  tools: CoinCopilotToolProgress[]
  canCancel: boolean
  cancelling: boolean
  truncated: boolean
  addingIdx: string | null
  addedSet: Set<string>
}>()

const emit = defineEmits<{
  cancel: []
  addToWishlist: [
    capability: CoinCopilotSpecialistCapability,
    item: CoinCopilotSpecialistEvidence,
    key: string,
  ]
}>()

// Dealer listings render through the same card the legacy chat uses, so the
// grid fetches thumbnails and the look matches. Wish list clicks are mapped
// back to the typed evidence item the parent re-checks before saving.
function eligibleDealerItems(tool: CoinCopilotToolProgress) {
  const capability = tool.specialistResult?.capability
  if (!capability) return []
  return (tool.specialistResult?.items ?? []).filter(item =>
    isEligibleCopilotDealerListing(capability, item))
}

function hasPartialDealerEvidence(tool: CoinCopilotToolProgress) {
  return eligibleDealerItems(tool).some(item => item.verificationState === 'partial')
}

function dealerSuggestions(tool: CoinCopilotToolProgress) {
  return eligibleDealerItems(tool).map(copilotDealerListingToSuggestion)
}

function gridKey(toolIndex: number, itemIndex: number) {
  return `${toolIndex}-${itemIndex}`
}

function gridAddedSet(tool: CoinCopilotToolProgress, toolIndex: number) {
  const keys = new Set<string>()
  eligibleDealerItems(tool).forEach((item, index) => {
    if (props.addedSet.has(copilotDealerListingKey(tool.toolCallId, item.sourceUrl))) {
      keys.add(gridKey(toolIndex, index))
    }
  })
  return keys
}

function gridAddingIdx(tool: CoinCopilotToolProgress, toolIndex: number) {
  const index = eligibleDealerItems(tool).findIndex(
    item => props.addingIdx === copilotDealerListingKey(tool.toolCallId, item.sourceUrl),
  )
  return index === -1 ? null : gridKey(toolIndex, index)
}

function emitGridWishlist(tool: CoinCopilotToolProgress, key: string) {
  const index = Number(key.split('-').pop())
  const item = eligibleDealerItems(tool)[index]
  if (!item || !tool.specialistResult) return
  emit(
    'addToWishlist',
    tool.specialistResult.capability,
    item,
    copilotDealerListingKey(tool.toolCallId, item.sourceUrl),
  )
}

const statusLabel = computed(() => {
  switch (props.run?.status) {
    case 'queued': return 'Queued'
    case 'running': return 'Working'
    case 'paused': return 'Waiting for your answer'
    case 'cancel_requested': return 'Cancelling'
    case 'completed': return 'Complete'
    case 'failed': return 'Stopped'
    case 'cancelled': return 'Cancelled'
    default: return ''
  }
})

function toolLabel(name: string) {
  return name
    .split('_')
    .map(part => part ? part.charAt(0).toUpperCase() + part.slice(1) : part)
    .join(' ')
}

// Display lines are built here from typed fields; the agent service sends the
// evidence, not a pre-rendered list.
function itemFacts(item: CoinCopilotSpecialistEvidence): string[] {
  const money = (amount?: number) =>
    amount == null ? '' : `${item.currency ? `${item.currency} ` : ''}${amount}`
  const lines: [string, string | number | undefined][] = [
    ['Dealer', item.dealerName],
    ['Price', money(item.listedPrice)],
    ['Availability', item.availability],
    ['Auction house', item.auctionHouse],
    ['Sale', item.saleName],
    ['Lot', item.lotNumber],
    ['Sale date', item.saleDate],
    ['Estimate', money(item.estimate)],
    ['Current bid', money(item.currentBid)],
    ['Amount', money(item.amount)],
    ['Price basis', item.priceBasis],
    ['Ruler', item.ruler],
    ['Denomination', item.denomination],
    ['Era', item.era],
    ['Material', item.material],
  ]
  return lines
    .filter(([, value]) => value !== undefined && value !== null && String(value).trim() !== '')
    .map(([label, value]) => `${label}: ${titleCase(String(value))}`)
    .slice(0, 10)
}

function titleCase(value: string) {
  return value.charAt(0).toUpperCase() + value.slice(1).replaceAll('_', ' ')
}

function outcomeLabel(outcome: 'complete' | 'partial' | 'no_match' | 'unavailable') {
  switch (outcome) {
    case 'complete': return 'Complete'
    case 'partial': return 'Partial'
    case 'no_match': return 'No matching evidence'
    case 'unavailable': return 'Sources unavailable'
  }
}

function deepOutcomeLabel(outcome: DeepAnalysisHandoffResult['outcome']) {
  const labels: Record<DeepAnalysisHandoffResult['outcome'], string> = {
    accepted: 'Accepted',
    reused_active: 'In progress',
    reused_result: 'Result available',
    status: 'Status',
    retry_available: 'Retry available',
    missing_images: 'Images required',
    target_unavailable: 'Target unavailable',
    not_eligible: 'Not eligible',
    unavailable: 'Unavailable',
    cancelled: 'Cancelled',
  }
  return labels[outcome]
}

function deepStatusLabel(status: NonNullable<Extract<DeepAnalysisHandoffResult, { schema_version: 1 }>['job']>['status']) {
  return `Job status: ${titleCase(status)}`
}

function validatedReviewUrl(result: DeepAnalysisHandoffResult): string | null {
  if (!('schema_version' in result) || !result.job || !result.review_url) return null
  const expected = `/deep-analysis/${result.job.id}`
  return Number.isSafeInteger(result.job.id) && result.job.id > 0 && result.review_url === expected
    ? expected
    : null
}

function omittedCount(truncation: DeepAnalysisHandoffTruncation) {
  return truncation.omitted_fields +
    truncation.omitted_evidence +
    truncation.omitted_disagreements +
    truncation.omitted_questions
}

function dealerListingKey(toolCallId: string, sourceUrl: string) {
  return copilotDealerListingKey(toolCallId, sourceUrl)
}

function deepLimitations(result: Extract<DeepAnalysisHandoffResult, { schema_version: 1 }>) {
  return [...result.limitations, ...(result.result?.limitations ?? [])]
}

function formatObservedAt(value: string) {
  return formatDate(value)
}

function formatDate(value: string) {
  return new Intl.DateTimeFormat(undefined, {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    timeZone: 'UTC',
  }).format(new Date(value))
}

function formatAmount(value: number, currency: string | null) {
  if (!currency) return String(value)
  return new Intl.NumberFormat(undefined, {
    style: 'currency',
    currency,
    maximumFractionDigits: 2,
  }).format(value)
}
</script>
