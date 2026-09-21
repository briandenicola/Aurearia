<template>
  <div class="capture-shell">
    <div class="shell-frame">
      <p class="sr-only">Step {{ step + 1 }} of 3</p>
      <h2 class="sr-only">{{ currentStep.title }}</h2>

      <header class="shell-top">
        <button type="button" class="icon-round" aria-label="Close capture" @click="closeShell">
          <X :size="20" aria-hidden="true" />
        </button>

        <div class="segmented" role="tablist" aria-label="Capture mode">
          <button
            type="button"
            role="tab"
            class="segment"
            :class="{ 'is-active': purpose === 'intake' }"
            :aria-selected="purpose === 'intake'"
            @click="switchPurpose('intake')"
          >
            Add
          </button>
          <button
            type="button"
            role="tab"
            class="segment"
            :class="{ 'is-active': purpose === 'identify' }"
            :aria-selected="purpose === 'identify'"
            @click="switchPurpose('identify')"
          >
            Identify
          </button>
        </div>

        <button type="button" class="icon-round" aria-label="Quick Capture" title="Quick Capture" @click="openQuickCapture">
          <Zap :size="20" aria-hidden="true" />
        </button>
      </header>

      <ol class="capture-rail" :aria-label="purpose === 'intake' ? 'Coin intake progress' : 'Identification progress'">
        <li v-for="(wizardStep, index) in steps" :key="wizardStep.role">
          <button
            type="button"
            class="rail-step"
            :class="{ 'is-active': index === step, 'is-complete': stepImage(wizardStep.role) !== null }"
            :aria-label="wizardStep.navLabel"
            :aria-current="index === step ? 'step' : undefined"
            :disabled="index > 0 && !obverse"
            @click="step = index"
          >
            <span class="rail-bar"></span>
            <span class="rail-label">{{ index + 1 }} {{ wizardStep.label }}</span>
          </button>
        </li>
      </ol>

      <section class="capture-stage">
        <label v-if="showNotesField" class="stage-notes">
          <span class="sr-only">Identification notes</span>
          <textarea
            class="form-input"
            :value="notes"
            maxlength="2000"
            placeholder="Weight, diameter, provenance, visible text, suspected ruler&hellip;"
            @input="$emit('update:notes', ($event.target as HTMLTextAreaElement).value)"
          ></textarea>
          <span class="stage-notes-count">{{ notes.length }} / 2000</span>
        </label>

        <div class="stage-view">
          <img
            v-if="currentImage"
            :src="currentImage.preview"
            :alt="`${currentStep.label} coin image`"
            class="stage-photo"
          />
          <InlineCameraCapturePanel
            v-else
            ref="cameraPanel"
            immersive
            :filename-prefix="`${purpose}-${currentStep.role}`"
            :image-role="currentStep.role"
            :instruction="currentStep.hint"
            @captured="(file, role) => $emit('captured', role, file)"
            @upload="pickFromLibrary"
          />

          <svg v-if="!currentImage && showGuideRing" class="guide-ring" viewBox="0 0 272 272" aria-hidden="true" focusable="false">
            <circle cx="136" cy="136" r="135" stroke-opacity="0.65" />
            <circle cx="136" cy="136" r="118.5" class="guide-ring-inner" stroke-opacity="0.3" />
          </svg>
        </div>

        <span class="stage-chip">
          <i class="stage-chip-dot" :class="{ 'is-met': stepImage(currentStep.role) }"></i>
          {{ currentStep.chip }}
        </span>

        <button
          v-if="currentImage"
          type="button"
          class="stage-remove"
          :aria-label="`Remove ${currentStep.label.toLowerCase()} image`"
          @click="$emit('remove', currentStep.role)"
        >
          <X :size="18" aria-hidden="true" />
        </button>

        <div class="stage-foot">
          <p v-if="stageError" class="stage-error" role="alert">
            <AlertCircle :size="16" class="stage-error-icon" aria-hidden="true" />
            <span>{{ stageError }}</span>
          </p>
          <p v-else class="stage-hint">{{ hintText }}</p>

          <button
            v-if="obverse"
            type="button"
            class="stage-action"
            :disabled="submitting || preparingImage"
            @click="$emit('analyze')"
          >
            <span v-if="busy" class="stage-action-spinner" aria-hidden="true"></span>
            {{ actionLabel }}
          </button>
        </div>
      </section>

      <footer class="shell-bottom">
        <button
          type="button"
          class="tool-btn"
          aria-label="Choose from library"
          :disabled="submitting || preparingImage"
          @click="pickFromLibrary"
        >
          <Images :size="20" aria-hidden="true" />
          <span>Library</span>
        </button>

        <button
          type="button"
          class="shutter"
          :aria-label="shutterLabel"
          :disabled="submitting || preparingImage"
          @click="handleShutter"
        >
          <span class="shutter-core"></span>
        </button>

        <button
          v-if="purpose === 'intake'"
          type="button"
          class="tool-btn"
          aria-label="Use manual entry"
          @click="$emit('manual')"
        >
          <Pencil :size="20" aria-hidden="true" />
          <span>Manual</span>
        </button>
        <button
          v-else-if="deepAnalysisEnabled"
          type="button"
          class="tool-btn"
          aria-label="Deep Analysis"
          :title="deepAnalysisDisabled ? deepAnalysisDisabledTitle : 'Deep Analysis'"
          :disabled="submitting || preparingImage || deepAnalysisDisabled"
          @click="startDeepAnalysis"
        >
          <Microscope :size="20" aria-hidden="true" />
          <span>Deep</span>
        </button>
        <span v-else class="tool-btn-spacer" aria-hidden="true"></span>
      </footer>

      <input
        ref="fileInput"
        type="file"
        accept="image/*"
        :disabled="submitting || preparingImage"
        class="hidden"
        @change="handleFileSelection"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { AlertCircle, Images, Microscope, Pencil, X, Zap } from 'lucide-vue-next'
