<template>
  <a v-if="safeHref" :href="safeHref" :target="target" :rel="computedRel">
    <slot />
  </a>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { sanitizeExternalUrl } from '@/composables/useSafeExternalLink'

const props = withDefaults(defineProps<{
  href: string | null | undefined
  target?: string
  rel?: string
}>(), {
  target: '_blank',
  rel: 'noopener noreferrer',
})

const safeHref = computed(() => sanitizeExternalUrl(props.href))

const computedRel = computed(() => {
  const relTokens = new Set(
    (props.rel ?? '')
      .split(/\s+/)
      .map(token => token.trim())
      .filter(Boolean),
  )

  if (props.target === '_blank') {
    relTokens.add('noopener')
    relTokens.add('noreferrer')
  }

  return Array.from(relTokens).join(' ')
})
</script>
