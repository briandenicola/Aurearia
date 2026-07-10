<template>
  <article
    :class="[
      'grid gap-3 rounded-md border border-border-subtle bg-card p-4',
      candidate.lifecycleState === 'suppressed' ? 'opacity-75' : '',
    ]"
  >
    <header class="flex flex-col gap-4 md:flex-row md:items-start md:justify-between">
      <div>
        <p class="section-label">{{ candidate.sourceName || 'Unknown source' }}</p>
        <h3 class="mt-1">{{ candidate.title || 'Unknown title' }}</h3>
      </div>
      <div class="flex flex-wrap items-center gap-2">
        <span
          :class="[
            'chip-sm border',
            candidate.provenanceStatus === 'verified'
              ? 'border-gold text-gold'
              : candidate.provenanceStatus === 'partial'
                ? 'border-bronze text-bronze'
                : 'border-border-subtle text-text-muted',
          ]"
        >
          {{ provenanceLabel }}
        </span>
        <span class="chip-sm border border-border-subtle text-text-secondary">{{ stateLabel }}</span>
      </div>
    </header>

    <div class="grid gap-3 md:grid-cols-2">
      <div class="grid gap-1 rounded-sm border border-border-subtle bg-input p-3"><span class="text-label font-semibold uppercase tracking-[0.08em] text-text-muted">Price</span><strong>{{ priceLabel }}</strong></div>
      <div class="grid gap-1 rounded-sm border border-border-subtle bg-input p-3"><span class="text-label font-semibold uppercase tracking-[0.08em] text-text-muted">Last seen</span><strong>{{ formatDate(candidate.lastSeenAt) }}</strong></div>
      <div v-if="candidate.matchingWishlistCoinId" class="grid gap-1 rounded-sm border border-border-subtle bg-input p-3"><span class="text-label font-semibold uppercase tracking-[0.08em] text-text-muted">Duplicate warning</span><strong>Matches wishlist coin #{{ candidate.matchingWishlistCoinId }}</strong></div>
      <div v-if="candidate.duplicateOfCandidateId" class="grid gap-1 rounded-sm border border-border-subtle bg-input p-3"><span class="text-label font-semibold uppercase tracking-[0.08em] text-text-muted">Suppressed duplicate</span><strong>Candidate #{{ candidate.duplicateOfCandidateId }}</strong></div>
    </div>

    <p class="m-0 text-body text-text-secondary">{{ candidate.reasonForMatch || 'No source-backed match reason provided.' }}</p>

    <SafeExternalLink :href="candidate.sourceUrl" class="break-words text-gold">Open source listing</SafeExternalLink>

    <dl v-if="sourceFields.length" class="grid gap-3 md:grid-cols-2">
      <div
        v-for="[field, value] in sourceFields"
        :key="field"
        class="grid gap-1 rounded-sm border border-border-subtle bg-input p-3"
      >
        <dt class="text-label font-semibold uppercase tracking-[0.08em] text-text-muted">{{ formatFieldLabel(field) }}</dt>
        <dd class="m-0 text-text-primary">{{ value }}</dd>
      </div>
    </dl>

    <details v-if="candidate.provenance?.length" class="grid gap-2">
      <summary class="cursor-pointer text-gold focus:outline-none focus-visible:ring-2 focus-visible:ring-[var(--accent-gold)]">Provenance</summary>
      <ul class="mt-3 grid gap-2 pl-5 text-body text-text-secondary">
        <li v-for="item in candidate.provenance" :key="item.id">
          <span>{{ item.field }}:</span> {{ item.value || 'Unknown' }}
          <SafeExternalLink :href="item.sourceUrl" class="text-gold">source</SafeExternalLink>
          <span class="text-text-muted">{{ item.verificationState }}, {{ item.confidence || 'unknown confidence' }}</span>
        </li>
      </ul>
    </details>
    <p v-else class="m-0 text-body text-text-secondary">No detailed provenance was returned for this candidate.</p>

    <section v-if="candidate.lifecycleState !== 'converted'" class="grid gap-3 border-t border-border-subtle pt-3">
      <div v-if="candidate.lifecycleState === 'dismissed'" class="flex flex-wrap items-center gap-2">
        <button class="btn btn-secondary btn-sm focus:outline-none focus-visible:ring-2 focus-visible:ring-[var(--accent-gold)]" type="button" :disabled="busy" @click="$emit('restore', candidate)">Restore</button>
      </div>
      <template v-else>
        <div class="flex flex-col gap-2 md:flex-row md:flex-wrap md:items-center">
          <select
            v-model="dismissReason"
            class="form-select md:min-w-[11rem] focus:outline-none focus-visible:ring-2 focus-visible:ring-[var(--accent-gold)]"
            :disabled="busy"
          >
            <option value="irrelevant">Irrelevant</option>
            <option value="duplicate">Duplicate</option>
            <option value="price_too_high">Price too high</option>
            <option value="poor_provenance">Poor provenance</option>
            <option value="other">Other</option>
          </select>
          <input
            v-model.trim="dismissNotes"
            class="form-input md:flex-1 focus:outline-none focus-visible:ring-2 focus-visible:ring-[var(--accent-gold)]"
            :disabled="busy"
            maxlength="300"
            placeholder="Optional note"
          />
          <button class="btn btn-secondary btn-sm focus:outline-none focus-visible:ring-2 focus-visible:ring-[var(--accent-gold)]" type="button" :disabled="busy" @click="emitDismiss">Dismiss</button>
        </div>
        <details class="grid gap-3">
          <summary class="cursor-pointer text-gold focus:outline-none focus-visible:ring-2 focus-visible:ring-[var(--accent-gold)]">Convert to wishlist item</summary>
          <div class="mt-3 grid gap-3 md:grid-cols-2">
            <label class="grid gap-1 text-base text-text-secondary">Name <input v-model.trim="coin.name" class="form-input focus:outline-none focus-visible:ring-2 focus-visible:ring-[var(--accent-gold)]" /></label>
            <label class="grid gap-1 text-base text-text-secondary">Category <input v-model.trim="coin.category" class="form-input focus:outline-none focus-visible:ring-2 focus-visible:ring-[var(--accent-gold)]" /></label>
            <label class="grid gap-1 text-base text-text-secondary">Denomination <input v-model.trim="coin.denomination" class="form-input focus:outline-none focus-visible:ring-2 focus-visible:ring-[var(--accent-gold)]" /></label>
            <label class="grid gap-1 text-base text-text-secondary">Ruler <input v-model.trim="coin.ruler" class="form-input focus:outline-none focus-visible:ring-2 focus-visible:ring-[var(--accent-gold)]" /></label>
            <label class="grid gap-1 text-base text-text-secondary">Era <input v-model.trim="coin.era" class="form-input focus:outline-none focus-visible:ring-2 focus-visible:ring-[var(--accent-gold)]" /></label>
            <label class="grid gap-1 text-base text-text-secondary">Mint <input v-model.trim="coin.mint" class="form-input focus:outline-none focus-visible:ring-2 focus-visible:ring-[var(--accent-gold)]" /></label>
            <label class="grid gap-1 text-base text-text-secondary">Material <input v-model.trim="coin.material" class="form-input focus:outline-none focus-visible:ring-2 focus-visible:ring-[var(--accent-gold)]" /></label>
            <label class="grid gap-1 text-base text-text-secondary">Grade <input v-model.trim="coin.grade" class="form-input focus:outline-none focus-visible:ring-2 focus-visible:ring-[var(--accent-gold)]" /></label>
            <label class="grid gap-1 text-base text-text-secondary">Price <input v-model.number="coin.purchasePrice" class="form-input focus:outline-none focus-visible:ring-2 focus-visible:ring-[var(--accent-gold)]" type="number" min="0" step="0.01" /></label>
            <div class="grid gap-1 text-base text-text-secondary">
              <span>Source</span>
              <SafeExternalLink v-if="coin.referenceUrl" :href="coin.referenceUrl" class="break-words text-gold">{{ coin.referenceUrl }}</SafeExternalLink>
              <span v-else class="text-body text-text-secondary">No source URL provided</span>
            </div>
          </div>
          <p class="m-0 text-body text-text-secondary">Review missing or uncertain fields before saving. Only source-backed candidate fields are prefilled.</p>
          <p v-if="convertError" class="m-0 text-body text-bronze">{{ convertError }}</p>
          <label v-if="showDuplicateAck" class="mt-3 flex items-center gap-2 text-base text-text-secondary">
            <input
              v-model="ackDuplicate"
              class="h-4 w-4 rounded-sm border border-border-subtle bg-input accent-[var(--accent-gold)] focus:outline-none focus-visible:ring-2 focus-visible:ring-[var(--accent-gold)]"
              type="checkbox"
            />
            I acknowledge this may duplicate an existing wishlist item.
          </label>
          <div class="flex flex-wrap items-center gap-2">
            <button class="btn btn-primary btn-sm focus:outline-none focus-visible:ring-2 focus-visible:ring-[var(--accent-gold)]" type="button" :disabled="busy || !canConvert" @click="emitConvert">Save as Wishlist Item</button>
          </div>
        </details>
      </template>
    </section>
    <router-link
      v-else-if="candidate.convertedCoinId"
      class="btn btn-secondary btn-sm w-fit focus:outline-none focus-visible:ring-2 focus-visible:ring-[var(--accent-gold)]"
      :to="`/coin/${candidate.convertedCoinId}`"
    >
      Open converted wishlist item
    </router-link>
  </article>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import SafeExternalLink from '@/components/SafeExternalLink.vue'
