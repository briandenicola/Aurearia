<template>
  <section class="tags-section">
    <h3>Tags & Sets</h3>
    <div class="detail-tags">
      <span
        v-for="tag in tags"
        :key="`tag-${tag.id}`"
        class="detail-tag-chip"
        :style="{ backgroundColor: tag.color + '22', color: tag.color, borderColor: tag.color + '44' }"
      >
        {{ tag.name }}
        <button class="tag-remove" type="button" :aria-label="`Remove ${tag.name} tag`" @click="handleRemoveTag(tag.id)">x</button>
      </span>
      <span
        v-for="set in sets"
        :key="`set-${set.id}`"
        class="detail-tag-chip set-chip"
        :style="{ backgroundColor: set.color + '22', color: set.color, borderColor: set.color + '44' }"
      >
        {{ set.name }}
        <button
          v-if="set.setType !== 'smart'"
          class="tag-remove"
          type="button"
          :aria-label="`Remove ${set.name} set`"
          @click="handleRemoveSet(set.id)"
        >
          x
        </button>
      </span>
      <button v-if="!showTagPicker" class="btn-tag-add" type="button" @click="showTagPicker = true">+ Tag or Set</button>
    </div>
    <div v-if="showTagPicker" class="tag-picker">
      <select v-model="itemToAdd" class="tag-picker-select" @change="handleAddItem">
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
      <button class="btn-tag-cancel" type="button" @click="showTagPicker = false">Cancel</button>
    </div>
  </section>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { getTags, getSets, addTagToCoin, removeTagFromCoin, addCoinToSet, removeCoinFromSet } from '@/api/client'
import type { CoinSetSummary, Tag } from '@/types'

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
const showTagPicker = ref(false)
const itemToAdd = ref('')

const availableTags = computed(() => {
  const coinTagIds = new Set(props.tags.map(t => t.id))
  return userTags.value.filter(t => !coinTagIds.has(t.id))
})

const availableSets = computed(() => {
  const coinSetIds = new Set(props.sets.map(s => s.id))
  return userSets.value.filter(s => s.setType !== 'smart' && !coinSetIds.has(s.id))
})

onMounted(async () => {
  try {
    const [tagsRes, setsRes] = await Promise.all([getTags(), getSets()])
    userTags.value = tagsRes.data?.tags ?? []
    userSets.value = setsRes.data?.sets ?? []
  } catch { /* ignore */ }
})

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
}

async function handleRemoveTag(tagId: number) {
  try {
    await removeTagFromCoin(props.coinId, tagId)
    emit('tagsChanged')
  } catch { /* ignore */ }
}

async function handleRemoveSet(setId: number) {
  try {
    await removeCoinFromSet(setId, props.coinId)
    emit('tagsChanged')
  } catch { /* ignore */ }
}
</script>

<style scoped>
.tags-section {
  margin-bottom: 1.5rem;
}

.tags-section h3 {
  margin-bottom: 0.75rem;
  font-size: 1rem;
}

.detail-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
  align-items: center;
}

.detail-tag-chip {
  font-size: 0.8rem;
  padding: 0.35rem 0.85rem;
  border-radius: var(--radius-full);
  border: 1px solid;
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
}

.set-chip {
  border-style: dashed;
}

.tag-remove {
  background: none;
  border: none;
  color: inherit;
  cursor: pointer;
  font-size: 0.75rem;
  padding: 0;
  opacity: 0.6;
  line-height: 1;
}

.tag-remove:hover {
  opacity: 1;
}

.btn-tag-add {
  background: none;
  border: 1px dashed var(--border-subtle);
  border-radius: var(--radius-full);
  color: var(--text-secondary);
  font-size: 0.8rem;
  padding: 0.35rem 0.85rem;
  cursor: pointer;
}

.btn-tag-add:hover {
  color: var(--text-primary);
  border-color: var(--text-primary);
}

.tag-picker {
  display: flex;
  gap: 0.5rem;
  align-items: center;
  margin-top: 0.75rem;
}

.tag-picker-select {
  padding: 0.35rem 0.6rem;
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
  background: var(--bg-card);
  color: var(--text-primary);
  font-size: 0.8rem;
}

.btn-tag-cancel {
  background: none;
  border: none;
  color: var(--text-secondary);
  font-size: 0.75rem;
  cursor: pointer;
}
</style>
