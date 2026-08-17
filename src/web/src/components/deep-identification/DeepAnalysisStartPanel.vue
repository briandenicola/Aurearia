<template>
  <div class="grid gap-5">
    <p class="text-body text-text-secondary">
      Deep Analysis runs multiple catalog providers in the background and returns a narrative
      report plus proposed fields for your review. It never saves or updates your coin automatically.
    </p>

    <div
      v-if="reuseCapturedEvidence"
      class="grid min-w-0 gap-1 rounded-sm border border-border-subtle bg-input p-3"
      data-testid="reused-capture-summary"
    >
      <strong class="text-base text-text-primary">Using photos from Identify Coin</strong>
      <span class="text-sm text-text-secondary">{{ reusedEvidenceSummary }}</span>
    </div>

    <div v-else class="grid min-w-0 gap-4 [grid-template-columns:repeat(auto-fit,minmax(min(180px,100%),1fr))]">
      <div
        v-if="!hasExistingObverse"
        class="relative grid min-h-[170px] cursor-pointer gap-3 rounded-sm border border-dashed border-border-accent bg-card p-4 transition-[border-color,background] duration-200 hover:border-gold hover:bg-card-hover"
      >
        <span class="text-base font-semibold text-heading">Obverse *</span>
        <img v-if="obverseUrl" :src="obverseUrl" alt="Obverse preview" class="aspect-square w-full rounded-sm border border-border-subtle object-cover">
        <span v-else class="grid min-h-20 place-items-center rounded-sm border border-dashed border-border-subtle p-3 text-center text-body text-text-secondary">Required obverse photo</span>
        <span class="chip pointer-events-none justify-self-start">{{ obverseImage ? 'Replace photo' : 'Choose photo' }}</span>
        <button v-if="obverseImage" type="button" class="relative z-10 min-h-[44px] justify-self-start bg-transparent px-2 text-sm text-byzantine underline" aria-label="Remove obverse photo" @click="obverseImage = null">Remove</button>
        <input class="absolute inset-0 cursor-pointer opacity-0" type="file" accept="image/*" capture="environment" aria-label="Choose obverse photo" @change="onSingleFile('obverse', $event)">
      </div>
      <div v-else class="grid min-h-[170px] gap-3 rounded-sm border border-border-subtle bg-card p-4">
        <span class="text-base font-semibold text-heading">Obverse</span>
        <span class="grid min-h-20 place-items-center rounded-sm border border-border-subtle bg-input p-3 text-center text-body text-text-secondary">Using this coin's existing obverse photo</span>
      </div>
      <div
        v-if="!hasExistingReverse"
        class="relative grid min-h-[170px] cursor-pointer gap-3 rounded-sm border border-dashed border-border-accent bg-card p-4 transition-[border-color,background] duration-200 hover:border-gold hover:bg-card-hover"
      >
        <span class="text-base font-semibold text-heading">Reverse *</span>
        <img v-if="reverseUrl" :src="reverseUrl" alt="Reverse preview" class="aspect-square w-full rounded-sm border border-border-subtle object-cover">
        <span v-else class="grid min-h-20 place-items-center rounded-sm border border-dashed border-border-subtle p-3 text-center text-body text-text-secondary">Required reverse photo</span>
        <span class="chip pointer-events-none justify-self-start">{{ reverseImage ? 'Replace photo' : 'Choose photo' }}</span>
        <button v-if="reverseImage" type="button" class="relative z-10 min-h-[44px] justify-self-start bg-transparent px-2 text-sm text-byzantine underline" aria-label="Remove reverse photo" @click="reverseImage = null">Remove</button>
        <input class="absolute inset-0 cursor-pointer opacity-0" type="file" accept="image/*" capture="environment" aria-label="Choose reverse photo" @change="onSingleFile('reverse', $event)">
      </div>
      <div v-else class="grid min-h-[170px] gap-3 rounded-sm border border-border-subtle bg-card p-4">
        <span class="text-base font-semibold text-heading">Reverse</span>
        <span class="grid min-h-20 place-items-center rounded-sm border border-border-subtle bg-input p-3 text-center text-body text-text-secondary">Using this coin's existing reverse photo</span>
      </div>
      <div class="relative grid min-h-[170px] cursor-pointer gap-3 rounded-sm border border-dashed border-border-accent bg-card p-4 transition-[border-color,background] duration-200 hover:border-gold hover:bg-card-hover">
        <span class="text-base font-semibold text-heading">Hint photos</span>
        <span class="grid min-h-20 place-items-center rounded-sm border border-dashed border-border-subtle p-3 text-center text-body text-text-secondary">{{ hintCountText }}</span>
        <span class="chip pointer-events-none justify-self-start">{{ hintImages.length ? 'Replace hints' : 'Choose hints' }}</span>
        <button v-if="hintImages.length" type="button" class="relative z-10 min-h-[44px] justify-self-start bg-transparent px-2 text-sm text-byzantine underline" aria-label="Remove hint photos" @click="hintImages = []">Remove</button>
        <input class="absolute inset-0 cursor-pointer opacity-0" type="file" accept="image/*" multiple aria-label="Choose hint photos" @change="onHintFiles">
      </div>
    </div>
    <p v-if="!reuseCapturedEvidence" class="text-sm text-text-secondary">
      Hint photos (labels, envelopes, references) are used only during analysis and are never saved to your coin.
    </p>

    <label v-if="!reuseCapturedEvidence" class="grid gap-2">
      <span class="text-base font-semibold text-heading">Notes (optional)</span>
      <textarea
        v-model="notes"
        maxlength="2000"
        rows="3"
        class="rounded-sm border border-border-subtle bg-card p-3 text-body text-text-primary"
        placeholder="Anything providers should know (mint marks, inscriptions, provenance, etc.)"
      />
    </label>

    <fieldset class="grid gap-2 rounded-sm border border-border-subtle p-3">
      <legend class="px-1 text-base font-semibold text-heading">Providers (optional override)</legend>
      <p class="text-sm text-text-secondary">Leave all unchecked to let Deep Analysis choose providers automatically.</p>
      <label v-for="providerId in eligibleProviders" :key="providerId" class="flex min-h-[44px] items-center gap-2 text-body text-text-primary">
        <input type="checkbox" :value="providerId" :checked="selectedProviders.includes(providerId)" @change="toggleProvider(providerId)">
        {{ providerLabel(providerId) }}
      </label>
    </fieldset>

    <p v-if="validationError" role="alert" class="text-body text-byzantine">{{ validationError }}</p>
    <div v-if="submitError" role="alert" class="grid gap-2">
      <p class="m-0 text-body text-byzantine">{{ submitError }}</p>
      <RouterLink
        v-if="conflictJobId"
        class="btn btn-secondary btn-sm justify-self-start"
        :to="`/deep-analysis/${conflictJobId}`"
      >
        View running analysis
      </RouterLink>
    </div>

    <div class="flex flex-wrap justify-end gap-3">
      <BaseButton type="button" variant="ghost" @click="$emit('cancel')">Cancel</BaseButton>
      <BaseButton type="button" variant="primary" :loading="submitting" @click="onSubmit">Start Deep Analysis</BaseButton>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { RouterLink } from 'vue-router'
