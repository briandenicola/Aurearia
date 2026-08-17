<template>
  <div class="detail-actions">
    <div class="section-content-card">
      <div class="mb-6">
        <h3 class="mb-3 text-base font-medium text-text-primary">Upload Images</h3>
        <div>
          <div class="action-row">
            <select v-model="uploadType" class="form-select action-input flex-1">
              <option value="obverse">Obverse</option>
              <option value="reverse">Reverse</option>
              <option value="detail">Detail</option>
              <option value="other">Other</option>
            </select>
            <label class="btn btn-secondary btn-sm action-btn cursor-pointer whitespace-nowrap">
              Choose File
              <input type="file" accept="image/*" hidden @change="handleImageUpload" />
            </label>
            <button
              v-if="isPwa"
              type="button"
              class="btn btn-secondary btn-sm action-btn inline-flex items-center gap-1 whitespace-nowrap"
              @click="showCameraModal = true"
            >
              <Camera :size="14" /> Photo
            </button>
          </div>

          <div class="mt-2 action-row">
            <input
              v-model="imageUrl"
              type="url"
              class="form-input action-input min-w-0 flex-1 text-[0.82rem]"
              placeholder="Or paste an image URL..."
              @keydown.enter="handleUrlUpload"
            />
            <button
              class="btn btn-secondary btn-sm action-btn"
              :disabled="!imageUrl || urlLoading"
              @click="handleUrlUpload"
            >
              {{ urlLoading ? 'Fetching...' : 'Fetch' }}
            </button>
          </div>

          <p v-if="uploadStatus" class="mt-2 text-chip" :class="uploadError ? 'text-loss' : 'text-gold'">{{ uploadStatus }}</p>
        </div>
      </div>

      <div class="mb-6">
        <div class="mb-3 flex items-center justify-between gap-3 max-sm:flex-col max-sm:items-stretch">
          <h3 class="m-0 text-base font-medium text-text-primary">AI Value Estimate</h3>
          <button
            class="btn btn-secondary btn-sm action-btn"
            :disabled="estimating"
            @click="handleEstimateValue"
          >
            {{ estimating ? 'Estimating...' : 'Estimate Value' }}
          </button>
        </div>
        <div v-if="estimating" class="flex items-center gap-3 rounded-sm border border-border-subtle bg-card p-4 text-text-secondary">
          <div class="h-5 w-5 animate-spin rounded-full border-2 border-border-subtle border-t-gold" />
          <span>{{ estimateStatusMessage || 'Estimating current market value...' }}</span>
        </div>
        <div v-if="estimateError" class="px-2 py-2 text-base text-loss">{{ estimateError }}</div>
        <div v-if="valueEstimate" class="rounded-sm border border-border-subtle bg-card p-4">
          <div class="mb-3 flex items-center gap-3 max-sm:flex-col max-sm:items-start">
            <span class="text-xl font-bold text-gold">{{ valueEstimate.estimatedValue ? formatCurrency(valueEstimate.estimatedValue) : 'N/A' }}</span>
            <span
              class="rounded-full px-[0.6rem] py-[0.2rem] text-sm font-semibold uppercase tracking-[0.03em]"
              :class="{
                'bg-gold-glow text-gold': valueEstimate.confidence === 'high',
                'bg-gold-dim text-text-primary': valueEstimate.confidence === 'medium',
                'bg-input text-text-secondary': valueEstimate.confidence === 'low',
              }"
            >
              {{ valueEstimate.confidence }} confidence
            </span>
          </div>
          <p class="mb-3 text-base leading-6 text-text-secondary">{{ valueEstimate.reasoning }}</p>
          <div v-if="valueEstimate.comparables?.length" class="mb-3">
            <h4 class="mb-2 text-label font-semibold uppercase tracking-[0.08em] text-text-muted">Comparable Listings</h4>
            <div v-for="(comp, i) in valueEstimate.comparables" :key="i" class="flex items-center justify-between gap-3 border-b border-border-subtle py-1.5 last:border-b-0">
              <SafeExternalLink
                v-if="safeComparableUrl(comp.url)"
                :href="comp.url"
                target="_blank"
                rel="noopener"
                class="text-body text-gold no-underline hover:underline"
              >
                {{ comp.source }}
              </SafeExternalLink>
              <span v-else class="text-body text-gold">{{ comp.source }}</span>
              <span class="text-body font-semibold text-text-primary">{{ comp.price }}</span>
            </div>
          </div>
          <div class="mt-3 flex gap-2 max-sm:flex-col">
            <button class="btn btn-primary btn-sm" @click="handleApplyEstimate">
              Apply as Current Value
            </button>
            <button class="btn btn-ghost btn-sm" @click="dismissEstimate">
              Dismiss
            </button>
          </div>
        </div>
      </div>

      <div class="flex flex-wrap items-center justify-between gap-3 border-t border-border-subtle pt-4">
        <span class="text-body text-text-secondary">Catalog lookup and saved references</span>
        <RouterLink class="btn btn-ghost btn-sm" :to="`/coin/${coinId}#catalog-references`">
          Catalog References
        </RouterLink>
      </div>

      <div v-if="deepAnalysisEnabled" class="mt-4 flex flex-wrap items-center justify-between gap-3 border-t border-border-subtle pt-4">
        <span class="text-body text-text-secondary">Re-analyze this coin with multiple catalog providers</span>
        <DeepAnalysisEntryButton
          :disabled="Boolean(deepAnalysisDisabledTitle)"
          :title="deepAnalysisDisabledTitle"
          @click="showDeepAnalysisModal = true"
        />
      </div>
    </div>

    <div v-if="showDeepAnalysisModal" class="fixed inset-0 z-[1000] flex items-center justify-center bg-overlay p-4" @click.self="showDeepAnalysisModal = false">
      <div class="card max-h-[90vh] w-full max-w-[640px] overflow-y-auto p-6">
        <div class="mb-4 flex items-center justify-between">
          <h2 class="text-lg text-heading">Deep Analysis</h2>
          <AppIconButton title="Close" @click="showDeepAnalysisModal = false">
            <X :size="18" />
          </AppIconButton>
        </div>
        <DeepAnalysisStartPanel
          :coin-id="coinId"
          :has-existing-obverse="coinHasObverseImage"
          :has-existing-reverse="coinHasReverseImage"
          :eligible-providers="deepAnalysisProviders"
          :submitting="deepIdentification.starting.value"
          :submit-error="deepIdentification.error.value"
          :conflict-job-id="launcher.capacityConflictJobId.value"
          @submit="onDeepAnalysisSubmit"
          @cancel="showDeepAnalysisModal = false"
        />
      </div>
    </div>

    <CameraCaptureModal
      :is-open="showCameraModal"
      @close="showCameraModal = false"
      @captured="handleCameraCaptured"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { RouterLink } from 'vue-router'
