<template>
  <section class="card mt-6">
    <div class="mb-4 flex items-start justify-between gap-4">
      <div>
        <span class="section-label">Ready for cataloging</span>
        <h2 class="mt-1">Promote Draft</h2>
      </div>
    </div>

    <!-- Already promoted -->
    <template v-if="alreadyPromoted">
      <p class="text-base text-text-secondary">This draft was already promoted.</p>
      <RouterLink class="btn btn-primary" :to="`/coin/${promotedCoinId}`">View Coin</RouterLink>
    </template>

    <!-- Success -->
    <template v-else-if="successCoinId">
      <p class="text-base text-text-secondary">Draft promoted successfully.</p>
      <RouterLink class="btn btn-primary" :to="`/coin/${successCoinId}`">View Coin</RouterLink>
    </template>

    <!-- Promotion form -->
    <template v-else>
      <p class="mb-4 text-body text-text-secondary">Choose where this coin should land, review readiness, then promote it. Repeated promotion is safe.</p>

      <fieldset class="m-0 grid border-0 p-0 [grid-template-columns:repeat(auto-fit,minmax(220px,1fr))] gap-3">
        <legend class="section-label col-span-full mb-1">Promote to</legend>
        <label
          class="flex cursor-pointer items-start gap-3 rounded-sm border border-border-subtle bg-input p-3 text-text-primary transition-[border-color,background,box-shadow] duration-200"
          :class="target === 'collection' ? 'border-gold bg-gold-glow shadow-glow' : ''"
        >
          <input v-model="target" type="radio" value="collection" class="mt-[0.2rem] shrink-0 accent-gold">
          <Coins :size="20" class="shrink-0 text-gold" />
          <span class="grid gap-1">
            <strong class="text-base font-semibold text-text-primary">Collection</strong>
            <small class="text-sm text-text-secondary">Counts as an owned collection coin.</small>
          </span>
        </label>
        <label
          class="flex cursor-pointer items-start gap-3 rounded-sm border border-border-subtle bg-input p-3 text-text-primary transition-[border-color,background,box-shadow] duration-200"
          :class="target === 'wishlist' ? 'border-gold bg-gold-glow shadow-glow' : ''"
        >
          <input v-model="target" type="radio" value="wishlist" class="mt-[0.2rem] shrink-0 accent-gold">
          <Bookmark :size="20" class="shrink-0 text-gold" />
          <span class="grid gap-1">
            <strong class="text-base font-semibold text-text-primary">Wishlist</strong>
            <small class="text-sm text-text-secondary">Tracks as a wanted coin instead.</small>
          </span>
        </label>
        <span v-if="fieldErrors.target" class="col-span-full text-body text-warning">{{ fieldErrors.target }}</span>
      </fieldset>

      <div class="mt-4 grid gap-3 rounded-sm border border-border-subtle bg-input p-3" aria-live="polite">
        <div class="flex items-start gap-3 text-text-secondary" :class="{ 'text-text-primary': hasRequiredName }">
          <CheckCircle v-if="hasRequiredName" :size="18" class="mt-[0.15rem] shrink-0 text-gold" />
          <AlertCircle v-else :size="18" class="mt-[0.15rem] shrink-0 text-warning" />
          <span class="grid gap-1">
            <strong class="text-base font-semibold text-text-primary">{{ hasRequiredName ? 'Required title is ready' : 'Working title is required' }}</strong>
            <small class="text-sm text-text-secondary">{{ hasRequiredName ? 'Promotion uses the current draft title above.' : 'Add a working title in the draft form before promoting.' }}</small>
          </span>
        </div>
        <div class="flex items-start gap-3 text-text-primary">
          <ImageIcon :size="18" class="mt-[0.15rem] shrink-0 text-gold" />
          <span class="grid gap-1">
            <strong class="text-base font-semibold text-text-primary">{{ imageCountLabel }}</strong>
            <small class="text-sm text-text-secondary">Saved draft images will move with the promoted coin.</small>
          </span>
        </div>
        <p class="m-0 text-sm text-text-secondary">Draft fields stay editable in the form above. This panel only chooses the destination and confirms the final action.</p>
        <span v-if="fieldErrors.name" class="text-body text-warning">{{ fieldErrors.name }}</span>
        <div v-if="readinessFieldErrors.length" class="grid text-body text-warning">
          <span v-for="[field, message] in readinessFieldErrors" :key="field">{{ message }}</span>
        </div>
      </div>

      <label class="my-4 flex cursor-pointer items-start gap-[0.6rem] text-base text-text-secondary">
        <input v-model="confirmed" type="checkbox" class="mt-[0.15rem] shrink-0 accent-gold">
        <span>I confirm promotion to {{ destinationLabel }}. This creates a permanent coin record.</span>
      </label>

      <p v-if="promoteError" class="mb-4 text-base text-warning">{{ promoteError }}</p>

      <div class="flex justify-end">
        <button
          type="button"
          class="btn btn-primary w-full justify-center sm:w-auto"
          :disabled="!confirmed || promoting || !hasRequiredName"
          @click="doPromote"
        >
          {{ promoting ? 'Promoting...' : `Promote to ${destinationLabel}` }}
        </button>
      </div>
    </template>
  </section>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { RouterLink } from 'vue-router'
