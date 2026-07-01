<template>
  <article class="candidate-card" :class="candidate.lifecycleState">
    <header class="candidate-header">
      <div>
        <p class="section-label">{{ candidate.sourceName || 'Unknown source' }}</p>
        <h3>{{ candidate.title || 'Unknown title' }}</h3>
      </div>
      <div class="badges">
        <span class="chip-sm" :class="candidate.provenanceStatus">{{ provenanceLabel }}</span>
        <span class="chip-sm">{{ stateLabel }}</span>
      </div>
    </header>

    <div class="candidate-meta">
      <div><span class="info-label">Price</span><strong>{{ priceLabel }}</strong></div>
      <div><span class="info-label">Last seen</span><strong>{{ formatDate(candidate.lastSeenAt) }}</strong></div>
      <div v-if="candidate.matchingWishlistCoinId"><span class="info-label">Duplicate warning</span><strong>Matches wishlist coin #{{ candidate.matchingWishlistCoinId }}</strong></div>
      <div v-if="candidate.duplicateOfCandidateId"><span class="info-label">Suppressed duplicate</span><strong>Candidate #{{ candidate.duplicateOfCandidateId }}</strong></div>
    </div>

    <p class="reason">{{ candidate.reasonForMatch || 'No source-backed match reason provided.' }}</p>

    <SafeExternalLink :href="candidate.sourceUrl" class="source-link">Open source listing</SafeExternalLink>

    <dl v-if="sourceFields.length" class="source-fields">
      <div v-for="[field, value] in sourceFields" :key="field">
        <dt>{{ formatFieldLabel(field) }}</dt>
        <dd>{{ value }}</dd>
      </div>
    </dl>

    <details v-if="candidate.provenance?.length" class="provenance">
      <summary>Provenance</summary>
      <ul>
        <li v-for="item in candidate.provenance" :key="item.id">
          <span>{{ item.field }}:</span> {{ item.value || 'Unknown' }}
          <SafeExternalLink :href="item.sourceUrl" class="evidence-link">source</SafeExternalLink>
          <span class="muted">{{ item.verificationState }}, {{ item.confidence || 'unknown confidence' }}</span>
        </li>
      </ul>
    </details>
    <p v-else class="muted">No detailed provenance was returned for this candidate.</p>

    <section v-if="candidate.lifecycleState !== 'converted'" class="review-actions">
      <div v-if="candidate.lifecycleState === 'dismissed'" class="actions-row">
        <button class="btn btn-secondary btn-sm" type="button" :disabled="busy" @click="$emit('restore', candidate)">Restore</button>
      </div>
      <template v-else>
        <div class="dismiss-controls">
          <select v-model="dismissReason" :disabled="busy">
            <option value="irrelevant">Irrelevant</option>
            <option value="duplicate">Duplicate</option>
            <option value="price_too_high">Price too high</option>
            <option value="poor_provenance">Poor provenance</option>
            <option value="other">Other</option>
          </select>
          <input v-model.trim="dismissNotes" :disabled="busy" maxlength="300" placeholder="Optional note" />
          <button class="btn btn-secondary btn-sm" type="button" :disabled="busy" @click="emitDismiss">Dismiss</button>
        </div>
        <details class="convert-box">
          <summary>Convert to wishlist item</summary>
          <div class="convert-grid">
            <label>Name <input v-model.trim="coin.name" /></label>
            <label>Category <input v-model.trim="coin.category" /></label>
            <label>Denomination <input v-model.trim="coin.denomination" /></label>
            <label>Ruler <input v-model.trim="coin.ruler" /></label>
            <label>Era <input v-model.trim="coin.era" /></label>
            <label>Mint <input v-model.trim="coin.mint" /></label>
            <label>Material <input v-model.trim="coin.material" /></label>
            <label>Grade <input v-model.trim="coin.grade" /></label>
            <label>Price <input v-model.number="coin.purchasePrice" type="number" min="0" step="0.01" /></label>
            <div class="convert-source">
              <span>Source</span>
              <SafeExternalLink v-if="coin.referenceUrl" :href="coin.referenceUrl" class="source-link">{{ coin.referenceUrl }}</SafeExternalLink>
              <span v-else class="muted">No source URL provided</span>
            </div>
          </div>
          <p class="muted">Review missing or uncertain fields before saving. Only source-backed candidate fields are prefilled.</p>
          <p v-if="convertError" class="error-text">{{ convertError }}</p>
          <label v-if="showDuplicateAck" class="ack"><input v-model="ackDuplicate" type="checkbox" /> I acknowledge this may duplicate an existing wishlist item.</label>
          <div class="actions-row">
            <button class="btn btn-primary btn-sm" type="button" :disabled="busy || !canConvert" @click="emitConvert">Save as Wishlist Item</button>
          </div>
        </details>
      </template>
    </section>
    <router-link v-else-if="candidate.convertedCoinId" class="btn btn-secondary btn-sm converted-link" :to="`/coin/${candidate.convertedCoinId}`">Open converted wishlist item</router-link>
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

