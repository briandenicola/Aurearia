<template>
  <section class="capture-wizard" aria-labelledby="capture-wizard-title">
    <h2 id="capture-wizard-title" class="sr-only">Coin photo steps</h2>

    <ol class="capture-progress" :aria-label="purpose === 'intake' ? 'Coin intake progress' : 'Identification progress'">
      <li
        v-for="(wizardStep, index) in steps"
        :key="wizardStep.role"
        :class="{ active: index === step, complete: stepImage(wizardStep.role) !== null }"
      >
        <span class="step-marker">{{ index + 1 }}</span>
        <span>{{ wizardStep.label }}</span>
      </li>
    </ol>

    <div class="capture-workspace">
      <aside class="capture-guidance">
        <div>
          <span class="section-label">Step {{ step + 1 }} of 3</span>
          <h2 class="mt-1 text-heading">{{ currentStep.title }}</h2>
          <p class="mt-1 text-small leading-5 text-text-secondary">{{ currentStep.description }}</p>
        </div>

        <div class="capture-evidence" aria-label="Captured evidence">
          <button
            v-for="(wizardStep, index) in steps.slice(0, 2)"
            :key="wizardStep.role"
            type="button"
            class="evidence-item"
            :class="{ active: index === step }"
            :disabled="index > 0 && !obverse"
            @click="step = index"
          >
            <img
              v-if="stepImage(wizardStep.role)"
              :src="stepImage(wizardStep.role)?.preview"
              :alt="`${wizardStep.label} thumbnail`"
            />
            <span v-else class="evidence-placeholder"><Camera :size="18" /></span>
            <span>
              <strong>{{ wizardStep.label }}</strong>
              <small>{{ stepImage(wizardStep.role) ? 'Image added' : wizardStep.required ? 'Required' : 'Optional' }}</small>
            </span>
            <Check v-if="stepImage(wizardStep.role)" :size="16" class="evidence-check" />
          </button>
        </div>

        <label v-if="currentStep.role === 'notes' && purpose === 'identify'" class="form-group">
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
      </aside>

      <div class="capture-stage">
        <div
          v-if="currentImage"
          class="capture-preview relative w-full overflow-hidden rounded-sm border border-border-accent bg-card"
        >
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

        <InlineCameraCapturePanel
          v-else
          ref="cameraPanel"
          class="w-full"
          desktop-workspace
          :filename-prefix="`lookup-${currentStep.role}`"
          :instruction="currentStep.instruction"
          @captured="$emit('captured', currentStep.role, $event)"
          @upload="fileInput?.click()"
        />

        <input
          ref="fileInput"
          type="file"
          accept="image/*"
          :disabled="submitting || preparingImage"
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

        <div class="workflow-actions sticky bottom-0 z-10 flex items-center gap-2">
          <button
            v-if="step > 0"
            type="button"
            class="btn btn-secondary min-h-11 shrink-0 justify-center"
            title="Previous step"
            aria-label="Previous step"
            @click="step -= 1"
          >
            <ChevronLeft :size="20" aria-hidden="true" />
            <span class="hidden sm:inline">Previous</span>
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
            {{ submitting ? 'Analyzing...' : preparingImage ? 'Preparing image...' : purpose === 'intake' ? 'Generate Intake Draft' : 'Analyze Photos' }}
          </button>

          <button
            v-if="obverse && deepAnalysisEnabled"
            type="button"
            class="btn btn-secondary min-w-0 flex-1 justify-center px-2 text-tiny sm:px-5 sm:text-base"
            :disabled="submitting || preparingImage || deepAnalysisDisabled"
            :title="deepAnalysisDisabled ? deepAnalysisDisabledTitle : undefined"
            @click="startDeepAnalysis"
          >
            <Microscope :size="19" class="hidden sm:block" aria-hidden="true" />
            Deep Analysis
          </button>

          <button
            v-if="step < 2"
            type="button"
            class="btn btn-secondary ml-auto min-h-11 shrink-0 justify-center"
            :disabled="!obverse || preparingImage"
            :title="step === 0 ? 'Add reverse image' : purpose === 'intake' ? 'Add coin card' : 'Add notes'"
            :aria-label="step === 0 ? 'Add reverse image' : purpose === 'intake' ? 'Add coin card' : 'Add notes'"
            @click="step += 1"
          >
            <span class="hidden sm:inline">{{ step === 0 ? 'Next: Reverse' : purpose === 'intake' ? 'Next: Card' : 'Next: Notes' }}</span>
            <ChevronRight :size="20" aria-hidden="true" />
          </button>
        </div>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { AlertCircle, Camera, Check, ChevronLeft, ChevronRight, Microscope, Search, X } from 'lucide-vue-next'
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
  purpose?: 'identify' | 'intake'
  /**
   * True when the user is already at the Deep Analysis `MaxActivePerUser`
   * limit (T088/F6) - disables the control instead of letting the click
   * round-trip to the backend and surface an error toast.
   */
  deepAnalysisDisabled?: boolean
  deepAnalysisDisabledTitle?: string
}>(), {
  deepAnalysisEnabled: false,
  purpose: 'identify',
  deepAnalysisDisabled: false,
  deepAnalysisDisabledTitle: undefined,
})

