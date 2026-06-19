<template>
  <img v-if="objectUrl" :src="objectUrl" :alt="alt" v-bind="$attrs" />
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useAuthenticatedMedia } from '@/composables/useAuthenticatedMedia'

defineOptions({ inheritAttrs: false })

const props = withDefaults(defineProps<{
  mediaPath: string | null | undefined
  alt?: string
}>(), {
  alt: '',
})

const mediaPathRef = computed(() => props.mediaPath)
const { objectUrl } = useAuthenticatedMedia(mediaPathRef)
</script>
