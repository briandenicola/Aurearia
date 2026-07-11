<template>
  <div class="fixed inset-0 z-[1000] flex items-center justify-center bg-overlay p-4" @click.self="$emit('close')">
    <div class="card w-full max-w-[420px] !p-8">
      <h3 class="mb-1 text-lg font-medium text-heading">Sell Coin</h3>
      <p class="mb-5 truncate text-base text-gold">{{ coin.name }}</p>

      <div class="form-group">
        <label class="form-label">Sale Price</label>
        <div class="relative flex items-center">
          <span class="pointer-events-none absolute left-3 text-base text-text-secondary">$</span>
          <input
            ref="priceInput"
            v-model="priceStr"
            type="number"
            step="0.01"
            min="0"
            class="form-input pl-6 [appearance:textfield] [&::-webkit-inner-spin-button]:appearance-none [&::-webkit-outer-spin-button]:appearance-none"
            placeholder="0.00"
          />
        </div>
      </div>

      <div class="form-group">
        <label class="form-label">Sold To</label>
        <input
          v-model="soldTo"
          type="text"
          class="form-input"
          placeholder="Buyer name (optional)"
        />
      </div>

      <div v-if="coin.purchasePrice" class="mb-4 flex justify-between rounded-sm bg-surface-secondary px-3 py-2 text-chip">
        <span class="text-text-secondary">Cost basis:</span>
        <span class="font-semibold text-gold">{{ formatCurrency(coin.purchasePrice) }}</span>
      </div>

      <div v-if="error" class="mb-3 text-[0.82rem] text-loss">{{ error }}</div>

      <div class="mt-6 flex justify-end gap-3">
        <button class="btn btn-secondary" @click="$emit('close')">Cancel</button>
        <button class="btn btn-primary" :disabled="submitting" @click="handleSubmit">
          {{ submitting ? 'Saving...' : 'Mark as Sold' }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import type { Coin } from '@/types'
import { formatCurrency } from '@/utils/format'

// Props are type-checked but not referenced directly in script
const _props = defineProps<{
  coin: Coin
}>()

const emit = defineEmits<{
  close: []
  confirm: [soldPrice: number | null, soldTo: string]
}>()

const priceStr = ref('')
const soldTo = ref('')
const error = ref('')
const submitting = ref(false)
const priceInput = ref<HTMLInputElement>()

onMounted(() => {
  priceInput.value?.focus()
})



function handleSubmit() {
  error.value = ''

  let soldPrice: number | null = null
  if (priceStr.value) {
    soldPrice = parseFloat(priceStr.value)
    if (isNaN(soldPrice) || soldPrice < 0) {
      error.value = 'Please enter a valid price'
      return
    }
  }

  submitting.value = true
  emit('confirm', soldPrice, soldTo.value.trim())
}
</script>
