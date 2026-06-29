<template>
  <div class="quick-capture-slots">
    <label class="slot-card">
      <span class="slot-title">Obverse</span>
      <img v-if="obverseUrl" :src="obverseUrl" alt="Obverse preview" class="slot-preview">
      <span v-else class="slot-empty">Take or upload obverse photo</span>
      <span class="slot-action">{{ obverseImage ? 'Replace photo' : 'Choose photo' }}</span>
      <button v-if="obverseImage" type="button" class="slot-clear" @click.prevent="emit('update:obverseImage', null)">Remove</button>
      <input class="slot-input" type="file" accept="image/*" capture="environment" @change="onFile('obverse', $event)">
    </label>
    <label class="slot-card">
      <span class="slot-title">Reverse</span>
      <img v-if="reverseUrl" :src="reverseUrl" alt="Reverse preview" class="slot-preview">
      <span v-else class="slot-empty">Optional reverse photo</span>
      <span class="slot-action">{{ reverseImage ? 'Replace photo' : 'Choose photo' }}</span>
      <button v-if="reverseImage" type="button" class="slot-clear" @click.prevent="emit('update:reverseImage', null)">Remove</button>
      <input class="slot-input" type="file" accept="image/*" capture="environment" @change="onFile('reverse', $event)">
    </label>
    <label class="slot-card detail">
      <span class="slot-title">Detail photos</span>
      <span class="slot-empty">{{ detailCountText }}</span>
      <span class="slot-action">{{ detailImages.length ? 'Replace details' : 'Choose details' }}</span>
      <button v-if="detailImages.length" type="button" class="slot-clear" @click.prevent="emit('update:detailImages', [])">Remove</button>
      <input class="slot-input" type="file" accept="image/*" multiple @change="onDetails">
    </label>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'

const props = defineProps<{
  obverseImage: File | null
  reverseImage: File | null
  detailImages: File[]
}>()

const emit = defineEmits<{
  'update:obverseImage': [file: File | null]
  'update:reverseImage': [file: File | null]
  'update:detailImages': [files: File[]]
}>()

const obverseUrl = ref('')
const reverseUrl = ref('')

function refreshUrl(target: 'obverse' | 'reverse', file: File | null) {
  const current = target === 'obverse' ? obverseUrl : reverseUrl
  if (current.value) URL.revokeObjectURL(current.value)
  current.value = file ? URL.createObjectURL(file) : ''
}

watch(() => props.obverseImage, file => refreshUrl('obverse', file), { immediate: true })
watch(() => props.reverseImage, file => refreshUrl('reverse', file), { immediate: true })
onBeforeUnmount(() => {
  if (obverseUrl.value) URL.revokeObjectURL(obverseUrl.value)
  if (reverseUrl.value) URL.revokeObjectURL(reverseUrl.value)
})

const detailCountText = computed(() => props.detailImages.length ? `${props.detailImages.length} selected` : 'Optional detail images')

function onFile(target: 'obverse' | 'reverse', event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0] ?? null
  if (target === 'obverse') {
    emit('update:obverseImage', file)
  } else {
    emit('update:reverseImage', file)
  }
}

function onDetails(event: Event) {
  const input = event.target as HTMLInputElement
  emit('update:detailImages', Array.from(input.files ?? []))
}
</script>

<style scoped>
.quick-capture-slots {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: 1rem;
}

.slot-card {
  position: relative;
  display: grid;
  gap: 0.75rem;
  min-height: 170px;
  border: 1px dashed var(--border-accent);
  border-radius: var(--radius-sm);
  padding: 1rem;
  background: var(--bg-card);
  cursor: pointer;
  transition: border-color var(--transition-fast), background var(--transition-fast);
}

.slot-card:hover {
  border-color: var(--accent-gold);
  background: var(--bg-card-hover);
}

.slot-title {
  color: var(--text-heading);
  font-size: 0.9rem;
  font-weight: 600;
}

.slot-preview {
  width: 100%;
  aspect-ratio: 1;
  object-fit: cover;
  border-radius: var(--radius-sm);
  border: 1px solid var(--border-subtle);
}

.slot-empty {
  display: grid;
  min-height: 5rem;
  place-items: center;
  border: 1px dashed var(--border-subtle);
  border-radius: var(--radius-sm);
  color: var(--text-secondary);
  font-size: 0.85rem;
  text-align: center;
  padding: 0.75rem;
}

.slot-action {
  justify-self: start;
  border-radius: var(--radius-full);
  border: 1px solid var(--border-accent);
  padding: 0.25rem 0.7rem;
  color: var(--accent-gold);
  font-size: 0.75rem;
  font-weight: 500;
}

.slot-input {
  position: absolute;
  inset: 0;
  opacity: 0;
  cursor: pointer;
}

.slot-clear {
  justify-self: start;
  border: 0;
  background: transparent;
  color: var(--cat-byzantine);
  cursor: pointer;
  font-size: 0.75rem;
  padding: 0;
  text-decoration: underline;
  z-index: 1;
}
</style>
