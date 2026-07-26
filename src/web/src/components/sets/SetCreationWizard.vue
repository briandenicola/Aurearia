<template>
  <div class="w-full">
    <form @submit.prevent="submit">
      <div class="form-group mb-4">
        <label for="setType" class="form-label mb-2 block">Set type</label>
        <select id="setType" v-model="form.setType" class="form-input w-full">
          <option value="standard">Standard</option>
          <option value="goal">Goal</option>
          <option value="smart">Smart</option>
          <option value="tracker">Tracker</option>
        </select>
      </div>
      <div v-if="form.setType === 'goal'" class="form-group mb-4">
        <label for="templateId" class="form-label mb-2 block">Template</label>
        <select id="templateId" v-model="form.templateId" class="form-input w-full">
          <option value="">No template</option>
          <option v-for="template in templates" :key="template.id" :value="template.id">
            {{ template.name }}
          </option>
        </select>
      </div>
      <div v-if="form.setType === 'goal'" class="form-group mb-4">
        <label for="csvTargets" class="form-label mb-2 block">Custom CSV targets</label>
        <textarea
          id="csvTargets"
          v-model="csvTargets"
          rows="4"
          class="form-input w-full"
          placeholder="Label,Year,MintMark,Denomination,Country,Material"
        />
      </div>
      <div v-if="form.setType === 'goal'" class="form-group mb-4">
        <label for="targetCompletionDate" class="form-label mb-2 block">Target completion date</label>
        <input id="targetCompletionDate" v-model="form.targetCompletionDate" type="date" class="form-input w-full" />
      </div>
      <SetSmartRuleBuilder
        v-if="form.setType === 'smart'"
        @update="form.smartCriteria = $event"
      />
      <div v-if="form.setType === 'tracker'" class="form-group mb-4">
        <label for="trackerCreationMode" class="form-label mb-2 block">Tracker creation mode</label>
        <select id="trackerCreationMode" v-model="form.trackerCreationMode" class="form-input w-full">
          <option value="manual">Manual</option>
          <option value="dynamic">Dynamic (agentic)</option>
        </select>
      </div>
      <div v-if="form.setType === 'tracker' && form.trackerCreationMode === 'dynamic'" class="form-group mb-4">
        <label for="trackerPrompt" class="form-label mb-2 block">Dynamic builder prompt</label>
        <textarea
          id="trackerPrompt"
          v-model="form.trackerPrompt"
          rows="3"
          maxlength="500"
          class="form-input w-full"
          placeholder="Example: all American wheat pennies by date and mint"
        />
        <p class="mt-2 text-body text-text-secondary">A proposal will be generated for review before any tracker set is created.</p>
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
import { onMounted, reactive, ref, watch } from 'vue'
import { getSetTemplates } from '@/api/client'
import { normalizeCoinSetType } from '@/types'
import type { CoinSetTemplate, CreateCoinSetRequest, SmartCriteriaGroup } from '@/types'
import SetSmartRuleBuilder from '@/components/sets/SetSmartRuleBuilder.vue'

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
  color: props.initialValue?.color ?? '#6b7280',
  setType: props.initialValue?.setType ? normalizeCoinSetType(props.initialValue.setType) : 'standard',
  templateId: props.initialValue?.templateId ?? '',
  targetCompletionDate: props.initialValue?.targetCompletionDate ?? '',
  smartCriteria: props.initialValue?.smartCriteria as SmartCriteriaGroup | undefined,
  trackerCreationMode: props.initialValue?.creationMode ?? 'manual',
  trackerPrompt: props.initialValue?.trackerPrompt ?? '',
})
const templates = ref<CoinSetTemplate[]>([])
const csvTargets = ref('')

onMounted(async () => {
  try {
    const res = await getSetTemplates()
    templates.value = res.data.templates
  } catch {
    templates.value = []
  }
})

watch(() => props.initialValue, (value) => {
  form.name = value?.name ?? ''
  form.description = value?.description ?? ''
  form.color = value?.color ?? '#6b7280'
  form.setType = value?.setType ? normalizeCoinSetType(value.setType) : 'standard'
  form.templateId = value?.templateId ?? ''
  form.targetCompletionDate = value?.targetCompletionDate ?? ''
  form.smartCriteria = value?.smartCriteria as SmartCriteriaGroup | undefined
  form.trackerCreationMode = value?.creationMode ?? 'manual'
  form.trackerPrompt = value?.trackerPrompt ?? ''
})

function submit() {
  const name = form.name.trim()
  if (!name) return
  emit('submit', {
    name,
    description: form.description.trim(),
    color: form.color,
    setType: form.setType,
    templateId: form.templateId || undefined,
    targetCompletionDate: form.targetCompletionDate || undefined,
    smartCriteria: form.smartCriteria ?? undefined,
    creationMode: form.setType === 'tracker' ? form.trackerCreationMode : undefined,
    trackerPrompt: form.setType === 'tracker' && form.trackerCreationMode === 'dynamic'
      ? form.trackerPrompt.trim() || undefined
      : undefined,
  }, csvTargets.value.trim() || undefined)
}
</script>