import { formatCurrency } from '@/utils/format'
import CameraCaptureModal from '@/components/CameraCaptureModal.vue'
import SafeExternalLink from '@/components/SafeExternalLink.vue'
import { Camera, X } from 'lucide-vue-next'
import { sanitizeExternalUrl } from '@/composables/useSafeExternalLink'
import { useDeepAnalysisLauncher } from '@/composables/useDeepAnalysisLauncher'
import { useCoinImageUpload } from '@/composables/useCoinImageUpload'
import { useCoinValueEstimate } from '@/composables/useCoinValueEstimate'
import DeepAnalysisEntryButton from '@/components/deep-identification/DeepAnalysisEntryButton.vue'
import DeepAnalysisStartPanel from '@/components/deep-identification/DeepAnalysisStartPanel.vue'
import AppIconButton from '@/components/ui/AppIconButton.vue'
import type { Coin, CreateDeepIdentificationJobInput } from '@/types'

const props = defineProps<{
  coinId: number
  coinName: string
  coinRuler?: string | null
  coinDenomination?: string | null
  coinMint?: string | null
  coinDateRange?: string | null
  coinMaterial: Coin['material']
  coinObverseInscription?: string | null
  coinReverseInscription?: string | null
  imageCount: number
  coinHasObverseImage?: boolean
  coinHasReverseImage?: boolean
  isPwa: boolean
}>()

const emit = defineEmits<{
  imagesChanged: []
  estimateApplied: []
}>()

const launcher = useDeepAnalysisLauncher()
const {
  deepIdentification,
  deepAnalysisEnabled,
  deepAnalysisProviders,
  showDeepAnalysisModal,
  activeJobCount,
} = launcher
const deepAnalysisDisabledTitle = computed(() =>
  activeJobCount.value > 0
    ? 'You already have an active Deep Analysis job. Wait for it to finish before starting another.'
    : undefined,
)
const coinHasObverseImage = computed(() => props.coinHasObverseImage ?? false)
const coinHasReverseImage = computed(() => props.coinHasReverseImage ?? false)

const showCameraModal = ref(false)

const {
  uploadType,
  uploadStatus,
  uploadError,
  imageUrl,
  urlLoading,
  handleImageUpload,
  handleCameraCaptured,
  handleUrlUpload,
} = useCoinImageUpload(
  () => props.coinId,
  () => props.imageCount,
  { onUploaded: () => emit('imagesChanged') },
)

const {
  estimating,
  estimateStatusMessage,
  estimateError,
  valueEstimate,
  handleEstimateValue,
  handleApplyEstimate,
  dismissEstimate,
} = useCoinValueEstimate(
  () => props.coinId,
  { onApplied: () => emit('estimateApplied') },
)

function safeComparableUrl(url: string | null | undefined): string | null {
  return sanitizeExternalUrl(url)
}

async function onDeepAnalysisSubmit(input: CreateDeepIdentificationJobInput) {
  await launcher.submitDeepAnalysis({ ...input, coinId: props.coinId })
}
</script>

<style scoped>
.action-row {
  display: flex;
  align-items: stretch;
  gap: 0.5rem;
}

.action-input {
  min-height: 2rem;
}

.action-btn {
  min-height: 2rem;
  justify-content: center;
}

@media (max-width: 575px) {
  .action-row {
    flex-direction: column;
  }
}
</style>
