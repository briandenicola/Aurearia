<template>
  <div
    class="fixed inset-0 z-[1000] flex items-center justify-center bg-overlay p-4"
    @click.self="handleClose"
    @keydown.esc="handleClose"
  >
    <div
      ref="dialogEl"
      role="dialog"
      :aria-labelledby="titleId"
      aria-modal="true"
      class="card w-full max-w-[420px] !p-6"
      tabindex="-1"
    >
      <div class="mb-4 flex items-center justify-between">
        <h3 :id="titleId" class="m-0 font-display text-lg font-medium text-heading">
          {{ existingReminder ? 'Edit Reminder' : 'Set Reminder' }}
        </h3>
        <button
          class="inline-flex h-8 w-8 items-center justify-center rounded-sm text-text-muted transition-colors hover:bg-white/5 hover:text-text-primary"
          title="Close"
          aria-label="Close"
          @click="handleClose"
        >
          <X :size="18" />
        </button>
      </div>

      <p class="mb-4 truncate text-base text-gold">{{ coinName }}</p>

      <div class="form-group">
        <label :for="dateInputId" class="form-label">Remind Date</label>
        <input
          :id="dateInputId"
          ref="dateInputEl"
          v-model="selectedDate"
          type="date"
          class="form-input"
          :min="today"
          aria-required="true"
          :aria-describedby="validationError ? errorId : undefined"
        />
        <p v-if="validationError" :id="errorId" role="alert" class="mt-1 text-[0.82rem] text-loss">
          {{ validationError }}
        </p>
      </div>

      <p v-if="saveError" role="alert" class="mb-3 text-[0.82rem] text-loss">{{ saveError }}</p>

      <div class="mt-5 flex flex-wrap items-center gap-2">
        <button
          v-if="existingReminder"
          class="btn btn-danger btn-sm mr-auto"
          :disabled="saving"
          @click="handleCancel"
        >
          Remove Reminder
        </button>
        <button class="btn btn-secondary btn-sm ml-auto" @click="handleClose">Close</button>
        <button
          class="btn btn-primary btn-sm"
          :disabled="saving || !selectedDate"
          @click="handleSubmit"
        >
          {{ saving ? 'Saving...' : existingReminder ? 'Save Changes' : 'Set Reminder' }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, nextTick } from 'vue'
import { X } from 'lucide-vue-next'
import type { PurchaseReminder } from '@/types/coin'
import { todayDateString } from '@/composables/usePurchaseReminder'

const props = defineProps<{
  coinId: number
  coinName: string
  existingReminder?: PurchaseReminder | null
  saving?: boolean
  saveError?: string
}>()

const emit = defineEmits<{
  save: [date: string]
  cancel: []
  close: []
}>()

// Stable IDs for ARIA associations
const titleId = `reminder-title-${props.coinId}`
const dateInputId = `reminder-date-${props.coinId}`
const errorId = `reminder-error-${props.coinId}`

const dialogEl = ref<HTMLElement | null>(null)
const dateInputEl = ref<HTMLInputElement | null>(null)

const today = todayDateString()
const selectedDate = ref(props.existingReminder?.remindDate ?? '')
const validationError = ref('')

const saveError = computed(() => props.saveError ?? '')

function validateDate(): boolean {
  if (!selectedDate.value) {
    validationError.value = 'Please select a date.'
    return false
  }
  if (selectedDate.value < today) {
    validationError.value = 'Reminder date must be today or in the future.'
    return false
  }
  validationError.value = ''
  return true
}

function handleSubmit() {
  if (!validateDate()) return
  emit('save', selectedDate.value)
}

function handleCancel() {
  emit('cancel')
}

function handleClose() {
  emit('close')
}

onMounted(async () => {
  await nextTick()
  dateInputEl.value?.focus()
})
</script>
