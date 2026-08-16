<template>
  <section class="flex flex-col gap-6" aria-labelledby="capture-wizard-title">
    <div class="flex flex-col gap-4">
      <div>
        <h2 id="capture-wizard-title" class="sr-only">Coin photo steps</h2>
        <span class="section-label">Step {{ step + 1 }} of 3</span>
        <h2 class="mt-1 text-heading">{{ currentStep.title }}</h2>
        <p class="mt-2 text-base leading-6 text-text-secondary">{{ currentStep.description }}</p>
      </div>

      <div v-if="currentImage" class="relative aspect-[4/3] overflow-hidden rounded-sm border border-border-accent bg-card">
        <img :src="currentImage.preview" :alt="`${currentStep.label} coin image`" class="h-full w-full object-contain" />
        <button
          type="button"
          class="absolute right-2 top-2 flex min-h-11 min-w-11 items-center justify-center rounded-sm bg-overlay text-text-primary"
          :aria-label="`Remove ${currentStep.label.toLowerCase()} image`"
          @click="$emit('remove', currentStep.role)"
        >
          <X :size="18" />
        </button>
      </div>

      <label v-if="currentStep.role === 'notes'" class="form-group">
        <span class="section-label">Identification notes</span>
        <textarea
          :value="notes"
          class="form-input min-h-[120px] resize-y"
          maxlength="2000"
          placeholder="Add weight, diameter, provenance, visible text, suspected ruler, denomination, or anything else that may help."
          @input="$emit('update:notes', ($event.target as HTMLTextAreaElement).value)"
        ></textarea>
        <span class="text-right text-tiny text-text-muted">{{ notes.length }} / 2000</span>
      </label>

      <p v-if="currentStep.role === 'notes'" class="text-small text-text-secondary">
        You may add text, one supporting image, or both.
      </p>

      <InlineCameraCapturePanel
        v-if="!currentImage"
        ref="cameraPanel"
        :filename-prefix="`lookup-${currentStep.role}`"
        :instruction="currentStep.instruction"
        @captured="$emit('captured', currentStep.role, $event)"
        @upload="fileInput?.click()"
      />

      <input
        ref="fileInput"
        type="file"
        accept="image/*"
        class="hidden"
        @change="handleFileSelection"
      />

      <div v-if="uploadError" class="flex items-center gap-3 rounded-md border border-border-accent bg-input p-4 text-base text-text-primary" role="alert">
        <AlertCircle :size="20" class="shrink-0 text-byzantine" />
        <span>{{ uploadError }}</span>
      </div>

      <div v-if="deepRequirementError" class="flex items-center gap-3 rounded-md border border-border-accent bg-input p-4 text-base text-text-primary" role="alert">
        <AlertCircle :size="20" class="shrink-0 text-byzantine" />
        <span>{{ deepRequirementError }}</span>
      </div>

      <div class="flex items-center gap-2">
        <button
          v-if="step > 0"
          type="button"
          class="btn btn-secondary min-h-11 min-w-11 shrink-0 justify-center px-3"
          title="Previous step"
          aria-label="Previous step"
          @click="step -= 1"
        >
          <ChevronLeft :size="20" aria-hidden="true" />
        </button>

        <button
          v-if="obverse"
          type="button"
          class="btn btn-primary min-w-0 flex-1 justify-center px-2 text-tiny sm:px-5 sm:text-base"
          :disabled="submitting || preparingImage"
          @click="$emit('analyze')"
        >
          <span v-if="submitting" class="inline-block h-[14px] w-[14px] animate-spin rounded-full border-2 border-border-subtle border-t-gold"></span>
          <span v-else-if="preparingImage" class="inline-block h-[14px] w-[14px] animate-spin rounded-full border-2 border-border-subtle border-t-gold"></span>
          <Search v-else :size="19" class="hidden sm:block" />
          {{ submitting ? 'Analyzing...' : preparingImage ? 'Preparing image...' : 'Analyze Photos' }}
        </button>

        <button
          v-if="obverse && deepAnalysisEnabled"
          type="button"
          class="btn btn-secondary min-w-0 flex-1 justify-center px-2 text-tiny sm:px-5 sm:text-base"
          :disabled="submitting || preparingImage"
          @click="startDeepAnalysis"
        >
          <Microscope :size="19" class="hidden sm:block" aria-hidden="true" />
          Deep Analysis
        </button>

        <button
          v-if="step < 2"
          type="button"
          class="btn btn-secondary ml-auto min-h-11 min-w-11 shrink-0 justify-center px-3"
          :disabled="!obverse || preparingImage"
          :title="step === 0 ? 'Add reverse image' : 'Add notes'"
          :aria-label="step === 0 ? 'Add reverse image' : 'Add notes'"
          @click="step += 1"
        >
          <ChevronRight :size="20" aria-hidden="true" />
        </button>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { AlertCircle, ChevronLeft, ChevronRight, Microscope, Search, X } from 'lucide-vue-next'
import InlineCameraCapturePanel from '@/components/InlineCameraCapturePanel.vue'
import type { CoinLookupImageRole } from '@/types'

interface CaptureImage {
  file: File
  preview: string
}

const props = withDefaults(defineProps<{
  obverse: CaptureImage | null
  reverse: CaptureImage | null
  notesImage: CaptureImage | null
  notes: string
  submitting: boolean
  preparingImage: boolean
  uploadError: string
  deepAnalysisEnabled?: boolean
}>(), {
  deepAnalysisEnabled: false,
})

const emit = defineEmits<{
  captured: [role: CoinLookupImageRole, file: File]
  selected: [role: CoinLookupImageRole, file: File]
  remove: [role: CoinLookupImageRole]
  analyze: []
  deepAnalyze: []
  'update:notes': [value: string]
}>()

const steps = [
  {
    role: 'obverse' as const,
    label: 'Obverse',
    required: true,
    title: 'Add the obverse',
    description: 'Photograph or upload the front of the coin. This is the only required image.',
    instruction: 'Center the obverse in the circle',
  },
  {
    role: 'reverse' as const,
    label: 'Reverse',
    required: false,
    title: 'Add the reverse',
    description: 'Optional for quick analysis and required for Deep Analysis. Legends and designs significantly improve attribution.',
    instruction: 'Center the reverse in the circle',
  },
  {
    role: 'notes' as const,
    label: 'Notes',
    required: false,
    title: 'Add supporting evidence',
    description: 'Provide any additional evidence that may help identify the coin.',
    instruction: 'Capture a label, edge, measurement, or other detail',
  },
] as const

const step = ref(0)
const deepRequirementError = ref('')
const fileInput = ref<HTMLInputElement | null>(null)
const cameraPanel = ref<InstanceType<typeof InlineCameraCapturePanel> | null>(null)
const currentStep = computed(() => {
  if (step.value === 1) return steps[1]
  if (step.value === 2) return steps[2]
  return steps[0]
})
const currentImage = computed(() => {
  if (currentStep.value.role === 'obverse') return props.obverse
  if (currentStep.value.role === 'reverse') return props.reverse
  return props.notesImage
})

function handleFileSelection(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (file) {
    emit('selected', currentStep.value.role, file)
  }
  input.value = ''
}

function startDeepAnalysis() {
  deepRequirementError.value = ''
  if (!props.reverse) {
    step.value = 1
    deepRequirementError.value = 'Add a reverse image before starting Deep Analysis.'
    return
  }
  emit('deepAnalyze')
}

watch(() => props.reverse, (reverse) => {
  if (reverse) deepRequirementError.value = ''
})

function stopCamera() {
  cameraPanel.value?.stopCamera()
}

defineExpose({ stopCamera })
</script>
