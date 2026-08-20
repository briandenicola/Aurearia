<template>
  <section class="mb-6">
    <h3 class="mb-3 text-base font-medium text-text-primary">Tags & Sets</h3>
    <div class="flex flex-wrap items-center gap-2">
      <span
        v-for="tag in tags"
        :key="`tag-${tag.id}`"
        class="chip inline-flex items-center gap-[0.35rem]"
        :style="{ backgroundColor: tag.color + '22', color: tag.color, borderColor: tag.color + '44' }"
      >
        {{ tag.name }}
        <button class="bg-transparent p-0 text-sm leading-none opacity-60 transition-opacity hover:opacity-100" type="button" :aria-label="`Remove ${tag.name} tag`" @click="handleRemoveTag(tag.id)">x</button>
      </span>
      <span
        v-for="set in sets"
        :key="`set-${set.id}`"
        class="chip inline-flex items-center gap-[0.35rem]"
        :style="{ backgroundColor: set.color + '22', color: set.color, borderColor: set.color + '44', borderStyle: 'dashed' }"
      >
        {{ set.name }}
        <button
          v-if="canManageSetMembership(set.setType)"
          class="bg-transparent p-0 text-sm leading-none opacity-60 transition-opacity hover:opacity-100"
          type="button"
          :aria-label="`Remove ${set.name} set`"
          @click="handleRemoveSet(set.id)"
        >
          x
        </button>
      </span>
      <button
        v-if="!showTagPicker"
        class="rounded-full border border-dashed border-border-subtle px-[0.85rem] py-[0.35rem] text-chip text-text-secondary transition-colors hover:border-text-primary hover:text-text-primary"
        type="button"
        @click="showTagPicker = true"
      >
        + Tag or Set
      </button>
    </div>
    <div v-if="showTagPicker" class="mt-3 flex items-center gap-2 max-sm:flex-col max-sm:items-stretch">
      <select v-model="itemToAdd" class="form-select min-w-0 flex-1" @change="handleAddItem">
        <option value="" disabled>Select a tag or set...</option>
        <optgroup v-if="availableTags.length" label="Tags">
          <option
            v-for="tag in availableTags"
            :key="`tag-option-${tag.id}`"
            :value="`tag:${tag.id}`"
          >
            {{ tag.name }}
          </option>
        </optgroup>
        <optgroup v-if="availableSets.length" label="Sets">
          <option
            v-for="set in availableSets"
            :key="`set-option-${set.id}`"
            :value="`set:${set.id}`"
          >
            {{ set.name }}
          </option>
        </optgroup>
        <option
          v-if="!availableTags.length && !availableSets.length"
          value=""
          disabled
        >
          No tags or sets available
        </option>
      </select>
      <button class="btn btn-ghost btn-xs" type="button" @click="showTagPicker = false">Cancel</button>
    </div>

    <div class="mt-4 border-t border-border-subtle pt-4">
      <h4 class="mb-2 text-sm font-medium text-text-secondary">Suggested Tags & Sets</h4>
      <div v-if="recommendationsLoading" class="text-sm text-text-muted">Loading suggestions...</div>
      <div v-else-if="!recommendations.length" class="text-sm text-text-muted">No suggestions yet.</div>
      <div v-else class="flex flex-col gap-2">
        <div
          v-for="recommendation in recommendations"
          :key="recommendation.id"
          class="rounded-sm border border-border-subtle bg-card p-2"
        >
          <div class="mb-1 flex items-center gap-2">
            <span class="chip-sm">{{ recommendation.targetType === 'set' ? 'Set' : 'Tag' }}</span>
            <span class="text-sm font-medium text-text-primary">{{ recommendation.targetName }}</span>
            <span class="chip-sm ml-auto capitalize">{{ recommendation.confidence }}</span>
          </div>
          <p class="m-0 text-xs text-text-secondary">{{ recommendation.reasons.join(' • ') }}</p>
          <div class="mt-2 flex gap-2">
            <button class="btn btn-secondary btn-xs" type="button" @click="handleAcceptRecommendation(recommendation.id)">Accept</button>
            <button class="btn btn-ghost btn-xs" type="button" @click="handleRejectRecommendation(recommendation.id)">Reject</button>
          </div>
        </div>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { getTags, getSets, addTagToCoin, removeTagFromCoin, addCoinToSet, removeCoinFromSet, getCoinRecommendations, acceptCoinRecommendation, rejectCoinRecommendation } from '@/api/client'