import { MATERIALS, type AlertCandidate, type CandidateDismissalReason, type CoinMutationPayload, type Material } from '@/types'

const props = defineProps<{ candidate: AlertCandidate; busy?: boolean; duplicateWarnings?: string[] }>()
const emit = defineEmits<{
  dismiss: [candidate: AlertCandidate, reason: CandidateDismissalReason, notes: string]
  restore: [candidate: AlertCandidate]
  convert: [candidate: AlertCandidate, coin: CoinMutationPayload, acknowledgeDuplicateWarning: boolean]
}>()

const dismissReason = ref<CandidateDismissalReason>('irrelevant')
const dismissNotes = ref('')
const ackDuplicate = ref(false)
const coin = reactive<CoinMutationPayload>({})

const provenanceLabel = computed(() => props.candidate.provenanceStatus.replace(/_/g, ' '))
const stateLabel = computed(() => props.candidate.lifecycleState.replace(/_/g, ' '))
const priceLabel = computed(() => props.candidate.observedPrice == null ? 'Unknown' : `${props.candidate.observedPrice.toLocaleString()} ${props.candidate.observedCurrency || 'USD'}`)
const sourceFields = computed(() => Object.entries(props.candidate.fields ?? {}).filter(([, value]) => String(value ?? '').trim()))
const showDuplicateAck = computed(() => !!props.candidate.matchingWishlistCoinId || !!props.duplicateWarnings?.length)
const convertError = computed(() => {
  if (!String(coin.name ?? '').trim()) return 'Name is required before conversion.'
  if (!String(coin.category ?? '').trim()) return 'Category is required before conversion.'
  if (!String(coin.era ?? '').trim()) return 'Era is required before conversion.'
  if (showDuplicateAck.value && !ackDuplicate.value) return 'Acknowledge the duplicate warning before saving.'
  return ''
})
const canConvert = computed(() => !convertError.value)

