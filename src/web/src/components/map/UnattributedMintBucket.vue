<template>
  <section class="card p-4" aria-labelledby="unattributed-title">
    <button
      class="flex w-full cursor-pointer items-center justify-between gap-3 border-0 bg-transparent p-0 text-left text-text-primary"
      type="button"
      :aria-expanded="expanded"
      aria-controls="unattributed-content"
      @click="$emit('update:expanded', !expanded)"
    >
      <span>
        <span id="unattributed-title" class="section-label">Unattributed Mints</span>
        <strong class="mt-1 block text-base text-text-primary">{{ totalCount }} {{ totalCount === 1 ? 'coin needs' : 'coins need' }} mint review</strong>
      </span>
      <ChevronDown :size="18" class="shrink-0 text-gold transition-transform duration-200" :class="expanded ? 'rotate-180' : ''" />
    </button>

    <div v-if="expanded" id="unattributed-content" class="mt-4 flex flex-col gap-4">
      <div v-if="unknown.length" class="border-t border-border-subtle pt-3">
        <h3 class="mb-1">Unknown mint</h3>
        <ul class="m-0 flex flex-col gap-1.5 pl-4">
          <li v-for="coin in unknown" :key="coin.id">
            <router-link :to="`/coin/${coin.id}`">{{ coin.name }}</router-link>
          </li>
        </ul>
      </div>

      <div v-for="group in unmatched" :key="group.normalizedName" class="border-t border-border-subtle pt-3">
        <h3 class="mb-1">{{ group.originalNames.join(', ') }}</h3>
        <p class="mb-2 text-body text-text-secondary">No static mint coordinate matched this name.</p>
        <ul class="m-0 flex flex-col gap-1.5 pl-4">
          <li v-for="coin in group.coins" :key="coin.id">
            <router-link :to="`/coin/${coin.id}`">{{ coin.name }}</router-link>
          </li>
        </ul>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { ChevronDown } from 'lucide-vue-next'
import type { Coin } from '@/types'
import type { UnmatchedMintGroup } from '@/utils/mintMap'

const props = defineProps<{
  unknown: Coin[]
  unmatched: UnmatchedMintGroup[]
  expanded: boolean
}>()

defineEmits<{
  'update:expanded': [value: boolean]
}>()

const totalCount = computed(() =>
  props.unknown.length + props.unmatched.reduce((total, group) => total + group.coins.length, 0),
)
</script>