import BaseButton from '@/components/ui/BaseButton.vue'
import type { CreateDeepIdentificationJobInput, DeepProviderId } from '@/types'

const MAX_DEEP_HINT_IMAGES = 3

function providerLabel(id: DeepProviderId): string {
  switch (id) {
    case 'nomisma': return 'Nomisma'
    case 'numista': return 'Numista'
    case 'ngc': return 'NGC (link-out only)'
    case 'ocre': return 'OCRE'
    case 'rpc': return 'RPC (unavailable)'
    default: return id
  }
}

const props = withDefaults(defineProps<{
  coinId?: number | null
  hasExistingObverse?: boolean
  hasExistingReverse?: boolean
  reuseCapturedEvidence?: boolean
  initialObverseImage?: File | null
  initialReverseImage?: File | null
  initialHintImages?: File[]
  initialNotes?: string
  eligibleProviders?: DeepProviderId[]
  submitting?: boolean
  submitError?: string
  // Set only when submitError came from a `job_at_capacity` (HTTP 409)
  // conflict - the id of the already-running job to link the user to, so
  // they can go wait on it or cancel it instead of being stuck with a
  // dead-end error.
  conflictJobId?: number | null
}>(), {
  coinId: null,
  hasExistingObverse: false,
  hasExistingReverse: false,
  reuseCapturedEvidence: false,
  initialObverseImage: null,
  initialReverseImage: null,
  initialHintImages: () => [],
  initialNotes: '',
  eligibleProviders: () => ['nomisma', 'numista'],
  submitting: false,
  submitError: '',
  conflictJobId: null,
})