watch(() => props.candidate, resetCoin, { immediate: true })

function resetCoin(candidate: AlertCandidate) {
  coin.name = candidate.title || ''
  coin.category = candidateField(candidate, ['category'])
  coin.denomination = candidateField(candidate, ['denomination', 'coinType', 'type'])
  coin.ruler = candidateField(candidate, ['ruler', 'rulerOrIssuer', 'issuer'])
  coin.era = candidateField(candidate, ['era'])
  coin.mint = candidateField(candidate, ['mint'])
  coin.material = candidateMaterial(candidate)
  coin.grade = candidateField(candidate, ['grade', 'condition'])
  coin.purchasePrice = candidate.observedPrice
  coin.currentValue = candidate.observedPrice
  coin.purchaseLocation = candidate.sourceName || ''
  coin.referenceUrl = candidate.sourceUrl || ''
  coin.referenceText = `Source-backed candidate from wishlist search alert #${candidate.alertId}`
  coin.notes = `Converted from alert candidate #${candidate.id}`
  coin.isWishlist = true
  ackDuplicate.value = false
}

function formatDate(value: string) {
  return value ? new Date(value).toLocaleString() : 'Unknown'
}

function candidateField(candidate: AlertCandidate, keys: string[]) {
  const fields = candidate.fields ?? {}
  const entry = Object.entries(fields).find(([field, value]) =>
    keys.some((key) => field.toLowerCase() === key.toLowerCase()) && String(value ?? '').trim()
  )
  return entry?.[1] ?? ''
}

function candidateMaterial(candidate: AlertCandidate): Material | undefined {
  const value = candidateField(candidate, ['material', 'metal'])
  return MATERIALS.find((material) => material.toLowerCase() === value.toLowerCase())
}

function formatFieldLabel(field: string) {
  return field.replace(/([a-z])([A-Z])/g, '$1 $2').replace(/_/g, ' ')
}

function emitDismiss() {
  emit('dismiss', props.candidate, dismissReason.value, dismissNotes.value)
}

function emitConvert() {
  emit('convert', props.candidate, { ...coin }, ackDuplicate.value)
}
</script>