import { normalizeCoinSetType } from '@/types'
import type { CoinRecommendation, CoinSetSummary, Tag } from '@/types'

type SetChip = Pick<CoinSetSummary, 'id' | 'name' | 'color' | 'setType'>

const props = defineProps<{
  tags: Tag[]
  sets: SetChip[]
  coinId: number
}>()

const emit = defineEmits<{
  tagsChanged: []
}>()

const userTags = ref<Tag[]>([])
const userSets = ref<CoinSetSummary[]>([])
const recommendations = ref<CoinRecommendation[]>([])
const recommendationsLoading = ref(false)
const showTagPicker = ref(false)
const itemToAdd = ref('')

const availableTags = computed(() => {
  const coinTagIds = new Set(props.tags.map(t => t.id))
  return userTags.value.filter(t => !coinTagIds.has(t.id))
})

const availableSets = computed(() => {
  const coinSetIds = new Set(props.sets.map(s => s.id))
  return userSets.value.filter(s => canManageSetMembership(s.setType) && !coinSetIds.has(s.id))
})

function canManageSetMembership(setType: CoinSetSummary['setType']): boolean {
  const normalizedType = normalizeCoinSetType(setType)
  return normalizedType !== 'smart' && normalizedType !== 'agentic'
}

onMounted(async () => {
  await Promise.all([loadAvailableItems(), loadRecommendations()])
})

async function loadAvailableItems() {
  try {
    const [tagsRes, setsRes] = await Promise.all([getTags(), getSets()])
    userTags.value = tagsRes.data?.tags ?? []
    userSets.value = setsRes.data?.sets ?? []
  } catch { /* ignore */ }
}

async function loadRecommendations() {
  recommendationsLoading.value = true
  try {
    const response = await getCoinRecommendations(props.coinId)
    recommendations.value = response.data?.recommendations ?? []
  } catch {
    recommendations.value = []
  } finally {
    recommendationsLoading.value = false
  }
}

async function handleAddItem() {
  if (!itemToAdd.value) return
  const [source, id] = itemToAdd.value.split(':')
  try {
    if (source === 'set') {
      await addCoinToSet(Number(id), { coinId: props.coinId })
    } else {
      await addTagToCoin(props.coinId, Number(id))
    }
    emit('tagsChanged')
  } catch { /* ignore */ }
  itemToAdd.value = ''
  showTagPicker.value = false
  await Promise.all([loadAvailableItems(), loadRecommendations()])
}

async function handleRemoveTag(tagId: number) {
  try {
    await removeTagFromCoin(props.coinId, tagId)
    emit('tagsChanged')
  } catch { /* ignore */ }
  await Promise.all([loadAvailableItems(), loadRecommendations()])
}

async function handleRemoveSet(setId: number) {
  try {
    await removeCoinFromSet(setId, props.coinId)
    emit('tagsChanged')
  } catch { /* ignore */ }
  await Promise.all([loadAvailableItems(), loadRecommendations()])
}

async function handleAcceptRecommendation(recommendationId: number) {
  try {
    await acceptCoinRecommendation(props.coinId, recommendationId)
    emit('tagsChanged')
  } catch { /* ignore */ }
  await Promise.all([loadAvailableItems(), loadRecommendations()])
}

async function handleRejectRecommendation(recommendationId: number) {
  try {
    await rejectCoinRecommendation(props.coinId, recommendationId)
  } catch { /* ignore */ }
  await loadRecommendations()
}
</script>