import InlineCameraCapturePanel from '@/components/InlineCameraCapturePanel.vue'
import { useDialog } from '@/composables/useDialog'
import { useImmersiveShellClaim } from '@/composables/useImmersiveShell'
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
  manual: []
  'update:notes': [value: string]
}>()

const router = useRouter()
const { showConfirm } = useDialog()
useImmersiveShellClaim()

const step = ref(0)
const deepRequirementError = ref('')
const fileInput = ref<HTMLInputElement | null>(null)
const cameraPanel = ref<InstanceType<typeof InlineCameraCapturePanel> | null>(null)

const steps = computed(() => [
  {
    role: 'obverse' as const,
    label: 'Obverse',
    navLabel: 'Add obverse image',
    title: 'Add the obverse',
    chip: 'Obverse · required',
    hint: 'Fill the ring · soft, even light',
  },
  {
    role: 'reverse' as const,
    label: 'Reverse',
    navLabel: 'Add reverse image',
    title: 'Add the reverse',
    chip: 'Reverse · optional',
    hint: 'Fill the ring · soft, even light',
  },
  {
    role: 'notes' as const,
    label: 'Details',
    navLabel: props.purpose === 'intake' ? 'Add coin card' : 'Add notes',
    title: props.purpose === 'intake' ? 'Add a coin card' : 'Add supporting evidence',
    chip: props.purpose === 'intake' ? 'Card · optional' : 'Details · optional',
    hint: props.purpose === 'intake'
      ? 'Frame the coin card or label'
      : 'Capture a label, edge, or measurement',
  },
] as const)

const currentStep = computed(() => steps.value[step.value] ?? steps.value[0])
const currentImage = computed(() => stepImage(currentStep.value.role))
const showNotesField = computed(() => props.purpose === 'identify' && currentStep.value.role === 'notes')
const showGuideRing = computed(() => currentStep.value.role !== 'notes')
const busy = computed(() => props.submitting || props.preparingImage)
const stageError = computed(() => deepRequirementError.value || props.uploadError)

const actionLabel = computed(() => {
  if (props.submitting) return 'Analyzing...'
  if (props.preparingImage) return 'Preparing image...'
  return props.purpose === 'intake' ? 'Generate Intake Draft' : 'Analyze Photos'
})

const shutterLabel = computed(() => {
  if (currentImage.value) return 'Retake photo'
  return cameraPanel.value?.cameraActive ? 'Capture photo' : 'Start camera'
})

const hintText = computed(() => (
  currentImage.value
    ? `${currentStep.value.label} added · shutter retakes it`
    : currentStep.value.hint
))

function stepImage(role: CoinLookupImageRole) {
  if (role === 'obverse') return props.obverse
  if (role === 'reverse') return props.reverse
  return props.notesImage
}

/** True once anything has been captured or typed that leaving would discard. */
function hasWork() {
  return Boolean(props.obverse || props.reverse || props.notesImage || props.notes.trim())
}

async function confirmDiscard() {
  if (!hasWork()) return true
  return await showConfirm('Leaving now discards the photos you have taken.', {
    title: 'Discard capture?',
    confirmLabel: 'Discard',
    variant: 'danger',
  })
}