const emit = defineEmits<{
  captured: [role: CoinLookupImageRole, file: File]
  selected: [role: CoinLookupImageRole, file: File]
  remove: [role: CoinLookupImageRole]
  analyze: []
  deepAnalyze: []
  'update:notes': [value: string]
}>()

const steps = computed(() => [
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
    description: props.purpose === 'intake'
      ? 'Optional. Legends and designs on the back help prepare a more complete draft.'
      : 'Optional for quick analysis and required for Deep Analysis. Legends and designs significantly improve attribution.',
    instruction: 'Center the reverse in the circle',
  },
  {
    role: 'notes' as const,
    label: props.purpose === 'intake' ? 'Card' : 'Notes',
    required: false,
    title: props.purpose === 'intake' ? 'Add a coin card' : 'Add supporting evidence',
    description: props.purpose === 'intake'
      ? 'Optional. Photograph or upload a coin card or label. For a PDF card, use manual mode.'
      : 'Provide any additional evidence that may help identify the coin.',
    instruction: 'Capture a label, edge, measurement, or other detail',
  },
] as const)

const step = ref(0)
const deepRequirementError = ref('')
const fileInput = ref<HTMLInputElement | null>(null)
const cameraPanel = ref<InstanceType<typeof InlineCameraCapturePanel> | null>(null)
const currentStep = computed(() => {
  if (step.value === 1) return steps.value[1]
  if (step.value === 2) return steps.value[2]
  return steps.value[0]
})
const currentImage = computed(() => {
  if (currentStep.value.role === 'obverse') return props.obverse
  if (currentStep.value.role === 'reverse') return props.reverse
  return props.notesImage
})

function stepImage(role: CoinLookupImageRole) {
  if (role === 'obverse') return props.obverse
  if (role === 'reverse') return props.reverse
  return props.notesImage
}

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

<style scoped>
.capture-wizard {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.capture-progress {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  margin: 0;
  padding: 0;
  list-style: none;
}

.capture-progress li {
  position: relative;
  display: flex;
  align-items: center;
  gap: 0.5rem;
  color: var(--text-muted);
  font-size: 0.8rem;
}

.capture-progress li:not(:last-child)::after {
  content: '';
  position: absolute;
  left: 2rem;
  right: 0.5rem;
  top: 50%;
  height: 1px;
  background: var(--border-subtle);
}

.step-marker {
  position: relative;
  z-index: 1;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 1.75rem;
  height: 1.75rem;
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-full);
  background: var(--bg-primary);
}

.capture-progress li.active,
.capture-progress li.complete {
  color: var(--accent-gold);
}

.capture-progress li.active .step-marker,
.capture-progress li.complete .step-marker {
  border-color: var(--accent-gold);
  background: var(--accent-gold-dim);
}

.capture-workspace {
  display: grid;
  gap: 1rem;
}

.capture-guidance,
.capture-stage {
  min-width: 0;
}

.capture-guidance {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.capture-stage {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.capture-evidence {
  display: grid;
  gap: 0.5rem;
}

.evidence-item {
  display: grid;
  grid-template-columns: 2.75rem 1fr auto;
  align-items: center;
  gap: 0.75rem;
  width: 100%;
  padding: 0.5rem;
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
  background: var(--bg-input);
  color: var(--text-secondary);
  text-align: left;
  transition: all var(--transition-fast);
}

.evidence-item.active {
  border-color: var(--accent-gold);
  background: var(--accent-gold-glow);
}

.evidence-item:disabled {
  opacity: 0.55;
  cursor: not-allowed;
}

.evidence-item img,
.evidence-placeholder {
  width: 2.75rem;
  height: 2.75rem;
  border-radius: var(--radius-sm);
}

.evidence-item img {
  object-fit: cover;
}

.evidence-placeholder {
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--bg-card);
  color: var(--text-muted);
}

.evidence-item strong,
.evidence-item small {
  display: block;
}

.evidence-item strong {
  color: var(--text-primary);
  font-size: 0.85rem;
}

.evidence-item small {
  margin-top: 0.15rem;
  color: var(--text-muted);
  font-size: 0.75rem;
}

.evidence-check {
  color: var(--accent-gold);
}

.capture-preview {
  height: clamp(240px, 48vh, 420px);
}

.workflow-actions {
  margin: 0 -0.25rem -0.25rem;
  padding: 0.75rem 0.25rem 0.25rem;
  background: linear-gradient(to bottom, transparent, var(--bg-primary) 30%);
}

@media (max-height: 700px) {
  .capture-preview {
    height: 40vh;
  }
}

@media (min-width: 769px) {
  .capture-wizard {
    gap: 1.5rem;
  }

  .capture-progress {
    max-width: 36rem;
  }

  .capture-workspace {
    grid-template-columns: minmax(13rem, 15rem) minmax(0, 1fr);
    gap: 1.5rem;
    padding: 1rem;
    border: 1px solid var(--border-subtle);
    border-radius: var(--radius-md);
    background: var(--bg-card);
    box-shadow: var(--shadow-card);
  }

  .capture-guidance {
    padding: 0.5rem;
    border-right: 1px solid var(--border-subtle);
  }

  .capture-preview {
    height: min(48vh, 390px);
  }

  .workflow-actions {
    position: static;
    margin: 0;
    padding: 0.75rem 0 0;
    border-top: 1px solid var(--border-subtle);
    background: transparent;
  }
}
</style>
