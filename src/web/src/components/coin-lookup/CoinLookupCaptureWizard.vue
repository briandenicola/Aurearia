<template>
  <section class="flex flex-col gap-6" aria-labelledby="capture-wizard-title">
    <div class="card p-4">
      <h2 id="capture-wizard-title" class="sr-only">Coin photo steps</h2>
      <ol class="grid grid-cols-3 gap-2" aria-label="Coin identification progress">
        <li v-for="(item, index) in steps" :key="item.role">
          <button
            type="button"
            class="flex min-h-11 w-full items-center gap-2 rounded-sm border px-3 py-2 text-left transition-colors"
            :class="stepClass(index)"
            :disabled="index > 0 && !obverse"
            :aria-current="step === index ? 'step' : undefined"
            @click="step = index"
          >
            <span class="flex h-6 w-6 shrink-0 items-center justify-center rounded-full border border-current text-tiny font-semibold">
              <Check v-if="isComplete(item.role)" :size="14" aria-hidden="true" />
              <span v-else>{{ index + 1 }}</span>
            </span>
            <span class="min-w-0">
              <span class="block truncate text-small font-semibold">{{ item.label }}</span>
              <span class="block text-tiny text-text-muted">{{ item.required ? 'Required' : 'Optional' }}</span>
            </span>
          </button>
        </li>
      </ol>
    </div>

    <div class="flex flex-col gap-4">
      <div>
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

      <div class="flex flex-wrap items-center gap-3">
        <button
          v-if="step > 0"
          type="button"
          class="btn btn-secondary"
          @click="step -= 1"
        >
          <ChevronLeft :size="18" />
          Back
        </button>

        <button
          v-if="obverse"
          type="button"
          class="btn btn-primary flex-1 justify-center"
          :disabled="submitting || preparingImage"
          @click="$emit('analyze')"
        >
          <span v-if="submitting" class="inline-block h-[14px] w-[14px] animate-spin rounded-full border-2 border-border-subtle border-t-gold"></span>
          <span v-else-if="preparingImage" class="inline-block h-[14px] w-[14px] animate-spin rounded-full border-2 border-border-subtle border-t-gold"></span>
          <Search v-else :size="19" />
          {{ submitting ? 'Analyzing...' : preparingImage ? 'Preparing image...' : 'Analyze Photos' }}
        </button>

        <button
          v-if="step < 2"
          type="button"
          class="btn btn-secondary ml-auto"
          :disabled="!obverse || preparingImage"
          @click="step += 1"
        >
          {{ step === 0 ? 'Add Reverse' : 'Add Notes' }}
          <ChevronRight :size="18" />
        </button>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { AlertCircle, Check, ChevronLeft, ChevronRight, Search, X } from 'lucide-vue-next'
import InlineCameraCapturePanel from '@/components/InlineCameraCapturePanel.vue'
import type { CoinLookupImageRole } from '@/types'

interface CaptureImage {
  file: File
  preview: string
}

const props = defineProps<{
  obverse: CaptureImage | null
  reverse: CaptureImage | null
  notesImage: CaptureImage | null
  notes: string
  submitting: boolean
  preparingImage: boolean
  uploadError: string
}>()

const emit = defineEmits<{
  captured: [role: CoinLookupImageRole, file: File]
  selected: [role: CoinLookupImageRole, file: File]
  remove: [role: CoinLookupImageRole]
  analyze: []
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
    description: 'A reverse image is optional, but legends and designs can significantly improve attribution.',
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

function isComplete(role: CoinLookupImageRole) {
  if (role === 'obverse') return props.obverse !== null
  if (role === 'reverse') return props.reverse !== null
  return props.notesImage !== null || props.notes.trim().length > 0
}

function stepClass(index: number) {
  if (step.value === index) {
    return 'border-border-accent bg-gold-glow text-gold'
  }
  if (isComplete(steps[index]?.role ?? 'obverse')) {
    return 'border-border-subtle bg-card text-text-primary'
  }
  return 'border-border-subtle bg-input text-text-secondary'
}

function handleFileSelection(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (file) {
    emit('selected', currentStep.value.role, file)
  }
  input.value = ''
}

function stopCamera() {
  cameraPanel.value?.stopCamera()
}

defineExpose({ stopCamera })
</script>