<style scoped>
.candidate-card { border: 1px solid var(--border-subtle); border-radius: var(--radius-md); background: var(--bg-card); padding: 1rem; display: grid; gap: 0.75rem; }
.candidate-card.suppressed { opacity: 0.75; }
.candidate-header { display: flex; justify-content: space-between; gap: 1rem; align-items: flex-start; }
h3 { margin: 0.15rem 0 0; }
.section-label, .info-label { font-size: 0.7rem; font-weight: 600; text-transform: uppercase; letter-spacing: 0.08em; color: var(--text-muted); margin: 0; }
.badges, .actions-row, .dismiss-controls { display: flex; gap: 0.5rem; flex-wrap: wrap; align-items: center; }
.verified { color: var(--accent-gold); border-color: var(--accent-gold); }
.partial { color: var(--accent-bronze); border-color: var(--accent-bronze); }
.unverified { color: var(--text-muted); }
.candidate-meta, .source-fields { display: grid; grid-template-columns: repeat(auto-fit, minmax(150px, 1fr)); gap: 0.75rem; }
.candidate-meta div, .source-fields div { border: 1px solid var(--border-subtle); border-radius: var(--radius-sm); background: var(--bg-input); padding: 0.75rem; display: grid; gap: 0.25rem; }
.source-fields { margin: 0; }
.source-fields dt { color: var(--text-muted); font-size: 0.7rem; font-weight: 600; text-transform: uppercase; letter-spacing: 0.08em; }
.source-fields dd { margin: 0; color: var(--text-primary); }
.reason, .muted { color: var(--text-secondary); margin: 0; }
.source-link, .evidence-link, .converted-link { color: var(--accent-gold); }
.provenance summary, .convert-box summary { color: var(--accent-gold); cursor: pointer; }
.provenance ul { margin: 0.75rem 0 0; padding-left: 1.25rem; color: var(--text-secondary); }
.review-actions { border-top: 1px solid var(--border-subtle); padding-top: 0.75rem; display: grid; gap: 0.75rem; }
select, input { border: 1px solid var(--border-subtle); border-radius: var(--radius-sm); padding: 0.5rem; background: var(--bg-input); color: var(--text-primary); }
.convert-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(160px, 1fr)); gap: 0.75rem; margin-top: 0.75rem; }
.convert-grid label, .convert-source { display: grid; gap: 0.25rem; color: var(--text-secondary); }
.convert-source a { overflow-wrap: anywhere; }
.ack { display: flex; gap: 0.5rem; align-items: center; color: var(--text-secondary); margin-top: 0.75rem; }
.error-text { color: var(--accent-bronze); margin: 0.75rem 0 0; }
@media (max-width: 640px) { .candidate-header { flex-direction: column; } .dismiss-controls { align-items: stretch; } .dismiss-controls > * { width: 100%; } }
</style>