async function closeShell() {
  if (!await confirmDiscard()) return
  stopCamera()
  router.push('/')
}

async function switchPurpose(next: 'identify' | 'intake') {
  if (next === props.purpose) return
  if (!await confirmDiscard()) return
  stopCamera()
  router.push(next === 'intake' ? '/add' : '/lookup')
}

async function openQuickCapture() {
  if (!await confirmDiscard()) return
  stopCamera()
  router.push('/quick-capture')
}

function pickFromLibrary() {
  fileInput.value?.click()
}

async function handleShutter() {
  // With a photo on the stage the camera is unmounted, so the shutter's job is
  // to drop that photo and come straight back live rather than do nothing.
  if (currentImage.value) {
    emit('remove', currentStep.value.role)
    await nextTick()
    await cameraPanel.value?.startCamera()
    return
  }

  const panel = cameraPanel.value
  if (!panel) return
  if (panel.cameraActive) {
    await panel.captureFromCamera()
    return
  }
  await panel.startCamera()
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
/* Geometry below is taken from the approved 390x844 capture mockup; the
   comments give the mockup's CSS-pixel positions so future edits can be
   checked against it. */
.capture-shell {
  position: fixed;
  inset: 0;
  /* Above the app nav (z 100), below dialogs (z 2000) and toasts (z 1200). */
  z-index: 500;
  display: flex;
  flex-direction: column;
  background: var(--bg-primary);
  color: var(--text-primary);
  /* Mockup puts the header 62px down on a device with a 47px top inset. */
  padding-top: calc(max(env(safe-area-inset-top), 12px) + 15px);
  /* The shutter overhangs the 58px tool row, so the home-indicator inset is
     the whole bottom gap - adding to it would push the row off a short phone. */
  padding-bottom: max(env(safe-area-inset-bottom), 22px);
  padding-left: env(safe-area-inset-left);
  padding-right: env(safe-area-inset-right);
}

/* An installed desktop PWA reports the same standalone display mode as a phone,
   so cap the viewfinder at phone width instead of stretching it across a
   1440px window. */
.shell-frame {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-height: 0;
  width: 100%;
  max-width: 420px;
  margin-inline: auto;
}

/* ── Header: close, Add/Identify segments, Quick Capture ─────────────────── */
.shell-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  height: 46px;
  padding: 0 12px;
}

.icon-round {
  flex: none;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 44px;
  height: 44px;
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-full);
  background: var(--bg-input);
  color: var(--text-primary);
  cursor: pointer;
  transition: background var(--transition-fast);
}

.icon-round:active {
  background: var(--border-subtle);
}

.segmented {
  display: flex;
  flex: none;
  width: 200px;
  height: 46px;
  padding: 4px;
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-full);
  background: var(--bg-input);
}

.segment {
  flex: 1;
  border: 0;
  border-radius: var(--radius-full);
  background: transparent;
  color: var(--text-secondary);
  font-size: 15px;
  font-weight: 500;
  cursor: pointer;
  transition: background var(--transition-fast), color var(--transition-fast);
}

.segment.is-active {
  background: var(--accent-gold);
  color: var(--bg-primary);
  font-weight: 600;
}

/* ── Progress rail: 3px bar over a 12px label ────────────────────────────── */
.capture-rail {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 8px;
  margin: 9px 20px 0;
  padding: 0;
  list-style: none;
}

.rail-step {
  display: grid;
  gap: 4px;
  width: 100%;
  padding: 0;
  border: 0;
  background: transparent;
  text-align: left;
  cursor: pointer;
}

.rail-step:disabled {
  cursor: not-allowed;
}

.rail-bar {
  height: 3px;
  border-radius: var(--radius-full);
  background: var(--bg-input);
  transition: background var(--transition-fast);
}

.rail-step.is-active .rail-bar,
.rail-step.is-complete .rail-bar {
  background: var(--accent-gold);
}

.rail-label {
  font-size: 12px;
  line-height: 17px;
  color: var(--text-muted);
  white-space: nowrap;
}

.rail-step.is-active .rail-label {
  color: var(--accent-gold);
  font-weight: 500;
}


/* ── Stage: the viewfinder card ──────────────────────────────────────────── */
.capture-stage {
  position: relative;
  display: flex;
  flex-direction: column;
  flex: 1;
  min-height: 0;
  margin: 11px 12px 0;
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-xl);
  background: var(--bg-card);
  overflow: hidden;
}

.stage-view {
  position: relative;
  flex: 1;
  min-height: 0;
}

