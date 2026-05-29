<template>
  <div class="modal-overlay" @click.self="$emit('close')">
    <div class="modal card">
      <h3>Sell Coin</h3>
      <p class="sell-coin-name">{{ coin.name }}</p>

      <div class="form-group">
        <label class="form-label">Sale Price</label>
        <div class="price-input-wrapper">
          <span class="price-prefix">$</span>
          <input
            ref="priceInput"
            v-model="priceStr"
            type="number"
            step="0.01"
            min="0"
            class="form-input price-input"
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

      <div v-if="coin.purchasePrice" class="cost-basis-note">
        <span class="label">Cost basis:</span>
        <span class="value">{{ formatCurrency(coin.purchasePrice) }}</span>
      </div>

      <div v-if="error" class="sell-error">{{ error }}</div>

      <div class="modal-actions">
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

<style scoped>
.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.7);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.modal {
  max-width: 420px;
  width: 90%;
  padding: 2rem;
}

.modal h3 {
  margin-bottom: 0.25rem;
  font-size: 1.1rem;
}

.sell-coin-name {
  color: var(--accent-gold);
  font-size: 0.9rem;
  margin-bottom: 1.25rem;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.form-group {
  margin-bottom: 1rem;
}

.form-label {
  display: block;
  font-size: 0.82rem;
  font-weight: 600;
  margin-bottom: 0.35rem;
  color: var(--text-secondary);
}

.price-input-wrapper {
  position: relative;
  display: flex;
  align-items: center;
}

.price-prefix {
  position: absolute;
  left: 0.75rem;
  color: var(--text-secondary);
  font-size: 0.9rem;
  pointer-events: none;
}

.price-input {
  padding-left: 1.5rem;
}

/* Hide number input spinners */
.price-input::-webkit-inner-spin-button,
.price-input::-webkit-outer-spin-button {
  -webkit-appearance: none;
  margin: 0;
}
.price-input[type='number'] {
  -moz-appearance: textfield;
}

.cost-basis-note {
  display: flex;
  justify-content: space-between;
  font-size: 0.8rem;
  padding: 0.5rem 0.75rem;
  background: var(--bg-elevated);
  border-radius: var(--radius-sm);
  margin-bottom: 1rem;
}

.cost-basis-note .label {
  color: var(--text-secondary);
}

.cost-basis-note .value {
  color: var(--accent-gold);
  font-weight: 600;
}

.sell-error {
  color: #e74c3c;
  font-size: 0.82rem;
  margin-bottom: 0.75rem;
}

.modal-actions {
  display: flex;
  gap: 0.75rem;
  justify-content: flex-end;
  margin-top: 1.5rem;
}
</style>
