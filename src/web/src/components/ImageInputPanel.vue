<template>
  <div class="card">
    <h3 class="mb-4 text-base font-medium text-heading">Load Image</h3>
    <div class="flex flex-col gap-4">
      <label
        class="flex cursor-pointer flex-col items-center justify-center gap-3 rounded-md border-2 border-dashed border-border-subtle px-6 py-10 text-center text-text-muted transition-colors hover:border-gold hover:bg-gold-glow hover:text-gold"
        :class="{ 'border-gold bg-gold-glow text-gold': dragging }"
        @dragover.prevent="dragging = true"
        @dragleave="dragging = false"
        @drop.prevent="onDrop"
      >
        <Upload :size="32" />
        <span class="text-base">Drop an image here or click to browse</span>
        <input type="file" accept="image/*" hidden @change="onFileSelect" />
      </label>
      <div class="flex gap-2 max-sm:flex-col">
        <input
          v-model="url"
          type="url"
          class="form-input flex-1"
          placeholder="Or paste an image URL..."
          @keydown.enter="onUrlLoad"
        />
        <button class="btn btn-primary btn-sm" :disabled="!url || urlLoading" @click="onUrlLoad">
          {{ urlLoading ? 'Loading...' : 'Fetch' }}
        </button>
      </div>
      <p v-if="inputError" class="text-body text-loss">{{ inputError }}</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { Upload } from 'lucide-vue-next'

// Props are type-checked but not referenced directly in script
const _props = defineProps<{
  sourceImage: string | null
  urlLoading: boolean
  inputError: string
}>()

const emit = defineEmits<{
  'file-select': [file: File]
  'url-load': [url: string]
  'drop': [file: File]
}>()

const dragging = ref(false)
const url = ref('')

function onFileSelect(e: Event) {
  const file = (e.target as HTMLInputElement).files?.[0]
  if (file) emit('file-select', file)
}

function onDrop(e: DragEvent) {
  dragging.value = false
  const file = e.dataTransfer?.files?.[0]
  if (file) emit('drop', file)
}

function onUrlLoad() {
  if (!url.value) return
  emit('url-load', url.value)
}
</script>