/* The immersive camera panel is the stage's own root child and must fill it. */
.stage-view > :deep(.camera-first-card) {
  width: 100%;
  height: 100%;
}

.stage-photo {
  width: 100%;
  height: 100%;
  object-fit: contain;
}

/* Mockup ring: 272px across on a 390px viewport, centred in the stage. */
.guide-ring {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  width: min(272px, 75%, 40vh);
  aspect-ratio: 1;
  pointer-events: none;
}

.guide-ring circle {
  fill: none;
  stroke: var(--accent-gold);
  stroke-width: 2;
  vector-effect: non-scaling-stroke;
}

.guide-ring .guide-ring-inner {
  stroke-width: 1.5;
  stroke-dasharray: 3 3;
}

.stage-chip {
  position: absolute;
  top: 16px;
  left: 16px;
  display: inline-flex;
  align-items: center;
  gap: 8px;
  height: 32px;
  padding: 0 14px;
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-full);
  background: var(--bg-input);
  color: var(--text-primary);
  font-size: 12px;
  font-weight: 500;
}

.stage-chip-dot {
  width: 6px;
  height: 6px;
  border-radius: var(--radius-full);
  background: var(--accent-gold);
}

.stage-chip-dot.is-met {
  background: var(--confidence-high);
}

.stage-remove {
  position: absolute;
  top: 16px;
  right: 16px;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 44px;
  height: 44px;
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-full);
  background: var(--bg-input);
  color: var(--text-primary);
  cursor: pointer;
}

.stage-foot {
  position: absolute;
  inset: auto 16px 20px;
  display: grid;
  gap: 12px;
  justify-items: center;
  pointer-events: none;
}

.stage-hint,
.stage-error {
  margin: 0;
  font-size: 13px;
  text-align: center;
}

.stage-hint {
  color: var(--text-secondary);
}

.stage-error {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 14px;
  border: 1px solid var(--border-accent);
  border-radius: var(--radius-md);
  background: var(--bg-input);
  color: var(--text-primary);
  pointer-events: auto;
}

.stage-error-icon {
  flex-shrink: 0;
  color: var(--cat-byzantine);
}

.stage-action {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  min-height: 44px;
  padding: 0 24px;
  border: 0;
  border-radius: var(--radius-full);
  background: linear-gradient(135deg, var(--accent-gold), var(--accent-bronze));
  color: var(--bg-primary);
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  pointer-events: auto;
}

.stage-action:disabled {
  opacity: 0.7;
  cursor: progress;
}

.stage-action-spinner {
  width: 14px;
  height: 14px;
  border: 2px solid var(--bg-primary);
  border-top-color: transparent;
  border-radius: var(--radius-full);
  animation: spin 0.8s linear infinite;
}

/* Identify-only notes panel on step 3, above the viewfinder. */
.stage-notes {
  flex: none;
  display: grid;
  gap: 4px;
  /* Top padding clears the stage chip floating at top:16px, height 32px. */
  padding: 60px 16px 0;
}

.stage-notes textarea {
  height: 116px;
  font-size: 13px;
  resize: none;
}

.stage-notes-count {
  justify-self: end;
  font-size: 10px;
  color: var(--text-secondary);
}

/* ── Bottom bar: Library, shutter, Manual/Deep ───────────────────────────── */
.shell-bottom {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 17px 30px 0;
}

.tool-btn,
.tool-btn-spacer {
  width: 58px;
  height: 58px;
  flex: none;
}

.tool-btn {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 4px;
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-lg);
  background: var(--bg-input);
  color: var(--text-primary);
  font-size: 10px;
  line-height: 1.2;
  cursor: pointer;
}

.tool-btn:disabled {
  opacity: 0.45;
  cursor: not-allowed;
}

.shutter {
  display: grid;
  place-items: center;
  width: 78px;
  height: 78px;
  padding: 5px;
  border: 3px solid var(--text-primary);
  border-radius: var(--radius-full);
  background: transparent;
  cursor: pointer;
}

.shutter:disabled {
  opacity: 0.45;
  cursor: not-allowed;
}

.shutter-core {
  width: 100%;
  height: 100%;
  border-radius: var(--radius-full);
  background: linear-gradient(135deg, var(--accent-gold), var(--accent-bronze));
  transition: transform var(--transition-fast);
}

.shutter:active:not(:disabled) .shutter-core {
  transform: scale(0.92);
}

/* Landscape phones cannot fit the full-height ring; keep the stage usable. */
@media (max-height: 560px) {
  .capture-stage {
    margin-top: 8px;
  }

  .shell-bottom {
    padding-top: 10px;
  }
}
</style>
