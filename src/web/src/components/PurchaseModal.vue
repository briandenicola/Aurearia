<template>
  <div class="fixed inset-0 z-[1000] flex items-center justify-center bg-overlay p-4" @click.self="$emit('close')">
    <div class="card w-full max-w-[420px] !p-8">
      <h3 class="mb-1 text-lg font-medium text-heading">Move to Collection</h3>
      <p class="mb-5 truncate text-base text-gold">{{ coin.name }}</p>

      <div class="form-group">
        <label class="form-label">Purchase Price</label>
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
        <label class="form-label">Purchase Date</label>
        <input
          v-model="purchaseDate"
          type="date"
          class="form-input"
        />
      </div>

      <div class="form-group">
        <label class="form-label">Purchased From</label>
        <input
          v-model="purchaseLocation"
          type="text"
          class="form-input"
          placeholder="e.g. VCoins, Heritage Auctions"
        />
      </div>

      <p class="mb-3 text-[0.78rem] text-text-secondary">All fields are optional. You can update these later.</p>

      <div v-if="error" class="mb-3 text-[0.82rem] text-loss">{{ error }}</div>

      <div class="mt-6 flex justify-end gap-3">
        <button class="btn btn-secondary" @click="$emit('close')">Cancel</button>
        <button class="btn btn-primary" :disabled="submitting" @click="handleSubmit">
          {{ submitting ? 'Saving...' : 'Add to Collection' }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import type { Coin } from '@/types'

const props = defineProps<{
  coin: Coin
}>()

const emit = defineEmits<{
  close: []
  confirm: [data: { purchasePrice?: number; purchaseDate?: string; purchaseLocation?: string }]
}>()

const priceStr = ref('')
const purchaseDate = ref(new Date().toISOString().slice(0, 10))
const purchaseLocation = ref('')
const error = ref('')
const submitting = ref(false)
const priceInput = ref<HTMLInputElement>()

onMounted(() => {
  if (props.coin.purchasePrice) {
    priceStr.value = String(props.coin.purchasePrice)
  }
  if (props.coin.purchaseLocation) {
    purchaseLocation.value = props.coin.purchaseLocation
  }
  priceInput.value?.focus()
})

function handleSubmit() {
  error.value = ''

  const data: { purchasePrice?: number; purchaseDate?: string; purchaseLocation?: string } = {}

  if (priceStr.value) {
    const price = parseFloat(priceStr.value)
    if (isNaN(price) || price < 0) {
      error.value = 'Please enter a valid price'
      return
    }
    data.purchasePrice = price
  }

  if (purchaseDate.value) {
    data.purchaseDate = purchaseDate.value
  }

  if (purchaseLocation.value.trim()) {
    data.purchaseLocation = purchaseLocation.value.trim()
  }

  submitting.value = true
  emit('confirm', data)
}
</script>
