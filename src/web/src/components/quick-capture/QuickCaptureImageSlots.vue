<template>
  <div class="grid gap-4 [grid-template-columns:repeat(auto-fit,minmax(180px,1fr))]">
    <label class="relative grid min-h-[170px] cursor-pointer gap-3 rounded-sm border border-dashed border-border-accent bg-card p-4 transition-[border-color,background] duration-200 hover:border-gold hover:bg-card-hover">
      <span class="text-base font-semibold text-heading">Obverse</span>
      <img v-if="obverseUrl" :src="obverseUrl" alt="Obverse preview" class="aspect-square w-full rounded-sm border border-border-subtle object-cover">
      <span v-else class="grid min-h-20 place-items-center rounded-sm border border-dashed border-border-subtle p-3 text-center text-body text-text-secondary">Take or upload obverse photo</span>
      <span class="justify-self-start rounded-full border border-border-accent px-[0.7rem] py-1 text-sm font-medium text-gold">{{ obverseImage ? 'Replace photo' : 'Choose photo' }}</span>
      <button v-if="obverseImage" type="button" class="relative z-10 justify-self-start bg-transparent p-0 text-sm text-byzantine underline" @click.prevent="emit('update:obverseImage', null)">Remove</button>
      <input class="absolute inset-0 cursor-pointer opacity-0" type="file" accept="image/*" capture="environment" @change="onFile('obverse', $event)">
    </label>
    <label class="relative grid min-h-[170px] cursor-pointer gap-3 rounded-sm border border-dashed border-border-accent bg-card p-4 transition-[border-color,background] duration-200 hover:border-gold hover:bg-card-hover">
      <span class="text-base font-semibold text-heading">Reverse</span>
      <img v-if="reverseUrl" :src="reverseUrl" alt="Reverse preview" class="aspect-square w-full rounded-sm border border-border-subtle object-cover">
      <span v-else class="grid min-h-20 place-items-center rounded-sm border border-dashed border-border-subtle p-3 text-center text-body text-text-secondary">Optional reverse photo</span>
      <span class="justify-self-start rounded-full border border-border-accent px-[0.7rem] py-1 text-sm font-medium text-gold">{{ reverseImage ? 'Replace photo' : 'Choose photo' }}</span>
      <button v-if="reverseImage" type="button" class="relative z-10 justify-self-start bg-transparent p-0 text-sm text-byzantine underline" @click.prevent="emit('update:reverseImage', null)">Remove</button>
      <input class="absolute inset-0 cursor-pointer opacity-0" type="file" accept="image/*" capture="environment" @change="onFile('reverse', $event)">
    </label>
    <label class="relative grid min-h-[170px] cursor-pointer gap-3 rounded-sm border border-dashed border-border-accent bg-card p-4 transition-[border-color,background] duration-200 hover:border-gold hover:bg-card-hover">
      <span class="text-base font-semibold text-heading">Detail photos</span>
      <span class="grid min-h-20 place-items-center rounded-sm border border-dashed border-border-subtle p-3 text-center text-body text-text-secondary">{{ detailCountText }}</span>
      <span class="justify-self-start rounded-full border border-border-accent px-[0.7rem] py-1 text-sm font-medium text-gold">{{ detailImages.length ? 'Replace details' : 'Choose details' }}</span>
      <button v-if="detailImages.length" type="button" class="relative z-10 justify-self-start bg-transparent p-0 text-sm text-byzantine underline" @click.prevent="emit('update:detailImages', [])">Remove</button>
      <input class="absolute inset-0 cursor-pointer opacity-0" type="file" accept="image/*" multiple @change="onDetails">
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