const emit = defineEmits<{
  submit: [input: CreateDeepIdentificationJobInput]
  cancel: []
}>()

const obverseImage = ref<File | null>(props.initialObverseImage)
const reverseImage = ref<File | null>(props.initialReverseImage)
const hintImages = ref<File[]>([...props.initialHintImages])
const notes = ref(props.initialNotes)
const selectedProviders = ref<DeepProviderId[]>([])
const validationError = ref('')

const obverseUrl = ref('')
const reverseUrl = ref('')

function refreshUrl(target: 'obverse' | 'reverse', file: File | null) {
  const current = target === 'obverse' ? obverseUrl : reverseUrl
  if (current.value) URL.revokeObjectURL(current.value)
  current.value = file ? URL.createObjectURL(file) : ''
}

function onSingleFile(target: 'obverse' | 'reverse', event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0] ?? null
  if (target === 'obverse') obverseImage.value = file
  else reverseImage.value = file
}

function onHintFiles(event: Event) {
  const input = event.target as HTMLInputElement
  hintImages.value = Array.from(input.files ?? []).slice(0, MAX_DEEP_HINT_IMAGES)
}

function toggleProvider(id: DeepProviderId) {
  const idx = selectedProviders.value.indexOf(id)
  if (idx >= 0) selectedProviders.value.splice(idx, 1)
  else selectedProviders.value.push(id)
}

const hintCountText = computed(() => {
  if (!hintImages.value.length) return `Optional, up to ${MAX_DEEP_HINT_IMAGES}`
  return `${hintImages.value.length} of ${MAX_DEEP_HINT_IMAGES} selected`
})
const reusedEvidenceSummary = computed(() => {
  const parts: string[] = []
  if (obverseImage.value || props.hasExistingObverse) parts.push('obverse')
  if (reverseImage.value || props.hasExistingReverse) parts.push('reverse')
  if (hintImages.value.length > 0) {
    parts.push(`${hintImages.value.length} supporting ${hintImages.value.length === 1 ? 'image' : 'images'}`)
  }
  if (notes.value.trim()) parts.push('notes')
  return parts.join(', ') || 'No evidence selected'
})

function watchUrl(target: 'obverse' | 'reverse', file: File | null) {
  refreshUrl(target, file)
}

watch(obverseImage, file => watchUrl('obverse', file), { immediate: true })
watch(reverseImage, file => watchUrl('reverse', file), { immediate: true })
onBeforeUnmount(() => {
  if (obverseUrl.value) URL.revokeObjectURL(obverseUrl.value)
  if (reverseUrl.value) URL.revokeObjectURL(reverseUrl.value)
})

function onSubmit() {
  validationError.value = ''
  const obverseMissing = !props.hasExistingObverse && !obverseImage.value
  const reverseMissing = !props.hasExistingReverse && !reverseImage.value
  if (obverseMissing || reverseMissing) {
    validationError.value = 'Obverse and reverse photos are both required to start Deep Analysis.'
    return
  }
  if (hintImages.value.length > MAX_DEEP_HINT_IMAGES) {
    validationError.value = `You can attach at most ${MAX_DEEP_HINT_IMAGES} hint photos.`
    return
  }
  emit('submit', {
    coinId: props.coinId ?? undefined,
    obverseImage: obverseImage.value,
    reverseImage: reverseImage.value,
    hintImages: hintImages.value,
    notes: notes.value,
    providers: selectedProviders.value.length ? selectedProviders.value : undefined,
  })
}
</script>
