<template>
  <div class="image-processor">
    <!-- Step 1: Image Input -->
    <ImageInputPanel
      v-if="!sourceImage"
      :source-image="sourceImage"
      :url-loading="urlLoading"
      :input-error="inputError"
      @file-select="loadImageFromFile"
      @url-load="handleUrlLoad"
      @drop="loadImageFromFile"
    />

    <!-- Step 2: Processing & Preview -->
    <div v-if="sourceImage" class="processing-section">
      <div class="step-header">
        <button class="btn btn-secondary btn-sm" @click="reset">← Start Over</button>
        <div class="step-indicators">
          <span class="step" :class="{ active: step === 'preview', done: step !== 'preview' }">1. Original</span>
          <span class="step" :class="{ active: step === 'removing', done: step === 'crop' || step === 'done' }">2.
            Remove BG</span>
          <span class="step" :class="{ active: step === 'crop', done: step === 'done' }">3. Crop</span>
        </div>
      </div>

      <!-- Original preview + remove BG button -->
      <div v-if="step === 'preview'" class="preview-panel">
        <div class="image-preview">
          <img :src="sourceImage" alt="Original" />
        </div>
        <button class="btn btn-primary" @click="removeBackground">Remove Background</button>
      </div>

      <!-- Processing spinner -->
      <div v-if="step === 'removing'" class="preview-panel">
        <div class="image-preview processing">
          <img :src="sourceImage" alt="Processing" class="processing-img" />
          <div class="processing-overlay">
            <div class="spinner"></div>
            <p>Removing background...</p>
            <p class="processing-hint">First run downloads the ML model (~40MB)</p>
          </div>
        </div>
      </div>

      <!-- Crop step -->
      <div v-if="step === 'crop' || step === 'done'" class="crop-panel">
        <div class="crop-workspace">
          <canvas
            ref="cropCanvas" class="crop-canvas" @pointerdown="startCropDrag" @pointermove="onCropDrag"
            @pointerup="endCropDrag"
          />
        </div>
        <div class="crop-controls">
          <button class="btn btn-secondary btn-sm" @click="autoCrop">Auto Crop</button>
          <button class="btn btn-secondary btn-sm" @click="resetCrop">Reset Crop</button>
          <label class="padding-control">
            <span>Padding</span>
            <input v-model.number="cropPadding" type="range" min="0" max="50" />
            <span class="padding-value">{{ cropPadding }}px</span>
          </label>
        </div>

        <!-- Result preview -->
        <div class="result-row">
          <div class="result-preview">
            <h4>Result</h4>
            <canvas ref="resultCanvas" class="result-canvas" />
          </div>
          <div class="save-controls">
            <div class="save-tabs">
              <button class="save-tab" :class="{ active: saveTab === 'existing' }" @click="saveTab = 'existing'">Existing Coin</button>
              <button class="save-tab" :class="{ active: saveTab === 'new' }" @click="saveTab = 'new'">New Coin</button>
              <button class="save-tab" :class="{ active: saveTab === 'download' }" @click="saveTab = 'download'">Download</button>
            </div>

            <!-- Assign to existing coin -->
            <div v-if="saveTab === 'existing'" class="save-panel">
              <div class="coin-search">
                <input v-model="coinSearch" type="text" class="form-input" placeholder="Search coins..." @input="searchCoins" />
              </div>
              <div v-if="coinOptions.length" class="coin-list">
                <button v-for="c in coinOptions" :key="c.id" class="coin-option" :class="{ selected: selectedCoinId === c.id }" @click="selectedCoinId = c.id">
                  <span class="coin-option-name">{{ c.name }}</span>
                  <span class="coin-option-meta">{{ [c.ruler, c.era].filter(Boolean).join(' · ') }}</span>
                </button>
              </div>
              <p v-else-if="coinSearch && !coinsLoading" class="hint">No coins found</p>
              <p v-else-if="coinsLoading" class="hint">Searching...</p>
              <p v-else class="hint">Type to search your collection</p>
              <div v-if="selectedCoinId" class="type-row">
                <label class="radio-label">
                  <input v-model="saveImageType" type="radio" value="obverse" name="imgType" />
                  <span>Obverse</span>
                </label>
                <label class="radio-label">
                  <input v-model="saveImageType" type="radio" value="reverse" name="imgType" />
                  <span>Reverse</span>
                </label>
              </div>
              <button class="btn btn-primary" :disabled="!selectedCoinId || saving" @click="saveToExisting">
                {{ saving ? 'Saving...' : 'Upload to Coin' }}
              </button>
            </div>

            <!-- Create new coin -->
            <div v-if="saveTab === 'new'" class="save-panel">
              <label class="field-label">Coin Name</label>
              <input v-model="newCoinName" type="text" class="form-input" placeholder="e.g. Augustus Denarius" />
              <button class="btn btn-primary" :disabled="!newCoinName.trim() || saving" @click="saveToNewCoin">
                {{ saving ? 'Creating...' : 'Create Coin & Upload' }}
              </button>
              <p class="hint">Image will be saved as the obverse</p>
            </div>

            <!-- Download -->
            <div v-if="saveTab === 'download'" class="save-panel">
              <button class="btn btn-secondary" @click="downloadResult">💾 Download PNG</button>
            </div>

            <p v-if="saveMsg" class="msg" :class="{ error: saveError }">{{ saveMsg }}</p>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useImageProcessor } from '@/composables/useImageProcessor'
import ImageInputPanel from '@/components/ImageInputPanel.vue'

const props = defineProps<{
  coinId?: number
}>()

const emit = defineEmits<{
  saved: [coinId: number]
}>()

const cropCanvas = ref<HTMLCanvasElement | null>(null)
const resultCanvas = ref<HTMLCanvasElement | null>(null)

