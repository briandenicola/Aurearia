<template>
  <div class="mx-auto max-w-[700px]">
    <ImageInputPanel
      v-if="!sourceImage"
      :source-image="sourceImage"
      :url-loading="urlLoading"
      :input-error="inputError"
      @file-select="loadImageFromFile"
      @url-load="handleUrlLoad"
      @drop="loadImageFromFile"
    />

    <div v-if="sourceImage">
      <div class="mb-5 flex items-center justify-between gap-3 max-sm:flex-col max-sm:items-stretch">
        <button class="btn btn-secondary btn-sm" @click="reset">← Start Over</button>
        <div class="flex gap-2 max-sm:justify-center">
          <span class="rounded-full border px-[0.6rem] py-[0.3rem] text-sm" :class="step === 'preview' ? 'border-gold bg-gold-glow text-gold' : 'border-border-subtle bg-card text-text-muted'">1. Original</span>
          <span class="rounded-full border px-[0.6rem] py-[0.3rem] text-sm" :class="step === 'removing' ? 'border-gold bg-gold-glow text-gold' : step === 'crop' || step === 'done' ? 'border-gold-dim bg-card text-text-secondary' : 'border-border-subtle bg-card text-text-muted'">2. Remove BG</span>
          <span class="rounded-full border px-[0.6rem] py-[0.3rem] text-sm" :class="step === 'crop' ? 'border-gold bg-gold-glow text-gold' : step === 'done' ? 'border-gold-dim bg-card text-text-secondary' : 'border-border-subtle bg-card text-text-muted'">3. Crop</span>
        </div>
      </div>

      <div v-if="step === 'preview'" class="flex flex-col items-center gap-4">
        <div class="relative w-full max-w-[500px] overflow-hidden rounded-md border border-border-subtle">
          <img :src="sourceImage" alt="Original" class="block w-full" />
        </div>
        <button class="btn btn-primary" @click="removeBackground">Remove Background</button>
      </div>

      <div v-if="step === 'removing'" class="flex flex-col items-center gap-4">
        <div class="relative w-full max-w-[500px] overflow-hidden rounded-md border border-border-subtle">
          <img :src="sourceImage" alt="Processing" class="block w-full opacity-30 blur-[2px]" />
          <div class="absolute inset-0 flex flex-col items-center justify-center gap-3 text-gold">
            <div class="spinner"></div>
            <p class="text-base">Removing background...</p>
            <p class="text-sm text-text-muted">First run downloads the ML model (~40MB)</p>
          </div>
        </div>
      </div>

      <div v-if="step === 'crop' || step === 'done'" class="flex flex-col gap-4">
        <div class="flex justify-center">
          <canvas
            ref="cropCanvas"
            class="max-w-full cursor-crosshair touch-none rounded-md border border-border-subtle"
            @pointerdown="startCropDrag"
            @pointermove="onCropDrag"
            @pointerup="endCropDrag"
          />
        </div>
        <div class="flex flex-wrap items-center gap-3">
          <button class="btn btn-secondary btn-sm" @click="autoCrop">Auto Crop</button>
          <button class="btn btn-secondary btn-sm" @click="resetCrop">Reset Crop</button>
          <label class="ml-auto flex items-center gap-2 text-chip text-text-secondary">
            <span>Padding</span>
            <input v-model.number="cropPadding" type="range" min="0" max="50" class="w-[100px] accent-gold" />
            <span class="min-w-8 text-right">{{ cropPadding }}px</span>
          </label>
        </div>

        <div class="flex gap-6 rounded-md border border-border-subtle bg-card p-4 max-sm:flex-col">
          <div class="flex flex-col items-center gap-2">
            <h4 class="text-body text-text-secondary">Result</h4>
            <canvas ref="resultCanvas" class="rounded-sm border border-border-subtle" />
          </div>
          <div class="flex min-w-0 flex-1 flex-col gap-3">
            <div class="flex gap-[2px] rounded-sm bg-surface p-[2px]">
              <button class="flex-1 whitespace-nowrap rounded-sm px-2 py-1.5 text-sm transition-colors" :class="saveTab === 'existing' ? 'bg-card text-gold' : 'text-text-muted hover:text-text-secondary'" @click="saveTab = 'existing'">Existing Coin</button>
              <button class="flex-1 whitespace-nowrap rounded-sm px-2 py-1.5 text-sm transition-colors" :class="saveTab === 'new' ? 'bg-card text-gold' : 'text-text-muted hover:text-text-secondary'" @click="saveTab = 'new'">New Coin</button>
              <button class="flex-1 whitespace-nowrap rounded-sm px-2 py-1.5 text-sm transition-colors" :class="saveTab === 'download' ? 'bg-card text-gold' : 'text-text-muted hover:text-text-secondary'" @click="saveTab = 'download'">Download</button>
            </div>

            <div v-if="saveTab === 'existing'" class="flex flex-col gap-2">
              <div>
                <input v-model="coinSearch" type="text" class="form-input w-full text-body" placeholder="Search coins..." @input="searchCoins" />
              </div>
              <div v-if="coinOptions.length" class="flex max-h-[140px] flex-col gap-[2px] overflow-y-auto rounded-sm border border-border-subtle p-[2px]">
                <button v-for="c in coinOptions" :key="c.id" class="flex flex-col items-start gap-[0.1rem] rounded-sm px-2.5 py-1.5 text-left transition-colors hover:bg-gold-glow" :class="selectedCoinId === c.id ? 'bg-gold-glow outline outline-1 outline-gold-dim' : ''" @click="selectedCoinId = c.id">
                  <span class="text-chip text-text-primary">{{ c.name }}</span>
                  <span class="text-label text-text-muted">{{ [c.ruler, c.era].filter(Boolean).join(' · ') }}</span>
                </button>
              </div>
              <p v-else-if="coinSearch && !coinsLoading" class="text-sm text-text-muted">No coins found</p>
              <p v-else-if="coinsLoading" class="text-sm text-text-muted">Searching...</p>
              <p v-else class="text-sm text-text-muted">Type to search your collection</p>
              <div v-if="selectedCoinId" class="flex gap-4">
                <label class="flex cursor-pointer items-center gap-1 text-body text-text-secondary">
                  <input v-model="saveImageType" type="radio" value="obverse" name="imgType" class="accent-gold" />
                  <span>Obverse</span>
                </label>
                <label class="flex cursor-pointer items-center gap-1 text-body text-text-secondary">
                  <input v-model="saveImageType" type="radio" value="reverse" name="imgType" class="accent-gold" />
                  <span>Reverse</span>
                </label>
              </div>
              <button class="btn btn-primary" :disabled="!selectedCoinId || saving" @click="saveToExisting">
                {{ saving ? 'Saving...' : 'Upload to Coin' }}
              </button>
            </div>

            <div v-if="saveTab === 'new'" class="flex flex-col gap-2">
              <label class="text-chip text-text-muted">Coin Name</label>
              <input v-model="newCoinName" type="text" class="form-input" placeholder="e.g. Augustus Denarius" />
              <button class="btn btn-primary" :disabled="!newCoinName.trim() || saving" @click="saveToNewCoin">
                {{ saving ? 'Creating...' : 'Create Coin & Upload' }}
              </button>
              <p class="text-sm text-text-muted">Image will be saved as the obverse</p>
            </div>

            <div v-if="saveTab === 'download'" class="flex flex-col gap-2">
              <button class="btn btn-secondary" @click="downloadResult">Download PNG</button>
            </div>

            <p v-if="saveMsg" class="text-body" :class="saveError ? 'text-red-400' : 'text-gold'">{{ saveMsg }}</p>
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
