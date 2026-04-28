<template>
  <div class="detail-tags-section">
    <div class="detail-tags">
      <span class="badge" :class="`badge-${category.toLowerCase()}`">{{ category }}</span>
      <span
        v-for="tag in tags"
        :key="tag.id"
        class="detail-tag-chip"
        :style="{ backgroundColor: tag.color + '22', color: tag.color, borderColor: tag.color + '44' }"
      >
        {{ tag.name }}
        <button class="tag-remove" @click="handleRemoveTag(tag.id)">x</button>
      </span>
      <button v-if="!showTagPicker" class="btn-tag-add" @click="showTagPicker = true">+ Tag</button>
    </div>
    <div v-if="showTagPicker" class="tag-picker">
      <select v-model="tagToAdd" class="tag-picker-select" @change="handleAddTag">
        <option value="" disabled>Select a tag...</option>
        <option
          v-for="tag in availableTags"
          :key="tag.id"
          :value="tag.id"
        >{{ tag.name }}</option>
      </select>
      <button class="btn-tag-cancel" @click="showTagPicker = false">Cancel</button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { getTags, addTagToCoin, removeTagFromCoin } from '@/api/client'
import type { Tag } from '@/types'

const props = defineProps<{
  tags: Tag[]
  category: string
  coinId: number
}>()

const emit = defineEmits<{
  tagsChanged: []
}>()

const userTags = ref<Tag[]>([])
const showTagPicker = ref(false)
const tagToAdd = ref('')

const availableTags = computed(() => {
  const coinTagIds = new Set(props.tags.map(t => t.id))
  return userTags.value.filter(t => !coinTagIds.has(t.id))
})

onMounted(async () => {
  try {
    const res = await getTags()
    userTags.value = res.data?.tags ?? []
  } catch { /* ignore */ }
})

async function handleAddTag() {
  if (!tagToAdd.value) return
  try {
    await addTagToCoin(props.coinId, Number(tagToAdd.value))
    emit('tagsChanged')
  } catch { /* ignore */ }
  tagToAdd.value = ''
  showTagPicker.value = false
}

async function handleRemoveTag(tagId: number) {
  try {
    await removeTagFromCoin(props.coinId, tagId)
    emit('tagsChanged')
  } catch { /* ignore */ }
}
</script>

<style scoped>
.detail-tags-section {
  margin-bottom: 1rem;
}

.detail-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 0.35rem;
  align-items: center;
}

.detail-tag-chip {
  font-size: 0.75rem;
  padding: 0.15rem 0.5rem;
  border-radius: var(--radius-full);
  border: 1px solid;
  display: inline-flex;
  align-items: center;
  gap: 0.25rem;
}

.tag-remove {
  background: none;
  border: none;
  color: inherit;
  cursor: pointer;
  font-size: 0.7rem;
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
  font-size: 0.75rem;
  padding: 0.15rem 0.5rem;
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
  margin-top: 0.5rem;
}

.tag-picker-select {
  padding: 0.3rem 0.5rem;
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
