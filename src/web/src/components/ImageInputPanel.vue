<template>
  <div class="input-section card">
    <h3>Load Image</h3>
    <div class="input-methods">
      <label
        class="drop-zone" :class="{ dragging }" @dragover.prevent="dragging = true"
        @dragleave="dragging = false" @drop.prevent="onDrop"
      >
        <Upload :size="32" />
        <span>Drop an image here or click to browse</span>
        <input type="file" accept="image/*" hidden @change="onFileSelect" />
      </label>
      <div class="url-input-row">
        <input
          v-model="url" type="url" class="form-input" placeholder="Or paste an image URL..."
          @keydown.enter="onUrlLoad"
        />
        <button class="btn btn-primary btn-sm" :disabled="!url || urlLoading" @click="onUrlLoad">
          {{ urlLoading ? 'Loading...' : 'Fetch' }}
        </button>
      </div>
      <p v-if="inputError" class="msg error">{{ inputError }}</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { Upload } from 'lucide-vue-next'

const props = defineProps<{
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

<style scoped>
.input-section h3 {
  margin-bottom: 1rem;
}

.input-methods {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.drop-zone {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 0.75rem;
  padding: 2.5rem 1.5rem;
  border: 2px dashed var(--border-subtle);
  border-radius: var(--radius-md);
  color: var(--text-muted);
  cursor: pointer;
  transition: all var(--transition-fast);
  text-align: center;
}

.drop-zone:hover,
.drop-zone.dragging {
  border-color: var(--accent-gold);
  color: var(--accent-gold);
  background: var(--accent-gold-glow);
}

.url-input-row {
  display: flex;
  gap: 0.5rem;
}

.url-input-row .form-input {
  flex: 1;
}
</style>