const {
  sourceImage, urlLoading, inputError,
  step,
  cropPadding,
  saveTab, saveImageType, saving, saveMsg, saveError,
  coinSearch, coinOptions, coinsLoading, selectedCoinId,
  newCoinName,
  loadImageFromFile, handleUrlLoad,
  removeBackground,
  autoCrop, resetCrop, startCropDrag, onCropDrag, endCropDrag,
  saveToExisting: doSaveToExisting,
  saveToNewCoin: doSaveToNewCoin,
  downloadResult, reset, searchCoins,
} = useImageProcessor(cropCanvas, resultCanvas, { coinId: props.coinId })

async function saveToExisting() {
  const coinId = await doSaveToExisting()
  if (coinId != null) emit('saved', coinId)
}

async function saveToNewCoin() {
  const coinId = await doSaveToNewCoin()
  if (coinId != null) emit('saved', coinId)
}
</script>

<style scoped>
.image-processor {
  max-width: 700px;
  margin: 0 auto;
}

/* Steps */
.step-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1.25rem;
}

.step-indicators {
  display: flex;
  gap: 0.5rem;
}

.step {
  font-size: 0.75rem;
  padding: 0.3rem 0.6rem;
  border-radius: var(--radius-full);
  color: var(--text-muted);
  background: var(--bg-card);
  border: 1px solid var(--border-subtle);
}

.step.active {
  color: var(--accent-gold);
  border-color: var(--accent-gold);
  background: var(--accent-gold-glow);
}

.step.done {
  color: var(--text-secondary);
  border-color: var(--accent-gold-dim);
}

/* Preview */
.preview-panel {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 1rem;
}

.image-preview {
  position: relative;
  max-width: 500px;
  width: 100%;
  border-radius: var(--radius-md);
  overflow: hidden;
  border: 1px solid var(--border-subtle);
}

.image-preview img {
  width: 100%;
  display: block;
}

.image-preview.processing {
  position: relative;
}

.processing-img {
  opacity: 0.3;
  filter: blur(2px);
}

.processing-overlay {
  position: absolute;
  inset: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 0.75rem;
  color: var(--accent-gold);
}

.processing-overlay p {
  font-size: 0.9rem;
}

.processing-hint {
  font-size: 0.75rem !important;
  color: var(--text-muted) !important;
}

/* Crop */
.crop-panel {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.crop-workspace {
  display: flex;
  justify-content: center;
}

.crop-canvas {
  border-radius: var(--radius-md);
  border: 1px solid var(--border-subtle);
  cursor: crosshair;
  max-width: 100%;
  touch-action: none;
}

.crop-controls {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  flex-wrap: wrap;
}

.padding-control {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.8rem;
  color: var(--text-secondary);
  margin-left: auto;
}

.padding-control input[type="range"] {
  width: 100px;
  accent-color: var(--accent-gold);
}

.padding-value {
  min-width: 32px;
  text-align: right;
}

/* Result */
.result-row {
  display: flex;
  gap: 1.5rem;
  padding: 1rem;
  background: var(--bg-card);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
}

.result-preview {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.5rem;
}

.result-preview h4,
.save-controls h4 {
  font-size: 0.85rem;
  color: var(--text-secondary);
  margin-bottom: 0.25rem;
}

.result-canvas {
  border-radius: var(--radius-sm);
  border: 1px solid var(--border-subtle);
}

.save-controls {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  min-width: 0;
}

.save-tabs {
  display: flex;
  gap: 2px;
  background: var(--bg-primary);
  border-radius: var(--radius-sm);
  padding: 2px;
}

.save-tab {
  flex: 1;
  padding: 0.4rem 0.5rem;
  font-size: 0.75rem;
  background: transparent;
  border: none;
  border-radius: var(--radius-sm);
  color: var(--text-muted);
  cursor: pointer;
  transition: all var(--transition-fast);
  white-space: nowrap;
}

.save-tab:hover {
  color: var(--text-secondary);
}

.save-tab.active {
  background: var(--bg-card);
  color: var(--accent-gold);
}

.save-panel {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.coin-search .form-input {
  width: 100%;
  font-size: 0.85rem;
}

.coin-list {
  max-height: 140px;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 2px;
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
  padding: 2px;
}

.coin-option {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 0.1rem;
  padding: 0.4rem 0.6rem;
  background: transparent;
  border: none;
  border-radius: var(--radius-sm);
  cursor: pointer;
  text-align: left;
  transition: all var(--transition-fast);
}

.coin-option:hover {
  background: var(--accent-gold-glow);
}

.coin-option.selected {
  background: var(--accent-gold-glow);
  outline: 1px solid var(--accent-gold-dim);
}

.coin-option-name {
  font-size: 0.8rem;
  color: var(--text-primary);
}

.coin-option-meta {
  font-size: 0.7rem;
  color: var(--text-muted);
}

.type-row {
  display: flex;
  gap: 1rem;
}

.radio-label {
  display: flex;
  align-items: center;
  gap: 0.3rem;
  font-size: 0.85rem;
  color: var(--text-secondary);
  cursor: pointer;
}

.radio-label input[type="radio"] {
  accent-color: var(--accent-gold);
}

.field-label {
  font-size: 0.8rem;
  color: var(--text-muted);
}

.hint {
  font-size: 0.75rem;
  color: var(--text-muted);
}

.msg {
  font-size: 0.85rem;
  color: var(--accent-gold);
}

.msg.error {
  color: #e74c3c;
}

@media (max-width: 640px) {
  .result-row {
    flex-direction: column;
  }

  .step-header {
    flex-direction: column;
    gap: 0.75rem;
    align-items: stretch;
  }

  .step-indicators {
    justify-content: center;
  }
}
</style>
