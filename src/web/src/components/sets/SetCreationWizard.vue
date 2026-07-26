<template>
  <div class="w-full">
    <form @submit.prevent="submit">
      <div class="form-group mb-4">
        <label for="setType" class="form-label mb-2 block">Set type</label>
        <select id="setType" v-model="form.setType" class="form-input w-full">
          <option value="standard">Standard</option>
          <option value="goal">Goal</option>
          <option value="smart">Smart</option>
          <option value="agentic">Agentic</option>
        </select>
      </div>
      <div v-if="form.setType === 'goal'" class="form-group mb-4 rounded-sm border border-border-subtle bg-card p-4">
        <span class="section-label">How goal completion works</span>
        <p class="mt-2 mb-0 text-body text-text-secondary">
          Goal sets track both collection and wishlist members.
        </p>
        <p class="mt-2 mb-0 text-body text-text-secondary">
          Completion is calculated as collection items divided by collection plus wishlist items.
        </p>
        <p class="mt-2 mb-0 text-chip text-text-muted">
          Example: 2 collection and 5 wishlist = 2 / (2 + 5) = 28.6%.
        </p>
      </div>
      <SetSmartRuleBuilder
        v-if="form.setType === 'smart'"
        @update="form.smartCriteria = $event"
      />
      <div v-if="form.setType === 'agentic'" class="form-group mb-4 rounded-sm border border-border-subtle bg-card p-4">
        <span class="section-label">How agentic sets work</span>
        <p class="mt-2 mb-0 text-body text-text-secondary">
          Describe the set you want and the agent will propose matching coins for you to review.
        </p>
        <p class="mt-2 mb-0 text-body text-text-secondary">
          No set is created immediately. A proposal request is submitted for the agent to work through.
        </p>
      </div>
      <div v-if="form.setType === 'agentic'" class="form-group mb-4">
        <label for="agenticPrompt" class="form-label mb-2 block">Agentic prompt</label>
        <textarea
          id="agenticPrompt"
          v-model="form.agenticPrompt"
          rows="3"
          maxlength="500"
          class="form-input w-full"
          placeholder="Example: All US silver quarters from 1940s to 1960s"
          :required="form.setType === 'agentic'"
        />
      </div>
      <div class="form-group mb-4">
        <label for="setName" class="form-label mb-2 block">Name</label>
        <input
          id="setName"
          v-model="form.name"
          type="text"
          required
          maxlength="80"
          class="form-input w-full"
        />
      </div>
      <div class="form-group mb-4">
        <label for="setDescription" class="form-label mb-2 block">Description</label>
        <textarea
          id="setDescription"
          v-model="form.description"
          rows="3"
          maxlength="2000"
          class="form-input w-full"
        />
      </div>
      <div class="form-group mb-4">
        <label for="setColor" class="form-label mb-2 block">Color</label>
        <input
          id="setColor"
          v-model="form.color"
          type="color"
          class="h-10 w-full rounded-sm border border-border-subtle bg-input p-1 text-text-primary"
        />
      </div>
      <div class="mt-6 flex justify-end gap-2">
        <button type="button" class="btn btn-secondary" @click="$emit('cancel')">Cancel</button>
        <button type="submit" class="btn btn-primary" :disabled="!form.name.trim()">
          {{ submitLabel }}
        </button>
      </div>
    </form>
  </div>
</template>

<script setup lang="ts">
import { reactive, watch } from 'vue'
import { normalizeCoinSetType } from '@/types'
import type { CreateCoinSetRequest, SmartCriteriaGroup } from '@/types'
import SetSmartRuleBuilder from '@/components/sets/SetSmartRuleBuilder.vue'
import { randomSetColor } from '@/utils/setColors'

const props = withDefaults(defineProps<{
  initialValue?: Partial<CreateCoinSetRequest>
  submitLabel?: string
}>(), {
  submitLabel: 'Create',
})

const emit = defineEmits<{
  submit: [value: CreateCoinSetRequest, csv?: string]
  cancel: []
}>()

const form = reactive({
  name: props.initialValue?.name ?? '',
  description: props.initialValue?.description ?? '',
  color: props.initialValue?.color ?? randomSetColor(),
  setType: props.initialValue?.setType ? normalizeCoinSetType(props.initialValue.setType) : 'standard',
  templateId: props.initialValue?.templateId ?? '',
  targetCompletionDate: props.initialValue?.targetCompletionDate ?? '',
  smartCriteria: props.initialValue?.smartCriteria as SmartCriteriaGroup | undefined,
  agenticPrompt: props.initialValue?.agenticPrompt ?? '',
})

watch(() => props.initialValue, (value) => {
  form.name = value?.name ?? ''
  form.description = value?.description ?? ''
  form.color = value?.color ?? randomSetColor()
  form.setType = value?.setType ? normalizeCoinSetType(value.setType) : 'standard'
  form.templateId = value?.templateId ?? ''
  form.targetCompletionDate = value?.targetCompletionDate ?? ''
  form.smartCriteria = value?.smartCriteria as SmartCriteriaGroup | undefined
  form.agenticPrompt = value?.agenticPrompt ?? ''
})

function submit() {
  const name = form.name.trim()
  if (!name) return
  emit('submit', {
    name,
    description: form.description.trim(),
    color: form.color,
    setType: form.setType,
    templateId: form.setType !== 'goal' ? (form.templateId || undefined) : undefined,
    targetCompletionDate: form.setType !== 'goal' ? (form.targetCompletionDate || undefined) : undefined,
    smartCriteria: form.smartCriteria ?? undefined,
    agenticPrompt: form.setType === 'agentic' ? form.agenticPrompt.trim() || undefined : undefined,
  }, undefined)
}
</script>
