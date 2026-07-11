<template>
  <img v-if="objectUrl" :src="objectUrl" :alt="alt" v-bind="$attrs" />
  <slot v-else-if="loadFailed" name="fallback" />
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useAuthenticatedMedia } from '@/composables/useAuthenticatedMedia'
import { useNetworkQuality } from '@/composables/useNetworkQuality'

defineOptions({ inheritAttrs: false })

const props = withDefaults(defineProps<{
  mediaPath: string | null | undefined
  alt?: string
}>(), {
  alt: '',
})

const { imageSize } = useNetworkQuality()

const effectivePath = computed(() => {
  const p = props.mediaPath
  if (!p || imageSize.value === 'full') return p
  const sep = p.includes('?') ? '&' : '?'
  return `${p}${sep}size=${imageSize.value}`
})

const { objectUrl, loadFailed } = useAuthenticatedMedia(effectivePath)
</script>
