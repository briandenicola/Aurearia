<template>
  <div class="mt-4">
    <span class="section-label">Smart rule</span>
    <div class="mt-2 flex flex-wrap gap-2">
      <select v-model="rule.field" class="rounded-sm border border-border-subtle bg-input px-[0.6rem] py-[0.45rem] text-base text-text-primary">
        <option value="material">Material</option>
        <option value="category">Category</option>
        <option value="mint">Mint</option>
        <option value="grade">Grade</option>
        <option value="currentValue">Current value</option>
        <option value="purchaseDate">Purchase date</option>
      </select>
      <select v-model="rule.op" class="rounded-sm border border-border-subtle bg-input px-[0.6rem] py-[0.45rem] text-base text-text-primary">
        <option value="eq">Equals</option>
        <option value="contains">Contains</option>
        <option value="gte">At least</option>
        <option value="lte">At most</option>
      </select>
      <input v-model="rule.value" class="rounded-sm border border-border-subtle bg-input px-[0.6rem] py-[0.45rem] text-base text-text-primary" placeholder="Value" @input="emitCriteria" />
      <button type="button" class="btn btn-secondary btn-sm" @click="preview">Preview</button>
    </div>
    <p v-if="previewResult" class="mt-2 text-body text-text-secondary">
      {{ previewResult.coinCount }} matching coins, ${{ previewResult.totalValue.toFixed(2) }} total value
    </p>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref, watch } from 'vue'
import { previewSmartSet } from '@/api/client'
import type { SmartCriteriaGroup, SmartCriteriaRule, SmartSetPreview } from '@/types'

const emit = defineEmits<{
  update: [criteria: SmartCriteriaGroup]
}>()

const rule = reactive<SmartCriteriaRule>({
  field: 'material',
  op: 'eq',
  value: 'Silver',
})
const previewResult = ref<SmartSetPreview | null>(null)

watch(rule, emitCriteria, { deep: true, immediate: true })

function buildCriteria(): SmartCriteriaGroup {
  return {
    operator: 'and',
    rules: [{ ...rule }],
  }
}

function emitCriteria() {
  emit('update', buildCriteria())
}

async function preview() {
  const res = await previewSmartSet(buildCriteria())
  previewResult.value = res.data
}
</script>