import { getApiErrorMessage, promoteQuickCaptureDraft } from '@/api/client'
import type { QuickCaptureDraft, QuickCapturePromoteOverrides } from '@/types'
import { AlertCircle, Bookmark, CheckCircle, Coins, Image as ImageIcon } from 'lucide-vue-next'

const props = withDefaults(
  defineProps<{ draft: QuickCaptureDraft; promotionOverrides?: QuickCapturePromoteOverrides }>(),
  { promotionOverrides: () => ({}) }
)
const emit = defineEmits<{ promoted: [coinId: number] }>()

type PromotionTarget = 'collection' | 'wishlist'

const alreadyPromoted = computed(
  () => props.draft.status === 'promoted' && props.draft.promotedCoinId != null
)
const promotedCoinId = computed(() => props.draft.promotedCoinId)

const target = ref<PromotionTarget>('collection')
const confirmed = ref(false)
const promoting = ref(false)
const promoteError = ref('')
const fieldErrors = ref<Record<string, string>>({})
const successCoinId = ref<number | null>(null)
const destinationLabel = computed(() => target.value === 'wishlist' ? 'Wishlist' : 'Collection')
const hasRequiredName = computed(() => (props.promotionOverrides.name ?? props.draft.workingTitle ?? '').trim().length > 0)
const imageCount = computed(() => props.draft.images?.length ?? 0)
const imageCountLabel = computed(() => `${imageCount.value} saved ${imageCount.value === 1 ? 'image' : 'images'}`)
const readinessFieldErrors = computed(() => Object.entries(fieldErrors.value)
  .filter(([field]) => !['target', 'name', 'confirm'].includes(field)))

async function doPromote() {
  promoteError.value = ''
  fieldErrors.value = {}
  if (!hasRequiredName.value) {
    fieldErrors.value = { name: 'Name is required' }
    promoteError.value = 'Complete required fields before promotion.'
    return
  }
  promoting.value = true
  try {
    const res = await promoteQuickCaptureDraft(props.draft.id, {
      confirm: true,
      target: target.value,
      overrides: props.promotionOverrides,
    })
    if (res.data.alreadyPromoted) {
      // trigger idempotent path in parent
      emit('promoted', res.data.coinId)
    } else {
      successCoinId.value = res.data.coinId
      emit('promoted', res.data.coinId)
    }
  } catch (err: unknown) {
    const errData = (err as { response?: { data?: { error?: string; fields?: Record<string, string> } } })
      ?.response?.data
    if (errData?.fields) {
      fieldErrors.value = errData.fields
      promoteError.value = errData.error ?? 'Complete required fields before promotion.'
    } else {
      promoteError.value = getApiErrorMessage(err) || 'Promotion failed. Please try again.'
    }
  } finally {
    promoting.value = false
  }
}
</script>
