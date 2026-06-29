<template>
  <div class="quick-capture-slots">
    <label class="slot-card">
      <span class="slot-title">Obverse</span>
      <img v-if="obverseUrl" :src="obverseUrl" alt="Obverse preview" class="slot-preview">
      <button v-if="obverseImage" type="button" class="slot-clear" @click.prevent="emit('update:obverseImage', null)">Remove obverse</button>
      <span v-else class="slot-empty">Take or upload obverse photo</span>
      <input type="file" accept="image/*" capture="environment" @change="onFile('obverse', $event)">
    </label>
    <label class="slot-card">
      <span class="slot-title">Reverse</span>
      <img v-if="reverseUrl" :src="reverseUrl" alt="Reverse preview" class="slot-preview">
      <button v-if="reverseImage" type="button" class="slot-clear" @click.prevent="emit('update:reverseImage', null)">Remove reverse</button>
      <span v-else class="slot-empty">Take or upload reverse photo</span>
      <input type="file" accept="image/*" capture="environment" @change="onFile('reverse', $event)">
    </label>
    <label class="slot-card detail">
      <span class="slot-title">Detail photos</span>
      <span class="slot-empty">{{ detailCountText }}</span>
      <button v-if="detailImages.length" type="button" class="slot-clear" @click.prevent="emit('update:detailImages', [])">Remove detail photos</button>
      <input type="file" accept="image/*" multiple @change="onDetails">
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
  grid-template-columns: repeat(auto-fit, minmax(140px, 1fr));
  gap: 1rem;
}
.slot-card {
  border: 1px dashed var(--color-border);
  border-radius: 1rem;
  padding: 1rem;
  background: var(--color-surface);
}
.slot-title {
  display: block;
  font-weight: 600;
  margin-bottom: 0.5rem;
}
.slot-preview {
  width: 100%;
  aspect-ratio: 1;
  object-fit: cover;
  border-radius: 0.75rem;
}
.slot-empty {
  display: block;
  min-height: 4rem;
  color: var(--color-text-muted);
}
input {
  margin-top: 0.75rem;
  max-width: 100%;
}
.slot-clear {
  display: block;
  margin-top: 0.5rem;
}
</style>
